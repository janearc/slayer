package gpu

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
)

// A scene may deliberately pair one density with another field's emission.
// That needs ordinary texture sampling, not a bake of only their shared prefix.
func TestDeferredFlamesKeepCustomFieldFormsAndUnequalSourceLists(t *testing.T) {
	fire := flame.Fire{Radius: 0.7, Height: 2.4, Absorption: 1,
		Hottest: measure.Kelvin(1750), Edge: measure.Kelvin(900), Seed: 5}
	cooler := fire
	cooler.Hottest = measure.Kelvin(1200)
	body, err := flame.NewBody(fire, 1)
	if err != nil {
		t.Fatal(err)
	}
	one, err := flame.NewField([]flame.Ignition{{Fire: fire}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	two, err := flame.NewField([]flame.Ignition{{Fire: fire}, {Fire: cooler}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	// NewField's documented upper endpoint must remain GPU-bakeable.
	maximum := make([]flame.Ignition, 64)
	for i := range maximum {
		maximum[i] = flame.Ignition{Fire: fire}
	}
	many, err := flame.NewField(maximum, 1)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(0, 0.5, 0)
	if !(one.Density(point) > 0) || one.Shine().At(point) == two.Shine().At(point) {
		t.Fatal("custom emission fixture has no visible difference")
	}
	box := func(med *flame.Field) boxed {
		bounds := med.Bounds()
		return boxed{med: med, shine: med.Shine(), low: bounds.Min,
			high: bounds.Max, cells: [3]int{3, 3, 3}}
	}
	for _, med := range []*flame.Field{one, two, many} {
		grid := box(med)
		if _, ok := flameField(grid); !ok {
			t.Fatal("matching field was not recognized")
		}
		density, shine, _ := grid.asked()
		if len(density) != 27 || len(shine) != 81 {
			t.Fatal("three-by-three-by-three reference grid changed")
		}
		encoded := newScene()
		const prefix = 0x12345678
		encoded.boxes.word(prefix)
		first, shone, ok := encoded.deferFlame(grid)
		if !ok {
			t.Fatal("matching field was not retained for baking")
		}
		// The shader reads 27 density floats followed by 81 RGB floats,
		// after the pre-existing word. The bake job must own that range.
		if first != 1 || shone != 28 || len(encoded.flames) != 1 ||
			len(encoded.boxes.bytes) != 436 {
			t.Fatalf("wrong deferred layout: density=%d shine=%d jobs=%d bytes=%d",
				first, shone, len(encoded.flames), len(encoded.boxes.bytes))
		}
		job := encoded.flames[0]
		if job.density != first || job.shine != shone ||
			job.grid.cells != grid.cells || job.grid.low != grid.low ||
			job.grid.high != grid.high ||
			binary.LittleEndian.Uint32(encoded.boxes.bytes[:4]) != prefix {
			t.Fatal("deferred job lost its grid, offsets or preceding data")
		}
	}
	for _, test := range []struct {
		name string
		grid boxed
	}{
		{"extra emission source", func() boxed { g := box(one); g.shine = two.Shine(); return g }()},
		{"missing emission source", func() boxed { g := box(two); g.shine = one.Shine(); return g }()},
		{"single density and composite emission", func() boxed { g := box(one); g.med = body; return g }()},
		{"composite density and single emission", func() boxed { g := box(one); g.shine = body.Shine(); return g }()},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, ok := flameField(test.grid); ok {
				t.Fatal("custom density/emission pair was replaced with a shared-prefix field")
			}
			encoded := newScene()
			if _, _, ok := encoded.deferFlame(test.grid); ok ||
				len(encoded.flames) != 0 || len(encoded.boxes.bytes) != 0 {
				t.Fatal("custom pair reserved a deferred bake instead of retaining CPU sampling")
			}
		})
	}
}

// Period is stored in nanoseconds. A public Body can have a smaller positive
// physical puff period, which truncates to zero and cannot drive the shader.
func TestFlamePairRefusesASubnanosecondPuffClock(t *testing.T) {
	fire := flame.Fire{Radius: 0.7, Height: 2.4, Absorption: 1,
		Hottest: measure.Kelvin(1750), Edge: measure.Kelvin(900), Seed: 5}
	ordinary, err := flame.NewBody(fire, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pairedFlame(flame.Piece{Body: ordinary}, flame.Piece{Body: ordinary}); !ok {
		t.Fatal("ordinary positive puff clock refused")
	}
	fire.Radius = 2e-18
	first, err := flame.NewBody(fire, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, _, firstPeriod, _ := first.Samples()
	if firstPeriod != time.Nanosecond {
		t.Fatalf("first positive clock fixture: %v", firstPeriod)
	}
	if _, ok := pairedFlame(flame.Piece{Body: first}, flame.Piece{Body: first}); !ok {
		t.Fatal("first positive puff clock refused")
	}
	fire.Radius = 1e-20
	tiny, err := flame.NewBody(fire, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, _, period, glow := tiny.Samples()
	if period != 0 || len(glow) != 256 {
		t.Fatalf("public tiny-body fixture lost its zero clock or full table: %v %d", period, len(glow))
	}
	if _, ok := pairedFlame(flame.Piece{Body: tiny}, flame.Piece{Body: tiny}); ok {
		t.Fatal("zero puff clock admitted to GPU baking")
	}
}
