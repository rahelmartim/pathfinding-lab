package hanoi

import "testing"

func TestBFSFindsTheShortestSolution(t *testing.T) {
	for disks := 1; disks <= MaxDisks; disks++ {
		r, err := BFS(disks)
		if err != nil {
			t.Fatal(err)
		}
		if want := 1<<disks - 1; len(r.Moves) != want {
			t.Fatalf("%d disks: %d moves, want %d", disks, len(r.Moves), want)
		}
		if r.States != pow3(disks) || r.Seen > r.States || r.Expanded > r.Seen {
			t.Fatalf("%d disks: %d states, %d seen, %d expanded", disks, r.States, r.Seen, r.Expanded)
		}
		// Play the moves on real rods, and check every rule.
		rods := [3][]int{}
		for d := disks - 1; d >= 0; d-- {
			rods[0] = append(rods[0], d)
		}
		for k, m := range r.Moves {
			from := rods[m.From]
			if len(from) == 0 || from[len(from)-1] != m.Disk {
				t.Fatalf("%d disks, move %d: disk %d is not on top of rod %d", disks, k, m.Disk, m.From)
			}
			if to := rods[m.To]; len(to) > 0 && to[len(to)-1] < m.Disk {
				t.Fatalf("%d disks, move %d: a big disk on a small one", disks, k)
			}
			rods[m.From] = from[:len(from)-1]
			rods[m.To] = append(rods[m.To], m.Disk)
		}
		if len(rods[2]) != disks {
			t.Fatalf("%d disks: rod 2 has %d disks at the end", disks, len(rods[2]))
		}
	}
}

func TestTheLimits(t *testing.T) {
	for _, n := range []int{0, -1, MaxDisks + 1} {
		if _, err := BFS(n); err == nil {
			t.Fatalf("%d disks must fail", n)
		}
	}
}

func TestTheOriginalGame(t *testing.T) {
	// The 2021 version solved 3 disks: 7 moves.
	r, _ := BFS(3)
	if len(r.Moves) != 7 || r.Moves[0] != (Move{Disk: 0, From: 0, To: 2}) {
		t.Fatalf("3 disks: %+v", r.Moves)
	}
}
