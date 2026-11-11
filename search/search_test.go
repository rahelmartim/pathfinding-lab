package search

import (
	"math"
	"testing"

	"github.com/rahelmartim/pathfinding-lab/grid"
)

const eps = 1e-9

// checkPath tests that a path goes from the start to the goal in legal
// moves, through open cells only.
func checkPath(t *testing.T, m *grid.Map, r Result, diagonal bool) {
	t.Helper()
	if !r.Found {
		return
	}
	if r.Path[0] != m.Start || r.Path[len(r.Path)-1] != m.Goal {
		t.Fatalf("%s: the path goes from %d to %d", r.Algorithm, r.Path[0], r.Path[len(r.Path)-1])
	}
	seen := map[int]bool{}
	for k, i := range r.Path {
		if m.Cells[i] == grid.Wall {
			t.Fatalf("%s: the path goes through a wall", r.Algorithm)
		}
		if seen[i] {
			t.Fatalf("%s: the path visits a cell twice", r.Algorithm)
		}
		seen[i] = true
		if k == 0 {
			continue
		}
		ax, ay := m.XY(r.Path[k-1])
		bx, by := m.XY(i)
		dx, dy := abs(ax-bx), abs(ay-by)
		if dx > 1 || dy > 1 || dx+dy == 0 || (!diagonal && dx+dy == 2) {
			t.Fatalf("%s: an illegal move from (%d,%d) to (%d,%d)", r.Algorithm, ax, ay, bx, by)
		}
		if dx+dy == 2 && (!m.Passable(ax, by) || !m.Passable(bx, ay)) {
			t.Fatalf("%s: a diagonal move cuts a wall corner", r.Algorithm)
		}
	}
	if r.Steps != len(r.Path)-1 {
		t.Fatalf("%s: %d steps for a path of %d cells", r.Algorithm, r.Steps, len(r.Path))
	}
}

func abs(v int) int { return max(v, -v) }

func run(t *testing.T, key string, m *grid.Map, o Options) Result {
	t.Helper()
	r, err := Run(key, m, o)
	if err != nil {
		t.Fatal(err)
	}
	checkPath(t, m, r, o.Diagonal)
	return r
}

func TestAllAlgorithmsOnRandomMaps(t *testing.T) {
	kinds := []string{"random", "maze", "rooms", "mud", "empty"}
	for seed := uint64(1); seed <= 200; seed++ {
		kind := kinds[seed%uint64(len(kinds))]
		m, err := grid.Generate(kind, 20+int(seed%17), 15+int(seed%13), 0.3, seed)
		if err != nil {
			t.Fatal(err)
		}
		for _, diagonal := range []bool{false, true} {
			o := Options{Diagonal: diagonal, MudCost: 5}
			best := run(t, "dijkstra", m, o)
			bfsR := run(t, "bfs", m, o)
			for _, a := range Algorithms {
				r := run(t, a.Key, m, o)
				if r.Found != best.Found {
					t.Fatalf("seed %d %s: %s found=%v, Dijkstra found=%v", seed, kind, a.Name, r.Found, best.Found)
				}
				if !r.Found {
					continue
				}
				switch a.Key {
				case "astar":
					// The default heuristic never overestimates: A* is optimal.
					if math.Abs(r.Cost-best.Cost) > eps {
						t.Fatalf("seed %d %s diagonal=%v: A* cost %.3f, Dijkstra %.3f", seed, kind, diagonal, r.Cost, best.Cost)
					}
				case "weighted":
					if r.Cost > 2*best.Cost+eps {
						t.Fatalf("seed %d: weighted A* cost %.3f is more than 2 × %.3f", seed, r.Cost, best.Cost)
					}
				case "bidirectional":
					if r.Steps != bfsR.Steps {
						t.Fatalf("seed %d %s diagonal=%v: bidirectional %d steps, BFS %d", seed, kind, diagonal, r.Steps, bfsR.Steps)
					}
				}
				if r.Cost < best.Cost-eps {
					t.Fatalf("seed %d: %s found a path cheaper than Dijkstra", seed, a.Name)
				}
				if r.Expanded < 1 || r.Expanded > len(m.Cells) {
					t.Fatalf("%s expanded %d cells", a.Name, r.Expanded)
				}
			}
			// Without mud, BFS counts moves, and a move costs 1: it is optimal.
			if !diagonal && kind != "mud" && best.Found && float64(bfsR.Steps) != best.Cost {
				t.Fatalf("seed %d %s: BFS %d steps, Dijkstra cost %.1f", seed, kind, bfsR.Steps, best.Cost)
			}
		}
	}
}

func TestAStarExpandsLessThanDijkstra(t *testing.T) {
	m := grid.New(60, 40)
	d := run(t, "dijkstra", m, Options{})
	a := run(t, "astar", m, Options{})
	g := run(t, "greedy", m, Options{})
	if !(a.Expanded < d.Expanded/4) || g.Expanded > a.Expanded {
		t.Fatalf("expanded: Dijkstra %d, A* %d, greedy %d", d.Expanded, a.Expanded, g.Expanded)
	}
	// On an open map, the straight line is the cheapest path.
	if a.Steps != 57 || a.Cost != 57 {
		t.Fatalf("A*: %d steps, cost %.1f; want 57", a.Steps, a.Cost)
	}
}

func TestMudChangesTheCheapestPath(t *testing.T) {
	m := grid.New(21, 9)
	// A wall of mud between the start and the goal, with a dry way around.
	for y := 0; y < 8; y++ {
		m.Cells[m.Index(10, y)] = grid.Mud
	}
	o := Options{MudCost: 10}
	d := run(t, "dijkstra", m, o)
	b := run(t, "bfs", m, o)
	if b.Steps != 18 || b.Cost != 27 {
		t.Fatalf("BFS goes through the mud: %d steps, cost %.0f; want 18 and 27", b.Steps, b.Cost)
	}
	if d.Cost >= b.Cost || d.Steps <= b.Steps {
		t.Fatalf("Dijkstra must go around: cost %.0f (BFS %.0f), steps %d (BFS %d)", d.Cost, b.Cost, d.Steps, b.Steps)
	}
	for _, i := range d.Path {
		if m.Cells[i] == grid.Mud {
			t.Fatal("Dijkstra walked into the mud")
		}
	}
}

func TestNoPath(t *testing.T) {
	m := grid.New(15, 9)
	gx, gy := m.XY(m.Goal)
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		m.Cells[m.Index(gx+d[0], gy+d[1])] = grid.Wall
	}
	for _, a := range Algorithms {
		r := run(t, a.Key, m, Options{Diagonal: true})
		if r.Found || r.Path != nil {
			t.Fatalf("%s found a path to a closed goal", a.Name)
		}
	}
}

func TestDiagonalsDoNotCutCorners(t *testing.T) {
	// The goal is one diagonal step away, but both corners are walls.
	m := grid.New(5, 5)
	m.Start, m.Goal = m.Index(1, 1), m.Index(2, 2)
	m.Cells[m.Index(2, 1)] = grid.Wall
	m.Cells[m.Index(1, 2)] = grid.Wall
	r := run(t, "astar", m, Options{Diagonal: true})
	if !r.Found || r.Steps == 1 {
		t.Fatalf("A* cut a corner, or found nothing: %+v", r.Path)
	}
}

func TestRecordAndOptions(t *testing.T) {
	m, _ := grid.Generate("random", 30, 20, 0.25, 9)
	r := run(t, "astar", m, Options{Record: true})
	if len(r.Visited) != r.Expanded || len(r.Seen) != len(r.SeenAt) || len(r.Seen) < r.Expanded {
		t.Fatalf("record: %d visited, %d expanded, %d seen", len(r.Visited), r.Expanded, len(r.Seen))
	}
	if r.Visited[0] != m.Start || r.Visited[len(r.Visited)-1] != m.Goal {
		t.Fatal("the record must start at the start and end at the goal")
	}
	quiet := run(t, "astar", m, Options{})
	if quiet.Visited != nil || quiet.Expanded != r.Expanded {
		t.Fatal("without Record, keep the counts but no cells")
	}

	o := Options{Weight: 99, MudCost: -3, Heuristic: "nope", Diagonal: true}.Normalized()
	if o.Weight != 10 || o.MudCost != 1 || o.Heuristic != "octile" {
		t.Fatalf("normalized %+v", o)
	}
	if _, err := Run("nope", m, Options{}); err == nil {
		t.Fatal("an unknown algorithm must fail")
	}
	bad := m.Clone()
	bad.Cells = bad.Cells[:3]
	if _, err := Run("bfs", bad, Options{}); err == nil {
		t.Fatal("a bad map must fail")
	}
}

func TestEveryHeuristicWorks(t *testing.T) {
	m, _ := grid.Generate("random", 40, 30, 0.3, 4)
	best := run(t, "dijkstra", m, Options{Diagonal: true})
	for _, h := range []string{"manhattan", "euclidean", "octile", "chebyshev"} {
		r := run(t, "astar", m, Options{Diagonal: true, Heuristic: h})
		if r.Found != best.Found {
			t.Fatalf("%s: found=%v", h, r.Found)
		}
		// Euclidean, octile, and Chebyshev never overestimate with diagonals.
		if h != "manhattan" && math.Abs(r.Cost-best.Cost) > eps {
			t.Fatalf("A* with %s: cost %.3f, Dijkstra %.3f", h, r.Cost, best.Cost)
		}
	}
}

func BenchmarkAlgorithms(b *testing.B) {
	m, _ := grid.Generate("random", 120, 80, 0.3, 1)
	for _, a := range Algorithms {
		b.Run(a.Key, func(b *testing.B) {
			for b.Loop() {
				if _, err := Run(a.Key, m, Options{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestBench(t *testing.T) {
	spec := BenchSpec{Kind: "random", W: 30, H: 20, Density: 0.3, Maps: 15, Seed: 5}
	rows := map[string]BenchRow{}
	for _, a := range Algorithms {
		row, err := Bench(a.Key, spec)
		if err != nil {
			t.Fatal(err)
		}
		if row.Maps != 15 || row.Found == 0 || row.Expanded == 0 {
			t.Fatalf("%s: %+v", a.Name, row)
		}
		rows[a.Key] = row
	}
	for _, key := range []string{"dijkstra", "astar"} {
		if rows[key].Extra > eps {
			t.Fatalf("%s is optimal, but costs %.2f%% more", key, rows[key].Extra)
		}
	}
	if rows["weighted"].Extra > 100 || rows["dfs"].Extra <= rows["astar"].Extra {
		t.Fatalf("extra cost: weighted %.1f%%, DFS %.1f%%", rows["weighted"].Extra, rows["dfs"].Extra)
	}
	for _, a := range Algorithms {
		if rows[a.Key].Found != rows["dijkstra"].Found {
			t.Fatalf("%s found %d paths, Dijkstra %d", a.Name, rows[a.Key].Found, rows["dijkstra"].Found)
		}
	}
	if _, err := Bench("nope", spec); err == nil {
		t.Fatal("an unknown algorithm must fail")
	}
}
