//go:build draw

package section_test

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

const (
	// missX, missY, missFar and missRadius are the pixel, the distance
	// along its ray and the radius of the first ball the close lens's
	// sight refuses at a side of sixteen, found by the sight test with
	// the seeds of numbers.yaml.
	missX      = 95
	missY      = 21
	missFar    = 0.0354
	missRadius = 0.0003
	// oddX, oddY, oddFar and oddRadius are the pixel, the distance along
	// its ray and the radius of the ball that the sight of the odd frame
	// refuses, found by the sight test with the seeds of numbers.yaml.
	oddX      = 53
	oddY      = 32
	oddFar    = 1.837
	oddRadius = 0.0269
)

// a ball that the ray of one pixel of a lens with a wide aperture meets,
// close to the glass and small, is in no kept section: it is drawn in the
// whole frame and not in the frame made by keeping what is kept. The
// pixel and the ball are the first refused by the sight test above.
func TestStrictHoldsForASmallBallNearAWideApertureLens(t *testing.T) {
	glass := thinLens(t, closeFocal, closeStop, closeFocus)
	view := eye{lens: glass, wide: trialWide, high: trialHigh}
	ray := view.look(missX, missY)
	centre := ray.At.Add(ray.Dir.Scale(missFar))
	speck := solid(missRadius, thing.Diffuse("speck", maths.Grey(nothing)),
		maths.NewSubspace(centre, maths.Orientation{}, nothing))
	speck.Material.Glow = maths.Grey(starGlow)
	before, after := worldOf(nil, nil), worldOf([]*thing.Thing{speck}, nil)
	try := newTrial("a speck near a wide aperture", before, after)
	try.view = view
	found := try.strict(t)
	if found.changed == nothing {
		t.Errorf("the speck changed no section: nothing was tried")
	}
}

// through a lens with an aperture, a mover keeps no section its rays can
// meet and no kept section differs in any bit.
func TestStrictThroughALensWithAnApertureHolds(t *testing.T) {
	base := append(stars(t), clay(t, clayRadius, "clay low"))
	spark := glowing(t, moverRadius, starGlow, "spark")
	spark.Motion = circling(t, nothing)
	before := worldOf(base, nil, sun(t))
	after := worldOf(append(base[:len(base):len(base)], spark), nil, sun(t))
	try := newTrial("an aperture", before, after)
	try.view.lens = thinLens(t, farFocal, farStop, farFocus)
	found := try.strict(t)
	tried(t, found, keptAmongFew)
}

// through a wide pinhole, a mover keeps no section its rays can meet and
// no kept section differs in any bit.
func TestStrictThroughAWideLensHolds(t *testing.T) {
	base := append(stars(t), clay(t, clayRadius, "clay low"))
	spark := glowing(t, moverRadius, starGlow, "spark")
	spark.Motion = circling(t, nothing)
	before := worldOf(base, nil, sun(t))
	after := worldOf(append(base[:len(base):len(base)], spark), nil, sun(t))
	try := newTrial("a wide lens", before, after)
	try.view.lens = film.Pinhole{Wide: degrees150 * maths.Degree}
	found := try.strict(t)
	tried(t, found, nothing)
}

// a small ball that the ray of one pixel of the odd frame meets is in no
// kept section: the pixel and ball are the first the sight refuses.
func TestStrictHoldsForABallInANearlyHalfTurnLens(t *testing.T) {
	view := oddView(nothing)
	ray := view.look(oddX, oddY)
	centre := ray.At.Add(ray.Dir.Scale(oddFar))
	speck := solid(oddRadius, thing.Diffuse("speck", maths.Grey(nothing)),
		maths.NewSubspace(centre, maths.Orientation{}, nothing))
	speck.Material.Glow = maths.Grey(starGlow)
	before, after := worldOf(nil, nil), worldOf([]*thing.Thing{speck}, nil)
	try := newTrial("a ball near a half turn", before, after)
	try.view, try.side = view, oddSide
	found := try.strict(t)
	if found.changed == nothing {
		t.Errorf("the ball changed no section: nothing was tried")
	}
}
