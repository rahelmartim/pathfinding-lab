package search

import (
	"fmt"
	"math"

	"github.com/rahelmartim/pathfinding-lab/grid"
)

// BenchSpec describes a set of random maps to test the algorithms on.
type BenchSpec struct {
	Kind    string  `json:"kind"`
	W       int     `json:"w"`
	H       int     `json:"h"`
	Density float64 `json:"density"`
	Maps    int     `json:"maps"` // from 1 to 200
	Seed    uint64  `json:"seed"`
	Options Options `json:"options"`
}

// BenchRow is the result of one algorithm on all the maps.
type BenchRow struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Maps  int    `json:"maps"`
	Found int    `json:"found"`
	// Averages over the maps with a path.
	Micros   float64 `json:"micros"`
	Expanded float64 `json:"expanded"`
	Steps    float64 `json:"steps"`
	Cost     float64 `json:"cost"`
	// Extra is the average extra cost over the cheapest path, in percent.
	Extra float64 `json:"extra"`
}

// Bench runs one algorithm on the maps of the spec. The same spec makes
// the same maps, so the rows of the algorithms compare.
func Bench(key string, spec BenchSpec) (BenchRow, error) {
	a, ok := Find(key)
	if !ok {
		return BenchRow{}, fmt.Errorf("unknown algorithm %q", key)
	}
	spec.Maps = max(1, min(200, spec.Maps))
	opt := spec.Options
	opt.Record = false
	row := BenchRow{Key: a.Key, Name: a.Name, Maps: spec.Maps}
	for k := 0; k < spec.Maps; k++ {
		m, err := grid.Generate(spec.Kind, spec.W, spec.H, spec.Density, spec.Seed+uint64(k))
		if err != nil {
			return row, err
		}
		r, err := Run(key, m, opt)
		if err != nil {
			return row, err
		}
		if !r.Found {
			continue
		}
		best, err := Run("dijkstra", m, opt)
		if err != nil {
			return row, err
		}
		row.Found++
		row.Micros += r.Micros
		row.Expanded += float64(r.Expanded)
		row.Steps += float64(r.Steps)
		row.Cost += r.Cost
		if best.Cost > 0 {
			row.Extra += (r.Cost/best.Cost - 1) * 100
		}
	}
	if row.Found > 0 {
		n := float64(row.Found)
		row.Micros /= n
		row.Expanded /= n
		row.Steps /= n
		row.Cost /= n
		row.Extra = math.Max(0, row.Extra/n)
	}
	return row, nil
}
