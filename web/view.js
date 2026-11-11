// GridView draws a map and the progress of a search on a canvas. It draws
// one pixel for each cell into a small image, then scales it up, so that
// a 200 × 200 map draws fast.

const COLORS = {
  open: [255, 255, 255],
  wall: [43, 48, 68],
  mud: [168, 123, 79],
  frontier: [251, 191, 36],
  start: [22, 163, 74],
  goal: [220, 38, 38],
};

// visitColor goes from light blue (early) to indigo (late).
function visitColor(t, mud) {
  const a = [165, 216, 255];
  const b = [129, 140, 248];
  const c = a.map((v, i) => Math.round(v + (b[i] - v) * t));
  // Mud stays a little brown under the search.
  return mud ? c.map((v, i) => Math.round(v * 0.6 + COLORS.mud[i] * 0.4)) : c;
}

export class GridView {
  constructor(canvas, { maxCell = 24, maxHeight = 620 } = {}) {
    this.canvas = canvas;
    this.context = canvas.getContext("2d");
    this.small = document.createElement("canvas");
    this.smallContext = this.small.getContext("2d");
    this.maxCell = maxCell;
    this.maxHeight = maxHeight;
    this.map = null;
    this.result = null;
    this.shown = 0;
  }

  setMap(map) {
    this.map = map;
    this.result = null;
    this.shown = 0;
    this.small.width = map.w;
    this.small.height = map.h;
    this.image = this.smallContext.createImageData(map.w, map.h);
    this.resize();
  }

  // setResult keeps the order of the search, for show().
  setResult(result) {
    this.result = result;
    const n = this.map.w * this.map.h;
    this.visitStep = new Int32Array(n).fill(-1);
    (result.visited ?? []).forEach((cell, k) => {
      if (this.visitStep[cell] === -1) this.visitStep[cell] = k;
    });
    this.seenStep = new Int32Array(n).fill(-1);
    (result.seen ?? []).forEach((cell, k) => {
      if (this.seenStep[cell] === -1) this.seenStep[cell] = result.seenAt[k];
    });
    this.shown = 0;
  }

  clearResult() {
    this.result = null;
    this.shown = 0;
    this.draw();
  }

  get total() {
    return this.result?.visited?.length ?? 0;
  }

  get done() {
    return !this.result || this.shown >= this.total;
  }

  // show draws the first n expansions. The path appears after the last one.
  show(n) {
    this.shown = Math.min(n, this.total);
    this.draw();
  }

  resize() {
    const width = this.canvas.parentElement.clientWidth - 16;
    const cell = Math.max(2, Math.min(this.maxCell, Math.floor(width / this.map.w), Math.floor(this.maxHeight / this.map.h)));
    this.cell = cell;
    const dpr = window.devicePixelRatio || 1;
    this.canvas.width = this.map.w * cell * dpr;
    this.canvas.height = this.map.h * cell * dpr;
    this.canvas.style.width = `${this.map.w * cell}px`;
    this.canvas.style.height = `${this.map.h * cell}px`;
    this.context.setTransform(dpr, 0, 0, dpr, 0, 0);
    this.draw();
  }

  draw() {
    const { w, h, cells, start, goal } = this.map;
    const data = this.image.data;
    const t = this.shown;
    const searching = this.result && t > 0;
    for (let i = 0; i < w * h; i++) {
      let c = cells[i] === 1 ? COLORS.wall : cells[i] === 2 ? COLORS.mud : COLORS.open;
      if (searching && cells[i] !== 1) {
        const v = this.visitStep[i];
        if (v !== -1 && v < t) c = visitColor(v / Math.max(1, this.total), cells[i] === 2);
        else if (this.seenStep[i] !== -1 && this.seenStep[i] <= t) c = COLORS.frontier;
      }
      if (i === start) c = COLORS.start;
      if (i === goal) c = COLORS.goal;
      data.set(c, i * 4);
      data[i * 4 + 3] = 255;
    }
    this.smallContext.putImageData(this.image, 0, 0);

    const ctx = this.context;
    const s = this.cell;
    ctx.imageSmoothingEnabled = false;
    ctx.clearRect(0, 0, w * s, h * s);
    ctx.drawImage(this.small, 0, 0, w * s, h * s);

    if (s >= 8) {
      ctx.strokeStyle = "rgba(28, 32, 48, 0.07)";
      ctx.lineWidth = 1;
      ctx.beginPath();
      for (let x = 1; x < w; x++) {
        ctx.moveTo(x * s + 0.5, 0);
        ctx.lineTo(x * s + 0.5, h * s);
      }
      for (let y = 1; y < h; y++) {
        ctx.moveTo(0, y * s + 0.5);
        ctx.lineTo(w * s, y * s + 0.5);
      }
      ctx.stroke();
    }

    if (this.result?.found && this.shown >= this.total) {
      ctx.strokeStyle = "#f97316";
      ctx.lineWidth = Math.max(2, s * 0.35);
      ctx.lineJoin = "round";
      ctx.lineCap = "round";
      ctx.beginPath();
      this.result.path.forEach((cell, k) => {
        const x = (cell % w) * s + s / 2;
        const y = Math.floor(cell / w) * s + s / 2;
        if (k === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.stroke();
    }
  }

  // cellAt returns the cell under a pointer event, or -1.
  cellAt(event) {
    const rect = this.canvas.getBoundingClientRect();
    const x = Math.floor((event.clientX - rect.left) / (rect.width / this.map.w));
    const y = Math.floor((event.clientY - rect.top) / (rect.height / this.map.h));
    if (x < 0 || y < 0 || x >= this.map.w || y >= this.map.h) return -1;
    return y * this.map.w + x;
  }
}
