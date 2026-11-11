// Searchgif writes an animated GIF for the README: BFS, Dijkstra, A*, and
// greedy best-first explore the same map at the same speed.
//
//	go run ./cmd/searchgif -out docs/race.gif
package main

import (
	"flag"
	"image"
	"image/color"
	"image/gif"
	"log"
	"os"

	"github.com/rahelmartim/pathfinding-lab/grid"
	"github.com/rahelmartim/pathfinding-lab/search"
)

var (
	white    = color.RGBA{255, 255, 255, 255}
	wall     = color.RGBA{43, 48, 68, 255}
	mud      = color.RGBA{168, 123, 79, 255}
	frontier = color.RGBA{251, 191, 36, 255}
	pathC    = color.RGBA{249, 115, 22, 255}
	start    = color.RGBA{22, 163, 74, 255}
	goal     = color.RGBA{220, 38, 38, 255}
	back     = color.RGBA{247, 248, 251, 255}
)

func main() {
	out := flag.String("out", "docs/race.gif", "file to write")
	scale := flag.Int("scale", 5, "pixels for one cell")
	flag.Parse()

	m, err := grid.Generate("mud", 49, 29, 0.3, 11)
	if err != nil {
		log.Fatal(err)
	}
	keys := []string{"bfs", "dijkstra", "astar", "greedy"}
	results := make([]search.Result, len(keys))
	for k, key := range keys {
		if results[k], err = search.Run(key, m, search.Options{Record: true}); err != nil {
			log.Fatal(err)
		}
	}

	// 16 shades from light blue to indigo for the expanded cells.
	palette := color.Palette{back, white, wall, mud, frontier, pathC, start, goal}
	shade0 := len(palette)
	for s := 0; s < 16; s++ {
		t := float64(s) / 15
		palette = append(palette, color.RGBA{uint8(165 - 36*t), uint8(216 - 76*t), uint8(255 - 7*t), 255})
	}

	const gap = 3
	cols := 2
	laneW, laneH := m.W**scale, m.H**scale
	width := cols*laneW + (cols-1)*gap**scale
	height := 2*laneH + gap**scale

	most := 0
	for _, r := range results {
		most = max(most, r.Expanded)
	}
	const frames = 70
	perFrame := max(1, most/(frames-12))

	anim := &gif.GIF{}
	for f := 0; f < frames; f++ {
		img := image.NewPaletted(image.Rect(0, 0, width, height), palette)
		for i := range img.Pix {
			img.Pix[i] = 0
		}
		shown := (f + 1) * perFrame
		for k, r := range results {
			left := (k % cols) * (laneW + gap**scale)
			top := (k / cols) * (laneH + gap**scale)
			lane(img, m, r, shown, left, top, *scale, shade0)
		}
		anim.Image = append(anim.Image, img)
		delay := 6
		if f == frames-1 {
			delay = 250 // hold the end, with the paths
		}
		anim.Delay = append(anim.Delay, delay)
	}

	file, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	if err := gif.EncodeAll(file, anim); err != nil {
		log.Fatal(err)
	}
}

// lane draws one search after its first n expansions.
func lane(img *image.Paletted, m *grid.Map, r search.Result, n, left, top, s, shade0 int) {
	n = min(n, len(r.Visited))
	color := make([]uint8, len(m.Cells))
	for i, c := range m.Cells {
		color[i] = [3]uint8{1, 2, 3}[c]
	}
	seenAt := map[int]int{}
	for k, c := range r.Seen {
		if _, ok := seenAt[c]; !ok {
			seenAt[c] = r.SeenAt[k]
		}
	}
	for c, at := range seenAt {
		if at <= n && m.Cells[c] != grid.Wall {
			color[c] = 4
		}
	}
	for k := 0; k < n; k++ {
		color[r.Visited[k]] = uint8(shade0 + k*15/max(1, len(r.Visited)-1))
	}
	if n == len(r.Visited) {
		for _, c := range r.Path {
			color[c] = 5
		}
	}
	color[m.Start], color[m.Goal] = 6, 7
	for i, c := range color {
		x, y := m.XY(i)
		for dy := 0; dy < s; dy++ {
			row := img.Pix[(top+y*s+dy)*img.Stride+left+x*s:]
			for dx := 0; dx < s; dx++ {
				row[dx] = c
			}
		}
	}
}
