package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// reachScale and reachRadius are the volume the box tests place: a puff of
// radius reachRadius in a space of scale reachScale, so a ball of their
// product in the world.
const reachScale, reachRadius = 2.0, 1.5

// reachAt is where the box tests place a volume, off the origin.
var reachAt = maths.V(30, -4, 12)

// puffRow is the row of one puff placed in space, boxes written.
func puffRow(space maths.Subspace) []byte {
	encoded := newScene()
	encoded.volume(space, haze{puff: medium.Puff{Radius: reachRadius}}, 0)
	encoded.boxVolumes()
	return encoded.volumes.bytes[:volumeStruct.size()]
}

// reachBox is the box a row keeps in the world, and whether it has one.
func reachBox(row []byte) (low, high [3]float64, walled bool) {
	return volumeCorner(row, "world_low"), volumeCorner(row, "world_high"),
		volumeNumber(row, "world_low", 3) == 1
}

// turnedSpace is a space at reachAt, turned and scaled.
func turnedSpace() maths.Subspace {
	return maths.Subspace{Origin: reachAt, Scale: reachScale,
		Orientation: maths.LookAt(reachAt, maths.V(0, 9, 0))}
}

func TestAPuffsBoxHoldsItsBallAndLittleMore(t *testing.T) {
	for name, space := range map[string]maths.Subspace{
		"upright": {Origin: reachAt, Scale: reachScale,
			Orientation: maths.Identity},
		"turned": turnedSpace()} {
		low, high, walled := reachBox(puffRow(space))
		if !walled {
			t.Errorf("%s: a puff has no box", name)
			continue
		}
		ball := reachScale * reachRadius
		centre := [3]float64{reachAt.X, reachAt.Y, reachAt.Z}
		for part := range centre {
			if low[part] > centre[part]-ball || high[part] < centre[part]+ball {
				t.Errorf("%s: axis %d box %v..%v is inside the ball %v",
					name, part, low[part], high[part], ball)
			}
			// a turned cube is a wider box, so the bound is the cube's
			if high[part]-low[part] > 2*ball*math.Sqrt(3)*1.01 {
				t.Errorf("%s: axis %d box is %v wide for a ball of %v",
					name, part, high[part]-low[part], 2*ball)
			}
		}
	}
}

// reachHit is whether a segment from at along way, as far as far, comes
// within radius of a centre: the ball a puff is, and what span finds.
func reachHit(at, way, centre [3]float64, radius, far float64) bool {
	var closest, square float64
	for part := range at {
		from := at[part] - centre[part]
		closest -= from * way[part]
		square += from * from
	}
	disc := closest*closest - (square - radius*radius)
	if disc < 0 {
		return false
	}
	root := math.Sqrt(disc)
	return closest+root >= 0 && closest-root <= far
}

// reachMissed is volume_missed in trace.metal, in float64.
func reachMissed(at, way, low, high [3]float64, far float64) bool {
	enter, leave := 0.0, far
	for part := range at {
		if way[part] == 0 {
			if at[part] < low[part] || at[part] > high[part] {
				return true
			}
			continue
		}
		one, other := (low[part]-at[part])/way[part],
			(high[part]-at[part])/way[part]
		enter, leave = max(enter, min(one, other)), min(leave, max(one, other))
	}
	return enter > leave
}

const (
	// reachRays is how many rays are sent at a puff's ball.
	reachRays = 20000
	// reachSpread is how far from the ball's centre, on each axis, a ray
	// may start: twice the ball's radius of 3, so rays start inside the
	// ball, beside it and clear of its box.
	reachSpread = 6
	// reachMet is the fewest rays that must meet the ball for the test to
	// show anything. A ray that starts inside the ball always meets it,
	// and the ball is a share of (4/3)*pi*27 over the 12 by 12 by 12 start
	// cube, about 6.5%: 1300 of 20000, before any ray that starts outside
	// and reaches it.
	reachMet = 1000
)

// reachLost sends reachRays rays from around a puff's ball, in a fixed
// order, and says how many met the ball and whether any of those was missed
// by the box low..high.
func reachLost(low, high [3]float64) (met int, lost bool) {
	centre := [3]float64{reachAt.X, reachAt.Y, reachAt.Z}
	ball := reachScale * reachRadius
	seed := uint64(7)
	next := func() float64 {
		seed = seed*6364136223846793005 + 1442695040888963407
		return float64(seed>>11)/(1<<53)*2 - 1
	}
	for range reachRays {
		at := [3]float64{centre[0] + reachSpread*next(),
			centre[1] + reachSpread*next(), centre[2] + reachSpread*next()}
		way := [3]float64{next(), next(), next()}
		length := math.Sqrt(way[0]*way[0] + way[1]*way[1] + way[2]*way[2])
		for part := range way {
			way[part] /= length
		}
		far := 8 * (next() + 1)
		if !reachHit(at, way, centre, ball, far) {
			continue
		}
		met++
		if reachMissed(at, way, low, high, far) {
			lost = true
		}
	}
	return met, lost
}

// no ray that meets a puff's ball is missed by its box: the box rejects
// only what the march would have given nothing.
func TestNoRayThatMeetsAPuffIsMissedByItsBox(t *testing.T) {
	row := puffRow(turnedSpace())
	low, high, _ := reachBox(row)
	met, lost := reachLost(low, high)
	if lost {
		t.Error("a ray meets the ball and misses the box")
	}
	if met < reachMet {
		t.Errorf("only %d of the rays met the ball: the test shows little",
			met)
	}
}

// Anti-test: the rays above do find a box that is too small: one that
// holds the middle of the ball and not the ball is missed by rays that
// meet the ball.
func TestARayThatMeetsTheBallIsMissedByABoxInsideIt(t *testing.T) {
	centre := [3]float64{reachAt.X, reachAt.Y, reachAt.Z}
	ball := reachScale * reachRadius
	var low, high [3]float64
	for part := range centre {
		low[part], high[part] = centre[part]-ball/3, centre[part]+ball/3
	}
	if _, lost := reachLost(low, high); !lost {
		t.Error("a box inside the ball lost no ray that meets the ball")
	}
}

// Anti-test: the box does reject, or it is no help: a ray well off to
// one side misses it.
func TestTheBoxMissesARayThatPassesFarOff(t *testing.T) {
	row := puffRow(turnedSpace())
	low, high, _ := reachBox(row)
	at := [3]float64{reachAt.X + 40, reachAt.Y, reachAt.Z}
	if !reachMissed(at, [3]float64{0, 1, 0}, low, high, 100) {
		t.Error("a ray 40 metres to the side meets the box")
	}
	if !reachMissed([3]float64{reachAt.X - 20, reachAt.Y, reachAt.Z},
		[3]float64{1, 0, 0}, low, high, 5) {
		t.Error("a ray that stops short meets the box")
	}
}

// unboxedRow is the row of one puff placed in space as the encoder writes
// it before boxVolumes has run: with no box in the world, nought in both
// corners.
func unboxedRow(space maths.Subspace) []byte {
	encoded := newScene()
	encoded.volume(space, haze{puff: medium.Puff{Radius: reachRadius}}, 0)
	return append([]byte(nil), encoded.volumes.bytes[:volumeStruct.size()]...)
}

// patched is a puff's row, not yet boxed, with fields written over it.
func patched(given values) []byte {
	row := unboxedRow(maths.Subspace{Origin: reachAt, Scale: 1,
		Orientation: maths.Identity})
	volumeStruct.patch(row, given)
	return row
}

// Anti-test: reachBox tells a row with a box from one with none, which
// the test of a volume with no box below reads: a puff's row has none
// until boxVolumes has run, and one after.
func TestAPuffRowIsBoxedOnlyOnceBoxVolumesHasRun(t *testing.T) {
	space := turnedSpace()
	if _, _, walled := reachBox(unboxedRow(space)); walled {
		t.Error("a row the encoder has not boxed has a box")
	}
	if _, _, walled := reachBox(puffRow(space)); !walled {
		t.Error("a puff's row has no box after boxVolumes")
	}
}

func TestEachShapeSpanClipsToHasItsBox(t *testing.T) {
	cases := map[string]struct {
		given     values
		low, high [3]float64
	}{
		"a cylinder": {values{"info": {2, 0, boxedKind, 0},
			"puff": {0, 2, 6}}, [3]float64{-2, -3, -2},
			[3]float64{2, 3, 2}},
		"cells": {values{"info": {0, 0, boxedKind, 0},
			"low": {-1, -2, -3}, "high": {4, 5, 6}},
			[3]float64{-1, -2, -3}, [3]float64{4, 5, 6}},
		"a ring": {values{"info": {0, 0, ringKind, 0},
			"low": {-1, -2, -3}, "high": {4, 5, 6}},
			[3]float64{-1, -2, -3}, [3]float64{4, 5, 6}},
		"a ball in a boxed medium": {values{"info": {1, 0, boxedKind, 0},
			"puff": {2}, "low": {-9, -9, -9}, "high": {9, 9, 9}},
			[3]float64{-2, -2, -2}, [3]float64{2, 2, 2}},
		"a wake": {values{"info": {0, 0, wakeKind, 0}, "puff": {5}},
			[3]float64{-5, -5, -5}, [3]float64{5, 5, 5}},
	}
	centre := [3]float64{reachAt.X, reachAt.Y, reachAt.Z}
	for name, one := range cases {
		low, high, walled := volumeReach(patched(one.given))
		if !walled {
			t.Errorf("%s has no box", name)
			continue
		}
		for part := range low {
			if low[part] > centre[part]+one.low[part] ||
				high[part] < centre[part]+one.high[part] {
				t.Errorf("%s: axis %d box %v..%v is inside %v..%v", name,
					part, low[part], high[part],
					centre[part]+one.low[part], centre[part]+one.high[part])
			}
			if low[part] < centre[part]+one.low[part]-1 ||
				high[part] > centre[part]+one.high[part]+1 {
				t.Errorf("%s: axis %d box %v..%v is loose", name, part,
					low[part], high[part])
			}
		}
	}
}

// Anti-test: a volume the kernel cannot be told a box of keeps none, and
// its row says so, so every ray is let into it as before.
func TestAVolumeWithNoSoundBoxKeepsNone(t *testing.T) {
	for name, given := range map[string]values{
		"axes that are not a turn": {"x": {2, 0, 0}},
		"axes that lean":           {"y": {0.6, 0.8, 0}},
		"a scale below nought":     {"origin": {30, -4, 12, -1}},
		"a radius that is not one": {"puff": {math.NaN()}},
		"a radius without end":     {"puff": {math.Inf(1)}},
	} {
		if _, _, walled := volumeReach(patched(given)); walled {
			t.Errorf("%s has a box", name)
		}
	}
	row := patched(values{"x": {2, 0, 0}})
	encoded := newScene()
	encoded.volumes.bytes = row
	encoded.boxVolumes()
	low, high, walled := reachBox(encoded.volumes.bytes)
	if walled {
		t.Error("a volume with no box is marked as having one")
	}
	if low != [3]float64{} || high != [3]float64{} {
		t.Errorf("a volume with no box has corners %v..%v", low, high)
	}
}

func TestAWorldWithNoVolumesHasNoBoxes(t *testing.T) {
	encoded := newScene()
	encoded.boxVolumes()
	if len(encoded.volumes.bytes) != 0 {
		t.Errorf("%d bytes of volumes from none", len(encoded.volumes.bytes))
	}
}
