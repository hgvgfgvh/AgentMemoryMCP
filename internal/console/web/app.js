(function () {
  const groupColors = {
    fact: "#6ecf9a",
    episode: "#c9a227",
    tag: "#b07ce8",
    tool: "#f0874d",
    source: "#7eb8da",
  };

  const edgeColors = {
    from_episode: "#6a8caf",
    has_tag: "#b07ce8",
    used_tool: "#f0874d",
    source: "#7eb8da",
    same_correlation: "#e06c75",
    similar: "#5a6a7a",
  };

  const API = {
    graph: () =>
      fetch("/console/api/graph?limit=200").then((r) => {
        if (!r.ok) throw new Error("graph API " + r.status);
        return r.json();
      }),
    search: (q) =>
      fetch("/console/api/search?q=" + encodeURIComponent(q) + "&limit=25").then((r) => {
        if (!r.ok) throw new Error("search API " + r.status);
        return r.json();
      }),
    stats: () =>
      fetch("/console/api/stats").then((r) => {
        if (!r.ok) throw new Error("stats API " + r.status);
        return r.json();
      }),
    mcpRetrieve: (context, queryHint) =>
      fetch("/console/api/mcp_retrieve", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ context: context, query_hint: queryHint || "" }),
      }).then((r) => {
        if (!r.ok) {
          return r.json().then((j) => {
            throw new Error(j.error || "mcp_retrieve API " + r.status);
          });
        }
        return r.json();
      }),
  };

  let graph3d = null;
  let allNodes = [];
  let nodeById = {};
  let highlightIds = new Set();

  const graphEl = document.getElementById("graph");
  const statusEl = document.getElementById("status");
  const statsEl = document.getElementById("stats");
  const graphErrorEl = document.getElementById("graphError");
  const emptyStateEl = document.getElementById("emptyState");
  const detailBody = document.getElementById("detailBody");
  const searchInput = document.getElementById("searchInput");
  const searchResults = document.getElementById("searchResults");

  function check3d() {
    if (typeof ForceGraph3D !== "function") {
      throw new Error(
        "3d-force-graph 未加载。请确认 vendor/three.min.js 与 vendor/3d-force-graph.min.js 可访问，并重新 go build。"
      );
    }
  }

  function showGraphError(msg) {
    graphErrorEl.textContent = msg;
    graphErrorEl.classList.remove("hidden");
  }

  function hideGraphError() {
    graphErrorEl.classList.add("hidden");
  }

  function nodeColor(group, id) {
    if (highlightIds.has(id)) return "#ffffff";
    return groupColors[group] || groupColors.fact;
  }

  function toGraphData(view) {
    nodeById = {};
    allNodes = view.nodes || [];
    const nodes = allNodes.map((n) => {
      const node = {
        id: n.id,
        label: n.label,
        group: n.group,
        title: n.title || n.label,
        data: n.data,
        val: Math.max(2, (n.size || 10) * 0.35),
      };
      nodeById[n.id] = node;
      return node;
    });
    const links = (view.edges || []).map((e, i) => ({
      id: "link-" + i,
      source: e.from,
      target: e.to,
      label: e.label,
      group: e.group,
    }));
    return { nodes, links };
  }

  function initGraph3d() {
    if (graph3d) return graph3d;
    graph3d = ForceGraph3D()(graphEl)
      .backgroundColor("#0f1419")
      .showNavInfo(false)
      .nodeLabel((n) => n.title || n.label || n.id)
      .nodeVal((n) => n.val || 4)
      .nodeColor((n) => nodeColor(n.group, n.id))
      .nodeOpacity(0.92)
      .linkColor((l) => edgeColors[l.group] || "#5a6a7a")
      .linkOpacity(0.45)
      .linkWidth(0.6)
      .linkDirectionalParticles((l) => (l.group === "similar" ? 0 : 1))
      .linkDirectionalParticleWidth(2)
      .linkDirectionalParticleSpeed(0.006)
      .onNodeClick((n) => showDetail(n.id))
      .onBackgroundClick(() => {
        highlightIds.clear();
        graph3d.nodeColor((node) => nodeColor(node.group, node.id));
      });
    return graph3d;
  }

  function renderGraph(view) {
    check3d();
    hideGraphError();
    highlightIds.clear();

    if (!(view.nodes || []).length) {
      emptyStateEl.classList.remove("hidden");
      statusEl.textContent = "无节点可显示（facts.jsonl 为空或路径不对）";
      return;
    }
    emptyStateEl.classList.add("hidden");

    const g = initGraph3d();
    const data = toGraphData(view);
    g.graphData(data);

    const physicsOn = document.getElementById("physicsToggle").checked;
    if (physicsOn) {
      g.resumeAnimation();
    } else {
      g.pauseAnimation();
    }

    g.onEngineStop(() => {
      if (!document.getElementById("physicsToggle").checked) {
        g.pauseAnimation();
      }
    });

    const s = view.stats || {};
    statusEl.textContent =
      "3D 图 · 节点 " +
      (view.nodes?.length || 0) +
      " · 边 " +
      (view.edges?.length || 0) +
      " · 库内事实 " +
      (s.facts ?? "—") +
      "（拖拽旋转 · 滚轮缩放 · 右键平移）";
  }

  function showDetail(nodeId) {
    const raw = allNodes.find((n) => n.id === nodeId) || nodeById[nodeId];
    if (!raw) {
      detailBody.textContent = "（无数据）";
      return;
    }
    const lines = ["id: " + (raw.id || nodeId), "group: " + (raw.group || "")];
    if (raw.label) lines.push("label: " + raw.label);
    if (raw.title) lines.push("\n--- hover ---\n" + raw.title);
    const data = raw.data;
    if (data && typeof data === "object") {
      lines.push("\n--- fields ---\n" + JSON.stringify(data, null, 2));
    }
    detailBody.textContent = lines.join("\n");
  }

  function focusNode(nodeId) {
    if (!graph3d) return;
    const node = nodeById[nodeId];
    if (!node || node.x === undefined) {
      showDetail(nodeId);
      return;
    }
    highlightIds = new Set([nodeId]);
    graph3d.nodeColor((n) => nodeColor(n.group, n.id));

    const dist = 140;
    const r = Math.hypot(node.x, node.y, node.z) || 1;
    const ratio = 1 + dist / r;
    graph3d.cameraPosition(
      { x: node.x * ratio, y: node.y * ratio, z: node.z * ratio },
      node,
      1200
    );
    showDetail(nodeId);
  }

  function clearHighlight() {
    highlightIds.clear();
    if (graph3d) graph3d.nodeColor((n) => nodeColor(n.group, n.id));
    searchResults.classList.add("hidden");
    searchResults.innerHTML = "";
  }

  function resetCamera() {
    if (!graph3d) return;
    graph3d.cameraPosition({ x: 0, y: 0, z: 280 }, { x: 0, y: 0, z: 0 }, 1200);
  }

  async function loadStats() {
    try {
      const s = await API.stats();
      statsEl.innerHTML =
        "<strong>数据目录</strong><br/>" +
        escapeHtml(s.data_dir || "") +
        "<br/><strong>事实条数</strong> " +
        (s.facts_count ?? 0) +
        "<br/><small>" +
        escapeHtml(s.facts_path || "") +
        "</small>" +
        "<br/><br/><strong>3D 操作</strong><br/>左键拖拽旋转<br/>滚轮缩放<br/>右键拖拽平移";
    } catch (e) {
      statsEl.textContent = "统计加载失败: " + e.message;
    }
  }

  async function refresh() {
    statusEl.textContent = "加载 3D 图…";
    hideGraphError();
    try {
      await loadStats();
      check3d();
      const view = await API.graph();
      renderGraph(view);
      setTimeout(resetCamera, 800);
    } catch (e) {
      statusEl.textContent = "加载失败: " + e.message;
      showGraphError(e.message);
      console.error("[memory-console]", e);
    }
  }

  async function runSearch() {
    const q = searchInput.value.trim();
    if (!q) {
      clearHighlight();
      return;
    }
    try {
      const res = await API.search(q);
      const hits = res.hits || [];
      if (hits.length === 0) {
        searchResults.classList.remove("hidden");
        searchResults.innerHTML = '<div class="hit">无匹配节点</div>';
        return;
      }
      searchResults.classList.remove("hidden");
      searchResults.innerHTML = hits
        .map(
          (h) =>
            '<div class="hit" data-id="' +
            escapeAttr(h.node_id) +
            '"><strong>' +
            escapeHtml(h.label || h.node_id) +
            '</strong><div class="meta">' +
            escapeHtml(h.group) +
            " · score " +
            h.score.toFixed(1) +
            "</div></div>"
        )
        .join("");

      searchResults.querySelectorAll(".hit[data-id]").forEach((el) => {
        el.addEventListener("click", () => focusNode(el.getAttribute("data-id")));
      });

      highlightIds = new Set(hits.map((h) => h.node_id));
      if (graph3d) graph3d.nodeColor((n) => nodeColor(n.group, n.id));
      focusNode(hits[0].node_id);
    } catch (e) {
      searchResults.classList.remove("hidden");
      searchResults.innerHTML =
        '<div class="hit">搜索失败: ' + escapeHtml(e.message) + "</div>";
    }
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function escapeAttr(s) {
    return escapeHtml(s).replace(/'/g, "&#39;");
  }

  document.getElementById("refreshBtn").addEventListener("click", refresh);
  document.getElementById("searchBtn").addEventListener("click", runSearch);
  document.getElementById("clearSearchBtn").addEventListener("click", () => {
    searchInput.value = "";
    clearHighlight();
  });
  document.getElementById("resetViewBtn").addEventListener("click", resetCamera);
  searchInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter") runSearch();
  });
  let debounce;
  searchInput.addEventListener("input", () => {
    clearTimeout(debounce);
    debounce = setTimeout(() => {
      if (searchInput.value.trim().length >= 2) runSearch();
    }, 400);
  });
  const retrievePanel = document.getElementById("mcpRetrievePanel");
  const retrieveBody = document.getElementById("mcpRetrieveBody");
  const retrieveContext = document.getElementById("retrieveContext");
  const retrieveQueryHint = document.getElementById("retrieveQueryHint");
  const retrieveOutput = document.getElementById("retrieveOutput");
  const retrieveStatus = document.getElementById("retrieveStatus");

  function syncRetrievePanelHeight() {
    if (!retrievePanel || retrievePanel.classList.contains("collapsed")) {
      document.documentElement.style.setProperty("--retrieve-panel-h", "52px");
      return;
    }
    const h = retrievePanel.offsetHeight;
    document.documentElement.style.setProperty("--retrieve-panel-h", h + "px");
  }

  async function runMCPRetrieve() {
    const ctx = retrieveContext.value.trim();
    if (!ctx) {
      retrieveStatus.textContent = "请填写 context";
      return;
    }
    retrieveStatus.textContent = "检索中…";
    retrieveOutput.classList.add("hidden");
    try {
      const res = await API.mcpRetrieve(ctx, retrieveQueryHint.value.trim());
      const mcp = res.mcp || {};
      const route = res.route;
      let html =
        '<div class="meta-row">' +
        "<span>phase: <strong>" +
        escapeHtml(mcp.phase || "") +
        "</strong></span>" +
        "<span>skipped: " +
        escapeHtml(mcp.skipped || "false") +
        "</span>" +
        "<span>engine: " +
        escapeHtml(res.engine || "") +
        "</span>" +
        "<span>" +
        (res.duration_ms != null ? res.duration_ms + " ms" : "") +
        "</span></div>";
      if (mcp.skip_reason) {
        html += '<p class="meta-row">skip_reason: ' + escapeHtml(mcp.skip_reason) + "</p>";
      }
      if (route) {
        const yes = route.exec_simple_match === "yes";
        html +=
          '<p><span class="route-badge ' +
          (yes ? "yes" : "no") +
          '">exec_simple_match=' +
          escapeHtml(route.exec_simple_match || "?") +
          "</span> " +
          "confidence=" +
          (route.confidence != null ? route.confidence.toFixed(3) : "?") +
          (route.fact_ids && route.fact_ids.length
            ? " · fact_ids=" + escapeHtml(route.fact_ids.join(", "))
            : "") +
          "</p>";
      }
      html += "<h3>【跨会话事实参考】正文</h3><pre>" + escapeHtml(res.hints_body || mcp.hints || "") + "</pre>";
      html += "<h3>MCP 原始 JSON</h3><pre>" + escapeHtml(JSON.stringify(mcp, null, 2)) + "</pre>";
      if (res.note) {
        html += '<p class="meta-row">' + escapeHtml(res.note) + "</p>";
      }
      retrieveOutput.innerHTML = html;
      retrieveOutput.classList.remove("hidden");
      retrieveStatus.textContent = "完成";
      if (route && route.fact_ids && route.fact_ids.length) {
        highlightIds = new Set(route.fact_ids.map((id) => "fact:" + id));
        if (graph3d) graph3d.nodeColor((n) => nodeColor(n.group, n.id));
        const nid = "fact:" + route.fact_ids[0];
        if (nodeById[nid]) focusNode(nid);
      }
    } catch (e) {
      retrieveStatus.textContent = "失败";
      retrieveOutput.innerHTML = '<pre class="err">' + escapeHtml(e.message) + "</pre>";
      retrieveOutput.classList.remove("hidden");
    }
    syncRetrievePanelHeight();
  }

  document.getElementById("retrieveRunBtn").addEventListener("click", runMCPRetrieve);
  document.getElementById("retrieveSampleBtn").addEventListener("click", () => {
    retrieveContext.value =
      "【跨会话事实参考】\n(无)\n\n---\n用户本轮输入:\n列出 WorkSpace 目录下的文件，把清单保存到 WorkSpace/dev_retrieve_test.txt";
    retrieveQueryHint.value = "列出 WorkSpace 目录";
  });
  document.getElementById("toggleRetrievePanel").addEventListener("click", () => {
    retrievePanel.classList.toggle("collapsed");
    const collapsed = retrievePanel.classList.contains("collapsed");
    document.getElementById("toggleRetrievePanel").textContent = collapsed ? "展开" : "收起";
    syncRetrievePanelHeight();
  });

  window.addEventListener("load", syncRetrievePanelHeight);
  if (typeof ResizeObserver !== "undefined" && retrievePanel) {
    new ResizeObserver(syncRetrievePanelHeight).observe(retrievePanel);
  }

  document.getElementById("physicsToggle").addEventListener("change", (e) => {
    if (!graph3d) return;
    if (e.target.checked) graph3d.resumeAnimation();
    else graph3d.pauseAnimation();
  });
  window.addEventListener("resize", () => {
    try {
      if (graph3d) graph3d.width(graphEl.clientWidth).height(graphEl.clientHeight);
    } catch (_) {}
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", refresh);
  } else {
    refresh();
  }
})();
