/* Knowledge Graph Explorer — vanilla-JS controller. No external deps.
   Loads ../graph.json, ../meta.json, ../backlinks.json and renders an
   interactive directed graph with search, filter toggles, focus mode,
   and keyboard accessibility. */
(function () {
  'use strict';

  var LARGE_THRESHOLD = 500;
  // Orphan rule: a page is considered orphan when it has zero inbound
  // links. The homepage (page id "index") is deliberately excluded so a
  // freshly-seeded site that links out from index does not flag index as
  // orphan. The rule is documented once here and surfaced via the
  // toggle "Orphans" filter for users.
  var HOMEPAGE_ORPHAN_EXEMPT = 'index';
  var NS = 'http://www.w3.org/2000/svg';

  var state = {
    graph: null,
    backlinks: null,
    meta: null,
    nodes: [],
    nodeIndex: {},
    edges: [],
    selected: null,
    palette: { w: 800, h: 600 },
    filter: {
      render: true, raw: true, stub: true, orphan: true,
      query: '', focus: readQueryParam('focus') === 'true'
    }
  };

  var els = {};

  function $(id) { return document.getElementById(id); }

  function init() {
    els.searchInput = $('kgx-search-input');
    els.suggestions = $('kgx-suggestions');
    els.toggleRender = $('kgx-toggle-render');
    els.toggleRaw = $('kgx-toggle-raw');
    els.toggleStub = $('kgx-toggle-stub');
    els.toggleOrphan = $('kgx-toggle-orphan');
    els.toggleFocus = $('kgx-toggle-focus-mode');
    els.toggleHelp = $('kgx-toggle-help');
    els.help = $('kgx-help');
    els.canvasWrap = $('kgx-canvas-wrap');
    els.svg = $('kgx-svg');
    els.edgesLayer = $('kgx-edges');
    els.nodesLayer = $('kgx-nodes');
    els.detail = $('kgx-detail');
    els.detailClose = $('kgx-detail-close');
    els.main = $('kgx-main');
    els.loading = $('kgx-loading');
    els.error = $('kgx-error');
    els.empty = $('kgx-empty');
    els.noResults = $('kgx-no-results');
    els.status = $('kgx-status');
    els.counts = $('kgx-counts');
    bindEvents();
    loadData();
  }

  function assetURL(rel) { return rel; }

  function loadData() {
    var urls = [
      assetURL('../graph.json'),
      assetURL('../backlinks.json'),
      assetURL('../meta.json')
    ];
    show(els.loading); hide(els.svg); hide(els.detail);
    Promise.all(urls.map(fetchJSON))
      .then(function (parts) {
        state.graph = parts[0];
        state.backlinks = parts[1];
        state.meta = parts[2] || {};
        buildIndex();
        afterLoad();
      })
      .catch(function (err) { renderError(err); });
  }

  function fetchJSON(url) {
    return fetch(url, { credentials: 'same-origin', cache: 'no-cache' })
      .then(function (r) {
        if (!r.ok) throw new Error('Failed to fetch ' + url + ' (' + r.status + ')');
        return r.json();
      });
  }

  function buildIndex() {
    var nodesRaw = (state.graph && state.graph.nodes) || {};
    state.nodes = [];
    state.nodeIndex = {};
    var keys = Object.keys(nodesRaw).sort();
    for (var i = 0; i < keys.length; i++) {
      var id = keys[i];
      var n = nodesRaw[id] || {};
      var md = state.meta[id] || {};
      var isRender = n.render !== false;
      var isStub = (n.type === 'stub') || n.missing === true;
      var tags = (md && md.tags) || [];
      var categories = (md && md.categories) || [];
      var inbound = (state.backlinks && state.backlinks[id]) || [];
      var isOrphan = inbound.length === 0 && id !== HOMEPAGE_ORPHAN_EXEMPT;
      state.nodes.push({
        id: id,
        type: n.type || 'page',
        render: isRender,
        missing: !!isStub,
        raw: !isRender,
        stub: !!isStub,
        orphan: isOrphan,
        title: md.title || deriveTitle(id),
        tags: tags,
        categories: categories,
        author: md.author || '',
        date: md.date || '',
        word_count: md.word_count || 0,
        url: isRender ? deriveUrl(id) : '',
        inbound: inbound,
        outbound: [],
        x: 0, y: 0, vx: 0, vy: 0
      });
      state.nodeIndex[id] = state.nodes[state.nodes.length - 1];
    }
    var edges = (state.graph && state.graph.edges) || [];
    state.edges = [];
    for (var j = 0; j < edges.length; j++) {
      var e = edges[j];
      if (!e || e.length < 2) continue;
      state.edges.push([e[0], e[1]]);
      var node = state.nodeIndex[e[1]];
      if (node) node.outbound.push(e[0]);
    }
  }

  function deriveTitle(id) {
    var base = id.split('/').pop().replace(/\.md$/, '');
    return base.replace(/[-_]+/g, ' ').replace(/\b\w/g, function (c) { return c.toUpperCase(); });
  }

  // Compute a root-relative public URL from the page id only. Slug-aware
  // linking would have required extending meta.json; we keep the JSON
  // contracts stable by computing the URL from the id. Sites using
  // front-matter slug overrides show the unslugged link in the detail
  // panel; this is documented as a known limitation in publishing.md.
  function deriveUrl(id) {
    if (!id) return '';
    if (id === HOMEPAGE_ORPHAN_EXEMPT) return '/';
    return '/' + id + '/';
  }

  function afterLoad() {
    hide(els.loading);
    if (!state.nodes.length) { show(els.empty); return; }
    hide(els.empty);
    state.selected = readQueryParam('node') || null;
    applyFilter();
    renderControls();
    reseedLayout();
    if (state.nodes.length >= LARGE_THRESHOLD) {
      focusSearch();
      updateStatus('Large site (' + state.nodes.length + ' nodes). Search to explore.');
    } else {
      startLayout();
    }
    renderGraph();
  }

  function focusSearch() {
    if (els.searchInput) els.searchInput.focus();
  }

  function applyFilter() {
    var q = (state.filter.query || '').trim().toLowerCase();
    var parts = q ? q.split(/\s+/) : [];
    var f = state.filter;
    for (var i = 0; i < state.nodes.length; i++) {
      var n = state.nodes[i];
      var visible = true;
      if (!f.render && n.render && !n.stub) visible = false;
      if (!f.raw && n.raw) visible = false;
      if (!f.stub && n.stub) visible = false;
      if (!f.orphan && n.orphan) visible = false;
      if (q && !matchQuery(n, parts)) visible = false;
      n._visible = visible;
    }
    applyFocusMode();
  }

  // Focus mode composes with the existing filters: when on AND a node is
  // selected, hide every node that is not the selected node or one of its
  // 1-degree inbound/outbound neighbors. The detail panel + URL deep-link
  // remain unchanged. If nothing is selected the focus toggle is disabled.
  function applyFocusMode() {
    if (!state.filter.focus || !state.selected) return;
    var sel = state.nodeIndex[state.selected];
    if (!sel) return;
    var allowed = {};
    allowed[sel.id] = true;
    for (var i = 0; i < sel.inbound.length; i++) allowed[sel.inbound[i]] = true;
    for (var j = 0; j < sel.outbound.length; j++) allowed[sel.outbound[j]] = true;
    for (var k = 0; k < state.nodes.length; k++) {
      var n = state.nodes[k];
      if (n._visible && !allowed[n.id]) n._visible = false;
    }
  }

  function matchQuery(n, parts) {
    for (var i = 0; i < parts.length; i++) {
      var p = parts[i];
      var idx = p.indexOf(':');
      if (idx > 0) {
        var kind = p.slice(0, idx).toLowerCase();
        var value = p.slice(idx + 1).toLowerCase();
        if (!value) return false;
        if (kind === 'tag') { if (!containsCI(n.tags, value)) return false; continue; }
        if (kind === 'category' || kind === 'cat') { if (!containsCI(n.categories, value)) return false; continue; }
        if (kind === 'author') { if ((n.author || '').toLowerCase() !== value) return false; continue; }
        if (kind === 'id') { if ((n.id || '').toLowerCase().indexOf(value) === -1) return false; continue; }
      }
      var hay = (n.title + ' ' + n.id + ' ' + (n.tags || []).join(' ') + ' ' + (n.categories || []).join(' ') + ' ' + (n.author || '')).toLowerCase();
      if (hay.indexOf(p) === -1) return false;
    }
    return true;
  }

  function containsCI(arr, needle) {
    for (var i = 0; i < arr.length; i++) {
      if (String(arr[i]).toLowerCase() === needle) return true;
    }
    return false;
  }

  /* -- Layout: deterministic-seeded force-directed simulation. -- */
  function reseedLayout() {
    var n = state.nodes.length;
    for (var i = 0; i < n; i++) {
      var node = state.nodes[i];
      var theta = (i + 0.5) / Math.max(1, n) * 2 * Math.PI;
      var r = 220 + 80 * hash(i);
      node.x = state.palette.w / 2 + Math.cos(theta) * r;
      node.y = state.palette.h / 2 + Math.sin(theta) * r;
      node.vx = 0;
      node.vy = 0;
    }
  }

  function hash(i) {
    var x = Math.sin(i * 9999.131) * 43758.5453;
    return x - Math.floor(x);
  }

  function startLayout() {
    if (state._layoutRaf) return;
    var ticks = 0;
    var nodeTotal = state.nodes.length;
    var desired = 90;
    function tick() {
      ticks++;
      var W = state.palette.w, H = state.palette.h;
      for (var i = 0; i < nodeTotal; i++) {
        var n = state.nodes[i];
        var fx = 0, fy = 0;
        for (var j = 0; j < nodeTotal; j++) {
          if (i === j) continue;
          var m = state.nodes[j];
          var dx = n.x - m.x, dy = n.y - m.y;
          var d2 = dx * dx + dy * dy;
          if (d2 < 0.001) d2 = 0.001;
          var f = 1800 / d2;
          fx += dx * f;
          fy += dy * f;
        }
        fx += (W / 2 - n.x) * 0.012;
        fy += (H / 2 - n.y) * 0.012;
        n.vx = (n.vx + fx * 0.0008) * 0.85;
        n.vy = (n.vy + fy * 0.0008) * 0.85;
      }
      var edges = state.edges;
      for (var k = 0; k < edges.length; k++) {
        var e = edges[k];
        var a = state.nodeIndex[e[0]];
        var b = state.nodeIndex[e[1]];
        if (!a || !b) continue;
        var ddx = b.x - a.x, ddy = b.y - a.y;
        var d = Math.sqrt(ddx * ddx + ddy * ddy);
        if (d < 0.001) d = 0.001;
        var diff = (d - desired) / d;
        var force = diff * 0.06;
        var fx2 = ddx * force, fy2 = ddy * force;
        a.vx += fx2; a.vy += fy2;
        b.vx -= fx2; b.vy -= fy2;
      }
      for (var l = 0; l < nodeTotal; l++) {
        var nd = state.nodes[l];
        nd.x += nd.vx;
        nd.y += nd.vy;
        if (nd.x < 8) { nd.x = 8; nd.vx *= -0.5; }
        if (nd.y < 8) { nd.y = 8; nd.vy *= -0.5; }
        if (nd.x > W - 8) { nd.x = W - 8; nd.vx *= -0.5; }
        if (nd.y > H - 8) { nd.y = H - 8; nd.vy *= -0.5; }
      }
      if (ticks % 2 === 0) updatePositions();
      if (ticks < 400) {
        state._layoutRaf = requestAnimationFrame(tick);
      } else {
        state._layoutRaf = null;
        updatePositions();
      }
    }
    state._layoutRaf = requestAnimationFrame(tick);
  }

  function stopLayout() {
    if (state._layoutRaf) { cancelAnimationFrame(state._layoutRaf); state._layoutRaf = null; }
  }

  /* -- Rendering -- */
  function renderControls() {
    if (els.counts) {
      var c = countVisible();
      var suffix = state.filter.focus ? ' (focus mode)' : '';
      els.counts.textContent = c + ' / ' + state.nodes.length + ' nodes · ' + state.edges.length + ' edges' + suffix;
    }
    setAria(els.toggleRender, !!state.filter.render);
    setAria(els.toggleRaw, !!state.filter.raw);
    setAria(els.toggleStub, !!state.filter.stub);
    setAria(els.toggleOrphan, !!state.filter.orphan);
    if (els.toggleFocus) {
      setAria(els.toggleFocus, !!state.filter.focus);
      els.toggleFocus.disabled = !state.selected;
    }
  }

  function countVisible() {
    var n = 0;
    for (var i = 0; i < state.nodes.length; i++) if (state.nodes[i]._visible) n++;
    return n;
  }

  function setAria(el, p) { if (el) el.setAttribute('aria-pressed', p ? 'true' : 'false'); }

  function renderGraph() {
    show(els.svg);
    if (countVisible() === 0) { show(els.noResults); els.noResults.textContent = 'No pages match the current filters.'; }
    else { hide(els.noResults); }
    drawEdges();
    drawNodes();
    if (state.selected && state.nodeIndex[state.selected]) {
      renderDetail(state.nodeIndex[state.selected]);
    } else {
      hide(els.detail);
    }
  }

  function drawEdges() {
    if (!els.edgesLayer) return;
    while (els.edgesLayer.firstChild) els.edgesLayer.removeChild(els.edgesLayer.firstChild);
    var es = state.edges;
    for (var i = 0; i < es.length; i++) {
      var e = es[i];
      var a = state.nodeIndex[e[0]];
      var b = state.nodeIndex[e[1]];
      if (!a || !b) continue;
      if (!a._visible || !b._visible) continue;
      var line = document.createElementNS(NS, 'line');
      line.setAttribute('class', 'kgx-edge' + (b.raw ? ' to-raw' : '') + (b.stub ? ' to-stub' : ''));
      line.setAttribute('data-src', a.id);
      line.setAttribute('data-tgt', b.id);
      line.setAttribute('x1', a.x);
      line.setAttribute('y1', a.y);
      line.setAttribute('x2', b.x);
      line.setAttribute('y2', b.y);
      line.setAttribute('focusable', 'false');
      line.setAttribute('aria-hidden', 'true');
      els.edgesLayer.appendChild(line);
    }
  }

  function drawNodes() {
    if (!els.nodesLayer) return;
    while (els.nodesLayer.firstChild) els.nodesLayer.removeChild(els.nodesLayer.firstChild);
    var ns = state.nodes;
    for (var i = 0; i < ns.length; i++) {
      var n = ns[i];
      if (!n._visible) continue;
      var g = document.createElementNS(NS, 'g');
      var klass = 'kgx-node';
      if (n.render && !n.stub) klass += ' render';
      if (n.raw) klass += ' raw';
      if (n.stub) klass += ' stub';
      if (n.orphan) klass += ' orphan';
      if (state.selected === n.id) klass += ' selected';
      g.setAttribute('class', klass);
      g.setAttribute('data-id', n.id);
      g.setAttribute('tabindex', '0');
      g.setAttribute('role', 'button');
      g.setAttribute('aria-label', describeNode(n));
      var c = document.createElementNS(NS, 'circle');
      c.setAttribute('r', n.stub ? 6 : (n.raw ? 7 : 9));
      c.setAttribute('cx', n.x);
      c.setAttribute('cy', n.y);
      c.setAttribute('focusable', 'false');
      c.setAttribute('aria-hidden', 'true');
      g.appendChild(c);
      var t = document.createElementNS(NS, 'text');
      t.setAttribute('x', n.x + 12);
      t.setAttribute('y', n.y + 4);
      t.setAttribute('focusable', 'false');
      t.setAttribute('aria-hidden', 'true');
      t.textContent = truncate(n.title || n.id, 28);
      g.appendChild(t);
      g.addEventListener('click', onNodeActivate);
      g.addEventListener('keydown', onNodeKey);
      els.nodesLayer.appendChild(g);
    }
  }

  function describeNode(n) {
    var label = (n.title || n.id) + ' — ';
    if (n.stub) label += 'missing-stub node';
    else if (n.raw) label += 'unrendered raw markdown';
    else if (n.orphan) label += 'orphan page';
    else label += 'rendered page';
    return label;
  }

  function truncate(s, n) {
    if (!s) return '';
    return s.length > n ? s.slice(0, n - 1) + '…' : s;
  }

  function updatePositions() {
    if (!els.nodesLayer) return;
    var circles = els.nodesLayer.querySelectorAll('g.kgx-node');
    for (var i = 0; i < circles.length; i++) {
      var g = circles[i];
      var id = g.getAttribute('data-id');
      var n = state.nodeIndex[id];
      if (!n) continue;
      var c = g.querySelector('circle');
      var t = g.querySelector('text');
      if (c) { c.setAttribute('cx', n.x); c.setAttribute('cy', n.y); }
      if (t) { t.setAttribute('x', n.x + 12); t.setAttribute('y', n.y + 4); }
    }
    if (!els.edgesLayer) return;
    var lines = els.edgesLayer.childNodes;
    for (var k = 0; k < lines.length; k++) {
      var line = lines[k];
      var sa = state.nodeIndex[line.getAttribute('data-src')];
      var sb = state.nodeIndex[line.getAttribute('data-tgt')];
      if (!sa || !sb || !sa._visible || !sb._visible) {
        line.setAttribute('display', 'none');
        continue;
      }
      line.removeAttribute('display');
      line.setAttribute('x1', sa.x);
      line.setAttribute('y1', sa.y);
      line.setAttribute('x2', sb.x);
      line.setAttribute('y2', sb.y);
    }
  }

  function renderDetail(n) {
    if (!n) { hide(els.detail); state.selected = null; updateUrl(); return; }
    show(els.detail);
    els.main.classList.add('with-detail');
    var html = '';
    html += '<header>';
    html += '<h2>' + escape(n.title || n.id) + '</h2>';
    html += '<div class="kgx-id">' + escape(n.id) + '</div>';
    html += '<div class="kgx-tags">';
    (n.tags || []).forEach(function (t) { html += '<span class="tag">#' + escape(t) + '</span>'; });
    (n.categories || []).forEach(function (c) { html += '<span class="tag" style="background:rgba(201,123,255,.12);border-color:rgba(201,123,255,.4)">category:' + escape(c) + '</span>'; });
    html += '</div></header>';
    html += '<dl>';
    if (n.url) html += '<dt>URL</dt><dd><a href="' + escape(n.url) + '">' + escape(n.url) + '</a></dd>';
    else html += '<dt>URL</dt><dd><span aria-label="Unrendered page — no HTML link">' + escape(n.raw ? 'raw markdown only' : 'missing stub') + '</span></dd>';
    html += '<dt>Type</dt><dd>' + escape(describeNodeShort(n)) + '</dd>';
    if (n.author) html += '<dt>Author</dt><dd>' + escape(n.author) + '</dd>';
    if (n.date) html += '<dt>Date</dt><dd>' + escape(n.date) + '</dd>';
    if (n.word_count) html += '<dt>Words</dt><dd>' + escape(String(n.word_count)) + '</dd>';
    html += '</dl>';
    html += '<section class="kgx-links">';
    html += '<h3>Outbound (' + n.outbound.length + ')</h3>';
    if (n.outbound.length) {
      html += '<ul>';
      n.outbound.slice().sort().forEach(function (id) {
        var o = state.nodeIndex[id];
        if (!o) return;
        html += '<li><span class="badge">' + (o.stub ? 'stub' : (o.raw ? 'raw' : 'page')) + '</span><a href="#" data-node-link="' + escape(id) + '">' + escape(o.title || id) + '</a></li>';
      });
      html += '</ul>';
    } else { html += '<p>No outbound links.</p>'; }
    html += '<h3>Inbound (' + n.inbound.length + ')</h3>';
    if (n.inbound.length) {
      html += '<ul>';
      n.inbound.slice().sort().forEach(function (id) {
        var o = state.nodeIndex[id];
        if (!o) return;
        html += '<li><span class="badge">' + (o.stub ? 'stub' : (o.raw ? 'raw' : 'page')) + '</span><a href="#" data-node-link="' + escape(id) + '">' + escape(o.title || id) + '</a></li>';
      });
      html += '</ul>';
    } else { html += '<p>No inbound links.</p>'; }
    html += '</section>';
    els.detail.innerHTML = html;
    var links = els.detail.querySelectorAll('[data-node-link]');
    for (var i = 0; i < links.length; i++) {
      links[i].addEventListener('click', function (e) {
        e.preventDefault();
        var id = this.getAttribute('data-node-link');
        selectNode(id, true);
      });
    }
  }

  function describeNodeShort(n) {
    if (n.stub) return 'stub (missing target)';
    if (n.raw) return 'raw markdown (render: false)';
    if (n.orphan) return 'orphan rendered page';
    return 'rendered page';
  }

  function escape(s) {
    return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  /* -- Events -- */
  function bindEvents() {
    if (els.searchInput) {
      els.searchInput.addEventListener('input', onSearchInput);
      els.searchInput.addEventListener('keydown', onSearchKey);
    }
    var btns = [els.toggleRender, els.toggleRaw, els.toggleStub, els.toggleOrphan];
    btns.forEach(function (b) {
      if (!b) return;
      b.addEventListener('click', function () {
        var key = this.getAttribute('data-filter');
        state.filter[key] = !state.filter[key];
        applyFilter();
        renderControls();
        renderGraph();
      });
    });
    if (els.toggleHelp) {
      els.toggleHelp.addEventListener('click', function () {
        setAria(this, this.getAttribute('aria-pressed') !== 'true');
        if (els.help) {
          if (els.help.hasAttribute('hidden')) els.help.removeAttribute('hidden');
          else els.help.setAttribute('hidden', '');
        }
      });
    }
    if (els.toggleFocus) {
      els.toggleFocus.addEventListener('click', function () {
        if (!state.selected) return;
        state.filter.focus = !state.filter.focus;
        applyFilter();
        renderControls();
        renderGraph();
        updateUrl();
      });
    }
    if (els.detailClose) els.detailClose.addEventListener('click', function () {
      hide(els.detail); els.main.classList.remove('with-detail'); state.selected = null; updateUrl();
    });
    window.addEventListener('resize', onResize);
    window.addEventListener('popstate', function () {
      state.selected = readQueryParam('node'); selectNode(state.selected, false);
    });
  }

  function onSearchInput(e) {
    state.filter.query = e.target.value;
    if (state.filter.query && state.filter.focus) {
      state.filter.focus = false;
    }
    applyFilter();
    renderControls();
    showSuggestions();
    renderGraph();
    updateUrl();
  }

  function onSearchKey(e) {
    if (e.key === 'Enter') {
      var first = null;
      for (var i = 0; i < state.nodes.length; i++) if (state.nodes[i]._visible) { first = state.nodes[i]; break; }
      if (first) { selectNode(first.id, true); hideSuggestions(); e.preventDefault(); }
    } else if (e.key === 'Escape') {
      state.filter.query = '';
      els.searchInput.value = '';
      applyFilter(); renderControls(); renderGraph();
    }
  }

  function showSuggestions() {
    if (!els.suggestions) return;
    var matches = [];
    for (var i = 0; i < state.nodes.length; i++) if (state.nodes[i]._visible) matches.push(state.nodes[i]);
    matches = matches.slice(0, 20);
    if (!matches.length) { els.suggestions.classList.remove('open'); els.suggestions.innerHTML = ''; return; }
    var html = '';
    for (var k = 0; k < matches.length; k++) {
      var n = matches[k];
      html += '<button class="kgx-suggestion" data-suggest="' + escape(n.id) + '">' + escape(n.title || n.id) + '<div class="meta">' + escape(n.id) + '</div></button>';
    }
    els.suggestions.innerHTML = html;
    els.suggestions.classList.add('open');
    var btns = els.suggestions.querySelectorAll('[data-suggest]');
    for (var j = 0; j < btns.length; j++) {
      btns[j].addEventListener('click', function () {
        selectNode(this.getAttribute('data-suggest'), true);
        hideSuggestions();
      });
    }
  }

  function hideSuggestions() { els.suggestions && els.suggestions.classList.remove('open'); }

  function onNodeActivate(e) {
    var id = e.currentTarget.getAttribute('data-id');
    selectNode(id, true);
  }
  function onNodeKey(e) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      selectNode(e.currentTarget.getAttribute('data-id'), true);
    }
  }

  function selectNode(id, push) {
    state.selected = id;
    if (state.filter.focus) {
      applyFilter();
      renderControls();
      renderGraph();
    } else {
      // Even when focus mode is off, redrawing the SVG highlights the new
      // selected node via CSS .selected.
      drawNodes();
      if (els.edgesLayer) redrawEdgesAll();
    }
    updateUrl();
    var n = state.nodeIndex[id];
    if (!n) { hide(els.detail); els.main.classList.remove('with-detail'); return; }
    renderDetail(n);
  }

  function redrawEdgesAll() {
    if (!els.edgesLayer) return;
    var lines = els.edgesLayer.childNodes;
    for (var i = 0; i < lines.length; i++) {
      var line = lines[i];
      var sa = state.nodeIndex[line.getAttribute('data-src')];
      var sb = state.nodeIndex[line.getAttribute('data-tgt')];
      if (!sa || !sb || !sa._visible || !sb._visible) {
        line.setAttribute('display', 'none');
        continue;
      }
      line.removeAttribute('display');
    }
  }

  function readQueryParam(name) {
    var u = new URLSearchParams(window.location.search);
    return u.get(name);
  }
  function updateUrl() {
    var u = new URLSearchParams(window.location.search);
    if (state.selected) u.set('node', state.selected); else u.delete('node');
    if (state.filter.focus) u.set('focus', 'true'); else u.delete('focus');
    var next = u.toString();
    var newUrl = window.location.pathname + (next ? '?' + next : '');
    if (newUrl !== window.location.pathname + window.location.search) {
      window.history.replaceState({}, '', newUrl);
    }
  }

  function onResize() {
    if (!els.canvasWrap) return;
    var r = els.canvasWrap.getBoundingClientRect();
    if (r.width < 1 || r.height < 1) return;
    state.palette.w = Math.max(400, r.width);
    state.palette.h = Math.max(300, r.height);
    if (els.svg) els.svg.setAttribute('viewBox', '0 0 ' + state.palette.w + ' ' + state.palette.h);
    updatePositions();
  }

  function renderError(err) {
    hide(els.loading);
    show(els.error);
    els.error.textContent = 'Error loading graph: ' + (err && err.message ? err.message : String(err));
  }

  function show(el) { if (el) el.removeAttribute('hidden'); }
  function hide(el) { if (el) el.setAttribute('hidden', ''); }
  function updateStatus(text) { if (els.status) els.status.textContent = text || ''; }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else { init(); }
})();
