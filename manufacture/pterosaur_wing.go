package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The wing's skin is two panels per side. The main panel (the
// brachiopatagium) hangs from the whole wing chain: arm, forearm,
// metacarpal and wing finger. Its far side is the body flank at the
// shoulder, the hind limb's ankle, and the free trailing edge from the
// ankle to the tip. The forward panel (the propatagium) runs from the
// arm and forearm to the neck's base, held out by the pteroid bone.
// Palmer and Dyke 2010 (PMC2842769) describe this arrangement for
// pterosaurs; where Hatz's skin attached is not known, so the ankle and
// the neck are authored attachments.

// The sample counts and offsets of the panels.
const (
	// mainChordSteps and forwardChordSteps are how many pieces each
	// chord of a panel is drawn in. Authored.
	mainChordSteps    = 48
	forwardChordSteps = 16
	// tipBehind is how far behind the wing finger's tip the trailing
	// edge ends, in metres, so the last chord has length. Authored.
	tipBehind = .05
)

// The number of samples taken along each bone of the leading edge: the
// four bones of the wing chain for the main panel, the first two for the
// forward panel. They are in proportion to the bones' lengths.
var (
	mainSpanCounts    = []int{16, 20, 22, 28}
	forwardSpanCounts = []int{18, 22}
)

// jointArcs are the arc lengths along a chain from its first joint to
// each of the joints after the first count of them, up to but not
// including the chain's end.
func jointArcs(count int) []float64 {
	arcs := make([]float64, 0, count)
	run := 0.0
	for i := 0; i < count; i++ {
		run += hatzSourceBones[i]
		arcs = append(arcs, run)
	}
	return arcs
}

// pleatAway is the direction the skin's pleats fall toward, for a side:
// up and outward, so they stand off the body and not into it.
func pleatAway(side float64) maths.Vec {
	const upShare = .6
	return maths.V(0, upShare, side)
}

// chainSamples is the points along a chain of joints, counts[i] pieces
// along bone i, and each point's share of the chain's length.
func chainSamples(joints []maths.Vec, counts []int) ([]maths.Vec, []float64) {
	var points []maths.Vec
	total := 0.0
	for i := range counts {
		total += joints[i+1].Sub(joints[i]).Len()
	}
	var fractions []float64
	run := 0.0
	for i, count := range counts {
		bone := joints[i+1].Sub(joints[i])
		for k := 0; k < count; k++ {
			share := float64(k) / float64(count)
			points = append(points,
				joints[i].Add(bone.Scale(share)))
			fractions = append(fractions,
				(run+share*bone.Len())/total)
		}
		run += bone.Len()
	}
	points = append(points, joints[len(counts)])
	return points, append(fractions, 1)
}

// trailingAnchor is where the free trailing edge ends, just behind the
// wing finger's tip.
func (pose pterosaurPose) trailingAnchor(sideIndex int) maths.Vec {
	return pose.arm[sideIndex][4].Add(maths.V(-tipBehind, 0, 0))
}

// mainEdgeRest is the rest length of the free trailing edge: the glide
// distance from ankle to tip plus the edge's slack.
func mainEdgeRest(sideIndex int) float64 {
	glide := pterosaurFlight(0, 0)
	ankle := glide.leg[sideIndex][2]
	reach := glide.trailingAnchor(sideIndex).Sub(ankle).Len()
	return reach * (1 + pterosaurEdgeSlack)
}

// mainBoundaries are the main panel's leading boundary samples, and the
// far boundary: flank, ankle and trailing edge, at the same shares.
func (pose pterosaurPose) mainBoundaries(sideIndex int) ([]maths.Vec,
	[]maths.Vec) {
	lead, fractions := chainSamples(pose.arm[sideIndex][:], mainSpanCounts)
	ankle, anchor := pose.leg[sideIndex][2], pose.trailingAnchor(sideIndex)
	away := pleatAway(pterosaurSides[sideIndex])
	middle := lead[len(lead)/2].Sub(ankle.Add(anchor).Scale(.5))
	normal := planeNormal(anchor.Sub(ankle), middle, away)
	path := slackPolyline(ankle, anchor, normal, mainEdgeRest(sideIndex))
	return lead, atFractions(path, fractions)
}

// mainRest is the rest length of every chord of the main panel: its
// length in the glide pose plus the chord slack.
func mainRest(sideIndex int) []float64 {
	lead, far := pterosaurFlight(0, 0).mainBoundaries(sideIndex)
	rest := make([]float64, len(lead))
	for i := range rest {
		rest[i] = far[i].Sub(lead[i]).Len() * (1 + pterosaurChordSlack)
	}
	return rest
}

// mainPanel is the main membrane of a side in this pose.
func (pose pterosaurPose) mainPanel(sideIndex int) membraneSpec {
	lead, far := pose.mainBoundaries(sideIndex)
	return membraneSpec{lead: lead, far: far, rest: mainRest(sideIndex),
		away:  pleatAway(pterosaurSides[sideIndex]),
		steps: mainChordSteps, joints: jointArcs(3)}
}

// forwardBoundaries are the forward panel's leading boundary, which is the
// arm and forearm, and its far boundary, which runs from the neck's base
// to the pteroid's tip.
func (pose pterosaurPose) forwardBoundaries(sideIndex int) ([]maths.Vec,
	[]maths.Vec) {
	lead, fractions := chainSamples(pose.arm[sideIndex][:3],
		forwardSpanCounts)
	anchor, tip := pose.forwardAnchor(sideIndex), pose.pteroidTip(sideIndex)
	away := pleatAway(pterosaurSides[sideIndex])
	middle := lead[len(lead)/2].Sub(anchor.Add(tip).Scale(.5))
	normal := planeNormal(tip.Sub(anchor), middle, away)
	glide := pterosaurFlight(0, 0)
	rest := glide.pteroidTip(sideIndex).Sub(glide.forwardAnchor(sideIndex)).
		Len() * (1 + pterosaurEdgeSlack)
	path := slackPolyline(anchor, tip, normal, rest)
	return lead, atFractions(path, fractions)
}

// forwardRest is the rest length of every chord of the forward panel.
func forwardRest(sideIndex int) []float64 {
	lead, far := pterosaurFlight(0, 0).forwardBoundaries(sideIndex)
	rest := make([]float64, len(lead))
	for i := range rest {
		rest[i] = far[i].Sub(lead[i]).Len() * (1 + pterosaurChordSlack)
	}
	return rest
}

// forwardPanel is the forward membrane of a side in this pose.
func (pose pterosaurPose) forwardPanel(sideIndex int) membraneSpec {
	lead, far := pose.forwardBoundaries(sideIndex)
	return membraneSpec{lead: lead, far: far, rest: forwardRest(sideIndex),
		away:  pleatAway(pterosaurSides[sideIndex]),
		steps: forwardChordSteps, joints: jointArcs(1)}
}

// stretchOf is the largest ratio, over a panel's chords, of the distance
// between a chord's ends to its rest length: above 1 the skin is pulled
// tight past its length and cannot reach without stretching.
func (spec membraneSpec) stretchOf() float64 {
	worst := 0.0
	for i := range spec.lead {
		reach := spec.far[i].Sub(spec.lead[i]).Len()
		worst = math.Max(worst, reach/spec.rest[i])
	}
	return worst
}
