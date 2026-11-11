// Pathfinding Lab. The searches run in Go (cmd/lab-wasm). This file draws
// them and holds the settings.
import { GridView } from "./view.js";

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];

function el(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value === undefined || value === null || value === false) continue;
    if (key === "class") node.className = value;
    // The page policy blocks style attributes, so set the styles one by one.
    else if (key === "style") Object.assign(node.style, value);
    else if (key.startsWith("on")) node.addEventListener(key.slice(2), value);
    else node.setAttribute(key, value === true ? "" : String(value));
  }
  node.append(...children);
  return node;
}

/* Go */

async function loadGo() {
  await new Promise((resolve, reject) => {
    const script = el("script", { src: "wasm_exec.js" });
    script.onload = resolve;
    script.onerror = () => reject(new Error("cannot load wasm_exec.js"));
    document.head.append(script);
  });
  const go = new window.Go();
  const ready = new Promise((resolve) => window.addEventListener("lab-ready", resolve, { once: true }));
  const result = await WebAssembly.instantiateStreaming(fetch("lab.wasm"), go.importObject);
  go.run(result.instance);
  await ready;
}

function call(name, arg) {
  const input = arg === undefined ? undefined : typeof arg === "number" ? arg : JSON.stringify(arg);
  const reply = JSON.parse(window.lab[name](input));
  if (!reply.ok) throw new Error(reply.error);
  return reply.result;
}

/* Settings */

let algorithms = [];
let map = null;
let tool = "wall";

const num = (id) => Number($(id).value);

function options(record = true) {
  return {
    diagonal: $("#diagonal").checked,
    heuristic: $("#heuristic").value,
    weight: num("#weight"),
    mudCost: num("#mud"),
    record,
  };
}

const perFrame = () => 2 ** num("#speed");
const clampSize = (v) => Math.max(5, Math.min(200, Math.round(v) || 5));

function newMap() {
  const w = clampSize(num("#width"));
  const h = clampSize(num("#height"));
  $("#width").value = w;
  $("#height").value = h;
  map = call("generate", { kind: $("#kind").value, w, h, density: num("#density") / 100, seed: Math.floor(Math.random() * 1e9) });
  explore.view.setMap(map);
  explore.reset();
  race.reset();
}

function bindOutputs() {
  const show = () => {
    $("#density-out").textContent = `${$("#density").value}%`;
    $("#weight-out").textContent = $("#weight").value;
    $("#mud-out").textContent = $("#mud").value;
    $("#speed-out").textContent = `${perFrame()} cell${perFrame() > 1 ? "s" : ""} / frame`;
    $("#disks-out").textContent = $("#disks").value;
  };
  for (const id of ["#density", "#weight", "#mud", "#speed", "#disks"]) $(id).addEventListener("input", show);
  show();
  // A change of the search settings makes old results wrong.
  for (const id of ["#diagonal", "#heuristic", "#weight", "#mud"]) {
    $(id).addEventListener("change", () => {
      explore.reset();
      race.reset();
    });
  }
}

/* Tabs */

function selectTab(name) {
  $$(".tabs button").forEach((b) => b.setAttribute("aria-selected", String(b.dataset.tab === name)));
  $$(".view").forEach((v) => (v.hidden = v.dataset.view !== name));
  $$(".settings .group").forEach((g) => (g.hidden = !g.dataset.for.split(" ").includes(name)));
  $("#settings").hidden = name === "hanoi";
  $("#layout").style.gridTemplateColumns = name === "hanoi" ? "minmax(0, 1fr)" : "";
  if (name === "explore") explore.view.resize();
  if (name === "race") race.lanes.forEach((l) => l.view.resize());
}

/* Stats */

const fmt = (v, digits = 0) => (Number.isFinite(v) ? v.toLocaleString("en-US", { maximumFractionDigits: digits, minimumFractionDigits: digits }) : "–");
const time = (micros) => (micros >= 1000 ? `${fmt(micros / 1000, 2)} ms` : `${fmt(micros, 0)} µs`);

function stats(node, items) {
  node.replaceChildren(...items.map(([k, v]) => el("div", {}, el("dt", {}, k), el("dd", {}, v))));
}

/* Explore */

const explore = {
  view: null,
  algorithm: "astar",
  result: null,
  frame: 0,

  init() {
    this.view = new GridView($("#explore-canvas"));
    $("#explore-algos").replaceChildren(
      ...algorithms.map((a) =>
        el("button", {
          type: "button", role: "radio", "aria-checked": String(a.key === this.algorithm), "data-key": a.key,
          onclick: () => this.choose(a.key),
        }, a.name)),
    );
    $("#explore-run").addEventListener("click", () => this.run(false));
    $("#explore-instant").addEventListener("click", () => this.run(true));
    $("#explore-clear").addEventListener("click", () => this.reset());
    this.bindDrawing();
    this.choose(this.algorithm);
  },

  choose(key) {
    this.algorithm = key;
    $$("#explore-algos button").forEach((b) => b.setAttribute("aria-checked", String(b.dataset.key === key)));
    const a = algorithms.find((x) => x.key === key);
    $("#explore-summary").replaceChildren(el("strong", {}, `${a.name}. `), a.summary, ` Cheapest path: ${a.optimal}.`);
    this.reset();
  },

  reset() {
    cancelAnimationFrame(this.frame);
    this.result = null;
    if (this.view?.map) this.view.clearResult();
    stats($("#explore-stats"), [["Cells expanded", "–"], ["Frontier at most", "–"], ["Moves", "–"], ["Cost", "–"], ["Time in Go", "–"]]);
  },

  run(instant) {
    this.reset();
    try {
      this.result = call("run", { map, algorithm: this.algorithm, options: options() });
    } catch (error) {
      stats($("#explore-stats"), [["Error", error.message]]);
      return;
    }
    this.view.setResult(this.result);
    if (instant) {
      this.view.show(this.view.total);
      this.report();
      return;
    }
    let shown = 0;
    const tick = () => {
      shown += perFrame();
      this.view.show(shown);
      this.report();
      if (!this.view.done) this.frame = requestAnimationFrame(tick);
    };
    tick();
  },

  report() {
    const r = this.result;
    const done = this.view.done;
    stats($("#explore-stats"), [
      ["Cells expanded", `${fmt(this.view.shown)}${done ? "" : ` / ${fmt(r.expanded)}`}`],
      ["Frontier at most", fmt(r.maxFrontier)],
      ["Moves", done ? (r.found ? fmt(r.steps) : "no path") : "…"],
      ["Cost", done ? (r.found ? fmt(r.cost, 1) : "–") : "…"],
      ["Time in Go", time(r.micros)],
    ]);
  },

  bindDrawing() {
    const canvas = $("#explore-canvas");
    let drawing = false;
    const paint = (event) => {
      const i = this.view.cellAt(event);
      if (i < 0) return;
      if (tool === "start" || tool === "goal") {
        if (i === map.start || i === map.goal) return;
        map.cells[i] = 0;
        map[tool] = i;
      } else if (i !== map.start && i !== map.goal) {
        const value = { wall: 1, mud: 2, erase: 0 }[tool];
        if (map.cells[i] === value) return;
        map.cells[i] = value;
      }
      if (this.result) this.reset();
      race.reset();
      this.view.draw();
    };
    canvas.addEventListener("pointerdown", (event) => {
      drawing = true;
      try {
        canvas.setPointerCapture(event.pointerId);
      } catch {
        // Some pointers cannot be captured. Drawing still works.
      }
      paint(event);
    });
    canvas.addEventListener("pointermove", (event) => drawing && paint(event));
    canvas.addEventListener("pointerup", () => (drawing = false));
    canvas.addEventListener("pointercancel", () => (drawing = false));
  },
};

/* Race */

const race = {
  chosen: new Set(),
  lanes: [],
  frame: 0,

  init() {
    for (const a of algorithms) this.chosen.add(a.key);
    $("#race-algos").replaceChildren(
      ...algorithms.map((a) =>
        el("button", {
          type: "button", "aria-pressed": "true", "data-key": a.key,
          onclick: (event) => {
            if (this.chosen.has(a.key) && this.chosen.size > 1) this.chosen.delete(a.key);
            else this.chosen.add(a.key);
            event.currentTarget.setAttribute("aria-pressed", String(this.chosen.has(a.key)));
            this.reset();
          },
        }, a.name)),
    );
    $("#race-run").addEventListener("click", () => this.run());
  },

  reset() {
    cancelAnimationFrame(this.frame);
    this.lanes = [];
    $("#lanes").replaceChildren();
    $("#race-table").hidden = true;
  },

  run() {
    this.reset();
    const opts = options();
    for (const a of algorithms) {
      if (!this.chosen.has(a.key)) continue;
      const result = call("run", { map, algorithm: a.key, options: opts });
      const canvas = el("canvas");
      const count = el("span", { class: "count" });
      const badges = el("div", { class: "badges" });
      const node = el("article", { class: "lane", "data-key": a.key }, el("h3", {}, a.name, count), el("div", { class: "lane-board" }, canvas), badges);
      $("#lanes").append(node);
      const view = new GridView(canvas, { maxCell: 12, maxHeight: 240 });
      view.setMap(map);
      view.setResult(result);
      this.lanes.push({ algorithm: a, result, view, count, badges });
    }
    let shown = 0;
    const tick = () => {
      shown += perFrame();
      for (const lane of this.lanes) {
        lane.view.show(shown);
        lane.count.textContent = `${fmt(lane.view.shown)} cells`;
      }
      if (this.lanes.some((l) => !l.view.done)) this.frame = requestAnimationFrame(tick);
      else this.finish();
    };
    tick();
  },

  finish() {
    const found = this.lanes.filter((l) => l.result.found);
    const min = (list, f) => Math.min(...list.map(f));
    const fewest = min(this.lanes, (l) => l.result.expanded);
    const cheapest = min(found, (l) => l.result.cost);
    const shortest = min(found, (l) => l.result.steps);
    const fastest = min(this.lanes, (l) => l.result.micros);
    for (const lane of this.lanes) {
      const r = lane.result;
      const marks = [];
      if (!r.found) marks.push(el("span", { class: "badge none" }, "no path"));
      if (r.expanded === fewest) marks.push(el("span", { class: "badge" }, "fewest cells"));
      if (r.found && Math.abs(r.cost - cheapest) < 1e-9) marks.push(el("span", { class: "badge" }, "cheapest path"));
      if (r.found && r.steps === shortest) marks.push(el("span", { class: "badge" }, "fewest moves"));
      if (r.micros === fastest) marks.push(el("span", { class: "badge" }, "fastest"));
      lane.badges.replaceChildren(...marks);
    }
    const rows = [...this.lanes].sort((a, b) => a.result.expanded - b.result.expanded);
    $("#race-table tbody").replaceChildren(
      ...rows.map((l) =>
        el("tr", { class: l.result.found && Math.abs(l.result.cost - cheapest) < 1e-9 ? "best" : "" },
          el("td", {}, l.algorithm.name),
          el("td", { class: "num" }, fmt(l.result.expanded)),
          el("td", { class: "num" }, l.result.found ? fmt(l.result.steps) : "no path"),
          el("td", { class: "num" }, l.result.found ? fmt(l.result.cost, 1) : "–"),
          el("td", { class: "num" }, time(l.result.micros)))),
    );
    $("#race-table").hidden = false;
  },
};

/* Benchmark */

const bench = {
  init() {
    $("#bench-run").addEventListener("click", () => this.run());
  },

  async run() {
    const button = $("#bench-run");
    button.disabled = true;
    const spec = {
      kind: $("#bench-kind").value,
      w: clampSize(num("#bench-w")),
      h: clampSize(num("#bench-h")),
      density: Math.max(0, Math.min(60, num("#bench-density"))) / 100,
      maps: Math.max(1, Math.min(200, Math.round(num("#bench-maps")) || 1)),
      seed: Math.floor(Math.random() * 1e9),
      options: options(false),
    };
    $("#bench-progress").hidden = false;
    const bar = $("#bench-progress div");
    const rows = [];
    for (const [k, a] of algorithms.entries()) {
      button.textContent = `Running ${a.name}…`;
      bar.style.width = `${(k / algorithms.length) * 100}%`;
      await new Promise((r) => setTimeout(r, 20)); // let the page draw
      rows.push(call("bench", { algorithm: a.key, spec }));
    }
    bar.style.width = "100%";
    this.show(rows, spec);
    button.disabled = false;
    button.textContent = "Run the benchmark";
  },

  show(rows, spec) {
    $("#bench-table tbody").replaceChildren(
      ...rows.map((r) =>
        el("tr", {},
          el("td", {}, r.name),
          el("td", { class: "num" }, `${r.found} / ${r.maps}`),
          el("td", { class: "num" }, time(r.micros)),
          el("td", { class: "num" }, fmt(r.expanded)),
          el("td", { class: "num" }, fmt(r.steps, 1)),
          el("td", { class: "num" }, fmt(r.cost, 1)),
          el("td", { class: "num" }, `${fmt(r.extra, 1)}%`))),
    );
    $("#bench-table").hidden = false;
    const chart = (title, value, label) => {
      const top = Math.max(...rows.map(value), 1e-9);
      return el("div", { class: "chart" },
        el("h3", {}, title),
        ...rows.map((r) =>
          el("div", { class: "bar-row" },
            el("span", {}, r.name),
            el("div", {}, el("div", { class: "bar", style: { width: `${Math.max(1, (value(r) / top) * 100)}%` } })),
            el("span", { class: "value" }, label(r)))));
    };
    $("#bench-charts").replaceChildren(
      chart("Average time", (r) => r.micros, (r) => time(r.micros)),
      chart("Average cells expanded", (r) => r.expanded, (r) => fmt(r.expanded)),
      chart("Extra cost over the cheapest path", (r) => r.extra, (r) => `${fmt(r.extra, 1)}%`),
    );
    $("#bench-progress").hidden = true;
    $("#bench-table").dataset.spec = `${spec.maps} ${spec.kind} maps of ${spec.w} × ${spec.h}`;
  },
};

/* Tower of Hanoi */

const tower = {
  solution: null,
  move: 0,
  timer: 0,

  init() {
    $("#disks").addEventListener("input", () => this.solve());
    $("#hanoi-play").addEventListener("click", () => this.play());
    $("#hanoi-step").addEventListener("click", () => {
      this.stop();
      this.stepOnce();
    });
    this.solve();
  },

  solve() {
    this.stop();
    this.solution = call("hanoi", num("#disks"));
    this.move = 0;
    this.render();
  },

  rods() {
    const n = this.solution.disks;
    const rods = [[], [], []];
    for (let d = n - 1; d >= 0; d--) rods[0].push(d);
    for (const m of this.solution.moves.slice(0, this.move)) rods[m.to].push(rods[m.from].pop());
    return rods;
  },

  render() {
    const n = this.solution.disks;
    const last = this.solution.moves[this.move - 1];
    $("#rods").replaceChildren(
      ...this.rods().map((rod, r) =>
        el("div", { class: "rod" },
          ...rod.map((d) =>
            el("div", {
              class: `disk${last && last.disk === d && last.to === r ? " moved" : ""}`,
              style: { width: `${28 + (62 * (d + 1)) / n}%`, background: `hsl(${240 - (d * 200) / Math.max(1, n - 1)} 70% 55%)` },
            }, String(d + 1))),
          el("span", { class: "name" }, "ABC"[r]))),
    );
    const s = this.solution;
    stats($("#hanoi-stats"), [
      ["Move", `${this.move} / ${s.moves.length}`],
      ["Shortest solution", `${fmt(s.moves.length)} moves (2^${n} − 1)`],
      ["States expanded by BFS", fmt(s.expanded)],
      ["States reached", fmt(s.seen)],
      ["All states", `${fmt(s.states)} (3^${n})`],
    ]);
    $("#hanoi-play").textContent = this.timer ? "Pause" : this.move >= s.moves.length ? "Play again" : "Play the solution";
  },

  stepOnce() {
    if (this.move >= this.solution.moves.length) this.move = 0;
    else this.move++;
    this.render();
  },

  play() {
    if (this.timer) {
      this.stop();
      return;
    }
    if (this.move >= this.solution.moves.length) this.move = 0;
    // Short games play slowly, long games play faster.
    const every = Math.max(25, Math.min(500, 12000 / this.solution.moves.length));
    this.timer = setInterval(() => {
      this.move++;
      if (this.move >= this.solution.moves.length) this.stop();
      this.render();
    }, every);
    this.render();
  },

  stop() {
    clearInterval(this.timer);
    this.timer = 0;
    if (this.solution) this.render();
  },
};

/* Start */

async function start() {
  try {
    await loadGo();
  } catch (error) {
    $("#loading").textContent = `The page could not start: ${error.message}`;
    return;
  }
  algorithms = call("algorithms");
  const kinds = ["random", "maze", "rooms", "mud", "empty"];
  for (const id of ["#kind", "#bench-kind"]) {
    $(id).replaceChildren(...kinds.map((k) => el("option", { value: k }, k)));
  }
  $("#loading").hidden = true;
  $("#layout").hidden = false;

  bindOutputs();
  $$("#tools button").forEach((b) =>
    b.addEventListener("click", () => {
      tool = b.dataset.tool;
      $$("#tools button").forEach((x) => x.setAttribute("aria-pressed", String(x === b)));
    }));
  $("#new-map").addEventListener("click", newMap);
  $("#kind").addEventListener("change", newMap);
  $("#clear-map").addEventListener("click", () => {
    map.cells.fill(0);
    explore.reset();
    race.reset();
    explore.view.draw();
  });
  $$(".tabs button").forEach((b) => b.addEventListener("click", () => selectTab(b.dataset.tab)));
  window.addEventListener("resize", () => {
    explore.view.resize();
    race.lanes.forEach((l) => l.view.resize());
  });

  explore.init();
  race.init();
  bench.init();
  tower.init();
  newMap();
  selectTab("explore");
}

start();
