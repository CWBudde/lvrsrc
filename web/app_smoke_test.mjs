import assert from "node:assert/strict";

class FakeClassList {
  constructor(initial = []) {
    this.set = new Set(initial);
  }

  add(...tokens) {
    for (const token of tokens) this.set.add(token);
  }

  remove(...tokens) {
    for (const token of tokens) this.set.delete(token);
  }

  toggle(token, force) {
    if (force === undefined) {
      if (this.set.has(token)) {
        this.set.delete(token);
        return false;
      }
      this.set.add(token);
      return true;
    }
    if (force) this.set.add(token);
    else this.set.delete(token);
    return force;
  }

  contains(token) {
    return this.set.has(token);
  }
}

class FakeElement {
  constructor(id = "", classes = []) {
    this.id = id;
    this.dataset = {};
    this.innerHTML = "";
    this.textContent = "";
    this.value = "";
    this.files = [];
    this.style = {};
    this.hidden = false;
    this.children = [];
    this.classList = new FakeClassList(classes);
    this.listeners = new Map();
  }

  addEventListener(type, handler) {
    const list = this.listeners.get(type) || [];
    list.push(handler);
    this.listeners.set(type, list);
  }

  dispatchEvent(type, payload = {}) {
    for (const handler of this.listeners.get(type) || []) {
      handler(payload);
    }
  }

  appendChild(child) {
    this.children.push(child);
    return child;
  }
}

class FakeCanvasElement extends FakeElement {
  constructor() {
    super("", []);
    this.width = 0;
    this.height = 0;
  }

  getContext(kind) {
    if (kind !== "2d") {
      return null;
    }
    return {
      scale() {},
      translate() {},
      clearRect() {},
      save() {},
      restore() {},
      beginPath() {},
      moveTo() {},
      arcTo() {},
      closePath() {},
      fill() {},
      stroke() {},
      fillText() {},
      setLineDash() {},
      putImageData() {},
      createImageData(width, height) {
        return { data: new Uint8ClampedArray(width * height * 4) };
      },
      textBaseline: "alphabetic",
      font: "",
      fillStyle: "",
      strokeStyle: "",
      lineWidth: 1,
    };
  }

  toDataURL() {
    return "data:image/png;base64,";
  }
}

function makeDocument() {
  const elements = new Map();
  const ids = [
    "drop-zone", "file-input", "loading", "loading-message", "error", "error-message",
    "results", "file-name", "file-kind", "file-compression", "clear-btn",
    "info-hero", "info-description-card", "info-description", "info-deps-card", "info-deps",
    "info-connector-card", "info-connector", "info-types-card", "info-types",
    "schema-diagram", "resources-list",
    "fp-empty", "fp-summary-card", "fp-histogram", "fp-warning-card", "fp-warnings",
    "fp-scene-card", "fp-scene", "fp-canvas-card", "fp-canvas-note", "fp-canvas", "fp-tree-card", "fp-tree",
    "bd-empty", "bd-summary-card", "bd-histogram", "bd-warning-card", "bd-warnings",
    "bd-scene-card", "bd-scene", "bd-canvas-card", "bd-canvas-note", "bd-canvas", "bd-tree-card", "bd-tree",
    "tab-info", "tab-frontpanel", "tab-blockdiagram", "tab-structure",
  ];
  for (const id of ids) {
    elements.set(id, new FakeElement(id, id.startsWith("tab-") ? ["tab-content"] : []));
  }
  elements.get("tab-info").classList.add("active");

  const tabs = [
    ["info", "active"],
    ["frontpanel"],
    ["blockdiagram"],
    ["structure"],
  ].map(([tab, active]) => {
    const el = new FakeElement("", ["tab"]);
    if (active) el.classList.add(active);
    el.dataset.tab = tab;
    return el;
  });

  const heapModeButtons = [];
  for (const panel of ["frontpanel", "blockdiagram"]) {
    for (const mode of ["visual", "canvas", "tree"]) {
      const el = new FakeElement("", ["heap-mode-btn"]);
      if (mode === "visual") el.classList.add("active");
      el.dataset.panel = panel;
      el.dataset.mode = mode;
      heapModeButtons.push(el);
    }
  }

  return {
    getElementById(id) {
      if (!elements.has(id)) {
        elements.set(id, new FakeElement(id));
      }
      return elements.get(id);
    },
    querySelectorAll(selector) {
      switch (selector) {
        case ".tab":
          return tabs;
        case ".tab-content":
          return [elements.get("tab-info"), elements.get("tab-frontpanel"), elements.get("tab-blockdiagram"), elements.get("tab-structure")];
        case ".heap-mode-btn":
          return heapModeButtons;
        default:
          if (selector.startsWith('.heap-mode-btn[data-panel="')) {
            const panel = selector.slice('.heap-mode-btn[data-panel="'.length, -2);
            return heapModeButtons.filter((b) => b.dataset.panel === panel);
          }
          return [];
      }
    },
    createElement(tag) {
      if (tag === "canvas") return new FakeCanvasElement();
      return new FakeElement();
    },
  };
}

globalThis.document = makeDocument();
globalThis.window = { devicePixelRatio: 1 };
globalThis.fetch = async () => ({});
globalThis.WebAssembly = {
  instantiateStreaming: async () => ({ instance: {} }),
};
globalThis.Go = class {
  constructor() {
    this.importObject = {};
  }
  run() {}
};

globalThis.parseVI = () => JSON.stringify({ success: true, data: {} });

const app = await import(new URL("./app.js", import.meta.url));

const syntheticParse = {
  kind: "VI",
  compression: "none",
  header: {
    magic: "RSRC",
    format_version: 3,
    type: "LVIN",
    creator: "LBVW",
    info_offset: 128,
    info_size: 256,
    data_offset: 32,
    data_size: 512,
  },
  summary: {
    block_count: 2,
    resource_count: 2,
    named_resource_count: 0,
    name_count: 0,
    total_payload_bytes: 512,
    decoded_count: 2,
  },
  resources: [],
  info: {
    display_name: "Smoke",
    has_desc: false,
    deps: { front_panel: [], block_diagram: [] },
    front_panel: {
      nodes: [
        { tag_name: "SL__object", scope: "open", parent: -1, children: [1], content_size: 0 },
        { tag_name: "Tag(99999)", scope: "leaf", parent: 0, children: [], content_size: 4 },
      ],
      roots: [0],
      histogram: { SL__object: 1 },
    },
    block_diagram: {
      nodes: [
        { tag_name: "SL__object", scope: "open", parent: -1, children: [1], content_size: 0 },
        { tag_name: "Tag(99999)", scope: "leaf", parent: 0, children: [], content_size: 4 },
      ],
      roots: [0],
      histogram: { SL__object: 1 },
    },
    front_panel_scene: {
      view: "front-panel",
      view_box: { x: 0, y: 0, width: 320, height: 180 },
      prefer_canvas: false,
      nodes: [
        { kind: "box", label: "SL__object", bounds: { x: 24, y: 24, width: 180, height: 80 }, z: 1, parent: 0, heap_index: 0 },
        { kind: "label", label: "SL__object", bounds: { x: 40, y: 40, width: 120, height: 18 }, z: 2, parent: 0, heap_index: 0 },
      ],
      roots: [0],
      warnings: ["heuristic layout"],
    },
    block_diagram_scene: {
      view: "block-diagram",
      view_box: { x: 0, y: 0, width: 1800, height: 1200 },
      prefer_canvas: true,
      nodes: [
        { kind: "box", label: "SL__object", bounds: { x: 24, y: 24, width: 300, height: 120 }, z: 1, parent: 0, heap_index: 0, placeholder: true },
        { kind: "label", label: "Tag(99999)", bounds: { x: 48, y: 52, width: 160, height: 18 }, z: 2, parent: 0, heap_index: 1, placeholder: true },
      ],
      roots: [0],
      warnings: ["heuristic layout", "wire routing omitted"],
    },
    front_panel_svg: "<svg><title>fp</title></svg>",
    block_diagram_svg: "<svg><title>bd</title></svg>",
    front_panel_warnings: ["heuristic layout"],
    block_diagram_warnings: ["heuristic layout", "wire routing omitted"],
  },
};

app.state.primary = { name: "smoke.vi", parse: syntheticParse };
app.render();

assert.equal(app.refs.fpSceneCard.classList.contains("hidden"), false, "front-panel visual card should be visible by default");
assert.equal(app.refs.fpTreeCard.classList.contains("hidden"), true, "front-panel tree card should be hidden by default");
assert.equal(app.refs.bdSceneCard.classList.contains("hidden"), false, "block-diagram visual card should be visible by default");
assert.equal(app.refs.bdTreeCard.classList.contains("hidden"), true, "block-diagram tree card should be hidden by default");

app.setHeapMode("frontpanel", "tree");
assert.equal(app.refs.fpTreeCard.classList.contains("hidden"), false, "front-panel tree card should be visible in tree mode");
assert.equal(app.refs.fpSceneCard.classList.contains("hidden"), true, "front-panel visual card should hide in tree mode");

app.setHeapMode("frontpanel", "canvas");
assert.equal(app.refs.fpCanvasCard.classList.contains("hidden"), false, "front-panel canvas card should be visible in canvas mode");

app.setHeapMode("blockdiagram", "tree");
assert.equal(app.refs.bdTreeCard.classList.contains("hidden"), false, "block-diagram tree card should be visible in tree mode");

app.setHeapMode("blockdiagram", "canvas");
assert.equal(app.refs.bdCanvasCard.classList.contains("hidden"), false, "block-diagram canvas card should be visible in canvas mode");
assert.equal(app.refs.bdCanvasNote.classList.contains("hidden"), false, "block-diagram canvas recommendation note should be visible");

assert.equal(app.refs.fpWarningCard.classList.contains("hidden"), false, "front-panel warnings should render");
assert.equal(app.refs.bdWarningCard.classList.contains("hidden"), false, "block-diagram warnings should render");
