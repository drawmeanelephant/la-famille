// Dependency-free controller tests. The minimal DOM records the nodes the UI
// creates; browser layout is deliberately outside this test's scope.
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

class Element {
  constructor(tag = "div") {
    this.tagName = tag;
    this.children = [];
    this.events = {};
    this.attributes = {};
    this.dataset = {};
    this.style = {};
    this.hidden = false;
    this.disabled = false;
    this.checked = false;
    this.value = "";
    this.classList = { add() {} };
  }
  appendChild(child) { this.children.push(child); }
  setAttribute(name, value) { this.attributes[name] = value; }
  addEventListener(name, fn) { this.events[name] = fn; }
  focus() {}
  set innerHTML(value) { this.html = value; this.children = []; }
  set textContent(value) { this.text = value; this.children = []; }
}

function descendants(el) {
  return [el, ...el.children.flatMap(descendants)];
}

async function harness({ embeddings = false, graphDefault = false } = {}) {
  const elements = new Map();
  const requests = [];
  const document = {
    readyState: "complete",
    getElementById(id) {
      if (!elements.has(id)) elements.set(id, new Element());
      return elements.get(id);
    },
    createElement(tag) { return new Element(tag); },
    addEventListener() {},
  };
  const nodes = [
    { page_id: "a", title: "A", chunk_id: "a#1", key: "1", url: "/a/" },
    { page_id: "c", title: "C", chunk_id: "c#1", key: "2", url: "/c/" },
    { page_id: "b", title: "B", chunk_id: "b#1", key: "3", url: "/b/" },
  ];
  const fetch = async (url, opts) => {
    if (url === "/api/status") {
      return { ok: true, json: async () => ({
        ready: true, embeddings, graph_expansion: graphDefault, graph_edges: 2,
      }) };
    }
    const req = JSON.parse(opts.body);
    requests.push(req);
    const graph = Boolean(req.graph_expansion);
    return { ok: true, json: async () => ({
      status: "answered",
      answer: graph ? "A [1] → C [2] → B [3]" : "Lexical [1]",
      sources: graph ? nodes.map((n) => ({
        title: n.title, key: "[" + n.key + "]", chunk_id: n.chunk_id, url: n.url,
      })) : [],
      paths: graph ? [{ nodes, edges: [["a", "c"], ["c", "b"]] }] : [],
      diagnostics: { retrieval_mode: graph ? "lexical+graph" : "lexical" },
    }) };
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, "ui/app.js"), "utf8"), {
    document, fetch, URL, location: { href: "http://127.0.0.1:8090/" },
    performance: { now: () => 1 }, navigator: {}, setTimeout, clearTimeout,
  });
  // loadStatus has two awaited operations.
  await new Promise((resolve) => setImmediate(resolve));
  const $ = (id) => document.getElementById(id);
  const submit = async () => {
    $("question").value = "How are A and B connected?";
    await $("ask-form").events.submit({ preventDefault() {} });
  };
  const toggle = (enabled) => {
    $("graph-expansion").checked = enabled;
    $("graph-expansion").events.change();
  };
  return { $, submit, toggle, requests, nodes };
}

async function main() {
  const h = await harness();
  await h.submit();
  assert.equal(h.requests[0].graph_expansion, false);
  assert.equal(descendants(h.$("answer-content")).filter((el) => el.className === "ask-paths").length, 0);
  h.toggle(true);
  await h.submit();
  assert.equal(h.requests[1].graph_expansion, true);
  let rendered = descendants(h.$("answer-content"));
  assert.equal(rendered.filter((el) => el.className === "ask-paths").length, 1);
  const route = rendered.find((el) => el.className === "ask-graph-path");
  assert.equal(route.tagName, "ol");
  assert.deepEqual(route.children.map((item) => item.children.at(-1).text), ["A [1]", "C [2]", "B [3]"]);
  assert.equal(rendered.filter((el) => el.className === "path-arrow").length, 2);
  assert.equal(rendered.find((el) => el.className === "ask-paths").attributes["aria-label"], "How I got there");
  assert.equal(h.$("diag-retrieval-mode").text, "lexical+graph");
  assert.equal(h.$("ask-submit").disabled, false);
  h.toggle(false);
  await h.submit();
  assert.equal(h.requests[2].graph_expansion, false);
  assert.equal(descendants(h.$("answer-content")).filter((el) => el.className === "ask-paths").length, 0);

  const hybrid = await harness({ embeddings: true });
  await hybrid.submit();
  assert.equal("graph_expansion" in hybrid.requests[0], false, "untouched checkbox must preserve hybrid default");
  hybrid.toggle(true);
  await hybrid.submit();
  hybrid.toggle(false);
  await hybrid.submit();
  assert.equal(hybrid.requests[1].graph_expansion, true);
  assert.equal(hybrid.requests[2].graph_expansion, false);

  const defaultOn = await harness({ graphDefault: true });
  assert.equal(defaultOn.$("graph-expansion").checked, true);
  defaultOn.nodes[1].title = "<script>not executable</script>";
  defaultOn.nodes[1].url = "javascript:alert(1)";
  await defaultOn.submit();
  rendered = descendants(defaultOn.$("answer-content"));
  const unsafe = rendered.find((el) => el.text === "<script>not executable</script> [2]");
  assert.ok(unsafe);
  assert.equal(unsafe.href, undefined, "graph links must reject script URLs");
  defaultOn.nodes[1].url = "http://[";
  await defaultOn.submit();
  assert.equal(descendants(defaultOn.$("answer-content")).filter((el) => el.className === "ask-paths").length, 1);
  console.log("Ask graph UI: off/on/off, route DOM, hybrid defaults, and safe text passed");
}

main().catch((err) => { console.error(err); process.exitCode = 1; });
