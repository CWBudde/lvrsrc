// RSRC Viewer — WebAssembly frontend

let wasmReady = false;
let wasmError = null;

const MAX_FILE_BYTES = 50 * 1024 * 1024; // 50 MiB

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

// DOM refs
const dropZone = document.getElementById("drop-zone");
const fileInput = document.getElementById("file-input");
const loadingEl = document.getElementById("loading");
const errorEl = document.getElementById("error");
const errorMsg = document.getElementById("error-message");
const resultsEl = document.getElementById("results");
const fileNameEl = document.getElementById("file-name");
const fileKindEl = document.getElementById("file-kind");
const clearBtn = document.getElementById("clear-btn");

// Drop zone
dropZone.addEventListener("click", () => fileInput.click());
dropZone.addEventListener("dragover", (e) => {
  e.preventDefault();
  dropZone.classList.add("drag-over");
});
dropZone.addEventListener("dragleave", () => dropZone.classList.remove("drag-over"));
dropZone.addEventListener("drop", (e) => {
  e.preventDefault();
  dropZone.classList.remove("drag-over");
  const file = e.dataTransfer.files[0];
  if (file) processFile(file);
});
fileInput.addEventListener("change", () => {
  if (fileInput.files[0]) processFile(fileInput.files[0]);
});

clearBtn.addEventListener("click", () => {
  resultsEl.classList.add("hidden");
  errorEl.classList.add("hidden");
  fileInput.value = "";
});

// Tab switching
document.querySelectorAll(".tab").forEach((btn) => {
  btn.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach((t) => t.classList.remove("active"));
    document.querySelectorAll(".tab-content").forEach((c) => c.classList.remove("active"));
    btn.classList.add("active");
    document.getElementById("tab-" + btn.dataset.tab).classList.add("active");
  });
});

async function processFile(file) {
  hideAll();
  loadingEl.classList.remove("hidden");

  if (file.size > MAX_FILE_BYTES) {
    loadingEl.classList.add("hidden");
    showError(
      `File is ${(file.size / (1024 * 1024)).toFixed(1)} MiB; limit is ${MAX_FILE_BYTES / (1024 * 1024)} MiB.`,
    );
    return;
  }

  try {
    if (!wasmReady) {
      await waitForWasm();
    }
    const buffer = await file.arrayBuffer();
    const bytes = new Uint8Array(buffer);
    const json = parseVI(bytes);
    const result = JSON.parse(json);

    loadingEl.classList.add("hidden");

    if (!result.success) {
      showError(result.error);
      return;
    }

    renderResults(file.name, result.data);
  } catch (err) {
    loadingEl.classList.add("hidden");
    showError(err.message);
  }
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

function renderResults(name, data) {
  fileNameEl.textContent = name;
  fileKindEl.textContent = data.kind;

  renderHeader("primary-header", data.header);
  renderHeader("secondary-header", data.secondary_header);
  renderResources(data.resources);

  resultsEl.classList.remove("hidden");
}

function renderHeader(tableId, h) {
  const table = document.getElementById(tableId);
  const rows = [
    ["Magic", h.magic],
    ["Format version", h.format_version],
    ["Type", h.type],
    ["Creator", h.creator],
    ["Info offset", "0x" + Number(h.info_offset).toString(16).padStart(8, "0")],
    ["Info size", Number(h.info_size) + " bytes"],
    ["Data offset", "0x" + Number(h.data_offset).toString(16).padStart(8, "0")],
    ["Data size", Number(h.data_size) + " bytes"],
  ];
  table.innerHTML = rows
    .map(
      ([label, val]) =>
        `<tr><td>${escHtml(String(label))}</td><td>${escHtml(String(val))}</td></tr>`,
    )
    .join("");
}

function renderResources(resources) {
  const container = document.getElementById("resources-list");
  if (!resources || resources.length === 0) {
    container.innerHTML = "<p>No resources found.</p>";
    return;
  }

  const rows = resources
    .map(
      (r) => `
      <tr>
        <td><span class="type-tag">${escHtml(String(r.type))}</span></td>
        <td>${Number(r.id)}</td>
        <td>${escHtml(String(r.name || ""))}</td>
        <td>${Number(r.size)}</td>
        <td class="hex-preview">${escHtml(String(r.payload || ""))}</td>
      </tr>`,
    )
    .join("");

  container.innerHTML = `
    <table class="resource-table">
      <thead>
        <tr>
          <th>Type</th>
          <th>ID</th>
          <th>Name</th>
          <th>Size (bytes)</th>
          <th>Hex preview</th>
        </tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>`;
}

function showError(msg) {
  errorMsg.textContent = msg;
  errorEl.classList.remove("hidden");
}

function hideAll() {
  loadingEl.classList.add("hidden");
  errorEl.classList.add("hidden");
  resultsEl.classList.add("hidden");
}

function escHtml(s) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
