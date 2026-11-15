//go:build draw

package section_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/section"
)

// What the trials of what a section can reach pick, each by the test
// author.
const (
	// clayRadius is the size of the lit ball that a shadow falls on.
	clayRadius = 1.0
	// casterRadius is the size of the ball whose shadow falls on it.
	casterRadius = 0.7
	// mirrorRadius and glassRadius are the sizes of the mirror and the
	// glass ball.
	mirrorRadius = 1.2
	glassRadius  = 1.2
	// brighter is the sun's strength after it changes: more than half
	// again, so the picture of anything it lights moves.
	brighter = 5.0
	// glassThrough is the share of light a clear dielectric lets
	// through, defined as one: all that is not reflected.
	glassThrough = 1.0
	// glassGloss is how much of the Fresnel reflection glass keeps, a
	// third: picked, so it both reflects and lets through.
	glassGloss = 0.3
	// puffRadius and puffThick are the size and density of a puff the
	// trials of volumes use.
	puffRadius = 1.5
	puffThick  = 1.0
	// keptAmongFew is the fewest sections kept when a few of twenty-four
	// are what an object can show in: picked after counting.
	keptAmongFew = 6
	// hazeDensity is the haze's extinction a scene unit: a haze that
	// dims a ball ten units off by two fifths.
	hazeDensity = 0.05
	// hazeGrey is the haze's colour, a mid grey.
	hazeGrey = 0.3
	// zenithBefore and zenithAfter are the sky's colour straight up
	// before and after it changes.
	zenithBefore = 0.1
	zenithAfter  = 0.4
	// fieldMass and fieldReach are what bends the rays in a world with
	// a field, as the existing section tests have it.
	fieldMass  = 1.0
	fieldReach = 1.0
)

// clay is a ball that a lamp lights, at a named place.
func clay(t *testing.T, radius float64, name string) *thing.Thing {
	t.Helper()
	made := thing.Diffuse(name, maths.Grey(picked(t, "clay-grey")))
	return solid(radius, made, placed(t, name))
}

// a ball that shadows the clay ball from the sun, put between them in
// the second frame and in sight elsewhere, changes the clay ball's
// section and no kept section.
func TestAShadowFromAMoverChangesNoKeptSection(t *testing.T) {
	base := append(stars(t), clay(t, clayRadius, "clay low"))
	caster := clay(t, casterRadius, "caster")
	before := worldOf(base, nil, sun(t))
	after := worldOf(append(slices.Clone(base), caster), nil, sun(t))
	found := newTrial("a shadow falls", before, after).strict(t)
	tried(t, found, keptAmongFew)
}

// a mirror reflects a ball behind the eye that no section can see: the
// ball moves, no section's rays meet it, and the mirror's section is
// still drawn again, because the mirror is no self-lit body.
func TestAMirrorShowsAMoverNoSectionSees(t *testing.T) {
	mirror := thing.Polished("mirror", maths.Grey(nothing), 1, nothing)
	base := append(stars(t), solid(mirrorRadius, mirror, placed(t, "mirror")))
	orbiter := glowing(t, casterRadius, starGlow, "behind centre")
	back := glowing(t, casterRadius*two, starGlow, "behind centre")
	before := worldOf(append(slices.Clone(base), orbiter), nil, sun(t))
	after := worldOf(append(slices.Clone(base), back), nil, sun(t))
	found := newTrial("a mirror shows a mover", before, after).strict(t)
	tried(t, found, keptAmongFew)
}

// a self-lit ball seen through a glass ball: the ball behind moves; the
// sections the glass is in are never kept, and no kept section differs.
// The glass both reflects and lets light through, or only lets it through.
func TestASelfLitBodyThroughGlassIsNotKept(t *testing.T) {
	for _, gloss := range []float64{glassGloss, nothing} {
		throughGlass(t, gloss)
	}
}

// throughGlass is the trial of a ball behind glass of a gloss, that
// moves.
func throughGlass(t *testing.T, gloss float64) {
	t.Helper()
	glass := thing.Polished("glass", maths.Grey(nothing), gloss, nothing)
	glass.Through = glassThrough
	behind := glowing(t, starRadius, starGlow, "star behind glass")
	base := append(stars(t), solid(glassRadius, glass, placed(t, "glass")),
		behind)
	before := worldOf(base, nil, sun(t))
	moved := *behind
	moved.Motion = circling(t, nothing)
	swapped := append(slices.Clone(base[:len(base)-one]), &moved)
	again := worldOf(swapped, nil, sun(t))
	name := fmt.Sprintf("a ball behind glass of gloss %g moves", gloss)
	found := newTrial(name, before, again).strict(t)
	tried(t, found, keptAmongFew)
}

// a glass ball that only lets light through bends the rays of its
// sections to a lit ball that none of their sights reach; when the sun
// changes, the glass's sections change, and none is kept.
func TestGlassShowsALitBodyOutsideItsSights(t *testing.T) {
	glass := thing.Polished("glass", maths.Grey(nothing), nothing, nothing)
	glass.Through = glassThrough
	base := append(stars(t), solid(glassRadius, glass, placed(t, "glass")),
		clay(t, clayRadius, "clay beside glass"))
	before := worldOf(base, nil, sun(t))
	after := worldOf(base, nil, thing.Sun(place(t, "sun toward"),
		maths.Grey(brighter)))
	found := newTrial("glass bends rays to a lit ball", before, after).strict(t)
	tried(t, found, keptAmongFew)
}

// a self-lit ball behind a steady volume: when a lamp changes, the
// sections the volume is in are not kept, with no drift, and none kept
// differs.
func TestASelfLitBodyBehindAVolumeIsNotKept(t *testing.T) {
	puff := puffAt(t, puffRadius, puffThick, "puff centre")
	behind := glowing(t, starRadius, starGlow, "star behind puff")
	things := append(stars(t), behind)
	volumes := []*thing.Volume{puff}
	before := worldOf(things, volumes, sun(t))
	after := worldOf(things, volumes, thing.Sun(place(t, "sun toward"),
		maths.Grey(brighter)))
	found := newTrial("a lamp changes, puff", before, after)
	got := found.strict(t)
	tried(t, got, keptAmongFew)
	if inside := found.keptInSightOf(t, nothing, "puff centre",
		puffRadius); inside != nothing {
		t.Errorf("%d sections the puff is in were kept, with no drift",
			inside)
	}
	same := newTrial("nothing changes, puff", before, before)
	if inside := same.keptInSightOf(t, nothing, "puff centre",
		puffRadius); inside == nothing {
		t.Errorf("no section the puff is in is kept when nothing changes")
	}
}

// keptInSightOf is how many sections whose sight meets a ball at a named
// place are kept under a drift.
func (try trial) keptInSightOf(t *testing.T, drift float64, name string,
	radius float64) (kept int) {
	t.Helper()
	_, sights := try.sights()
	change := section.Between(try.before, try.after, try.was,
		try.now).Allowing(drift)
	for _, sight := range sights {
		if change.Keeps(sight) && sight.Meets(place(t, name), radius) {
			kept++
		}
	}
	return kept
}

// a changed sky keeps nothing, and the pictures differ.
func TestAChangedSkyKeepsNothing(t *testing.T) {
	base := stars(t)
	before, after := worldOf(base, nil), worldOf(base, nil)
	before.Sky = sky.Gradient{Zenith: maths.Grey(zenithBefore)}
	after.Sky = sky.Gradient{Zenith: maths.Grey(zenithAfter)}
	found := newTrial("a changed sky", before, after).strict(t)
	if found.kept != nothing || found.changed == nothing {
		t.Errorf("%d sections kept, %d changed: a changed sky keeps none",
			found.kept, found.changed)
	}
}

// a world with a field keeps nothing, changed or not, and the pictures
// differ when a lamp does.
func TestAFieldKeepsNothing(t *testing.T) {
	field := &thing.Field{Mass: fieldMass, Reach: fieldReach,
		Motion: thing.Still(placed(t, "orbit centre"))}
	base := append(stars(t), clay(t, clayRadius, "clay low"))
	before, after := worldOf(base, nil, sun(t)), worldOf(base, nil,
		thing.Sun(place(t, "sun toward"), maths.Grey(brighter)))
	before.Fields, after.Fields = []*thing.Field{field}, []*thing.Field{field}
	found := newTrial("a field, a lamp", before, after).strict(t)
	if found.kept != nothing || found.changed == nothing {
		t.Errorf("%d kept, %d changed in a world with a field",
			found.kept, found.changed)
	}
	still := newTrial("a field, nothing changes", before, before).strict(t)
	if still.kept != nothing {
		t.Errorf("%d kept in a world with a field and no change", still.kept)
	}
}

// a world with a haze keeps nothing when a lamp changes, and the
// pictures differ; and keeps all of it when nothing does.
func TestAHazeKeepsNothingWhenSomethingChanges(t *testing.T) {
	base := append(stars(t), clay(t, clayRadius, "clay low"))
	before, after := worldOf(base, nil, sun(t)), worldOf(base, nil,
		thing.Sun(place(t, "sun toward"), maths.Grey(brighter)))
	haze := thing.Haze{Density: hazeDensity, Colour: maths.Grey(hazeGrey)}
	before.Haze, after.Haze = haze, haze
	found := newTrial("a haze, a lamp", before, after).strict(t)
	if found.kept != nothing || found.changed == nothing {
		t.Errorf("%d kept, %d changed in a hazy world", found.kept,
			found.changed)
	}
	same := newTrial("a haze, nothing changes", before, before).strict(t)
	if same.kept != same.total {
		t.Errorf("%d of %d kept in a hazy world nothing changed in",
			same.kept, same.total)
	}
}

// widerPixel is a pixel's width a world may say, in radians: picked, any
// width that is not the unsaid zero.
const widerPixel = 0.01

// two worlds that say different widths for a pixel keep nothing: the
// width goes to every texture that paints by it, so no section can be
// known to be the same. This asks the rule alone; the stars here are not
// painted by their pixel's width, so their pictures do not differ.
func TestAChangedPixelWidthKeepsNothing(t *testing.T) {
	base := stars(t)
	before, after := worldOf(base, nil), worldOf(base, nil)
	after.Pixel = widerPixel
	found := newTrial("a changed pixel width", before, after).strict(t)
	if found.kept != nothing {
		t.Errorf("%d sections kept though the pixel's width changed",
			found.kept)
	}
	same := newTrial("the same pixel width", after, after).strict(t)
	if same.kept != same.total {
		t.Errorf("%d of %d kept with nothing changed", same.kept, same.total)
	}
}
