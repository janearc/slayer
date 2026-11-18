package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// cityLevelHashes are the hull and quick cities' geometry as next made
// them at d0f70ff8, before the street level, measured by
// TestCityDetailMeasures on seed 31.
var cityLevelHashes = map[string]string{
	"grid hull":             "9536c80d4a8bc10c",
	"grid quick":            "f3de34be159c6363",
	"clustered day hull":    "16f89b9e3e02f316",
	"clustered day quick":   "d9f4d3ef200731f4",
	"clustered night hull":  "b82b137dd5725577",
	"clustered night quick": "012c03ba727f7a6c",
}

// envelopeSlack is the rounding a placed corner may carry past the
// envelope, in metres.
const envelopeSlack = 1e-9

// detailSeeds is how many seeds of each family the street level is
// checked over.
const detailSeeds = 40

// The street level is new; the levels scenes already use are the same
// to the bit.
func TestCityExistingLevelsKeepTheirGeometry(t *testing.T) {
	night := DefaultClusteredCity()
	night.Night = true
	makers := map[string]cityLevelMaker{"grid": cityMaker{DefaultCityGrid()},
		"clustered day":   clusteredCityMaker{DefaultClusteredCity()},
		"clustered night": clusteredCityMaker{night}}
	for name, row := range makers {
		for _, detail := range []string{"hull", "quick"} {
			made, err := row.madeAt(Sow(31), detail)
			if err != nil {
				t.Fatal(name, detail, err)
			}
			key := name + " " + detail
			if got := cityGeometryHash(made); got != cityLevelHashes[key] {
				t.Errorf("%s changed: %s, was %s", key, got,
					cityLevelHashes[key])
			}
		}
	}
}

// Every part of every family at street level stays inside the envelope
// its lot was planned with, and the envelope is the quick one.
func TestCityStreetLevelStaysInItsEnvelope(t *testing.T) {
	for _, name := range []string{"city-rowhouse", "city-apartment",
		"city-tower"} {
		row, _ := KindNamed(name)
		for number := range uint64(detailSeeds) {
			made, err := MakeWithComplexity(row, Sow(number), cityStreetLevel)
			if err != nil {
				t.Fatal(name, number, err)
			}
			quick, _ := Make(row, Sow(number))
			if made.Bounds() != quick.Bounds() {
				t.Fatalf("%s %d envelope changed", name, number)
			}
			if _, err := made.Things(maths.Local()); err != nil {
				t.Fatal(name, number, err)
			}
			cityWithin(t, made, made.Bounds())
		}
	}
}

// cityWithin fails when any placed corner lies outside a box.
func cityWithin(t *testing.T, made good, envelope maths.Box) {
	t.Helper()
	inside := func(at maths.Vec) bool {
		low, high := envelope.Min, envelope.Max
		return at.X >= low.X-envelopeSlack && at.X <= high.X+envelopeSlack &&
			at.Y >= low.Y-envelopeSlack && at.Y <= high.Y+envelopeSlack &&
			at.Z >= low.Z-envelopeSlack && at.Z <= high.Z+envelopeSlack
	}
	for _, group := range made.groups {
		for index := range group.MemberCount() {
			member := group.Member(index)
			for _, point := range member.Mesh.(*mesh.Mesh).Points() {
				if at := member.Space.Out(point); !inside(at) {
					t.Fatalf("corner %v outside %+v", at, envelope)
				}
			}
		}
	}
}

// A street-level city places each module many times and pools it once.
func TestCityStreetLevelSharesItsModules(t *testing.T) {
	grid := DefaultCityGrid()
	grid.BlocksX, grid.BlocksZ, grid.RoundaboutEvery = 2, 3, 0
	row, err := City(grid)
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(row, Sow(42), cityStreetLevel)
	if err != nil {
		t.Fatal(err)
	}
	counted := census(made)
	if counted.members < cityModuleReuse*counted.unique {
		t.Fatalf("%d members of %d meshes: modules are not shared",
			counted.members, counted.unique)
	}
	quick, _ := Make(row, Sow(42))
	if counted.stored <= census(quick).stored {
		t.Fatalf("street level pools %d faces, quick %d", counted.stored,
			census(quick).stored)
	}
	if _, err := made.Things(maths.Local()); err != nil {
		t.Fatal(err)
	}
}

// cityModuleReuse is the least number of times the small city places
// each pooled mesh on average.
const cityModuleReuse = 4

// A tower has recessed windows on all four faces, each pane behind its
// wall plane.
func TestCityStreetLevelGlazesAllFourFaces(t *testing.T) {
	row, _ := KindNamed("city-tower")
	made, err := MakeWithComplexity(row, Sow(3), cityStreetLevel)
	if err != nil {
		t.Fatal(err)
	}
	design := cityBuildingDesign("city-tower", Sow(3))
	limit := design.width*spaceHalf - cityValue("facade-projection") -
		cityValue("reveal-depth") + cityValue("pane-offset") + envelopeSlack
	faces := map[[2]int]int{}
	glass := made.groups[cityGlass]
	for index := range glass.MemberCount() {
		panes := glass.Member(index).Mesh.(*mesh.Mesh)
		for face := range panes.FaceCount() {
			a, b, c := panes.Corners(face)
			normal := b.Sub(a).Cross(c.Sub(a)).Unit()
			faces[[2]int{int(math.Round(normal.X)),
				int(math.Round(normal.Z))}]++
			if math.Abs(a.X) > limit && math.Abs(normal.X) > spaceHalf {
				t.Fatalf("pane at %v stands proud of its reveal", a)
			}
		}
	}
	for _, way := range [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
		if faces[way] == 0 {
			t.Fatalf("no windows facing %v: %v", way, faces)
		}
	}
}

// Night keeps the same windows as day, each lit or unlit by its draw.
func TestCityStreetLevelKeepsSeededNightPanes(t *testing.T) {
	design := cityBuildingDesign("city-apartment", Sow(9))
	panes := func(night bool) map[slot]int {
		batch := newCityBatch()
		batch.detailed(design, Sow(9), maths.Local(), night)
		made, err := batch.finish()
		if err != nil {
			t.Fatal(err)
		}
		counts := map[slot]int{}
		for _, role := range []slot{cityGlass, cityWarmWindow,
			cityCoolWindow} {
			if group := made.groups[role]; group != nil {
				counts[role] = group.FaceCount()
			}
		}
		return counts
	}
	day, night := panes(false), panes(true)
	lit := night[cityWarmWindow] + night[cityCoolWindow]
	if day[cityGlass] != night[cityGlass]+lit || lit == 0 ||
		night[cityGlass] == 0 {
		t.Fatalf("day panes %v, night panes %v", day, night)
	}
}

// anti-test of TestCityStreetLevelStaysInItsEnvelope
func TestCityStreetLevelRefusesPartsThatDoNotFit(t *testing.T) {
	source := string(cityChoices)
	broken := []string{
		strings.Replace(source, "name: module-height\n  value: 2.9",
			"name: module-height\n  value: 3.2", 1),
		strings.Replace(source, "name: tower-window-width\n  value: 2.6",
			"name: tower-window-width\n  value: 3.4", 1),
		strings.Replace(source, "name: mast-height\n  value: 2.7",
			"name: mast-height\n  value: 3.5", 1),
		strings.Replace(source, "name: coolers\n  value: 3",
			"name: coolers\n  value: 2.5", 1),
		strings.Replace(source, "name: bay-pitch", "name: bay-width", 1),
	}
	for index, raw := range broken {
		if raw == source {
			t.Fatalf("broken recipe %d changed nothing", index)
		}
		if _, err := readCityChoices([]byte(raw)); err == nil {
			t.Fatalf("accepted broken street-level recipe %d", index)
		}
	}
	row, _ := KindNamed("city-tower")
	if _, err := MakeWithComplexity(row, seed{}, cityStreetLevel); err == nil {
		t.Fatal("accepted an unsown street-level seed")
	}
	if _, err := MakeWithComplexity(row, Sow(1), "clumps"); err == nil {
		t.Fatal("accepted a level the city does not make")
	}
}
