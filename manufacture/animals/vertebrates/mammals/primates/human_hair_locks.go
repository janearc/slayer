package primates

import (
	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// mediumLock gives neighbouring followers a common shape through the
// hanging length. The chart footprints supply a finite width at the
// ends, and the resulting cluster still has individual shafts.
type mediumLock struct {
	targets            mediumTargets
	root, rootAcross   maths.Vec
	across             maths.Vec
	release, bend, tip maths.Vec // Displacements from the broad guide.
	width, lift        float64
}

func (spec ScalpHair) mediumLock(anchor hair.GuideAnchor, shape mediumReach, index int) mediumLock {
	targets := spec.mediumTargets(anchor.Point, anchor.Normal, shape)
	outward := maths.V(shape.side*(1-shape.behind), 0, -shape.behind).Unit()
	across := outward.Cross(maths.V(0, -1, 0)).Unit()
	variety := func(axis int64) float64 {
		return 2*noise.Hash3(spec.Seed^0x6c6f636b73686170, int64(index), axis, 0) - 1
	}
	nape := shape.behind * (1 - shape.crown)
	scale := mix(mix(spec.SideLength/.171, spec.NapeLength/.110, nape), spec.Length/.234, shape.crown)
	scale *= 1 - .55*shape.trim
	turn, body := variety(1), variety(2)
	fall := scale * (.004 + .021*variety(3))
	tuck := .002 + .006*body
	bow := (.004 + .010*spec.Curl) * (.74 + .26*body)
	return mediumLock{
		targets: targets, root: anchor.Point, rootAcross: anchor.Normal.Cross(targets.heading).Unit(),
		across: across, width: .23 + .075*variety(4),
		release: outward.Scale(.004 * body).Add(across.Scale(.004 * turn)),
		bend:    outward.Scale(bow).Add(across.Scale(-.010 * spec.Curl * turn)),
		tip:     outward.Scale(tuck).Add(across.Scale(.009 * turn)).Add(maths.V(0, -fall, 0)),
		lift:    .0019 * body,
	}
}

func (lock mediumLock) shape(root maths.Vec, guide mediumTargets, weight float64) mediumTargets {
	if weight == 0 {
		return guide
	}
	// Transport the offset across the root footprint into the lock's
	// sweep, retaining separation as the followers narrow toward the tip.
	spread := max(-.028, min(.028, root.Sub(lock.root).Dot(lock.rootAcross))) * lock.width
	release := lock.targets.release.Add(lock.release).Add(lock.across.Scale(1.4 * spread))
	tip := lock.targets.tip.Add(lock.tip).Add(lock.across.Scale(spread))
	guide.release = mixPoint(guide.release, release, .36*weight)
	guide.tip = mixPoint(guide.tip, tip, .84*weight)
	guide.tip.Y = min(guide.tip.Y, guide.release.Y-.014)
	guide.bend = guide.bend.Add(lock.bend.Scale(weight))
	guide.lift += lock.lift * weight
	return guide
}
