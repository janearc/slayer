package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The legacy stand chooses four authored seeded forms, then repeats them.
// Compare actual public geometry rather than only checking that some stock
// was reused. A fifth form would change both appearance and storage cost.
func TestScatteredStandRepeatsItsFourChosenSeededForms(t *testing.T) {
	given := mustParse(t)
	given.set = onFixture(t, "", "")
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld(),
		have: stock{}, radii: map[string]float64{}}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	stands := read.Stands()
	if len(stands) != 1 {
		t.Fatal("fixture lost its single actual fern stand")
	}
	if err := stage.scatter(stands[0]); err != nil {
		t.Fatal(err)
	}
	if stage.made.burning != 0 || len(stage.world.Volumes) != 0 || len(stage.world.Lamps) != 0 {
		t.Fatal("a stand requesting no fire ignited real specimens")
	}
	// Independent Ruby PCG128/DXSM fixture in testdata; not measured data.
	wantPlaces := [][3]float64{
		{1062.28056547676, 8960.671286481736, 2378.875},
		{1086.823009738792, 8955.673148228278, 2755.375},
		{1080.433178855137, 8957.86009282738, 2630.0},
		{1067.3096749883448, 8966.473958356159, 2004.125},
		{1009.061904110194, 8946.26596958035, 3250.5},
		{1029.6388028245467, 8946.455778871587, 3251.75},
		{1081.6909624417683, 8961.24256075157, 2380.0},
		{1087.101338287757, 8959.08421777064, 2505.375},
		{1063.1136697299164, 8953.574330943013, 2878.875},
		{1108.7956633941576, 8948.805177160459, 3131.75},
		{1065.7405995643444, 8944.453885961566, 3379.0},
		{1073.588693544726, 8955.67153482006, 2754.5},
	}
	if len(stage.stood) != len(wantPlaces) {
		t.Fatal("stand did not record its twelve actual places")
	}
	for i, expected := range wantPlaces {
		actual := stage.stood[i]
		if !(math.Abs(actual.east-expected[0]) <= 1e-9 &&
			math.Abs(actual.north-expected[1]) <= 1e-9 &&
			actual.height == expected[2]) {
			t.Fatalf("stand specimen %d moved from its authored seeded place: %+v, want %v", i, actual, expected)
		}
	}
	kind, err := manufacture.KindNamed("fern")
	if err != nil {
		t.Fatal(err)
	}
	var reference [4][]thing.Thing
	boxes := map[maths.Box]bool{}
	for i, seed := range []int{28, 29, 30, 31} {
		object, err := manufacture.MakeWithComplexity(kind,
			manufacture.Sow(stir(given.seed, seed)), "quick")
		if err != nil {
			t.Fatal(err)
		}
		reference[i], err = object.Things(maths.Subspace{})
		if err != nil || len(reference[i]) == 0 {
			t.Fatalf("public seeded reference has no real geometry: %v", err)
		}
		for _, part := range reference[i] {
			boxes[part.Shape.Bounds()] = true
		}
	}
	if len(boxes) < 2 {
		t.Fatal("reference forms do not expose changed seed selection")
	}
	yawTenths := []int{1767, 321, 3578, 468, 2254, 1199, 1561, 2391, 2713, 2043, 1872, 458}
	part := 1 // The actual terrain was added before scatter.
	for instance, choice := range []int{0, 1, 2, 3, 0, 1, 2, 3, 0, 1, 2, 3} {
		for _, expected := range reference[choice] {
			if part >= len(stage.world.Things) {
				t.Fatal("scattered stand omitted real geometry")
			}
			actual := stage.world.Things[part]
			part++
			// A positive turn about +Y sends +X towards -Z.
			radians := float64(yawTenths[instance]) * math.Pi / 1800
			wantRight := maths.V(math.Cos(radians), 0, -math.Sin(radians))
			if gap := actual.At(0).Orientation.Out(maths.XAxis).Sub(wantRight).Len(); !(gap <= 1e-12) {
				t.Fatalf("specimen %d lost its authored seeded yaw by %g", instance, gap)
			}
			if actual.Shape.Bounds() != expected.Shape.Bounds() ||
				actual.Material == nil || expected.Material == nil ||
				actual.Material.Colour != expected.Material.Colour {
				t.Fatalf("stand instance %d no longer matches its chosen public seeded form %d", instance, choice)
			}
		}
	}
	if part != len(stage.world.Things) || stage.made.things != 12 {
		t.Fatal("stand drew a different number of actual fern specimens")
	}
}
