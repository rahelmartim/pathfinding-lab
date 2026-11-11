// Pathfind runs the search algorithms on a map in the terminal, and draws
// the path of each one.
//
//	go run ./cmd/pathfind -kind maze -width 41 -height 21
//	go run ./cmd/pathfind -kind mud -algorithm astar -diagonal
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rahelmartim/pathfinding-lab/grid"
	"github.com/rahelmartim/pathfinding-lab/search"
)

func main() {
	kind := flag.String("kind", "random", "map kind: "+strings.Join(grid.Kinds, ", "))
	w := flag.Int("width", 40, "columns")
	h := flag.Int("height", 20, "rows")
	density := flag.Float64("density", 0.3, "share of walls, from 0 to 0.9")
	seed := flag.Uint64("seed", 1, "seed of the map")
	algorithm := flag.String("algorithm", "", "one algorithm; empty runs all of them")
	diagonal := flag.Bool("diagonal", false, "allow diagonal moves")
	mud := flag.Int("mud", 5, "cost to enter a mud cell")
	flag.Parse()

	m, err := grid.Generate(*kind, *w, *h, *density, *seed)
	if err != nil {
		log.Fatal(err)
	}
	opt := search.Options{Diagonal: *diagonal, MudCost: *mud, Record: true}
	for _, a := range search.Algorithms {
		if *algorithm != "" && a.Key != *algorithm {
			continue
		}
		r, err := search.Run(a.Key, m, opt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: ", a.Name)
		if !r.Found {
			fmt.Printf("no path, %d cells expanded\n\n", r.Expanded)
			continue
		}
		fmt.Printf("%d moves, cost %.1f, %d cells expanded, %.0f µs\n", r.Steps, r.Cost, r.Expanded, r.Micros)
		draw(m, r)
	}
}

// draw prints the map: # wall, ~ mud, . expanded, * path, S start, G goal.
func draw(m *grid.Map, r search.Result) {
	marks := make([]byte, len(m.Cells))
	for i, c := range m.Cells {
		marks[i] = " #~"[c]
	}
	for _, i := range r.Visited {
		if marks[i] == ' ' {
			marks[i] = '.'
		}
	}
	for _, i := range r.Path {
		marks[i] = '*'
	}
	marks[m.Start], marks[m.Goal] = 'S', 'G'
	var b strings.Builder
	for y := 0; y < m.H; y++ {
		b.Write(marks[y*m.W : (y+1)*m.W])
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	os.Stdout.WriteString(b.String())
}
