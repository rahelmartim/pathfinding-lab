// Tests the page in a headless Chrome, on the build in site/ (run
// "make web" first).

import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { after, before, describe, test } from "node:test";
import { findChrome, launchChrome, serveApp } from "./browser.js";

const chromePath = findChrome();
const built = existsSync(resolve(import.meta.dirname, "..", "..", "site", "lab.wasm"));
const skip = !chromePath ? "Chrome is not installed" : !built ? "run make web first" : false;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

describe("Pathfinding Lab in the browser", { skip }, () => {
  let chrome;
  let app;
  let page;

  before(async () => {
    app = await serveApp();
    chrome = await launchChrome(chromePath);
    page = await chrome.newPage({ width: 1400, height: 1000 });
    await page.goto(app.url);
    await page.waitFor("!document.querySelector('#layout').hidden", 30000);
  });

  after(async () => {
    page?.close();
    await chrome?.close();
    app?.server.close();
  });

  const text = (selector) => page.eval(`document.querySelector(${JSON.stringify(selector)})?.textContent ?? ""`);
  const stat = (name) =>
    page.eval(`[...document.querySelectorAll('#explore-stats div')].find((d) => d.querySelector('dt').textContent === ${JSON.stringify(name)})?.querySelector('dd').textContent`);
  const chooseAlgorithm = (name) => page.eval(`[...document.querySelectorAll('#explore-algos button')].find((b) => b.textContent === ${JSON.stringify(name)}).click(), true`);
  const tab = (name) => page.clickText(".tabs button", name);

  // orange counts the path pixels on a canvas.
  const orange = (selector) =>
    page.eval(`(() => {
      const c = document.querySelector(${JSON.stringify(selector)});
      const d = c.getContext('2d').getImageData(0, 0, c.width, c.height).data;
      let n = 0;
      for (let i = 0; i < d.length; i += 4) if (d[i] > 230 && d[i + 1] > 95 && d[i + 1] < 135 && d[i + 2] < 60) n++;
      return n;
    })()`);

  // drawColumn drags the pointer down one column of the map.
  const drawColumn = (x) =>
    page.eval(`(() => {
      const c = document.querySelector('#explore-canvas');
      const r = c.getBoundingClientRect();
      const w = Number(document.querySelector('#width').value);
      const h = Number(document.querySelector('#height').value);
      const at = (y) => ({ clientX: r.left + (${x} + 0.5) * (r.width / w), clientY: r.top + (y + 0.5) * (r.height / h), bubbles: true, pointerId: 1 });
      c.dispatchEvent(new PointerEvent('pointerdown', at(0)));
      for (let y = 1; y < h; y++) c.dispatchEvent(new PointerEvent('pointermove', at(y)));
      c.dispatchEvent(new PointerEvent('pointerup', at(h - 1)));
      return true;
    })()`);

  test("starts with seven algorithms and a map", async () => {
    assert.equal(await page.eval("document.querySelectorAll('#explore-algos button').length"), 7);
    assert.ok((await page.eval("document.querySelector('#explore-canvas').width")) > 100);
    assert.match(await text("#explore-summary"), /A\*/);
  });

  test("A* on an open map goes in a straight line", async () => {
    await page.click("#clear-map");
    await chooseAlgorithm("A*");
    await page.click("#explore-instant");
    assert.equal(await stat("Moves"), "38");
    assert.equal(await stat("Cost"), "38.0");
    assert.ok((await orange("#explore-canvas")) > 100, "the path is drawn");
  });

  test("the animation shows the expansions, then the path", async () => {
    await chooseAlgorithm("Dijkstra");
    await page.eval("document.querySelector('#speed').value = 0; document.querySelector('#speed').dispatchEvent(new Event('input')); true");
    await page.click("#explore-run");
    await sleep(150);
    assert.match(await stat("Cells expanded"), /^\d+ \/ [\d,]+$/);
    assert.equal(await stat("Moves"), "…");
    await page.eval("document.querySelector('#speed').value = 10; document.querySelector('#speed').dispatchEvent(new Event('input')); true");
    await page.click("#explore-run");
    await page.waitFor("[...document.querySelectorAll('#explore-stats dd')][2].textContent === '38'", 10000);
  });

  test("a wall drawn across the map blocks every search", async () => {
    await drawColumn(20);
    for (const name of ["BFS", "A*", "Bidirectional BFS"]) {
      await chooseAlgorithm(name);
      await page.click("#explore-instant");
      assert.equal(await stat("Moves"), "no path", name);
    }
    // Erase one cell of the wall: the searches find the gap.
    await page.clickText("#tools button", "Erase");
    await page.eval(`(() => {
      const c = document.querySelector('#explore-canvas');
      const r = c.getBoundingClientRect();
      const e = { clientX: r.left + 20.5 * (r.width / 41), clientY: r.top + 3.5 * (r.height / 25), bubbles: true, pointerId: 1 };
      c.dispatchEvent(new PointerEvent('pointerdown', e));
      c.dispatchEvent(new PointerEvent('pointerup', e));
      return true;
    })()`);
    await chooseAlgorithm("BFS");
    await page.click("#explore-instant");
    assert.equal(await stat("Moves"), "56"); // 38 across, and 2 × 9 rows to the gap and back
    await page.clickText("#tools button", "Wall");
  });

  test("the race runs every algorithm on the same map", async () => {
    await page.click("#new-map");
    await tab("Race");
    await page.eval("document.querySelector('#speed').value = 10; document.querySelector('#speed').dispatchEvent(new Event('input')); true");
    await page.click("#race-run");
    await page.waitFor("!document.querySelector('#race-table').hidden", 20000);
    assert.equal(await page.eval("document.querySelectorAll('.lane').length"), 7);
    assert.equal(await page.eval("document.querySelectorAll('#race-table tbody tr').length"), 7);
    const badges = (key) => page.eval(`document.querySelector('.lane[data-key="${key}"] .badges').textContent`);
    const dijkstra = await badges("dijkstra");
    assert.match(dijkstra, /cheapest path|no path/);
    if (!/no path/.test(dijkstra)) assert.match(await badges("astar"), /cheapest path/);
  });

  test("the race can leave algorithms out", async () => {
    await page.clickText("#race-algos button", "DFS");
    await page.clickText("#race-algos button", "Greedy best-first");
    await page.click("#race-run");
    await page.waitFor("!document.querySelector('#race-table').hidden", 20000);
    assert.equal(await page.eval("document.querySelectorAll('.lane').length"), 5);
    assert.equal(await page.eval("document.querySelector('.lane[data-key=\"dfs\"]')"), null);
  });

  test("the benchmark compares the algorithms on many maps", async () => {
    await tab("Benchmark");
    assert.equal(await page.eval("document.querySelector('[data-for=\"explore race\"]').hidden"), true);
    await page.eval("document.querySelector('#bench-maps').value = 6; true");
    await page.click("#bench-run");
    await page.waitFor("!document.querySelector('#bench-table').hidden", 60000);
    const rows = await page.eval("[...document.querySelectorAll('#bench-table tbody tr')].map((r) => [...r.children].map((c) => c.textContent))");
    assert.equal(rows.length, 7);
    const row = (name) => rows.find((r) => r[0] === name);
    assert.equal(row("Dijkstra")[6], "0.0%");
    assert.equal(row("A*")[6], "0.0%");
    assert.match(row("BFS")[1], /^\d+ \/ 6$/);
    assert.equal(await page.eval("document.querySelectorAll('#bench-charts .chart').length"), 3);
  });

  test("the Tower of Hanoi plays the BFS solution", async () => {
    await tab("Tower of Hanoi");
    assert.equal(await page.eval("document.querySelector('#settings').hidden"), true);
    await page.eval("document.querySelector('#disks').value = 3; document.querySelector('#disks').dispatchEvent(new Event('input')); true");
    assert.match(await text("#hanoi-stats"), /7 moves/);
    assert.match(await text("#hanoi-stats"), /27 \(3\^3\)/);
    for (let k = 0; k < 7; k++) await page.click("#hanoi-step");
    const rods = await page.eval("[...document.querySelectorAll('.rod')].map((r) => r.querySelectorAll('.disk').length)");
    assert.deepEqual(rods, [0, 0, 3]);
    assert.match(await text("#hanoi-stats"), /7 \/ 7/);
  });

  test("the page had no JavaScript errors", () => {
    assert.deepEqual(page.errors, []);
  });
});
