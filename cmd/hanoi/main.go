// Hanoi solves the Tower of Hanoi with a breadth-first search.
//
//	go run ./cmd/hanoi -disks 4
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/rahelmartim/pathfinding-lab/hanoi"
)

func main() {
	disks := flag.Int("disks", 3, fmt.Sprintf("number of disks, from 1 to %d", hanoi.MaxDisks))
	flag.Parse()
	r, err := hanoi.BFS(*disks)
	if err != nil {
		log.Fatal(err)
	}
	rods := "ABC"
	for k, m := range r.Moves {
		fmt.Printf("%3d. disk %d from %c to %c\n", k+1, m.Disk+1, rods[m.From], rods[m.To])
	}
	fmt.Printf("%d moves. BFS expanded %d of the %d states.\n", len(r.Moves), r.Expanded, r.States)
}
