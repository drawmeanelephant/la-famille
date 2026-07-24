// La Famille Ask This Site — local UI controller.
// All network calls stay loopback-only; this file never injects unauthenticated
// content into the DOM (only text we either wrote or rehydrated from our own
// JSON response).

(function () {
  "use strict";

  const $ = (id) => document.getElementById(id);

  const state = {
    busy: false,
    last: null,
  };

  function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;",
    })[c]);
  }

  function setStatus(label, mode) {
    $("status-label").textContent = label;
    $("status-dot").dataset.status = mode || "ready";
  }

  function showToast(text, ms) {
    const el = $("toast-region");
    el.textContent = text;
    el.hidden = false;
    if (showToast._t) clearTimeout(showToast._t);
    showToast._t = setTimeout(() => { el.hidden = true; }, ms || 2200);
  }

  async function fetchJSON(url, opts) {
    const res = await fetch(url, opts);
    if (!res.ok) {
      const text = await res.text().catch(() => res.statusText);
      throw new Error("HTTP " + res.status + ": " + text);
    }
    return res.json();
  }

  function renderAnswer(payload) {
    const card = $("answer-card");
    const content = $("answer-content");
    const warn = $("answer-warning");
    content.innerHTML = "";

    if (payload.no_answer) {
      warn.hidden = false;
      warn.textContent = payload.no_answer_message ||
        "This site does not provide enough information to answer that question.";
    } else {
      warn.hidden = true;
    }

    if (payload.answer) {
      const p = document.createElement("p");
      p.innerHTML = formatAnswerText(payload.answer);
      content.appendChild(p);
    }

    if (Array.isArray(payload.sources) && payload.sources.length > 0) {
      const heading = document.createElement("h3");
      heading.textContent = "Sources";
      heading.style.fontSize = "14px";
      heading.style.margin = "16px 0 6px";
      content.appendChild(heading);

      payload.sources.forEach((src) => {
        const card = document.createElement("article");
        card.className = "ask-source-card";

        const key = document.createElement("div");
        key.className = "source-key";
        key.textContent = src.key;
        card.appendChild(key);

        const title = document.createElement("div");
        title.className = "source-title";
        title.textContent = src.title || src.chunk_id || "Untitled source";
        card.appendChild(title);

        const heading = document.createElement("div");
        heading.className = "source-heading";
        heading.textContent = src.heading || "";
        card.appendChild(heading);

        if (src.excerpt) {
          const ex = document.createElement("div");
          ex.className = "source-excerpt";
          ex.textContent = src.excerpt;
          card.appendChild(ex);
        }

        const link = document.createElement("a");
        link.className = "source-link";
        if (src.url) {
          link.href = src.url;
          link.textContent = "Open source ↗";
          link.rel = "noopener noreferrer";
          link.target = "_blank";
        } else {
          link.textContent = "Generated URL not available";
          link.classList.add("unavailable");
          link.setAttribute("aria-disabled", "true");
        }
        card.appendChild(link);

        content.appendChild(card);
      });
    }

    card.hidden = false;
    $("empty-state").hidden = true;
  }

  // formatAnswerText converts [1]/[2]-style citations emitted by the model
  // into small inline tags so the reader can match them with the source
  // cards below the answer. We deliberately do NOT parse markdown further
  // to keep the local UI free of third-party libraries.
  function formatAnswerText(text) {
    const safe = escapeHTML(text);
    return safe.replace(/\[(\d{1,3})\]/g, (full, n) => {
      const id = "cite-" + n;
      return '<span class="citation-tag" id="' + id + '">[' + n + ']</span>';
    }).replace(/\n/g, "<br>");
  }

  async function submitQuestion(event) {
    event.preventDefault();
    if (state.busy) return;
    const q = $("question").value.trim();
    if (!q) return;

    state.busy = true;
    $("ask-submit").disabled = true;
    setStatus("Retrieving…", "loading");

    const t0 = performance.now();
    try {
      const payload = await fetchJSON("/api/ask", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question: q }),
      });
      const total = Math.round(performance.now() - t0);
      state.last = payload;
      renderAnswer(payload);

      const diagnostics = payload.diagnostics || {};
      $("diag-retrieve-ms").textContent = formatMs(diagnostics.retrieval_ms);
      $("diag-generate-ms").textContent = formatMs(diagnostics.generation_ms);

      if (payload.status === "no_answer") {
        setStatus("No corroborating sources in corpus", "error");
      } else if (payload.dropped_citations && payload.dropped_citations.length > 0) {
        setStatus("Answer ready — " + payload.dropped_citations.length + " unsupported citation(s) removed", "ready");
      } else {
        setStatus("Answer ready", "ready");
      }
      $("timing-label").textContent = total + " ms total";
    } catch (err) {
      setStatus("Error: " + err.message, "error");
      showToast("Couldn't reach the assistant — is the local server running?");
    } finally {
      state.busy = false;
      $("ask-submit").disabled = false;
    }
  }

  function formatMs(v) {
    if (v == null || isNaN(v)) return "—";
    return Math.round(v) + " ms";
  }

  async function loadStatus() {
    try {
      const data = await fetchJSON("/api/status");
      $("diag-corpus-version").textContent = data.corpus_version || "—";
      $("diag-docs").textContent = data.document_count ?? "—";
      $("diag-chunks").textContent = data.chunk_count ?? "—";
      $("diag-provider").textContent = data.provider || "—";
      $("diag-model").textContent = data.model || "—";
      $("diag-host").textContent = data.bind || "—";
      setStatus(data.ready ? "Ready" : "Unavailable", data.ready ? "ready" : "error");
    } catch (err) {
      setStatus("Status unavailable", "error");
    }
  }

  function copyAnswer(event) {
    event.preventDefault();
    if (!state.last) {
      showToast("Nothing to copy yet.");
      return;
    }
    const lines = [];
    if (state.last.answer) lines.push(state.last.answer);
    if (Array.isArray(state.last.sources)) {
      lines.push("");
      lines.push("Sources:");
      state.last.sources.forEach((s) => {
        const tag = s.key;
        const title = s.title || s.chunk_id;
        const url = s.url ? " — " + s.url : "";
        lines.push("  " + tag + " " + title + url);
      });
    }
    if (state.last.no_answer_message) lines.push("\n" + state.last.no_answer_message);
    const text = lines.join("\n");

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(() => showToast("Copied with citations."));
    } else {
      const ta = document.createElement("textarea");
      ta.value = text;
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand("copy"); showToast("Copied with citations."); }
      finally { document.body.removeChild(ta); }
    }
  }

  function toggleDiagnostics() {
    const drawer = $("diagnostics-drawer");
    const btn = $("diagnostics-toggle");
    const open = drawer.hidden;
    drawer.hidden = !open;
    btn.setAttribute("aria-expanded", String(open));
  }

  function bindKeys() {
    const q = $("question");
    q.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        $("ask-form").requestSubmit ? $("ask-form").requestSubmit() : $("ask-form").dispatchEvent(new Event("submit"));
      } else if (e.key === "Escape") {
        q.value = "";
        q.blur();
      }
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && !$("diagnostics-drawer").hidden) {
        toggleDiagnostics();
      }
    });
  }

  function init() {
    $("ask-form").addEventListener("submit", submitQuestion);
    $("copy-answer").addEventListener("click", copyAnswer);
    $("diagnostics-toggle").addEventListener("click", toggleDiagnostics);
    bindKeys();
    loadStatus();
    $("question").focus();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
