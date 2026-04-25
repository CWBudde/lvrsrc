// RSRC Viewer — WebAssembly frontend

let wasmReady = false;
let wasmError = null;

const MAX_FILE_BYTES = 50 * 1024 * 1024; // 50 MiB

const state = {
  activeTab: "info",
  primary: null,
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
  schemaDiagram: document.getElementById("schema-diagram"),
  resourcesList: document.getElementById("resources-list"),
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
  if (fp.length === 0 && bd.length === 0) {
    refs.infoDepsCard.classList.add("hidden");
    refs.infoDeps.innerHTML = "";
  } else {
    refs.infoDepsCard.classList.remove("hidden");
    refs.infoDeps.innerHTML = `
      ${renderDepGroup("Front panel imports", fp)}
      ${renderDepGroup("Block diagram imports", bd)}`;
  }
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
      return `
        <li class="info-dep-row">
          <div class="info-dep-row-main">
            <span class="info-dep-qualifier">${escHtml(qualifier) || "<em>unnamed</em>"}</span>
            ${entry.link_type ? `<span class="info-dep-kind">${escHtml(entry.link_type)}</span>` : ""}
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
