// RSRC Viewer — WebAssembly frontend

let wasmReady = false;
let wasmError = null;

const MAX_FILE_BYTES = 50 * 1024 * 1024; // 50 MiB

const state = {
  activeTab: "overview",
  resourceFilter: "",
  primary: null,
  compare: null,
  resourceDetail: null,
};

const refs = {
  dropZone: document.getElementById("drop-zone"),
  fileInput: document.getElementById("file-input"),
  compareInput: document.getElementById("compare-input"),
  compareBtn: document.getElementById("compare-btn"),
  compareClearBtn: document.getElementById("compare-clear-btn"),
  loadingEl: document.getElementById("loading"),
  loadingMessage: document.getElementById("loading-message"),
  errorEl: document.getElementById("error"),
  errorMsg: document.getElementById("error-message"),
  resultsEl: document.getElementById("results"),
  fileNameEl: document.getElementById("file-name"),
  fileKindEl: document.getElementById("file-kind"),
  fileCompressionEl: document.getElementById("file-compression"),
  compareStatusEl: document.getElementById("compare-status"),
  clearBtn: document.getElementById("clear-btn"),
  exportJsonBtn: document.getElementById("export-json-btn"),
  summaryCardsEl: document.getElementById("summary-cards"),
  resourceFilterInput: document.getElementById("resource-filter"),
  resourcesList: document.getElementById("resources-list"),
  validationSummary: document.getElementById("validation-summary"),
  validationList: document.getElementById("validation-list"),
  jsonOutput: document.getElementById("json-output"),
  diffSummary: document.getElementById("diff-summary"),
  diffList: document.getElementById("diff-list"),
  resourceModal: document.getElementById("resource-modal"),
  resourceModalTitle: document.getElementById("resource-modal-title"),
  resourceModalMeta: document.getElementById("resource-modal-meta"),
  resourceModalPayload: document.getElementById("resource-modal-payload"),
  resourceDownloadBtn: document.getElementById("resource-download-btn"),
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
      void processPrimaryFile(file);
    }
  });

  refs.fileInput.addEventListener("change", () => {
    if (refs.fileInput.files[0]) {
      void processPrimaryFile(refs.fileInput.files[0]);
    }
  });

  refs.compareBtn.addEventListener("click", () => {
    if (!state.primary) {
      showError("Load a primary file before selecting a comparison file.");
      return;
    }
    refs.compareInput.click();
  });

  refs.compareInput.addEventListener("change", () => {
    if (refs.compareInput.files[0]) {
      void processCompareFile(refs.compareInput.files[0]);
    }
  });

  refs.compareClearBtn.addEventListener("click", () => {
    state.compare = null;
    refs.compareInput.value = "";
    renderCompareStatus();
    renderDiff();
  });

  refs.clearBtn.addEventListener("click", resetState);

  refs.exportJsonBtn.addEventListener("click", () => {
    if (!state.primary?.dump) {
      return;
    }
    downloadText(
      buildDownloadName(state.primary.name, "json"),
      state.primary.dump.json,
      "application/json",
    );
  });

  refs.resourceFilterInput.addEventListener("input", (event) => {
    state.resourceFilter = event.target.value.trim();
    renderResources();
  });

  document.querySelectorAll(".tab").forEach((button) => {
    button.addEventListener("click", () => setActiveTab(button.dataset.tab));
  });

  refs.resourcesList.addEventListener("click", (event) => {
    const trigger = event.target.closest(
      "[data-resource-type][data-resource-id]",
    );
    if (!trigger) {
      return;
    }
    const resourceType = trigger.getAttribute("data-resource-type");
    const resourceID = Number(trigger.getAttribute("data-resource-id"));
    void openResourceDetail(resourceType, resourceID);
  });

  refs.resourceModal.addEventListener("click", (event) => {
    if (
      event.target === refs.resourceModal ||
      event.target.closest("[data-close-modal]")
    ) {
      closeResourceDetail();
    }
  });

  refs.resourceDownloadBtn.addEventListener("click", () => {
    if (!state.resourceDetail) {
      return;
    }
    const payload = hexToBytes(state.resourceDetail.payload);
    const fileName = `${buildBaseName(state.primary.name)}-${state.resourceDetail.type}-${state.resourceDetail.id}.bin`;
    downloadBlob(
      fileName,
      new Blob([payload], { type: "application/octet-stream" }),
    );
  });
}

async function processPrimaryFile(file) {
  clearError();
  closeResourceDetail();

  if (file.size > MAX_FILE_BYTES) {
    showError(
      `File is ${(file.size / (1024 * 1024)).toFixed(1)} MiB; limit is ${MAX_FILE_BYTES / (1024 * 1024)} MiB.`,
    );
    return;
  }

  try {
    await waitForWasm();
    const bytes = await readFileBytes(file);

    showLoading("Parsing file…");
    const parse = invokeWasm("parseVI", bytes).data;

    showLoading("Validating file…");
    const validation = invokeWasm("validateVI", bytes).data;

    showLoading("Building JSON view…");
    const dump = invokeWasm("dumpVI", bytes).data;

    state.primary = {
      name: file.name,
      bytes,
      parse,
      validation,
      dump,
    };

    if (state.compare?.bytes) {
      showLoading("Computing diff…");
      state.compare.diff = invokeWasm(
        "diffVI",
        state.primary.bytes,
        state.compare.bytes,
      ).data;
    }

    hideLoading();
    render();
  } catch (err) {
    hideLoading();
    showError(err.message);
  }
}

async function processCompareFile(file) {
  clearError();

  if (!state.primary) {
    showError("Load a primary file before selecting a comparison file.");
    return;
  }

  if (file.size > MAX_FILE_BYTES) {
    showError(
      `Comparison file is ${(file.size / (1024 * 1024)).toFixed(1)} MiB; limit is ${MAX_FILE_BYTES / (1024 * 1024)} MiB.`,
    );
    return;
  }

  try {
    await waitForWasm();
    const bytes = await readFileBytes(file);

    showLoading("Computing diff…");
    state.compare = {
      name: file.name,
      bytes,
      diff: invokeWasm("diffVI", state.primary.bytes, bytes).data,
    };

    hideLoading();
    renderCompareStatus();
    renderDiff();
  } catch (err) {
    hideLoading();
    showError(err.message);
  }
}

async function openResourceDetail(resourceType, resourceID) {
  if (!state.primary?.bytes) {
    return;
  }

  try {
    showLoading("Loading resource payload…");
    state.resourceDetail = invokeWasm(
      "resourcePayloadVI",
      state.primary.bytes,
      resourceType,
      resourceID,
    ).data;
    hideLoading();
    renderResourceDetail();
  } catch (err) {
    hideLoading();
    showError(err.message);
  }
}

function closeResourceDetail() {
  state.resourceDetail = null;
  refs.resourceModal.classList.add("hidden");
}

function resetState() {
  state.primary = null;
  state.compare = null;
  state.resourceDetail = null;
  state.resourceFilter = "";
  refs.fileInput.value = "";
  refs.compareInput.value = "";
  refs.resourceFilterInput.value = "";
  clearError();
  hideLoading();
  closeResourceDetail();
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
    refs.compareStatusEl.textContent = "";
    return;
  }

  refs.fileNameEl.textContent = state.primary.name;
  refs.fileKindEl.textContent = state.primary.parse.kind;
  refs.fileCompressionEl.textContent =
    state.primary.parse.compression || "uncompressed";
  refs.exportJsonBtn.disabled = !state.primary.dump;

  renderCompareStatus();
  renderSummaryCards();
  renderHeader("primary-header", state.primary.parse.header);
  renderHeader("secondary-header", state.primary.parse.secondary_header);
  renderResources();
  renderValidation();
  renderDump();
  renderDiff();

  refs.resultsEl.classList.remove("hidden");
}

function renderCompareStatus() {
  if (!state.primary) {
    refs.compareStatusEl.textContent = "";
    return;
  }
  if (!state.compare) {
    refs.compareStatusEl.textContent = "No comparison file loaded";
    return;
  }
  refs.compareStatusEl.textContent = `Comparing with ${state.compare.name}`;
}

function renderSummaryCards() {
  const summary = state.primary.parse.summary;
  const cards = [
    ["Blocks", formatCount(summary.block_count)],
    ["Resources", formatCount(summary.resource_count)],
    ["Named Resources", formatCount(summary.named_resource_count)],
    ["Name Entries", formatCount(summary.name_count)],
    ["Payload Bytes", formatBytes(summary.total_payload_bytes)],
    [
      "Validation",
      `${summary.error_count} errors / ${summary.warning_count} warnings`,
    ],
  ];

  refs.summaryCardsEl.innerHTML = cards
    .map(
      ([label, value]) => `
        <div class="summary-card">
          <span class="summary-label">${escHtml(label)}</span>
          <strong class="summary-value">${escHtml(value)}</strong>
        </div>`,
    )
    .join("");
}

function renderHeader(tableID, header) {
  const table = document.getElementById(tableID);
  const rows = [
    ["Magic", header.magic],
    ["Format version", header.format_version],
    ["Type", header.type],
    ["Creator", header.creator],
    ["Info offset", formatHex(header.info_offset)],
    ["Info size", formatBytes(header.info_size)],
    ["Data offset", formatHex(header.data_offset)],
    ["Data size", formatBytes(header.data_size)],
  ];

  table.innerHTML = rows
    .map(
      ([label, value]) =>
        `<tr><td>${escHtml(String(label))}</td><td>${escHtml(String(value))}</td></tr>`,
    )
    .join("");
}

function renderResources() {
  const resources = state.primary?.parse?.resources || [];
  const query = state.resourceFilter.toLowerCase();
  const filtered = resources.filter((resource) => {
    if (!query) {
      return true;
    }
    return [
      resource.type,
      String(resource.id),
      resource.name || "",
      resource.preview || "",
    ]
      .join(" ")
      .toLowerCase()
      .includes(query);
  });

  if (filtered.length === 0) {
    refs.resourcesList.innerHTML = query
      ? "<p>No resources match the current filter.</p>"
      : "<p>No resources found.</p>";
    return;
  }

  const rows = filtered
    .map(
      (resource) => `
        <tr>
          <td><span class="type-tag">${escHtml(String(resource.type))}</span></td>
          <td>${Number(resource.id)}</td>
          <td>${escHtml(String(resource.name || ""))}</td>
          <td>${formatBytes(resource.size)}</td>
          <td class="hex-preview">${escHtml(String(resource.preview || ""))}</td>
          <td>
            <button
              class="btn-inline"
              type="button"
              data-resource-type="${escAttr(String(resource.type))}"
              data-resource-id="${Number(resource.id)}"
            >
              View
            </button>
          </td>
        </tr>`,
    )
    .join("");

  refs.resourcesList.innerHTML = `
    <table class="resource-table">
      <thead>
        <tr>
          <th>Type</th>
          <th>ID</th>
          <th>Name</th>
          <th>Size</th>
          <th>Hex preview</th>
          <th></th>
        </tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>`;
}

function renderValidation() {
  const validation = state.primary?.validation;
  if (!validation) {
    refs.validationSummary.innerHTML = "";
    refs.validationList.innerHTML = "<p>No validation data loaded.</p>";
    return;
  }

  refs.validationSummary.innerHTML = `
    <div class="validation-pill validation-pill-error">${validation.summary.error_count} errors</div>
    <div class="validation-pill validation-pill-warning">${validation.summary.warning_count} warnings</div>
    <div class="validation-pill">${validation.summary.issue_count} total issues</div>`;

  if (validation.issues.length === 0) {
    refs.validationList.innerHTML =
      "<p>This file passed structural validation.</p>";
    return;
  }

  refs.validationList.innerHTML = validation.issues
    .map(
      (issue) => `
        <article class="issue-card issue-${escAttr(issue.severity)}">
          <div class="issue-row">
            <span class="issue-severity">${escHtml(issue.severity)}</span>
            <code>${escHtml(issue.code)}</code>
          </div>
          <p>${escHtml(issue.message)}</p>
          <p class="issue-location">${escHtml(formatLocation(issue.location))}</p>
        </article>`,
    )
    .join("");
}

function renderDump() {
  refs.jsonOutput.textContent = state.primary?.dump?.json || "";
}

function renderDiff() {
  if (!state.compare?.diff) {
    refs.diffSummary.innerHTML = `
      <div class="diff-callout">
        <p>Add a second file to compare structural differences.</p>
      </div>`;
    refs.diffList.innerHTML = "";
    return;
  }

  const summary = state.compare.diff.summary;
  refs.diffSummary.innerHTML = `
    <div class="validation-pill">${summary.item_count} diff items</div>
    <div class="validation-pill">${summary.header_count} header</div>
    <div class="validation-pill">${summary.block_count} block</div>
    <div class="validation-pill">${summary.section_count} section</div>
    <div class="validation-pill">${summary.added_count} added</div>
    <div class="validation-pill">${summary.removed_count} removed</div>
    <div class="validation-pill">${summary.modified_count} modified</div>`;

  const items = state.compare.diff.diff?.items || [];
  if (items.length === 0) {
    refs.diffList.innerHTML = "<p>No structural differences detected.</p>";
    return;
  }

  const rows = items
    .map(
      (item) => `
        <tr>
          <td><span class="type-tag">${escHtml(item.kind)}</span></td>
          <td>${escHtml(item.category)}</td>
          <td><code>${escHtml(item.path)}</code></td>
          <td>${escHtml(item.message || formatDiffChange(item.old, item.new))}</td>
        </tr>`,
    )
    .join("");

  refs.diffList.innerHTML = `
    <table class="resource-table">
      <thead>
        <tr>
          <th>Kind</th>
          <th>Category</th>
          <th>Path</th>
          <th>Change</th>
        </tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>`;
}

function renderResourceDetail() {
  if (!state.resourceDetail) {
    refs.resourceModal.classList.add("hidden");
    return;
  }

  refs.resourceModalTitle.textContent = `${state.resourceDetail.type} / ${state.resourceDetail.id}`;
  refs.resourceModalMeta.textContent = `${state.resourceDetail.name || "unnamed resource"} · ${formatBytes(state.resourceDetail.size)}`;
  refs.resourceModalPayload.textContent = chunkHex(
    state.resourceDetail.payload,
  );
  refs.resourceModal.classList.remove("hidden");
}

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
  let scaled = size;
  let unitIndex = -1;
  while (scaled >= 1024 && unitIndex < units.length - 1) {
    scaled /= 1024;
    unitIndex += 1;
  }
  return `${scaled.toFixed(scaled >= 10 ? 1 : 2)} ${units[unitIndex]}`;
}

function formatCount(value) {
  return Number(value).toLocaleString();
}

function formatLocation(location) {
  const parts = [];
  if (location.area) {
    parts.push(location.area);
  }
  if (location.blockType) {
    parts.push(`block ${location.blockType}`);
  }
  if (location.sectionIndex !== undefined && location.sectionIndex !== null) {
    parts.push(`section ${location.sectionIndex}`);
  }
  if (location.nameOffset) {
    parts.push(`name offset ${formatHex(location.nameOffset)}`);
  }
  parts.push(`offset ${formatHex(location.offset || 0)}`);
  return parts.join(" · ");
}

function formatDiffChange(oldValue, newValue) {
  const oldText =
    oldValue === undefined || oldValue === null
      ? "none"
      : JSON.stringify(oldValue);
  const newText =
    newValue === undefined || newValue === null
      ? "none"
      : JSON.stringify(newValue);
  return `${oldText} -> ${newText}`;
}

function buildBaseName(name) {
  return name.replace(/\.[^.]+$/, "");
}

function buildDownloadName(name, extension) {
  return `${buildBaseName(name)}.${extension}`;
}

function downloadText(name, content, mimeType) {
  downloadBlob(name, new Blob([content], { type: mimeType }));
}

function downloadBlob(name, blob) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = name;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

function chunkHex(hex) {
  const chunks = [];
  for (let index = 0; index < hex.length; index += 64) {
    chunks.push(hex.slice(index, index + 64));
  }
  return chunks.join("\n");
}

function hexToBytes(hex) {
  const clean = hex.trim();
  const output = new Uint8Array(clean.length / 2);
  for (let index = 0; index < clean.length; index += 2) {
    output[index / 2] = parseInt(clean.slice(index, index + 2), 16);
  }
  return output;
}

function escHtml(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function escAttr(value) {
  return escHtml(value);
}
