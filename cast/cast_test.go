package cast

import (
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
	s := hexes.Sphere{N: 8}
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
