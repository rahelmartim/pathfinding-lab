// Package search finds paths on a grid.Map with classic search algorithms,
// and records how each one explores the map.
package search

import (
	"container/heap"
	"fmt"
	"math"
	"time"

	"github.com/rahelmartim/pathfinding-lab/grid"
)

// Options changes how a search moves and what it records.
type Options struct {
	Diagonal  bool    `json:"diagonal"`  // also move to the 4 diagonal neighbors
	Heuristic string  `json:"heuristic"` // manhattan, euclidean, octile, chebyshev, or "" for the best fit
	Weight    float64 `json:"weight"`    // the weight of weighted A*, from 1 to 10
	MudCost   int     `json:"mudCost"`   // the cost to enter a mud cell, from 1 to 20
	Record    bool    `json:"record"`    // keep the order of the cells, for an animation
}

// Normalized fills in the defaults and keeps the values in their limits.
func (o Options) Normalized() Options {
	if o.Weight == 0 {
		o.Weight = 2
	}
	o.Weight = math.Max(1, math.Min(10, o.Weight))
	if o.MudCost == 0 {
		o.MudCost = 5
	}
	o.MudCost = max(1, min(20, o.MudCost))
	switch o.Heuristic {
	case "manhattan", "euclidean", "octile", "chebyshev":
	default:
		o.Heuristic = "manhattan"
		if o.Diagonal {
			o.Heuristic = "octile"
		}
	}
	return o
}

// Result is what a search found, and how much work it did.
type Result struct {
	Algorithm   string  `json:"algorithm"`
	Found       bool    `json:"found"`
	Path        []int   `json:"path"`     // cells from the start to the goal
	Steps       int     `json:"steps"`    // moves in the path
	Cost        float64 `json:"cost"`     // the sum of the move costs of the path
	Expanded    int     `json:"expanded"` // cells that the search took out of its frontier
	MaxFrontier int     `json:"maxFrontier"`
	Micros      float64 `json:"micros"` // the time of the search, in microseconds

	// With Options.Record:
	Visited []int `json:"visited"` // the expanded cells, in order
	Seen    []int `json:"seen"`    // the cells that joined the frontier, in order
	SeenAt  []int `json:"seenAt"`  // for each seen cell, the expansions before it
}

// Algorithm is one search algorithm.
type Algorithm struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Optimal string `json:"optimal"` // when it finds the cheapest path
	run     func(*problem)
}

// Algorithms lists every algorithm, in the order of the page.
var Algorithms = []Algorithm{
	{"bfs", "BFS", "Breadth-first search. It explores the map in rings around the start.", "fewest moves, but it does not see the cost of mud", bfs},
	{"dfs", "DFS", "Depth-first search. It follows one way to its end before it tries another.", "never: its path is often long", dfs},
	{"dijkstra", "Dijkstra", "Uniform-cost search. It always expands the cheapest cell so far.", "always", func(p *problem) { p.bestFirst(func(g, h float64) float64 { return g }) }},
	{"astar", "A*", "Dijkstra plus an estimate of the cost to the goal: f = g + h.", "when the heuristic never overestimates", func(p *problem) { p.bestFirst(func(g, h float64) float64 { return g + h }) }},
	{"weighted", "Weighted A*", "A* that trusts the estimate more: f = g + w·h. It is faster but not optimal.", "no: at most w times the cheapest cost", func(p *problem) { p.bestFirst(func(g, h float64) float64 { return g + p.opt.Weight*h }) }},
	{"greedy", "Greedy best-first", "It always goes to the cell that looks closest to the goal: f = h.", "never", func(p *problem) { p.bestFirst(func(g, h float64) float64 { return h }) }},
	{"bidirectional", "Bidirectional BFS", "Two BFS searches, from the start and from the goal, that meet in the middle.", "fewest moves, but it does not see the cost of mud", bidirectional},
}

// Find returns the algorithm with the key.
func Find(key string) (Algorithm, bool) {
	for _, a := range Algorithms {
		if a.Key == key {
			return a, true
		}
	}
	return Algorithm{}, false
}

// Run searches the map with one algorithm.
func Run(key string, m *grid.Map, opt Options) (Result, error) {
	a, ok := Find(key)
	if !ok {
		return Result{}, fmt.Errorf("unknown algorithm %q", key)
	}
	if err := m.Validate(); err != nil {
		return Result{}, err
	}
	p := newProblem(m, opt.Normalized())
	p.res.Algorithm = a.Name
	start := time.Now()
	a.run(p)
	p.res.Micros = float64(time.Since(start).Nanoseconds()) / 1000
	if p.res.Found {
		p.res.Steps = len(p.res.Path) - 1
		p.res.Cost = p.pathCost(p.res.Path)
	}
	return p.res, nil
}

/* The problem: the map, the moves, and the record */

type edge struct {
	to   int
	cost float64
}

type problem struct {
	m      *grid.Map
	opt    Options
	res    Result
	parent []int32
	edges  []edge
	gx, gy int
}

func newProblem(m *grid.Map, opt Options) *problem {
	p := &problem{m: m, opt: opt, parent: make([]int32, len(m.Cells))}
	for i := range p.parent {
		p.parent[i] = -1
	}
	p.gx, p.gy = m.XY(m.Goal)
	return p
}

var (
	straight = [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	diagonal = [][2]int{{1, -1}, {1, 1}, {-1, 1}, {-1, -1}}
)

// enterCost is the cost to enter a cell: 1, or the mud cost.
func (p *problem) enterCost(i int) float64 {
	if p.m.Cells[i] == grid.Mud {
		return float64(p.opt.MudCost)
	}
	return 1
}

// neighbors returns the moves from a cell. A diagonal move needs both
// straight cells next to it to be open, so it never cuts a wall corner.
func (p *problem) neighbors(i int) []edge {
	p.edges = p.edges[:0]
	x, y := p.m.XY(i)
	for _, d := range straight {
		if p.m.Passable(x+d[0], y+d[1]) {
			n := p.m.Index(x+d[0], y+d[1])
			p.edges = append(p.edges, edge{n, p.enterCost(n)})
		}
	}
	if p.opt.Diagonal {
		for _, d := range diagonal {
			if p.m.Passable(x+d[0], y+d[1]) && p.m.Passable(x+d[0], y) && p.m.Passable(x, y+d[1]) {
				n := p.m.Index(x+d[0], y+d[1])
				p.edges = append(p.edges, edge{n, math.Sqrt2 * p.enterCost(n)})
			}
		}
	}
	return p.edges
}

func (p *problem) heuristic(i int) float64 {
	x, y := p.m.XY(i)
	dx, dy := math.Abs(float64(x-p.gx)), math.Abs(float64(y-p.gy))
	switch p.opt.Heuristic {
	case "euclidean":
		return math.Hypot(dx, dy)
	case "octile":
		return math.Max(dx, dy) + (math.Sqrt2-1)*math.Min(dx, dy)
	case "chebyshev":
		return math.Max(dx, dy)
	}
	return dx + dy
}

func (p *problem) expand(i int) {
	p.res.Expanded++
	if p.opt.Record {
		p.res.Visited = append(p.res.Visited, i)
	}
}

func (p *problem) see(i int) {
	if p.opt.Record {
		p.res.Seen = append(p.res.Seen, i)
		p.res.SeenAt = append(p.res.SeenAt, p.res.Expanded)
	}
}

func (p *problem) frontier(n int) {
	p.res.MaxFrontier = max(p.res.MaxFrontier, n)
}

// finish follows the parents from the goal back to the start.
func (p *problem) finish() {
	p.res.Found = true
	var path []int
	for i := p.m.Goal; i != -1; i = int(p.parent[i]) {
		path = append(path, i)
		if i == p.m.Start {
			break
		}
	}
	for l, r := 0, len(path)-1; l < r; l, r = l+1, r-1 {
		path[l], path[r] = path[r], path[l]
	}
	p.res.Path = path
}

func (p *problem) pathCost(path []int) float64 {
	cost := 0.0
	for k := 1; k < len(path); k++ {
		ax, ay := p.m.XY(path[k-1])
		bx, by := p.m.XY(path[k])
		step := 1.0
		if ax != bx && ay != by {
			step = math.Sqrt2
		}
		cost += step * p.enterCost(path[k])
	}
	return cost
}

/* The algorithms */

// step:bfs:start
func bfs(p *problem) {
	seen := make([]bool, len(p.m.Cells))
	queue := []int{p.m.Start}
	seen[p.m.Start] = true
	p.see(p.m.Start)
	for len(queue) > 0 {
		p.frontier(len(queue))
		i := queue[0]
		queue = queue[1:]
		p.expand(i)
		if i == p.m.Goal {
			p.finish()
			return
		}
		for _, e := range p.neighbors(i) {
			if !seen[e.to] {
				seen[e.to] = true
				p.parent[e.to] = int32(i)
				p.see(e.to)
				queue = append(queue, e.to)
			}
		}
	}
}

// step:bfs:end

// step:dfs:start
func dfs(p *problem) {
	done := make([]bool, len(p.m.Cells))
	stack := []int{p.m.Start}
	p.see(p.m.Start)
	for len(stack) > 0 {
		p.frontier(len(stack))
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if done[i] {
			continue
		}
		done[i] = true
		p.expand(i)
		if i == p.m.Goal {
			p.finish()
			return
		}
		// Push in reverse, so that the first neighbor comes out first.
		edges := p.neighbors(i)
		for k := len(edges) - 1; k >= 0; k-- {
			if n := edges[k].to; !done[n] {
				p.parent[n] = int32(i)
				p.see(n)
				stack = append(stack, n)
			}
		}
	}
}

// step:dfs:end

type item struct {
	cell    int
	f, g, h float64
}

type queue []item

func (q queue) Len() int { return len(q) }
func (q queue) Less(a, b int) bool {
	if q[a].f != q[b].f {
		return q[a].f < q[b].f
	}
	return q[a].h < q[b].h // on a tie, the cell nearer the goal first
}
func (q queue) Swap(a, b int) { q[a], q[b] = q[b], q[a] }
func (q *queue) Push(x any)   { *q = append(*q, x.(item)) }
func (q *queue) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}

// step:bestfirst:start

// bestFirst always expands the cell with the smallest f. Dijkstra, A*,
// weighted A*, and greedy best-first differ only in f.
func (p *problem) bestFirst(f func(g, h float64) float64) {
	best := make([]float64, len(p.m.Cells))
	for i := range best {
		best[i] = math.Inf(1)
	}
	done := make([]bool, len(p.m.Cells))
	h := p.heuristic(p.m.Start)
	best[p.m.Start] = 0
	q := &queue{{p.m.Start, f(0, h), 0, h}}
	p.see(p.m.Start)
	for q.Len() > 0 {
		p.frontier(q.Len())
		it := heap.Pop(q).(item)
		if done[it.cell] {
			continue // an old entry: the cell was reached more cheaply
		}
		done[it.cell] = true
		p.expand(it.cell)
		if it.cell == p.m.Goal {
			p.finish()
			return
		}
		for _, e := range p.neighbors(it.cell) {
			g := it.g + e.cost
			if !done[e.to] && g < best[e.to] {
				best[e.to] = g
				p.parent[e.to] = int32(it.cell)
				h := p.heuristic(e.to)
				heap.Push(q, item{e.to, f(g, h), g, h})
				p.see(e.to)
			}
		}
	}
}

// step:bestfirst:end

// step:bidirectional:start

// bidirectional grows two BFS searches, one ring at a time, from the side
// with the smaller frontier. When a ring touches the other search, the
// best meeting cell of that ring joins the two paths.
func bidirectional(p *problem) {
	n := len(p.m.Cells)
	depth := [2][]int{make([]int, n), make([]int, n)}
	parent := [2][]int32{p.parent, make([]int32, n)}
	for i := 0; i < n; i++ {
		depth[0][i], depth[1][i], parent[1][i] = -1, -1, -1
	}
	rings := [2][]int{{p.m.Start}, {p.m.Goal}}
	depth[0][p.m.Start], depth[1][p.m.Goal] = 0, 0
	p.see(p.m.Start)
	p.see(p.m.Goal)
	for len(rings[0]) > 0 && len(rings[1]) > 0 {
		p.frontier(len(rings[0]) + len(rings[1]))
		s := 0
		if len(rings[1]) < len(rings[0]) {
			s = 1
		}
		meet, bestLen := -1, math.MaxInt
		var next []int
		for _, i := range rings[s] {
			p.expand(i)
			for _, e := range p.neighbors(i) {
				if depth[s][e.to] == -1 {
					depth[s][e.to] = depth[s][i] + 1
					parent[s][e.to] = int32(i)
					p.see(e.to)
					next = append(next, e.to)
				}
				if d := depth[1-s][e.to]; d != -1 && depth[s][e.to]+d < bestLen {
					meet, bestLen = e.to, depth[s][e.to]+d
				}
			}
		}
		if meet != -1 {
			// Join: start → meet with the start parents, then meet → goal.
			for i := meet; parent[1][i] != -1; i = int(parent[1][i]) {
				p.parent[parent[1][i]] = int32(i)
			}
			p.finish()
			return
		}
		rings[s] = next
	}
}

// step:bidirectional:end
