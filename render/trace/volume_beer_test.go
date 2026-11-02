package trace

import (
	"math"
	"sort"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// a lone volume that only glows, in a world with no lamp and no sky,
// takes the light behind it down by beer's law and adds its own glow by
// as much as it took.
//
// for a thickness k through a way d the depth is tau = k d, what is let
// through is exp(-tau), and what shows is the glow times 1 - exp(-tau).
//
// the ray runs through the middle of a block, or across a ball at an
// offset, and the thickness is per unit of the volume's own space, so a
// volume scaled up keeps its look: it is as thick through the middle at
// any scale.
//
// worked by hand: the block of half-width 1 is 2 across, and the ball of
// radius 1 at an offset of 0.6 is crossed by a chord of 2 root
// (1 - 0.36), 1.6.
func TestAVolumeDimsByBeersLawInItsOwnUnits(t *testing.T) {
	glow := maths.RGB(0.9, 0.5, 0.2)
	chord := 2 * math.Sqrt(1-0.6*0.6)
	for _, c := range []struct {
		name  string
		m     medium.Medium
		at    maths.Vec
		scale float64
		x, y  float64
		tau   float64
	}{
		{"a block", block{k: 0.5, half: 1}, maths.Vec{}, 1, 0, 0, 0.5 * 2},
		{"a block, doubled", block{k: 0.5, half: 1}, maths.Vec{}, 2, 0, 0, 0.5 * 2},
		{"a ball", ball{k: 0.8, radius: 1}, maths.Vec{}, 1, 0.6, 0, 0.8 * chord},
		{"a ball, tripled", ball{k: 0.8, radius: 1}, maths.Vec{}, 3, 1.8, 0, 0.8 * chord},
		{"a ball, moved and doubled", ball{k: 0.8, radius: 1}, maths.V(2, -1, -3),
			2, 3.2, -1, 0.8 * chord},
	} {
		w := thing.NewWorld()
		w.AddVolume(thing.Volume{Medium: c.m, Material: glowOf(glow),
			Motion: at(c.at, c.scale)})
		got := Trace(w, along(c.x, c.y), 0)
		seen := 1 - math.Exp(-c.tau)
		colours(t, c.name, got, glow.Scale(seen), 2e-6)
		if !within(got.Cover, seen, 2e-6) {
			t.Errorf("%s: covers %v, want %v", c.name, got.Cover, seen)
		}
	}
}

// a surface in the middle of a volume cuts it: only the way in front of
// the surface counts. a plane at y = 0 across a block or a ball of radius
// 1 scaled by 2 leaves, looking straight down, half of its way, which is
// 1 in its own units whatever the scale, so tau is k times 1.
func TestASurfaceCutsAVolumeInTwo(t *testing.T) {
	glow := maths.RGB(0.9, 0.5, 0.2)
	for _, c := range []struct {
		name string
		m    medium.Medium
		tau  float64
	}{
		{"a block", block{k: 0.5, half: 1}, 0.5},
		{"a ball", ball{k: 0.8, radius: 1}, 0.8},
	} {
		w := thing.NewWorld()
		w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Name: "coal"}})
		w.AddVolume(thing.Volume{Medium: c.m, Material: glowOf(glow),
			Motion: at(maths.Vec{}, 2)})
		got := Trace(w, maths.Ray{At: maths.V(0, 20, 0), Dir: maths.V(0, -1, 0)}, 0)
		colours(t, c.name, got, glow.Scale(1-math.Exp(-c.tau)), 2e-6)
		if got.Cover != 1 || got.Depth != 20 {
			t.Errorf("%s: the floor is covered %v at %v", c.name, got.Cover, got.Depth)
		}
	}
}

// a crossing is cut into 48 steps and no more: a volume that is as thick
// beyond its box as in it shows how many steps are taken, since each is a
// 48th of the way through the box. the thickness is 0.25 over 2, so tau
// is 0.5. this holds the 48 steps as the whole of what is summed.
func TestACrossingIsTakenInFortyEightSteps(t *testing.T) {
	w := thing.NewWorld()
	w.AddVolume(thing.Volume{Medium: leaky{block{k: 0.25, half: 1}},
		Material: glowOf(maths.Grey(1))})
	got := Trace(w, along(0, 0), 0)
	if want := 1 - math.Exp(-0.5); !within(got.R, want, 2e-6) {
		t.Errorf("the leaky block shows %v, want %v", got.R, want)
	}
}

// two volumes that overlap are added a step at a time, nearest step
// first. where two steps are as near as each other the volume the ray
// entered first goes first, and where they were entered together the one
// the world lists first.
//
// two blocks of 3 across, a 16th a step, the second set 0.25 behind the
// first, so the 5th step of the first is as near as the 1st of the
// second all the way down. every number here is a whole sixteenth, so
// the ties are exact.
//
// the first glows red, the second blue, and the answer is the steps
// walked in order, worked here by sorting them.
func TestOverlappingVolumesAreAddedNearestStepFirst(t *testing.T) {
	const k = 0.5
	redder, bluer := maths.RGB(1, 0, 0), maths.RGB(0, 0, 1)
	for _, c := range []struct {
		name string
		// where the blocks sit along the ray, red and blue, and the
		// entry of each along it; red is listed first in the world
		// when listRed
		redAt, blueAt float64
		listRed       bool
	}{
		{"red entered first, listed last", 0.125, -0.125, false},
		{"red entered first, listed first", 0.125, -0.125, true},
		{"entered together, red listed first", 0, 0, true},
		{"entered together, blue listed first", 0, 0, false},
	} {
		w := thing.NewWorld()
		red := thing.Volume{Medium: block{k: k, half: 1.5}, Material: glowOf(redder),
			Motion: at(maths.V(0, 0, c.redAt), 1)}
		blue := thing.Volume{Medium: block{k: k, half: 1.5}, Material: glowOf(bluer),
			Motion: at(maths.V(0, 0, c.blueAt), 1)}
		if c.listRed {
			w.AddVolume(red)
			w.AddVolume(blue)
		} else {
			w.AddVolume(blue)
			w.AddVolume(red)
		}
		got := Trace(w, along(0, 0), 0)

		// the entries of the two along the ray, 20 less the top face;
		// ties in entry go to the one listed first
		type step struct {
			at    float64
			rank  int // 0 for the volume entered first, then listed first
			isRed bool
		}
		entryRed, entryBlue := 20-(c.redAt+1.5), 20-(c.blueAt+1.5)
		listedRed, listedBlue := 0, 1
		if !c.listRed {
			listedRed, listedBlue = 1, 0
		}
		var steps []step
		for _, v := range []struct {
			enter  float64
			isRed  bool
			listed int
		}{{entryRed, true, listedRed}, {entryBlue, false, listedBlue}} {
			rank := v.listed
			if entryRed != entryBlue && v.enter == math.Min(entryRed, entryBlue) {
				rank = -1 // entered before the other
			}
			for i := 0; i < 48; i++ {
				steps = append(steps, step{v.enter + (float64(i)+0.5)/16, rank, v.isRed})
			}
		}
		sort.SliceStable(steps, func(i, j int) bool {
			if steps[i].at != steps[j].at {
				return steps[i].at < steps[j].at
			}
			return steps[i].rank < steps[j].rank
		})
		keep, red1, blue1 := 1.0, 0.0, 0.0
		for _, s := range steps {
			taken := 1 - math.Exp(-k/16)
			if s.isRed {
				red1 += keep * taken
			} else {
				blue1 += keep * taken
			}
			keep *= 1 - taken
		}
		colours(t, c.name, got, maths.RGB(red1, 0, blue1), 2e-6)
	}
}

// volumes that do not overlap are taken one after the other, and what is
// behind the first shows through it by what it let through: a red block
// of tau 1 in front of a blue block of tau 2 is red by 1 - exp(-1) and
// blue by exp(-1) times 1 - exp(-2).
func TestVolumesOneBehindAnotherDimOneAnother(t *testing.T) {
	w := thing.NewWorld()
	w.AddVolume(thing.Volume{Medium: block{k: 0.5, half: 1}, Material: glowOf(maths.RGB(1, 0, 0)),
		Motion: at(maths.V(0, 0, 4), 1)})
	w.AddVolume(thing.Volume{Medium: block{k: 1, half: 1}, Material: glowOf(maths.RGB(0, 0, 1))})
	got := Trace(w, along(0, 0), 0)
	colours(t, "two blocks in a row", got, maths.RGB(1-math.Exp(-1), 0,
		math.Exp(-1)*(1-math.Exp(-2))), 2e-6)
}

// a volume lit by a sun scatters its own colour of it, evenly every way,
// by half what lands on a surface, less what the volume itself takes from
// the sun on the way out.
//
// the sun is along plus x and the ray across the middle, so every step is
// 1 from the sun's side of the block, and the light on every step is the
// colour times the sun times 0.5 times exp(-k) for the way out; the pixel
// is that times 1 - exp(-tau) with tau = 2 k.
//
// a volume with no material is the default grey, 0.8; one with a texture
// takes its colour from the texture, and one with a glow gives that off
// beside.
func TestALitVolumeScattersHalfOfTheSunTimesWhatReachesIt(t *testing.T) {
	const k = 0.6
	sun := maths.RGB(1, 0.8, 0.5)
	reach := math.Exp(-k)
	seen := 1 - math.Exp(-2*k)
	for _, c := range []struct {
		name string
		m    *thing.Material
		want maths.Light
	}{
		{"painted", &thing.Material{Colour: maths.RGB(0.9, 0.6, 0.3)},
			maths.RGB(0.9, 0.6, 0.3).Times(sun).Scale(0.5 * reach * seen)},
		{"unpainted", nil, maths.Grey(0.8).Times(sun).Scale(0.5 * reach * seen)},
		{"textured", &thing.Material{Colour: maths.Grey(1), Texture: paint(maths.RGB(0.2, 0.4, 0.1))},
			maths.RGB(0.2, 0.4, 0.1).Times(sun).Scale(0.5 * reach * seen)},
		{"glowing", &thing.Material{Colour: maths.Grey(1), Glow: maths.Grey(0.3)},
			maths.Grey(1).Times(sun).Scale(0.5 * reach * seen).Add(maths.Grey(0.3 * seen))},
	} {
		w := thing.NewWorld()
		w.AddVolume(thing.Volume{Medium: block{k: k, half: 1}, Material: c.m})
		w.Lamp(thing.Sun(maths.V(1, 0, 0), sun))
		colours(t, c.name, Trace(w, along(0, 0), 0), c.want, 2e-6)
	}
}

// paint is a texture of one colour everywhere.
type paint maths.Light

func (p paint) At(maths.Vec) maths.Light { return maths.Light(p) }
