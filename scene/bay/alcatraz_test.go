package bay

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestAquaticParkAlcatrazGroundwork(t *testing.T) {
	grid := AquaticPark()
	pieces, err := Alcatraz("alcatraz-3dep.tif", grid)
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) < 15 {
		t.Fatalf("Alcatraz assembled only %d pieces", len(pieces))
	}
	land := pieces[0].Shape.(*mesh.Mesh)
	bounds := land.Bounds()
	if bounds.Min.Z > -1950 || bounds.Max.Z < -1610 ||
		bounds.Max.Y < 38 || bounds.Max.Y > 49 {
		t.Fatalf("unexpected island extents: %v", bounds)
	}
	light := grid.at(lighthouseEast, lighthouseNorth, 0)
	if math.Abs(light.Len()-1740.98) > .1 ||
		math.Abs(grid.LighthouseBearing()/maths.Degree-5.44) > .01 {
		t.Fatalf("unexpected swimmer sightline: %v, %v", light, grid.LighthouseBearing())
	}
	for _, piece := range pieces[1:] {
		if !piece.Shape.(*mesh.Mesh).Closed() {
			t.Fatal("an architecture piece is open")
		}
	}
	t.Logf("%d triangles, terrain %v, %d architecture pieces",
		land.FaceCount(), bounds, len(pieces)-1)
}

func TestAquaticParkPierFootprintsSitEitherSideOfTheSightline(t *testing.T) {
	grid := AquaticPark()
	piers, err := Piers(grid)
	if err != nil {
		t.Fatal(err)
	}
	municipal := grid.at(550658.85, 4184953.55, 0)
	hyde := grid.at(550819, 4184900, 0)
	if len(piers) < 50 || municipal.X >= 0 || hyde.X <= 0 {
		t.Fatalf("bad pier assembly or placement: %d pieces, %v and %v",
			len(piers), municipal, hyde)
	}
}
