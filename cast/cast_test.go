package cast

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// a band round a ring has no seam and no pentagons, so every tile sits in
// six triangles, each shared by three: two triangles a tile, less the two
// edge rows, which touch fewer
func TestABandCastsTwoTrianglesATile(t *testing.T) {
	b := hexes.Band{Radius: 10, Width: 4, Around: 120}
	inward := func(p maths.Vec) maths.Vec { return maths.V(-p.X, 0, -p.Z) }
	_, n, err := Mesh(b, inward, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	tiles := len(b.Tiles())
	rows := tiles / b.Around
	want := 2 * b.Around * (rows - 1)
	if n != want {
		t.Errorf("%d tiles in %d rows cast %d triangles, want %d", tiles, rows, n, want)
	}
}

// a sphere's tiling closes too, pentagons and all, and every triangle is
// turned outward, or NewMesh would have refused it
func TestASphereCastsClosed(t *testing.T) {
	s := hexes.Sphere{Splits: 8}
	outward := func(p maths.Vec) maths.Vec { return p }
	if _, n, err := Mesh(s, outward, nil, true); err != nil || n == 0 {
		t.Fatalf("cast %d triangles: %v", n, err)
	}
}

// a lift moves each centre along the way out, by as much as it says
func TestALiftRaisesTheCentres(t *testing.T) {
	b := hexes.Band{Radius: 10, Width: 4, Around: 60}
	inward := func(p maths.Vec) maths.Vec { return maths.V(-p.X, 0, -p.Z) }
	up := func(hexes.Tile, maths.Vec) float64 { return 0.5 }
	m, _, err := Mesh(b, inward, up, false)
	if err != nil {
		t.Fatal(err)
	}
	box := m.Bounds()
	if r := box.Max.X; r > 9.5+1e-9 {
		t.Errorf("a floor lifted half a unit toward the axis reaches out to %g, want at most 9.5", r)
	}
}

// loneTiles is a tiling whose tiles are in the wrong place to touch: the
// tiles are a sphere's, but none says it has a neighbour.
type loneTiles struct{ hexes.Tiling }

// Neighbours says no tile has a neighbour.
func (loneTiles) Neighbours(hexes.Tile) []hexes.Tile { return nil }

// tiles that touch no other close no triangle, and the cast says so and
// how many tiles it was given.
func TestTilesThatTouchNothingCastNothing(t *testing.T) {
	sphere := hexes.Sphere{Splits: 1}
	m, n, err := Mesh(loneTiles{sphere}, func(p maths.Vec) maths.Vec { return p }, nil, false)
	if err == nil || m != nil || n != 0 {
		t.Fatalf("cast %v with %d triangles: %v", m, n, err)
	}
	tiles := len(sphere.Tiles())
	want := fmt.Sprintf("the tiling's %d tiles close no triangles", tiles)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error %q lacks %q", err, want)
	}
}
