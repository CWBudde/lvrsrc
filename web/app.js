// RSRC Viewer — WebAssembly frontend

let wasmReady = false;
let wasmError = null;

const MAX_FILE_BYTES = 50 * 1024 * 1024; // 50 MiB

const state = {
  activeTab: "info",
  primary: null,
  heapMode: {
    frontpanel: "visual",
    blockdiagram: "visual",
  },
};

const refs = {
  dropZone: document.getElementById("drop-zone"),
  fileInput: document.getElementById("file-input"),
  loadingEl: document.getElementById("loading"),
  loadingMessage: document.getElementById("loading-message"),
  errorEl: document.getElementById("error"),
  errorMsg: document.getElementById("error-message"),
  resultsEl: document.getElementById("results"),
  fileNameEl: document.getElementById("file-name"),
  fileKindEl: document.getElementById("file-kind"),
  fileCompressionEl: document.getElementById("file-compression"),
  clearBtn: document.getElementById("clear-btn"),
  infoHero: document.getElementById("info-hero"),
  infoDescriptionCard: document.getElementById("info-description-card"),
  infoDescription: document.getElementById("info-description"),
  infoDepsCard: document.getElementById("info-deps-card"),
  infoDeps: document.getElementById("info-deps"),
  infoConnectorCard: document.getElementById("info-connector-card"),
  infoConnector: document.getElementById("info-connector"),
  infoTypesCard: document.getElementById("info-types-card"),
  infoTypes: document.getElementById("info-types"),
  schemaDiagram: document.getElementById("schema-diagram"),
  resourcesList: document.getElementById("resources-list"),
  fpEmpty: document.getElementById("fp-empty"),
  fpSummaryCard: document.getElementById("fp-summary-card"),
  fpHistogram: document.getElementById("fp-histogram"),
  fpWarningCard: document.getElementById("fp-warning-card"),
  fpWarnings: document.getElementById("fp-warnings"),
  fpSceneCard: document.getElementById("fp-scene-card"),
  fpScene: document.getElementById("fp-scene"),
  fpCanvasCard: document.getElementById("fp-canvas-card"),
  fpCanvasNote: document.getElementById("fp-canvas-note"),
  fpCanvas: document.getElementById("fp-canvas"),
  fpTreeCard: document.getElementById("fp-tree-card"),
  fpTree: document.getElementById("fp-tree"),
  bdEmpty: document.getElementById("bd-empty"),
  bdSummaryCard: document.getElementById("bd-summary-card"),
  bdHistogram: document.getElementById("bd-histogram"),
  bdWarningCard: document.getElementById("bd-warning-card"),
  bdWarnings: document.getElementById("bd-warnings"),
  bdSceneCard: document.getElementById("bd-scene-card"),
  bdScene: document.getElementById("bd-scene"),
  bdCanvasCard: document.getElementById("bd-canvas-card"),
  bdCanvasNote: document.getElementById("bd-canvas-note"),
  bdCanvas: document.getElementById("bd-canvas"),
  bdTreeCard: document.getElementById("bd-tree-card"),
  bdTree: document.getElementById("bd-tree"),
};

async function initWasm() {
  const go = new Go();
  const result = await WebAssembly.instantiateStreaming(
    fetch("lvrsrc.wasm"),
    go.importObject,
  );
  go.run(result.instance);
  wasmReady = true;
}

initWasm().catch((err) => {
  wasmError = err;
  showError("Failed to load WASM engine: " + err.message);
});

bindEvents();
setActiveTab(state.activeTab);
render();

function bindEvents() {
  refs.dropZone.addEventListener("click", () => refs.fileInput.click());
  refs.dropZone.addEventListener("dragover", (event) => {
    event.preventDefault();
    refs.dropZone.classList.add("drag-over");
  });
  refs.dropZone.addEventListener("dragleave", () =>
    refs.dropZone.classList.remove("drag-over"),
  );
  refs.dropZone.addEventListener("drop", (event) => {
    event.preventDefault();
    refs.dropZone.classList.remove("drag-over");
    const file = event.dataTransfer.files[0];
    if (file) {
      void processFile(file);
    }
  });

  refs.fileInput.addEventListener("change", () => {
    if (refs.fileInput.files[0]) {
      void processFile(refs.fileInput.files[0]);
    }
  });

  refs.clearBtn.addEventListener("click", resetState);

  document.querySelectorAll(".tab").forEach((button) => {
    button.addEventListener("click", () => setActiveTab(button.dataset.tab));
  });
  document.querySelectorAll(".heap-mode-btn").forEach((button) => {
    button.addEventListener("click", () => setHeapMode(button.dataset.panel, button.dataset.mode));
  });
}

async function processFile(file) {
  clearError();
  if (file.size > MAX_FILE_BYTES) {
    showError(`File too large (max ${formatBytes(MAX_FILE_BYTES)}).`);
    return;
  }

  showLoading(`Parsing ${file.name}…`);
  try {
    await waitForWasm();
    const bytes = await readFileBytes(file);
    const parse = invokeWasm("parseVI", bytes).data;
    state.primary = { name: file.name, parse };
    refs.fileInput.value = "";
    render();
  } catch (err) {
    showError(err.message || String(err));
    state.primary = null;
    render();
  } finally {
    hideLoading();
  }
}

function resetState() {
  state.primary = null;
  refs.fileInput.value = "";
  clearError();
  render();
}

function setActiveTab(tabName) {
  state.activeTab = tabName;
  document.querySelectorAll(".tab").forEach((button) => {
    button.classList.toggle("active", button.dataset.tab === tabName);
  });
  document.querySelectorAll(".tab-content").forEach((panel) => {
    panel.classList.toggle("active", panel.id === `tab-${tabName}`);
  });
}

function setHeapMode(panel, mode) {
  if (!panel || !mode) {
    return;
  }
  state.heapMode[panel] = mode;
  document.querySelectorAll(`.heap-mode-btn[data-panel="${panel}"]`).forEach((button) => {
    button.classList.toggle("active", button.dataset.mode === mode);
  });
  if (state.primary) {
    render();
  }
}

function render() {
  if (!state.primary) {
    refs.resultsEl.classList.add("hidden");
    return;
  }

  refs.fileNameEl.textContent = state.primary.name;
  refs.fileKindEl.textContent = state.primary.parse.kind;
  refs.fileCompressionEl.textContent =
    state.primary.parse.compression || "uncompressed";

  renderInfo();
  renderHeapTree(
    state.primary.parse.info.front_panel,
    refs.fpEmpty,
    refs.fpSummaryCard,
    refs.fpHistogram,
    refs.fpWarningCard,
    refs.fpWarnings,
    refs.fpSceneCard,
    refs.fpScene,
    refs.fpCanvasCard,
    refs.fpCanvasNote,
    refs.fpCanvas,
    refs.fpTreeCard,
    refs.fpTree,
    state.primary.parse.info.front_panel_scene,
    state.primary.parse.info.front_panel_svg,
    state.primary.parse.info.front_panel_warnings,
    state.heapMode.frontpanel,
    "fp",
  );
  renderHeapTree(
    state.primary.parse.info.block_diagram,
    refs.bdEmpty,
    refs.bdSummaryCard,
    refs.bdHistogram,
    refs.bdWarningCard,
    refs.bdWarnings,
    refs.bdSceneCard,
    refs.bdScene,
    refs.bdCanvasCard,
    refs.bdCanvasNote,
    refs.bdCanvas,
    refs.bdTreeCard,
    refs.bdTree,
    state.primary.parse.info.block_diagram_scene,
    state.primary.parse.info.block_diagram_svg,
    state.primary.parse.info.block_diagram_warnings,
    state.heapMode.blockdiagram,
    "bd",
  );
  renderStructure();

  refs.resultsEl.classList.remove("hidden");
}

// Info tab ------------------------------------------------------------------

function renderInfo() {
  const info = state.primary.parse.info;
  const fileName = state.primary.name;
  const displayName = info.display_name || fileName;

  refs.infoHero.innerHTML = `
    <div class="info-hero-icon">${renderIcon(info.icon)}</div>
    <div class="info-hero-meta">
      <h2 class="info-hero-title">${escHtml(displayName)}</h2>
      <p class="info-hero-subtitle">${escHtml(fileName)}</p>
      <dl class="info-hero-facts">
        ${renderFact("Kind", state.primary.parse.kind)}
        ${info.version ? renderFact("Version", info.version) : ""}
        ${renderFact("Format", formatHex(state.primary.parse.header.format_version))}
      </dl>
      ${renderFlagChips(info.flags)}
    </div>`;

  if (info.has_desc && info.description) {
    refs.infoDescriptionCard.classList.remove("hidden");
    refs.infoDescription.textContent = info.description;
  } else {
    refs.infoDescriptionCard.classList.add("hidden");
    refs.infoDescription.textContent = "";
  }

  const fp = info.deps?.front_panel ?? [];
  const bd = info.deps?.block_diagram ?? [];
  const vi = info.deps?.vi_dependencies ?? [];
  if (fp.length === 0 && bd.length === 0 && vi.length === 0) {
    refs.infoDepsCard.classList.add("hidden");
    refs.infoDeps.innerHTML = "";
  } else {
    refs.infoDepsCard.classList.remove("hidden");
    refs.infoDeps.innerHTML = `
      ${renderDepGroup("Front panel imports", fp)}
      ${renderDepGroup("Block diagram imports", bd)}
      ${renderDepGroup("VI dependencies", vi)}`;
  }

  renderConnectorPane(info.connector);
  renderTypesList(info.types);
}

function renderFact(label, value) {
  return `
    <div class="info-fact">
      <dt>${escHtml(label)}</dt>
      <dd>${escHtml(value)}</dd>
    </div>`;
}

// Maps each WASM-side flag field to its display label and chip variant.
// Variant controls colour: "warn" for safety/lock-relevant flags,
// "info" for benign settings, "debug" for debug-related state.
const FLAG_CHIPS = [
  { key: "password_protected", label: "password", variant: "warn" },
  { key: "locked", label: "locked", variant: "warn" },
  { key: "run_on_open", label: "run on open", variant: "warn" },
  { key: "saved_for_previous", label: "saved for previous", variant: "info" },
  { key: "separate_code", label: "separate code", variant: "info" },
  { key: "clear_indicators", label: "clear indicators on run", variant: "info" },
  { key: "auto_error_handling", label: "auto error handling", variant: "info" },
  { key: "suspend_on_run", label: "suspend on run", variant: "debug" },
  { key: "debuggable", label: "debuggable", variant: "debug" },
  { key: "has_breakpoints", label: "breakpoints", variant: "debug" },
];

function renderFlagChips(flags) {
  if (!flags) {
    return "";
  }
  const chips = FLAG_CHIPS.filter((c) => flags[c.key])
    .map(
      (c) =>
        `<span class="info-flag-chip info-flag-chip-${c.variant}">${escHtml(c.label)}</span>`,
    )
    .join("");
  if (chips === "") {
    return "";
  }
  return `<div class="info-flag-row">${chips}</div>`;
}

function renderIcon(icon) {
  if (!icon || !icon.rgba) {
    return `<div class="info-icon-placeholder" aria-hidden="true">VI</div>`;
  }
  const dataURL = iconRGBAToDataURL(icon);
  if (!dataURL) {
    return `<div class="info-icon-placeholder" aria-hidden="true">VI</div>`;
  }
  const badge = icon.fourcc
    ? `<span class="info-icon-variant">${escHtml(icon.fourcc)}</span>`
    : "";
  return `
    <div class="info-icon-stack">
      <img class="info-icon-img"
           src="${dataURL}"
           width="128" height="128"
           alt="VI icon (${escHtml(icon.fourcc || "icon")})" />
      ${badge}
    </div>`;
}

function iconRGBAToDataURL(icon) {
  const canvas = document.createElement("canvas");
  canvas.width = icon.width;
  canvas.height = icon.height;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    return null;
  }
  const raw = atob(icon.rgba);
  const expected = icon.width * icon.height * 4;
  if (raw.length !== expected) {
    return null;
  }
  const img = ctx.createImageData(icon.width, icon.height);
  for (let i = 0; i < expected; i++) {
    img.data[i] = raw.charCodeAt(i);
  }
  ctx.putImageData(img, 0, 0);
  return canvas.toDataURL("image/png");
}

function renderDepGroup(label, entries) {
  if (entries.length === 0) {
    return "";
  }
  const rows = entries
    .map((entry) => {
      const qualifier = (entry.qualifiers || []).filter(Boolean).join(" :: ");
      const path = renderDepPath(entry.primary_path);
      // Prefer the human-friendly LinkKind ("TypeDef → CustCtl") and
      // surface the 4-byte FourCC as a tooltip; fall back to the FourCC
      // itself if the kind hasn't been catalogued.
      const kindLabel = entry.link_kind || entry.link_type || "";
      const kindTitle = entry.link_type ? `title="${escHtml(entry.link_type)}"` : "";
      const kindChip = kindLabel
        ? `<span class="info-dep-kind" ${kindTitle}>${escHtml(kindLabel)}</span>`
        : "";
      // Typed-target hints: TDCC carries a TypeID (1-based VCTP index)
      // and an offset count. Render compactly to the right of the kind.
      const meta = [];
      if (entry.has_type_id) {
        meta.push(`type #${entry.type_id}`);
      }
      if (entry.offset_count) {
        meta.push(`${entry.offset_count} offset${entry.offset_count === 1 ? "" : "s"}`);
      }
      const metaChip = meta.length
        ? `<span class="info-dep-meta">${escHtml(meta.join(" · "))}</span>`
        : "";
      return `
        <li class="info-dep-row">
          <div class="info-dep-row-main">
            <span class="info-dep-qualifier">${escHtml(qualifier) || "<em>unnamed</em>"}</span>
            ${kindChip}
            ${metaChip}
          </div>
          ${path}
        </li>`;
    })
    .join("");
  return `
    <section class="info-dep-group">
      <h4>${escHtml(label)} <span class="info-dep-count">${entries.length}</span></h4>
      <ul class="info-dep-list">${rows}</ul>
    </section>`;
}

function renderDepPath(path) {
  if (!path) {
    return "";
  }
  const components = (path.components || []).filter((c) => c.length > 0);
  // Path display: PTH1 uses "/" as separator since TPIdent encodes the
  // root style; PTH0 treats components as drive/folder/file segments
  // joined with "/" too. Empty components are dropped (they are common
  // padding in Tools.lvlib paths).
  const rendered = components.map(escHtml).join(" / ");
  let prefix = "";
  if (path.is_absolute) prefix = "abs ";
  else if (path.is_relative) prefix = "rel ";
  else if (path.is_unc) prefix = "unc ";
  else if (path.is_not_a_path) prefix = "!pth";
  else if (path.is_phony) prefix = "phony ";
  if (rendered === "" && prefix === "") {
    return "";
  }
  return `<div class="info-dep-path"><code>${prefix}${rendered || "<em>empty</em>"}</code></div>`;
}

function renderConnectorPane(connector) {
  if (!connector) {
    refs.infoConnectorCard.classList.add("hidden");
    refs.infoConnector.innerHTML = "";
    return;
  }
  refs.infoConnectorCard.classList.remove("hidden");

  const layout = connectorLayout(connector.cpc2 || 0);
  const svg = renderConnectorSVG(layout);
  const summary = `
    <div class="info-connector-meta">
      <span class="info-connector-line">${layout.terminals} terminals · CPC2 = ${connector.cpc2 || 0}</span>
      <span class="subtle-text">CONP = ${connector.conp || 0}${
        connector.pane_type
          ? ` · resolved to <code>${escHtml(connector.pane_type.full_type)}</code>${
              connector.pane_type.label
                ? ` "<em>${escHtml(connector.pane_type.label)}</em>"`
                : ""
            }`
          : ""
      }</span>
    </div>`;
  refs.infoConnector.innerHTML = svg + summary;
}

// connectorLayout returns rows of terminal counts for a given CPC2 value.
// The classic LabVIEW pane shapes vary; we approximate by mapping the
// observed corpus values 1..4 to common pane patterns and falling back to
// a rough N-up grid for unfamiliar values.
function connectorLayout(cpc2) {
  switch (cpc2) {
    case 1:
      return { rows: [4, 2, 2, 4], terminals: 12 };
    case 2:
      return { rows: [4, 4], terminals: 8 };
    case 3:
      return { rows: [2, 1, 1, 2], terminals: 6 };
    case 4:
      return { rows: [3, 1, 1, 3], terminals: 8 };
    default:
      // Default: a single column of CPC2 placeholder cells (or 1 terminal).
      const n = Math.max(1, cpc2);
      return { rows: [Math.min(n, 8)], terminals: Math.min(n, 8) };
  }
}

function renderConnectorSVG(layout) {
  // Render rows × max-cols on a canvas. Cells are 22×16 with 2px gap;
  // SVG width depends on max columns.
  const cellW = 22;
  const cellH = 16;
  const gap = 2;
  const maxCols = Math.max(...layout.rows);
  const w = maxCols * (cellW + gap) + gap;
  const h = layout.rows.length * (cellH + gap) + gap;
  let cells = "";
  for (let r = 0; r < layout.rows.length; r++) {
    const cols = layout.rows[r];
    const xOff = (maxCols - cols) * (cellW + gap) / 2 + gap;
    const y = r * (cellH + gap) + gap;
    for (let c = 0; c < cols; c++) {
      const x = xOff + c * (cellW + gap);
      cells += `<rect x="${x}" y="${y}" width="${cellW}" height="${cellH}" rx="2"/>`;
    }
  }
  return `
    <svg class="info-connector-svg"
         viewBox="0 0 ${w} ${h}"
         width="${w}" height="${h}"
         role="img"
         aria-label="Connector pane (${layout.terminals} terminals)">
      <rect x="0" y="0" width="${w}" height="${h}" class="info-connector-bg"/>
      <g class="info-connector-cells">${cells}</g>
    </svg>`;
}

function renderTypesList(types) {
  if (!Array.isArray(types) || types.length === 0) {
    refs.infoTypesCard.classList.add("hidden");
    refs.infoTypes.innerHTML = "";
    return;
  }
  refs.infoTypesCard.classList.remove("hidden");
  // Show named typedescs first, then a collapsed-by-default block with
  // every entry. Limit the named list to 12 to keep the card compact.
  const named = types.filter((t) => t.label && t.label.length > 0);
  const namedLimit = 12;
  const namedRows = named
    .slice(0, namedLimit)
    .map(
      (t) => `
      <li class="info-type-row">
        <span class="info-type-index">[${t.index}]</span>
        <span class="info-type-kind">${escHtml(t.full_type)}</span>
        <span class="info-type-label">${escHtml(t.label)}</span>
      </li>`,
    )
    .join("");
  const namedHtml = named.length === 0
    ? `<p class="subtle-text">No named typedescs in this VI's pool.</p>`
    : `
      <ul class="info-type-list">${namedRows}</ul>
      ${named.length > namedLimit ? `<p class="subtle-text">Showing ${namedLimit} of ${named.length} named typedescs.</p>` : ""}`;
  // Histogram of all types.
  const counts = {};
  for (const t of types) counts[t.full_type] = (counts[t.full_type] || 0) + 1;
  const histRows = Object.entries(counts)
    .sort((a, b) => b[1] - a[1])
    .map(
      ([type, n]) =>
        `<span class="info-type-pill">${escHtml(type)} <span class="info-type-pill-count">${n}</span></span>`,
    )
    .join("");
  refs.infoTypes.innerHTML = `
    ${namedHtml}
    <div class="info-type-hist-label subtle-text">All ${types.length} typedescs by kind:</div>
    <div class="info-type-hist">${histRows}</div>`;
}

// Heap-tree tabs (Front Panel & Block Diagram) -----------------------------

// Cap on how many top-level open-scope nodes we render eagerly per panel.
// Heaps have hundreds of children at depth 1; rendering them all up front
// makes the tab feel laggy on large VIs. Beyond this, we render a "show
// more" affordance.
const HEAP_TOP_LEVEL_LIMIT = 200;

function renderHeapTree(
  tree,
  emptyEl,
  summaryCard,
  histogramEl,
  warningCard,
  warningEl,
  sceneCard,
  sceneEl,
  canvasCard,
  canvasNoteEl,
  canvasEl,
  treeCard,
  treeEl,
  sceneData,
  sceneSVG,
  sceneWarnings,
  heapMode,
  idPrefix,
) {
  if (!tree || !Array.isArray(tree.nodes) || tree.nodes.length === 0) {
    emptyEl.classList.remove("hidden");
    summaryCard.classList.add("hidden");
    warningCard.classList.add("hidden");
    sceneCard.classList.add("hidden");
    canvasCard.classList.add("hidden");
    treeCard.classList.add("hidden");
    histogramEl.innerHTML = "";
    warningEl.innerHTML = "";
    sceneEl.innerHTML = "";
    canvasNoteEl.textContent = "";
    canvasNoteEl.classList.add("hidden");
    canvasEl.innerHTML = "";
    treeEl.innerHTML = "";
    return;
  }
  emptyEl.classList.add("hidden");
  summaryCard.classList.remove("hidden");
  renderHeapWarnings(warningCard, warningEl, sceneWarnings);
  if (sceneSVG) {
    sceneCard.classList.toggle("hidden", heapMode !== "visual");
    sceneEl.innerHTML = sceneSVG;
  } else {
    sceneCard.classList.add("hidden");
    sceneEl.innerHTML = "";
  }
  if (sceneData && heapMode === "canvas") {
    canvasCard.classList.remove("hidden");
    renderHeapCanvas(canvasEl, canvasNoteEl, sceneData);
  } else {
    canvasCard.classList.add("hidden");
    canvasNoteEl.textContent = "";
    canvasNoteEl.classList.add("hidden");
    canvasEl.innerHTML = "";
  }
  treeCard.classList.toggle("hidden", heapMode !== "tree");

  // Histogram: top classes by count, sorted descending.
  const hist = tree.histogram || {};
  const sorted = Object.entries(hist).sort((a, b) => b[1] - a[1]);
  const totalOpens = sorted.reduce((s, [, n]) => s + n, 0);
  if (sorted.length === 0) {
    histogramEl.innerHTML =
      '<p class="subtle-text">No opening-scope nodes — the heap is degenerate.</p>';
  } else {
    histogramEl.innerHTML = `
      <p class="subtle-text">${totalOpens} opening-scope nodes across ${sorted.length} distinct tags.</p>
      <div class="heap-hist-grid">
        ${sorted
          .map(([name, n]) => {
            const cls = name.includes("(") ? "heap-hist-pill-fallback" : "heap-hist-pill";
            return `
              <span class="${cls}">${escHtml(name)}<span class="heap-hist-count">${n}</span></span>`;
          })
          .join("")}
      </div>`;
  }

  // Build the open-only structure: skip "close" and "leaf" nodes for the
  // tree view. Pure-leaf nodes (fields) are folded into their open
  // parents as a count badge.
  const openOnly = new Set();
  const leafCounts = new Array(tree.nodes.length).fill(0);
  for (let i = 0; i < tree.nodes.length; i++) {
    const n = tree.nodes[i];
    if (n.scope === "open") openOnly.add(i);
    else if (n.scope === "leaf" && n.parent >= 0) leafCounts[n.parent]++;
  }

  // Roots: filter to open-scope only; top-level leafs go into a special
  // "loose leafs" section if any (rare; keep simple).
  const openRoots = tree.roots.filter((i) => openOnly.has(i));
  const visibleRoots = openRoots.slice(0, HEAP_TOP_LEVEL_LIMIT);
  const truncated = openRoots.length - visibleRoots.length;

  treeEl.innerHTML = visibleRoots
    .map((idx) => renderHeapTreeNode(tree, idx, openOnly, leafCounts, idPrefix))
    .join("");
  if (truncated > 0) {
    treeEl.innerHTML += `<p class="subtle-text">… and ${truncated} more top-level objects (collapsed).</p>`;
  }
}

function renderHeapWarnings(cardEl, warningsEl, warnings) {
  const list = Array.isArray(warnings) ? warnings.filter(Boolean) : [];
  if (list.length === 0) {
    cardEl.classList.add("hidden");
    warningsEl.innerHTML = "";
    return;
  }
  cardEl.classList.remove("hidden");
  warningsEl.innerHTML = `<ul>${list.map((w) => `<li>${escHtml(w)}</li>`).join("")}</ul>`;
}

function renderHeapCanvas(containerEl, noteEl, scene) {
  containerEl.innerHTML = "";
  const canvas = document.createElement("canvas");
  const dpr = Math.max(1, window.devicePixelRatio || 1);
  const maxCSSWidth = Math.max(320, Math.min(960, scene.view_box?.width || 640));
  const scale = maxCSSWidth / Math.max(1, scene.view_box?.width || maxCSSWidth);
  const cssWidth = maxCSSWidth;
  const cssHeight = Math.max(180, (scene.view_box?.height || 360) * scale);
  canvas.width = Math.round(cssWidth * dpr);
  canvas.height = Math.round(cssHeight * dpr);
  canvas.style.width = `${cssWidth}px`;
  canvas.style.height = `${cssHeight}px`;
  canvas.className = "heap-canvas";
  containerEl.appendChild(canvas);

  if (scene.prefer_canvas) {
    noteEl.textContent = "Canvas is recommended for this scene because it is relatively large.";
    noteEl.classList.remove("hidden");
  } else {
    noteEl.textContent = "";
    noteEl.classList.add("hidden");
  }

  const ctx = canvas.getContext("2d");
  if (!ctx) {
    containerEl.innerHTML = '<p class="subtle-text">Canvas context unavailable in this browser.</p>';
    return;
  }
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, cssWidth, cssHeight);
  ctx.save();
  ctx.scale(scale, scale);
  ctx.translate(-(scene.view_box?.x || 0), -(scene.view_box?.y || 0));

  const nodes = [...(scene.nodes || [])].sort((a, b) => (a.z || 0) - (b.z || 0));
  for (const node of nodes) {
    drawSceneNodeToCanvas(ctx, node);
  }
  ctx.restore();
}

function drawSceneNodeToCanvas(ctx, node) {
  const bounds = node.bounds || { x: 0, y: 0, width: 0, height: 0 };
  if (node.kind === "box") {
    ctx.save();
    ctx.beginPath();
    roundedRectPath(ctx, bounds.x, bounds.y, bounds.width, bounds.height, 8);
    ctx.fillStyle = "#f4f0e8";
    ctx.strokeStyle = node.placeholder ? "#7d4f50" : "#5f4b32";
    ctx.lineWidth = 1.5;
    if (node.placeholder) {
      ctx.setLineDash([6, 4]);
    }
    ctx.fill();
    ctx.stroke();
    ctx.restore();
    return;
  }
  if (node.kind === "label") {
    ctx.save();
    ctx.fillStyle = node.placeholder ? "#7d4f50" : "#16324f";
    ctx.font = '13px "Helvetica Neue", Helvetica, Arial, sans-serif';
    ctx.textBaseline = "alphabetic";
    ctx.fillText(node.label || "", bounds.x, bounds.y + bounds.height - 4, Math.max(0, bounds.width));
    ctx.restore();
  }
}

function roundedRectPath(ctx, x, y, width, height, radius) {
  const r = Math.min(radius, width / 2, height / 2);
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + width, y, x + width, y + height, r);
  ctx.arcTo(x + width, y + height, x, y + height, r);
  ctx.arcTo(x, y + height, x, y, r);
  ctx.arcTo(x, y, x + width, y, r);
  ctx.closePath();
}

function renderHeapTreeNode(tree, idx, openOnly, leafCounts, idPrefix) {
  const n = tree.nodes[idx];
  const childOpens = (n.children || []).filter((i) => openOnly.has(i));
  const leafCount = leafCounts[idx];
  const tagName = n.tag_name || "Tag(?)";
  const isResolved = !tagName.includes("(");
  const tagClass = isResolved ? "heap-node-tag" : "heap-node-tag-fallback";

  const summary = `
    <span class="${tagClass}">${escHtml(tagName)}</span>
    ${
      childOpens.length > 0
        ? `<span class="heap-node-count">${childOpens.length} child${childOpens.length === 1 ? "" : "ren"}</span>`
        : ""
    }
    ${
      leafCount > 0
        ? `<span class="heap-node-leafcount" title="${leafCount} field/leaf entries">${leafCount} field${leafCount === 1 ? "" : "s"}</span>`
        : ""
    }
    ${
      n.content_size > 0
        ? `<span class="heap-node-size subtle-text">${n.content_size} B</span>`
        : ""
    }`;

  if (childOpens.length === 0) {
    return `<div class="heap-node heap-node-leaf">${summary}</div>`;
  }

  const childHtml = childOpens
    .map((ci) => renderHeapTreeNode(tree, ci, openOnly, leafCounts, idPrefix))
    .join("");
  return `
    <details class="heap-node">
      <summary class="heap-node-summary">${summary}</summary>
      <div class="heap-node-children">${childHtml}</div>
    </details>`;
}

// Structure tab -------------------------------------------------------------

function renderStructure() {
  renderSchemaDiagram();
  renderResourcesList();
}

function renderSchemaDiagram() {
  const { header, summary } = state.primary.parse;
  const headerRange = `bytes <code>${formatHex(0)}</code> – <code>${formatHex(32)}</code> (32 B)`;
  const dataRange = `bytes <code>${formatHex(header.data_offset)}</code> – <code>${formatHex(header.data_offset + header.data_size)}</code> (${formatBytes(header.data_size)})`;
  const infoRange = `bytes <code>${formatHex(header.info_offset)}</code> – <code>${formatHex(header.info_offset + header.info_size)}</code> (${formatBytes(header.info_size)})`;

  const headerFacts = [
    ["magic", escHtml(header.magic)],
    ["format_version", formatHex(header.format_version)],
    ["type", escHtml(header.type)],
    ["creator", escHtml(header.creator)],
    ["info_offset", formatHex(header.info_offset)],
    ["info_size", `${formatBytes(header.info_size)} (${header.info_size})`],
    ["data_offset", formatHex(header.data_offset)],
    ["data_size", `${formatBytes(header.data_size)} (${header.data_size})`],
  ];

  refs.schemaDiagram.innerHTML = `
    <div class="schema-blueprint">
      <div class="schema-region schema-region-primary">
        <div class="schema-region-header">
          <span class="schema-region-title">Primary Header</span>
          <span class="schema-region-range">${headerRange}</span>
        </div>
        <table class="info-table schema-header-fields">
          <tbody>
            ${headerFacts.map(([k, v]) => `<tr><td>${escHtml(k)}</td><td>${v}</td></tr>`).join("")}
          </tbody>
        </table>
      </div>

      <div class="schema-arrow"><code>data_offset</code> points here ↓</div>

      <div class="schema-region schema-region-data">
        <div class="schema-region-header">
          <span class="schema-region-title">Data Section</span>
          <span class="schema-region-range">${dataRange}</span>
        </div>
        <p class="subtle-text schema-region-desc">
          Contiguous block of section payloads. Descriptors in the metadata
          pool point into this region.
        </p>
        <div class="schema-pills">
          <span class="validation-pill">${summary.block_count} blocks</span>
          <span class="validation-pill">${summary.resource_count} sections</span>
          <span class="validation-pill">${summary.decoded_count} decoded · ${summary.resource_count - summary.decoded_count} opaque</span>
          <span class="validation-pill">${formatBytes(summary.total_payload_bytes)} payload</span>
        </div>
      </div>

      <div class="schema-arrow"><code>info_offset</code> points here ↓</div>

      <div class="schema-region schema-region-meta">
        <div class="schema-region-header">
          <span class="schema-region-title">Metadata Pool</span>
          <span class="schema-region-range">${infoRange}</span>
        </div>
        <p class="subtle-text schema-region-desc">
          Duplicate header plus the tables that describe every block and section.
        </p>
        <div class="schema-subregion-list">
          <div class="schema-subregion">
            <strong>Secondary Header</strong>
            <span class="subtle-text">32 B — duplicate of primary</span>
          </div>
          <div class="schema-subregion">
            <strong>Block Info List</strong>
            <span class="subtle-text">${summary.block_count} entries — per-FourCC type, count, offset</span>
          </div>
          <div class="schema-subregion">
            <strong>Section Descriptors</strong>
            <span class="subtle-text">${summary.resource_count} descriptors — id, name_offset, data_offset, size</span>
          </div>
          <div class="schema-subregion">
            <strong>Name Table</strong>
            <span class="subtle-text">${summary.name_count} Pascal-style strings</span>
          </div>
        </div>
      </div>
    </div>`;
}

function renderResourcesList() {
  const resources = state.primary.parse.resources || [];
  if (resources.length === 0) {
    refs.resourcesList.innerHTML = "<p>No resources in this file.</p>";
    return;
  }

  const groups = groupByType(resources);
  const rows = [...groups.entries()]
    .map(([type, sections]) => {
      const totalBytes = sections.reduce((n, s) => n + (s.size || 0), 0);
      const decoded = sections[0].decoded;
      const role = RESOURCE_ROLES[type] || "Opaque — bytes preserved";
      const tierClass = decoded ? "resource-row-decoded" : "resource-row-opaque";
      const decodedBadge = decoded
        ? `<span class="resource-decoded-badge">decoded</span>`
        : "";
      const sectionDetail =
        sections.length === 1
          ? `<span class="subtle-text">section ${sections[0].id}${sections[0].name ? ` · ${escHtml(sections[0].name)}` : ""}</span>`
          : `<span class="subtle-text">${sections.length} sections</span>`;
      return `
        <tr class="${tierClass}">
          <td><span class="type-tag">${escHtml(type)}</span></td>
          <td>
            <div class="resource-role">${escHtml(role)} ${decodedBadge}</div>
            ${sectionDetail}
          </td>
          <td class="resource-size">${formatBytes(totalBytes)}</td>
        </tr>`;
    })
    .join("");

  refs.resourcesList.innerHTML = `
    <table class="resource-table">
      <thead>
        <tr>
          <th>FourCC</th>
          <th>Role</th>
          <th>Size</th>
        </tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>`;
}

function groupByType(resources) {
  const groups = new Map();
  for (const r of resources) {
    const list = groups.get(r.type) || [];
    list.push(r);
    groups.set(r.type, list);
  }
  return groups;
}

// Catalogue of human-readable roles, indexed by FourCC. Sourced from
// docs/resource-registry.md — keep in sync.
const RESOURCE_ROLES = {
  LVSR: "LabVIEW Save Record — carries the VI display name",
  vers: "Version stamp",
  STRG: "VI description / string resource",
  LIBN: "Library-name list (.lvlib membership)",
  LIvi: "VI dependencies",
  LIfp: "Front-panel imports",
  LIbd: "Block-diagram imports",
  BDPW: "Block-diagram password hash",
  ICON: "1-bit VI icon",
  icl4: "4-bit colour icon",
  icl8: "8-bit colour icon",
  FPHb: "Front-panel heap",
  BDHb: "Block-diagram heap",
  VCTP: "Type descriptor pool",
  HIST: "Edit history counters",
  VITS: "VI settings",
  CONP: "Connector pane selector",
  CPC2: "Connector pane count / variant",
  RTSG: "Runtime signature",
  FTAB: "Font table",
  MUID: "Module unique ID",
  DTHP: "Default data-heap pointer",
  FPEx: "Front-panel extra",
  BDEx: "Block-diagram extra",
  FPSE: "Front-panel section entry",
  BDSE: "Block-diagram section entry",
  VPDP: "VI probe-data pointer",
};

// Utilities -----------------------------------------------------------------

function invokeWasm(name, ...args) {
  const fn = globalThis[name];
  if (typeof fn !== "function") {
    throw new Error(`Missing WASM export: ${name}`);
  }
  const response = JSON.parse(fn(...args));
  if (!response.success) {
    throw new Error(response.error || `WASM call ${name} failed`);
  }
  return response;
}

function waitForWasm() {
  return new Promise((resolve, reject) => {
    const check = () => {
      if (wasmError) {
        reject(new Error("WASM engine failed to load: " + wasmError.message));
        return;
      }
      if (wasmReady) {
        resolve();
        return;
      }
      setTimeout(check, 50);
    };
    check();
  });
}

async function readFileBytes(file) {
  const buffer = await file.arrayBuffer();
  return new Uint8Array(buffer);
}

function showLoading(message) {
  refs.loadingMessage.textContent = message;
  refs.loadingEl.classList.remove("hidden");
}

function hideLoading() {
  refs.loadingEl.classList.add("hidden");
}

function showError(message) {
  refs.errorMsg.textContent = message;
  refs.errorEl.classList.remove("hidden");
}

function clearError() {
  refs.errorMsg.textContent = "";
  refs.errorEl.classList.add("hidden");
}

function formatHex(value) {
  return "0x" + Number(value).toString(16).padStart(8, "0");
}

function formatBytes(value) {
  const size = Number(value);
  if (!Number.isFinite(size)) {
    return String(value);
  }
  if (size < 1024) {
    return `${size} bytes`;
  }
  const units = ["KiB", "MiB", "GiB"];
  let n = size / 1024;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(n < 10 ? 2 : 1)} ${units[i]}`;
}

function escHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export { refs, render, setHeapMode, state };
