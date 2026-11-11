// Package hanoi solves the Tower of Hanoi with a breadth-first search over
// the states of the game. Each state says on which rod each disk is.
package hanoi

import "fmt"

// MaxDisks keeps the search small: 3^10 states.
const MaxDisks = 10

// Move takes the top disk of one rod to another. Disk 0 is the smallest.
type Move struct {
	Disk int `json:"disk"`
	From int `json:"from"`
	To   int `json:"to"`
}

// Result is the solution and the size of the search.
type Result struct {
	Disks    int    `json:"disks"`
	Moves    []Move `json:"moves"`
	Expanded int    `json:"expanded"` // states that the search took out of its queue
	Seen     int    `json:"seen"`     // states that the search reached
	States   int    `json:"states"`   // all the states: 3^disks
}

// state has the rod of each disk, as digits in base 3: disk d is digit d.
type state int

func (s state) rod(disk int) int {
	for range disk {
		s /= 3
	}
	return int(s % 3)
}

func pow3(n int) int {
	p := 1
	for range n {
		p *= 3
	}
	return p
}

// moves returns the legal moves of a state: the smallest disk on a rod is
// its top, and it can go to a rod whose top is bigger, or to an empty rod.
func moves(s state, disks int) []Move {
	top := [3]int{-1, -1, -1}
	for d := disks - 1; d >= 0; d-- {
		top[s.rod(d)] = d
	}
	var out []Move
	for from := 0; from < 3; from++ {
		if top[from] == -1 {
			continue
		}
		for to := 0; to < 3; to++ {
			if to != from && (top[to] == -1 || top[to] > top[from]) {
				out = append(out, Move{top[from], from, to})
			}
		}
	}
	return out
}

func apply(s state, m Move) state {
	return s + state((m.To-m.From)*pow3(m.Disk))
}

// step:start

// BFS finds the shortest solution: all disks from rod 0 to rod 2.
func BFS(disks int) (Result, error) {
	if disks < 1 || disks > MaxDisks {
		return Result{}, fmt.Errorf("the number of disks must be from 1 to %d", MaxDisks)
	}
	goal := state(pow3(disks) - 1) // every digit is 2
	res := Result{Disks: disks, States: pow3(disks)}
	parent := make([]int32, res.States)
	how := make([]Move, res.States)
	for i := range parent {
		parent[i] = -1
	}
	parent[0] = 0
	queue := []state{0}
	res.Seen = 1
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		res.Expanded++
		if s == goal {
			break
		}
		for _, m := range moves(s, disks) {
			n := apply(s, m)
			if parent[n] == -1 {
				parent[n], how[n] = int32(s), m
				res.Seen++
				queue = append(queue, n)
			}
		}
	}
	for s := goal; s != 0; s = state(parent[s]) {
		res.Moves = append(res.Moves, how[s])
	}
	for l, r := 0, len(res.Moves)-1; l < r; l, r = l+1, r-1 {
		res.Moves[l], res.Moves[r] = res.Moves[r], res.Moves[l]
	}
	return res, nil
}

// step:end
