//go:build js && wasm

// The WebAssembly build for the page. It sets a global object "lab" with
// functions that take and return JSON text:
//
//	lab.algorithms()   → [{key, name, summary, optimal}]
//	lab.generate(json) → a map, from {kind, w, h, density, seed}
//	lab.run(json)      → a result, from {map, algorithm, options}
//	lab.bench(json)    → a row, from {algorithm, spec}
//	lab.hanoi(disks)   → a solution
//
// Every reply is {"ok": true, "result": ...} or {"ok": false, "error": "..."}.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/rahelmartim/pathfinding-lab/grid"
	"github.com/rahelmartim/pathfinding-lab/hanoi"
	"github.com/rahelmartim/pathfinding-lab/search"
)

// toJSON writes the cells as numbers. Go would write []Cell as numbers
// too, but the page reads one shape for maps in both directions.
func toJSON(m *grid.Map) mapJSON {
	cells := make([]int, len(m.Cells))
	for i, c := range m.Cells {
		cells[i] = int(c)
	}
	return mapJSON{W: m.W, H: m.H, Cells: cells, Start: m.Start, Goal: m.Goal}
}

func reply(result any, err error) string {
	out := map[string]any{"ok": err == nil}
	if err != nil {
		out["error"] = err.Error()
	} else {
		out["result"] = result
	}
	data, _ := json.Marshal(out)
	return string(data)
}

// handler reads the JSON argument into T and calls fn.
func handler[T any](fn func(T) (any, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		var in T
		if len(args) > 0 {
			if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
				return reply(nil, err)
			}
		}
		return reply(fn(in))
	})
}

// mapJSON is a map as the page sends and reads it.
type mapJSON struct {
	W     int   `json:"w"`
	H     int   `json:"h"`
	Cells []int `json:"cells"`
	Start int   `json:"start"`
	Goal  int   `json:"goal"`
}

// toMap converts it. Validate, in search.Run, refuses unknown cells.
func (j mapJSON) toMap() *grid.Map {
	m := &grid.Map{W: j.W, H: j.H, Cells: make([]grid.Cell, len(j.Cells)), Start: j.Start, Goal: j.Goal}
	for i, v := range j.Cells {
		m.Cells[i] = grid.Cell(max(0, min(255, v)))
	}
	return m
}

func main() {
	api := js.Global().Get("Object").New()
	api.Set("algorithms", js.FuncOf(func(js.Value, []js.Value) any {
		return reply(search.Algorithms, nil)
	}))
	api.Set("generate", handler(func(in struct {
		Kind    string  `json:"kind"`
		W       int     `json:"w"`
		H       int     `json:"h"`
		Density float64 `json:"density"`
		Seed    uint64  `json:"seed"`
	}) (any, error) {
		m, err := grid.Generate(in.Kind, in.W, in.H, in.Density, in.Seed)
		if err != nil {
			return nil, err
		}
		return toJSON(m), nil
	}))
	api.Set("run", handler(func(in struct {
		Map       mapJSON        `json:"map"`
		Algorithm string         `json:"algorithm"`
		Options   search.Options `json:"options"`
	}) (any, error) {
		return search.Run(in.Algorithm, in.Map.toMap(), in.Options)
	}))
	api.Set("bench", handler(func(in struct {
		Algorithm string           `json:"algorithm"`
		Spec      search.BenchSpec `json:"spec"`
	}) (any, error) {
		return search.Bench(in.Algorithm, in.Spec)
	}))
	api.Set("hanoi", js.FuncOf(func(_ js.Value, args []js.Value) any {
		disks := 3
		if len(args) > 0 {
			disks = args[0].Int()
		}
		return reply(hanoi.BFS(disks))
	}))
	js.Global().Set("lab", api)
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("lab-ready"))
	select {}
}
