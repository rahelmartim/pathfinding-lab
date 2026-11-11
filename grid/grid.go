// Package grid is a map of cells for path search: open cells, walls, and
// mud, with a start and a goal. It also makes maps: random walls, mazes,
// rooms, and mud.
package grid

import (
	"errors"
	"fmt"
	"math/rand/v2"
)

// Cell is the kind of one cell.
type Cell uint8

const (
	Open Cell = iota
	Wall
	Mud // open, but it costs more to enter
)

// Limits of the size of a map.
const (
	MinSize = 5
	MaxSize = 200
)

// Map is a grid of cells. A cell has an index: y*W + x.
type Map struct {
	W, H  int
	Cells []Cell
	Start int
	Goal  int
}

var ErrBadMap = errors.New("the map is not valid")

func clampSize(v int) int {
	return max(MinSize, min(MaxSize, v))
}

// New makes an open map with the start on the left and the goal on the
// right.
func New(w, h int) *Map {
	w, h = clampSize(w), clampSize(h)
	m := &Map{W: w, H: h, Cells: make([]Cell, w*h)}
	m.Start = m.Index(1, h/2)
	m.Goal = m.Index(w-2, h/2)
	return m
}

// Validate checks a map that came from outside, such as from the page.
func (m *Map) Validate() error {
	switch {
	case m.W < MinSize || m.W > MaxSize || m.H < MinSize || m.H > MaxSize:
		return fmt.Errorf("%w: the size must be from %d to %d", ErrBadMap, MinSize, MaxSize)
	case len(m.Cells) != m.W*m.H:
		return fmt.Errorf("%w: %d cells for a %d × %d map", ErrBadMap, len(m.Cells), m.W, m.H)
	case m.Start < 0 || m.Start >= len(m.Cells) || m.Goal < 0 || m.Goal >= len(m.Cells):
		return fmt.Errorf("%w: the start or the goal is outside the map", ErrBadMap)
	case m.Start == m.Goal:
		return fmt.Errorf("%w: the start and the goal are the same cell", ErrBadMap)
	}
	for _, c := range m.Cells {
		if c > Mud {
			return fmt.Errorf("%w: unknown cell %d", ErrBadMap, c)
		}
	}
	return nil
}

func (m *Map) Index(x, y int) int { return y*m.W + x }

func (m *Map) XY(i int) (int, int) { return i % m.W, i / m.W }

func (m *Map) In(x, y int) bool { return x >= 0 && y >= 0 && x < m.W && y < m.H }

// Passable reports whether a search can enter the cell at x, y.
func (m *Map) Passable(x, y int) bool { return m.In(x, y) && m.Cells[m.Index(x, y)] != Wall }

// Clone returns a copy that does not share cells.
func (m *Map) Clone() *Map {
	c := *m
	c.Cells = append([]Cell(nil), m.Cells...)
	return &c
}

// clearEnds opens the start and the goal. If one of them has no open
// neighbor, it also opens one neighbor, so that no generator walls them in.
func (m *Map) clearEnds() {
	for _, i := range []int{m.Start, m.Goal} {
		x, y := m.XY(i)
		m.Cells[i] = Open
		var walls []int
		open := false
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !m.In(nx, ny) {
				continue
			}
			if m.Cells[m.Index(nx, ny)] == Wall {
				walls = append(walls, m.Index(nx, ny))
			} else {
				open = true
			}
		}
		if !open && len(walls) > 0 {
			m.Cells[walls[0]] = Open
		}
	}
}

// Kinds of generated maps.
var Kinds = []string{"empty", "random", "maze", "rooms", "mud"}

// Generate makes a map of a kind. Density is the share of walls (or mud)
// from 0 to 1, for the kinds that use it. The same seed makes the same map.
func Generate(kind string, w, h int, density float64, seed uint64) (*Map, error) {
	m := New(w, h)
	rng := rand.New(rand.NewPCG(seed, seed^0x5851f42d4c957f2d))
	density = max(0, min(0.9, density))
	switch kind {
	case "empty":
	case "random":
		m.random(rng, density, Wall)
	case "maze":
		m.maze(rng)
	case "rooms":
		m.rooms(rng)
	case "mud":
		m.random(rng, density/2, Wall)
		m.blobs(rng, density)
	default:
		return nil, fmt.Errorf("unknown map kind %q", kind)
	}
	m.clearEnds()
	return m, nil
}

func (m *Map) random(rng *rand.Rand, density float64, c Cell) {
	for i := range m.Cells {
		if rng.Float64() < density {
			m.Cells[i] = c
		}
	}
}

// maze carves a perfect maze (one path between any two cells) with a
// depth-first search, then puts the start and the goal on carved cells.
func (m *Map) maze(rng *rand.Rand) {
	for i := range m.Cells {
		m.Cells[i] = Wall
	}
	type p struct{ x, y int }
	stack := []p{{1, 1}}
	m.Cells[m.Index(1, 1)] = Open
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		var next []p
		for _, d := range []p{{2, 0}, {-2, 0}, {0, 2}, {0, -2}} {
			n := p{c.x + d.x, c.y + d.y}
			if n.x > 0 && n.y > 0 && n.x < m.W-1 && n.y < m.H-1 && m.Cells[m.Index(n.x, n.y)] == Wall {
				next = append(next, n)
			}
		}
		if len(next) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		n := next[rng.IntN(len(next))]
		m.Cells[m.Index((c.x+n.x)/2, (c.y+n.y)/2)] = Open
		m.Cells[m.Index(n.x, n.y)] = Open
		stack = append(stack, n)
	}
	// The carved cells have odd coordinates.
	oddBelow := func(v int) int {
		if v%2 == 0 {
			v--
		}
		return max(1, v)
	}
	y := oddBelow(min(m.H-2, m.H/2))
	m.Start = m.Index(1, y)
	m.Goal = m.Index(oddBelow(m.W-2), y)
}

// rooms makes a grid of rooms with walls and a door in each wall.
func (m *Map) rooms(rng *rand.Rand) {
	size := max(4, min(m.W, m.H)/4)
	for x := size; x < m.W-1; x += size {
		for y := 0; y < m.H; y++ {
			m.Cells[m.Index(x, y)] = Wall
		}
	}
	for y := size; y < m.H-1; y += size {
		for x := 0; x < m.W; x++ {
			m.Cells[m.Index(x, y)] = Wall
		}
	}
	// One door in each wall segment between two crossings.
	for x := size; x < m.W-1; x += size {
		for y := 0; y < m.H; y += size {
			door := y + 1 + rng.IntN(max(1, min(size-1, m.H-y-1)))
			if door < m.H {
				m.Cells[m.Index(x, door)] = Open
			}
		}
	}
	for y := size; y < m.H-1; y += size {
		for x := 0; x < m.W; x += size {
			door := x + 1 + rng.IntN(max(1, min(size-1, m.W-x-1)))
			if door < m.W {
				m.Cells[m.Index(door, y)] = Open
			}
		}
	}
}

// blobs adds round patches of mud.
func (m *Map) blobs(rng *rand.Rand, density float64) {
	count := int(density*float64(m.W*m.H)/30) + 1
	for range count {
		cx, cy := rng.IntN(m.W), rng.IntN(m.H)
		r := 2 + rng.IntN(max(2, min(m.W, m.H)/6))
		for y := cy - r; y <= cy+r; y++ {
			for x := cx - r; x <= cx+r; x++ {
				if m.In(x, y) && (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r && m.Cells[m.Index(x, y)] == Open {
					m.Cells[m.Index(x, y)] = Mud
				}
			}
		}
	}
}
