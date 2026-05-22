(function () {
  function checkVis() {
    if (typeof vis === "undefined" || !vis.Network || !vis.DataSet) {
      throw new Error(
        "vis-network 未加载。请确认访问 /console/vendor/vis-network.min.js 可打开，并重新 go build memory-mcp.exe。"
      );
    }
  }

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
  };

  const groupColors = {
    fact: { background: "#2d4a3e", border: "#6ecf9a", highlight: { background: "#3d6a52", border: "#9ef0b8" } },
    episode: { background: "#4a4020", border: "#c9a227", highlight: { background: "#6a5830", border: "#e8c84a" } },
    tag: { background: "#3d2d4a", border: "#b07ce8", highlight: { background: "#5a3d6a", border: "#d4a8ff" } },
    tool: { background: "#4a3020", border: "#f0874d", highlight: { background: "#6a4530", border: "#ffb088" } },
    source: { background: "#243848", border: "#7eb8da", highlight: { background: "#355868", border: "#a8d4f0" } },
  };

  const edgeColors = {
    from_episode: "#6a8caf",
    has_tag: "#b07ce8",
    used_tool: "#f0874d",
    source: "#7eb8da",
    same_correlation: "#e06c75",
    similar: "#5a6a7a",
  };

  let network = null;
  let nodesData = null;
  let edgesData = null;
  let allNodes = [];

  const graphEl = document.getElementById("graph");
  const statusEl = document.getElementById("status");
  const statsEl = document.getElementById("stats");
  const graphErrorEl = document.getElementById("graphError");
  const emptyStateEl = document.getElementById("emptyState");

  function showGraphError(msg) {
    graphErrorEl.textContent = msg;
    graphErrorEl.classList.remove("hidden");
  }

  function hideGraphError() {
    graphErrorEl.classList.add("hidden");
  }
  const detailBody = document.getElementById("detailBody");
  const searchInput = document.getElementById("searchInput");
  const searchResults = document.getElementById("searchResults");

  function visNodes(raw) {
    return raw.map((n) => ({
      id: n.id,
      label: n.label,
      title: n.title || n.label,
      group: n.group,
      size: n.size || 12,
      font: { color: "#e7ecf3", size: 11, face: "Segoe UI" },
      color: groupColors[n.group] || groupColors.fact,
      data: n.data,
    }));
  }

  function visEdges(raw) {
    return raw.map((e) => ({
      from: e.from,
      to: e.to,
      title: e.label,
      arrows: "to",
      color: { color: edgeColors[e.group] || "#5a6a7a", highlight: "#9ecfff" },
      width: e.group === "similar" ? 1 : 1.5,
      dashes: e.group === "similar",
      smooth: { type: "continuous" },
    }));
  }

  function renderGraph(view) {
    checkVis();
    hideGraphError();
    allNodes = view.nodes || [];
    if (allNodes.length === 0) {
      emptyStateEl.classList.remove("hidden");
      statusEl.textContent = "无节点可显示（facts.jsonl 为空或路径不对）";
      return;
    }
    emptyStateEl.classList.add("hidden");
    nodesData = new vis.DataSet(visNodes(allNodes));
    edgesData = new vis.DataSet(visEdges(view.edges || []));

    const options = {
      nodes: { shape: "dot", scaling: { min: 8, max: 28 } },
      edges: { font: { size: 0 } },
      physics: {
        enabled: document.getElementById("physicsToggle").checked,
        barnesHut: { gravitationalConstant: -4200, springLength: 120, damping: 0.12 },
        stabilization: { iterations: 120 },
      },
      interaction: { hover: true, tooltipDelay: 120, navigationButtons: true },
    };

    if (network) {
      network.setData({ nodes: nodesData, edges: edgesData });
    } else {
      network = new vis.Network(graphEl, { nodes: nodesData, edges: edgesData }, options);
      network.on("click", (params) => {
        if (params.nodes.length === 1) showDetail(params.nodes[0]);
      });
      network.once("stabilizationIterationsDone", () => {
        try {
          network.fit({ animation: { duration: 400 } });
        } catch (_) {}
      });
    }
    setTimeout(() => {
      try {
        if (network) network.redraw();
        if (network) network.fit({ animation: false });
      } catch (_) {}
    }, 600);

    const s = view.stats || {};
    statusEl.textContent =
      "图内节点 " + (view.nodes?.length || 0) + " · 边 " + (view.edges?.length || 0) +
      " · 库内事实 " + (s.facts ?? "—");
  }

  async function loadStats() {
    try {
      const s = await API.stats();
      statsEl.innerHTML =
        "<strong>数据目录</strong><br/>" + escapeHtml(s.data_dir || "") +
        "<br/><strong>事实条数</strong> " + (s.facts_count ?? 0) +
        "<br/><small>" + escapeHtml(s.facts_path || "") + "</small>";
    } catch (e) {
      statsEl.textContent = "统计加载失败: " + e.message;
    }
  }

  async function refresh() {
    statusEl.textContent = "加载图…";
    hideGraphError();
    try {
      await loadStats();
      checkVis();
      const view = await API.graph();
      renderGraph(view);
    } catch (e) {
      statusEl.textContent = "加载失败: " + e.message;
      showGraphError(e.message);
      console.error("[memory-console]", e);
    }
  }

  function showDetail(nodeId) {
    const raw = allNodes.find((n) => n.id === nodeId);
    if (!raw) {
      detailBody.textContent = "（无数据）";
      return;
    }
    const lines = [
      "id: " + raw.id,
      "group: " + raw.group,
      "label: " + raw.label,
    ];
    if (raw.title) lines.push("\n--- hover ---\n" + raw.title);
    if (raw.data && typeof raw.data === "object") {
      lines.push("\n--- fields ---\n" + JSON.stringify(raw.data, null, 2));
    }
    detailBody.textContent = lines.join("\n");
  }

  function focusNode(nodeId) {
    if (!network || !nodesData) return;
    network.selectNodes([nodeId]);
    network.focus(nodeId, { scale: 1.2, animation: { duration: 400, easingFunction: "easeInOutQuad" } });
    showDetail(nodeId);
    highlightNodes([nodeId]);
  }

  function highlightNodes(ids) {
    const updates = allNodes.map((n) => {
      const on = ids.includes(n.id);
      const base = groupColors[n.group] || groupColors.fact;
      return {
        id: n.id,
        borderWidth: on ? 4 : 1,
        color: on ? base.highlight : base,
      };
    });
    nodesData.update(updates);
  }

  function clearHighlight() {
    if (nodesData) nodesData.update(visNodes(allNodes));
    searchResults.classList.add("hidden");
    searchResults.innerHTML = "";
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

      highlightNodes(hits.map((h) => h.node_id));
      focusNode(hits[0].node_id);
    } catch (e) {
      searchResults.classList.remove("hidden");
      searchResults.innerHTML = '<div class="hit">搜索失败: ' + escapeHtml(e.message) + "</div>";
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
  document.getElementById("physicsToggle").addEventListener("change", (e) => {
    if (network) network.setOptions({ physics: { enabled: e.target.checked } });
  });
  window.addEventListener("resize", () => {
    try {
      if (network) network.redraw();
    } catch (_) {}
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", refresh);
  } else {
    refresh();
  }
})();
