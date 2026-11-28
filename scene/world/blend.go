package world

import (
	"github.com/janearc/slayer/maths"
)

// blendReach is how far, in coarse spacings, a tile's height and
// wetness are felt: a point takes the bump-weighted mean of the tiles
// within it. It is under the distance at which a tile two rings out
// from one tile and three from its neighbour could be reached from the
// edge between them, so the mean has no step where the nearest tile
// changes. Set by eye (2026-11-26), then checked by a test.
const blendReach = 1.4

// blendRings is how many rings of neighbours are gathered round the
// nearest tile: two, which every tile within blendReach of a point in
// the nearest tile is among.
const blendRings = 2

// layReach lists, for every tile, itself and the tiles within blendRings
// rings of it, by index, once, so a point's blend is an array walk.
func (w *World) layReach() {
	w.reach = make([][]int, len(w.tiles))
	for i := range w.tiles {
		seen := map[int]bool{i: true}
		ring := []int{i}
		list := []int{i}
		for range blendRings {
			var next []int
			for _, from := range ring {
				for _, to := range w.near[from] {
					if !seen[to] {
						seen[to] = true
						next = append(next, to)
					}
				}
			}
			list = append(list, next...)
			ring = next
		}
		w.reach[i] = list
	}
}

// bump is the weight of a tile at a share of blendReach away: one at the
// middle, nought at and past the reach, smooth at both ends.
func bump(share float64) float64 {
	if share >= 1 {
		return 0
	}
	inner := 1 - share*share
	return inner * inner
}

// Ground is what the coarse world holds at a point, smoothed between its
// tiles: how high the ground is above the sea, how many tiles from the
// sea, and the tile that is nearest, by index, or minus one beyond the
// world's edge.
type Ground struct {
	Height, Wet float64
	Nearest     int
}

// Smoothed is the coarse world at a direction p, its tiles' heights and
// distances from the sea blended by distance, so it varies smoothly and
// shows no tile's edge. At a tile's middle it is near the tile's own.
// Beyond the world's edge it is the deep.
func (w *World) Smoothed(p maths.Vec) Ground {
	point := p.Unit()
	nearest := w.index(w.Tiling.Nearest(point))
	if nearest < 0 {
		return Ground{Height: -1, Nearest: -1}
	}
	reach := blendReach * w.Tiling.Spacing()
	var weight, height, wet float64
	for _, i := range w.reach[nearest] {
		share := point.Sub(w.middle[i]).Len() / reach
		part := bump(share)
		weight += part
		height += part * w.height[i]
		wet += part * float64(w.wet[i])
	}
	return Ground{Height: height / weight, Wet: wet / weight,
		Nearest: nearest}
}
