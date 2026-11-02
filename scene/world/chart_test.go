package world

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// chart is a tiling laid out by hand, so the plates, boundaries and
// classes of a world can be worked on paper. Tile I sits at at[I], is
// next to near[I], and has the latitude lat gives its middle. Distance
// is the straight line, Spacing is one, and Finer counts itself.
//
// Somewhere answers with the next of mids if there are any, and with its
// three numbers as a point if there are none, so a test can read the
// numbers a world drew its plates from.
type chart struct {
	at    []maths.Vec
	near  [][]int
	lat   map[maths.Vec]float64
	mids  []maths.Vec
	asked int
	fined int
}

// chained is a chart of n tiles at tiny steps along x, with the given pairs of
// tiles next to each other. The steps are a billionth, so the noise at a middle
// is zero for every purpose the tests have, and each middle is its own key for
// the latitude.
func chained(n int, pairs ...[2]int) *chart {
	c := &chart{at: make([]maths.Vec, n), near: make([][]int, n),
		lat: map[maths.Vec]float64{}}
	for i := range c.at {
		c.at[i] = maths.V(float64(i)*1e-9, 0, 0)
	}
	for _, p := range pairs {
		c.near[p[0]] = append(c.near[p[0]], p[1])
		c.near[p[1]] = append(c.near[p[1]], p[0])
	}
	return c
}

// along is the pairs that join tiles 0 to n-1 end to end.
func along(n int) [][2]int {
	var out [][2]int
	for i := 1; i < n; i++ {
		out = append(out, [2]int{i - 1, i})
	}
	return out
}

func (c *chart) Tiles() []hexes.Tile {
	out := make([]hexes.Tile, len(c.at))
	for i := range out {
		out[i] = hexes.Tile{I: i}
	}
	return out
}

func (c *chart) Centre(t hexes.Tile) maths.Vec { return c.at[t.I] }

func (c *chart) Neighbours(t hexes.Tile) []hexes.Tile {
	var out []hexes.Tile
	for _, j := range c.near[t.I] {
		out = append(out, hexes.Tile{I: j})
	}
	return out
}

func (c *chart) Nearest(p maths.Vec) hexes.Tile {
	best, at := math.Inf(1), 0
	for i, m := range c.at {
		if d := m.Sub(p).Len(); d < best {
			best, at = d, i
		}
	}
	return hexes.Tile{I: at}
}

func (c *chart) Edge(maths.Vec, hexes.Tile) float64 { return 1 }
func (c *chart) ID(t hexes.Tile) int64              { return int64(t.I) }
func (c *chart) Spacing() float64                   { return 1 }
func (c *chart) Finer() hexes.Tiling {
	c.fined++
	return c
}

func (c *chart) Distance(a, b maths.Vec) float64 { return a.Sub(b).Len() }

func (c *chart) Latitude(p, _ maths.Vec) float64 { return c.lat[p] }

func (c *chart) Somewhere(a, b, d float64) maths.Vec {
	if c.asked < len(c.mids) {
		c.asked++
		return c.mids[c.asked-1]
	}
	return maths.V(a, b, d)
}

// on is a world on the chart, laid out and nothing more.
func on(c *chart, seed uint64) *World {
	w := &World{Terrain: Terrain{Tiling: c, Seed: seed}}
	w.lay()
	return w
}

// ends runs call and, if it has not come back in a second, ends the
// whole run. a loop a change has made endless cannot be stopped, and
// some grow as they go, so failing the one test and carrying on, as the
// colour package does, would leave one filling the memory meanwhile.
func ends(t *testing.T, what string, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		call()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		fmt.Fprintf(os.Stderr, "FAIL: %s, in %s, did not finish in a second; "+
			"ending the run, for the loop cannot be stopped\n", what, t.Name())
		os.Exit(1)
	}
}

// close12 is whether two numbers are the same to a part in a trillion.
func close12(a, b float64) bool { return math.Abs(a-b) <= 1e-12*math.Max(1, math.Abs(b)) }
