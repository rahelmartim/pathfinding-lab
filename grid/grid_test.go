package grid

import (
	"math"
	"testing"
)

// reachable counts the open cells that the start can reach, with
// straight moves.
func reachable(m *Map) (int, bool) {
	seen := map[int]bool{m.Start: true}
	queue := []int{m.Start}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := m.XY(i)
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if m.Passable(x+d[0], y+d[1]) && !seen[m.Index(x+d[0], y+d[1])] {
				seen[m.Index(x+d[0], y+d[1])] = true
				queue = append(queue, m.Index(x+d[0], y+d[1]))
			}
		}
	}
	return len(seen), seen[m.Goal]
}

func count(m *Map, c Cell) int {
	n := 0
	for _, v := range m.Cells {
		if v == c {
			n++
		}
	}
	return n
}

func TestEveryKindIsValidAndKeepsTheEndsOpen(t *testing.T) {
	for _, kind := range Kinds {
		for seed := uint64(1); seed <= 30; seed++ {
			m, err := Generate(kind, 10+int(seed)*3, 8+int(seed)*2, 0.35, seed)
			if err != nil {
				t.Fatal(err)
			}
			if err := m.Validate(); err != nil {
				t.Fatalf("%s seed %d: %v", kind, seed, err)
			}
			if m.Cells[m.Start] != Open || m.Cells[m.Goal] != Open {
				t.Fatalf("%s seed %d: the start or the goal is not open", kind, seed)
			}
		}
	}
}

func TestMazesArePerfect(t *testing.T) {
	for seed := uint64(1); seed <= 20; seed++ {
		m, _ := Generate("maze", 31+int(seed%5), 21+int(seed%4), 0, seed)
		open := len(m.Cells) - count(m, Wall)
		got, goal := reachable(m)
		if got != open || !goal {
			t.Fatalf("seed %d: %d of %d open cells are reachable, goal=%v", seed, got, open, goal)
		}
		// A tree: the open cells have one fewer link than cells.
		links := 0
		for i, c := range m.Cells {
			x, y := m.XY(i)
			if c != Wall && m.Passable(x+1, y) {
				links++
			}
			if c != Wall && m.Passable(x, y+1) {
				links++
			}
		}
		if links != open-1 {
			t.Fatalf("seed %d: %d links for %d cells, so the maze has a loop", seed, links, open)
		}
	}
}

func TestRoomsAreConnected(t *testing.T) {
	for seed := uint64(1); seed <= 20; seed++ {
		m, _ := Generate("rooms", 40+int(seed), 30, 0, seed)
		got, goal := reachable(m)
		if !goal || got != len(m.Cells)-count(m, Wall) {
			t.Fatalf("seed %d: the rooms are not all connected (%d cells, goal=%v)", seed, got, goal)
		}
	}
}

func TestDensityAndMud(t *testing.T) {
	m, _ := Generate("random", 100, 100, 0.3, 1)
	if share := float64(count(m, Wall)) / 10000; math.Abs(share-0.3) > 0.03 {
		t.Fatalf("a density of 0.3 made %.2f walls", share)
	}
	mud, _ := Generate("mud", 80, 60, 0.4, 2)
	if count(mud, Mud) == 0 {
		t.Fatal("the mud map has no mud")
	}
	empty, _ := Generate("empty", 30, 20, 0.9, 3)
	if count(empty, Wall)+count(empty, Mud) != 0 {
		t.Fatal("the empty map is not empty")
	}
}

func TestTheSameSeedMakesTheSameMap(t *testing.T) {
	for _, kind := range Kinds {
		a, _ := Generate(kind, 40, 30, 0.3, 77)
		b, _ := Generate(kind, 40, 30, 0.3, 77)
		c, _ := Generate(kind, 40, 30, 0.3, 78)
		if string(cells(a)) != string(cells(b)) {
			t.Fatalf("%s: one seed made two maps", kind)
		}
		if kind != "empty" && string(cells(a)) == string(cells(c)) {
			t.Fatalf("%s: two seeds made one map", kind)
		}
	}
}

func cells(m *Map) []byte {
	out := make([]byte, len(m.Cells))
	for i, c := range m.Cells {
		out[i] = byte(c)
	}
	return out
}

func TestValidate(t *testing.T) {
	if _, err := Generate("lava", 10, 10, 0, 1); err == nil {
		t.Fatal("an unknown kind must fail")
	}
	m := New(1, 9999)
	if m.W != MinSize || m.H != MaxSize {
		t.Fatalf("New did not keep the size in its limits: %d × %d", m.W, m.H)
	}
	bad := []func(*Map){
		func(m *Map) { m.W = 3 },
		func(m *Map) { m.Cells = m.Cells[1:] },
		func(m *Map) { m.Start = -1 },
		func(m *Map) { m.Goal = len(m.Cells) },
		func(m *Map) { m.Goal = m.Start },
		func(m *Map) { m.Cells[0] = 9 },
	}
	for k, change := range bad {
		m := New(10, 10)
		change(m)
		if m.Validate() == nil {
			t.Fatalf("bad map %d passed", k)
		}
	}
	c := New(10, 10)
	d := c.Clone()
	d.Cells[0] = Wall
	if c.Cells[0] != Open {
		t.Fatal("Clone shares the cells")
	}
}
