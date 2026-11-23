package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func mediumEye(from, direction maths.Vec) maths.Subspace {
	return lookingAt(from, from.Add(direction))
}

func mediumWorld() *thing.World {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Name: "water surface", Through: 1, Gloss: 1, Index: 1.33,
	}})
	return w
}

func TestRenderWithInitialWaterUsesSnellAndFresnelAtFlatBoundary(t *testing.T) {
	w := mediumWorld()
	from := maths.V(0, -1, 0)
	direction := maths.V(math.Sin(maths.Degree*30),
		math.Cos(maths.Degree*30), 0)
	entryX := direction.X / direction.Y
	sineOut := 1.33 * direction.X
	cosineOut := math.Sqrt(1 - sineOut*sineOut)
	onPath := maths.V(entryX+3*sineOut/cosineOut, 3, 0)
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.25},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   resting(onPath)})

	got, err := RenderWithInitialMedia(w, mediumEye(from, direction),
		film.Pinhole{}, 0, 1, 1, []InitialMedium{{Index: 1.33}})
	if err != nil {
		t.Fatal(err)
	}
	// At thirty degrees in water, sin(theta_out)=1.33 sin(30). The
	// unpolarised Fresnel share from water to air is 0.025090, leaving
	// 0.974910 for the glowing sphere on the refracted path.
	const want = 0.974910
	red := float64(got.At(0, 0).R)
	if !within(red, want, 2e-6) {
		t.Fatalf("refracted path returned %g, want %g", got.At(0, 0).R,
			want)
	}
}

func TestRenderWithInitialWaterUsesTotalInternalReflection(t *testing.T) {
	w := mediumWorld()
	planeAt := resting(maths.V(0, -3, 0))
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   planeAt})
	from := maths.V(0, -1, 0)
	direction := maths.V(math.Sin(maths.Degree*60),
		math.Cos(maths.Degree*60), 0)
	got, err := RenderWithInitialMedia(w, mediumEye(from, direction),
		film.Pinhole{}, 0, 1, 1, []InitialMedium{{Index: 1.33}})
	if err != nil {
		t.Fatal(err)
	}
	if got.At(0, 0).R != 1 {
		t.Fatalf("TIR reflected %g of the light, want all of it",
			got.At(0, 0).R)
	}
}

func TestNestedInitialMediaUseRankAndEntryOrder(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.5, Rank: 2,
	}})
	planeAt := resting(maths.V(0, 3, 0))
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   planeAt})
	from := maths.V(0, -1, 0)
	got, err := RenderWithInitialMedia(w, mediumEye(from, maths.Up),
		film.Pinhole{}, 0, 1, 1, []InitialMedium{
			{Index: 1.33, Rank: 0}, {Index: 1.5, Rank: 2},
		})
	if err != nil {
		t.Fatal(err)
	}
	// The rank-2 glass is the active medium at the boundary. It leaves
	// into rank-0 water, so normal-incidence reflection is
	// ((1.5-1.33)/(1.5+1.33))^2 = 0.00360847.
	const want = 0.99639153
	red := float64(got.At(0, 0).R)
	if !within(red, want, 1e-6) {
		t.Fatalf("nested start state returned %g, want %g",
			got.At(0, 0).R, want)
	}
}

func TestInitialMediumStackUsesHighestRankThenLastEntry(t *testing.T) {
	inside, count, err := initialMedia([]InitialMedium{
		{Index: 1.5, Rank: 2}, {Index: 1.33, Rank: 0},
		{Index: 1.4, Rank: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	active, present := (look{inside: inside, holding: count}).medium()
	if !present || active != (held{index: 1.4, rank: 2}) {
		t.Fatalf("active initial medium = %+v, present %t", active, present)
	}
}

func TestEmptyInitialMediaIsBitIdenticalToExistingRender(t *testing.T) {
	w := thing.NewWorld()
	w.Sky = compass{}
	eye := lookingAt(maths.V(0, 0, -3), maths.Vec{})
	plain, err := Render(w, eye, film.Pinhole{Wide: 0.2}, 0, 7, 5)
	if err != nil {
		t.Fatal(err)
	}
	for name, initial := range map[string][]InitialMedium{
		"nil": nil, "empty": {},
	} {
		got, err := RenderWithInitialMedia(w, eye, film.Pinhole{Wide: 0.2},
			0, 7, 5, initial)
		if err != nil {
			t.Fatalf("%s initial state: %v", name, err)
		}
		for i := range plain.Pix {
			if got.Pix[i] != plain.Pix[i] {
				t.Fatalf("%s initial state changed pixel %d: got %+v, want %+v",
					name, i, got.Pix[i], plain.Pix[i])
			}
		}
	}

	plainSampled, err := Sampled(w, eye, film.Pinhole{Wide: 0.2}, 0,
		7, 5, 2)
	if err != nil {
		t.Fatal(err)
	}
	withEmpty, err := SampledWithInitialMedia(w, eye,
		film.Pinhole{Wide: 0.2}, 0, 7, 5, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := range plainSampled.Pix {
		if plainSampled.Pix[i] != withEmpty.Pix[i] {
			t.Fatalf("empty sampled state changed pixel %d", i)
		}
	}
}

func TestSampledWithInitialMediaSharesTheRenderPath(t *testing.T) {
	w := mediumWorld()
	planeAt := resting(maths.V(0, 3, 0))
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   planeAt})
	from := maths.V(0, -1, 0)
	eye := mediumEye(from, maths.Up)
	media := []InitialMedium{{Index: 1.33}}
	whole, err := SampledWithInitialMedia(w, eye, film.Pinhole{}, 0,
		1, 1, 2, media)
	if err != nil {
		t.Fatal(err)
	}
	each, err := RenderWithInitialMedia(w, eye, film.Pinhole{}, 0,
		2, 2, media)
	if err != nil {
		t.Fatal(err)
	}
	if want := each.At(0, 0); whole.At(0, 0) != want {
		t.Fatalf("one-pixel supersample = %+v, one ray = %+v",
			whole.At(0, 0), want)
	}
}

func TestInitialMediaRejectInvalidState(t *testing.T) {
	w := thing.NewWorld()
	eye := lookingAt(maths.V(0, 0, -3), maths.Vec{})
	tooMany := make([]InitialMedium, mostHeld+1)
	for i := range tooMany {
		tooMany[i] = InitialMedium{Index: float64(i + 1), Rank: i}
	}
	for _, test := range []struct {
		name  string
		media []InitialMedium
	}{
		{"zero index", []InitialMedium{{Index: 0}}},
		{"negative index", []InitialMedium{{Index: -1}}},
		{"nan index", []InitialMedium{{Index: math.NaN()}}},
		{"infinite index", []InitialMedium{{Index: math.Inf(1)}}},
		{"below bound", []InitialMedium{{Index: 1e-10}}},
		{"above bound", []InitialMedium{{Index: 1e10}}},
		{"duplicate medium", []InitialMedium{{Index: 1.33, Rank: 1},
			{Index: 1.33, Rank: 1}}},
		{"stack too deep", tooMany},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := RenderWithInitialMedia(w, eye, film.Pinhole{}, 0,
				1, 1, test.media); err == nil {
				t.Fatal("invalid start state accepted")
			}
		})
	}
}

func TestInitialMediaAcceptsDeclaredIndexEnvelope(t *testing.T) {
	w := thing.NewWorld()
	w.Sky = compass{}
	eye := lookingAt(maths.V(0, 0, -3), maths.Vec{})
	for _, index := range []float64{1e-9, 1e9} {
		if _, err := RenderWithInitialMedia(w, eye, film.Pinhole{}, 0,
			1, 1, []InitialMedium{{Index: index}}); err != nil {
			t.Errorf("endpoint index %g refused: %v", index, err)
		}
	}
}
