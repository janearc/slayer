package weathering

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// SpoutMaterial is the name of the material of a thing that stands where
// a building lets water go: the lip of a gargoyle's mouth. Apply takes
// the place of each thing of this name, as a spout, for the agent Spout
// to run a streak from. The name is held in the manufactory by the same
// words, and a test there says they are the same.
const SpoutMaterial = "gargoyle spout"

// the numbers the rule of a spout is drawn with
const (
	// streakCore is how much of a streak's width, from its middle, is as
	// dark as it gets before the edge softens. picked: a streak with a
	// body and a soft edge, not a hair.
	streakCore = 0.35
	// streakBody is how much of a streak is its steady dark, the rest
	// being the patchy stain that runs through it. picked: mostly steady.
	streakBody = 0.55
)

// spout is a dark streak on a wall below each spout of the layer: it runs
// straight down from the spout, a width of Spacing at the top and wider
// as it falls by Spread metres a metre, over a Length, fading as it goes,
// and wavers with the patchy stain in it. A surface that faces away from
// the spout has none. The wind leans a streak by how it blows across.
func (layer Layer) spout(agent Agent, site *Site) float64 {
	if !layer.wall(site, agent.Level) || !(agent.Length > 0) {
		return 0
	}
	var most float64
	for _, source := range layer.Spouts {
		most = math.Max(most, layer.runFrom(agent, site, source))
	}
	return most
}

// runFrom is how much of a streak from one source is at a site: nought
// off its length, to the side of it or on a surface turned from it.
func (layer Layer) runFrom(agent Agent, site *Site, source maths.Vec) float64 {
	fall := source.Sub(site.Point).Dot(layer.Up)
	if !(fall > 0) || fall > agent.Length ||
		!(source.Sub(site.Point).Dot(site.Normal) > 0) {
		return 0
	}
	windUp := layer.Wind.Dot(layer.Up)
	lean := layer.Wind.Sub(layer.Up.Scale(windUp))
	lay := source.Add(lean.Scale(agent.Upwind * fall))
	aside := site.Point.Sub(lay)
	asideUp := aside.Dot(layer.Up)
	aside = aside.Sub(layer.Up.Scale(asideUp))
	// a spout stands out from the wall, and what it lets go runs down
	// the wall below its place on the wall: how far out it stands is not
	// how far to the side the streak is.
	asideOut := aside.Dot(site.Normal)
	aside = aside.Sub(site.Normal.Scale(asideOut))
	half := math.Max((agent.Spacing/2)+agent.Spread*fall, tiny)
	across := ramp(1-aside.Len()/half, 0, 1-streakCore)
	stain := streakBody + (1-streakBody)*
		layer.patch(agent, site.Point, agent.Grain, agent.Share)
	return across * (1 - fall/agent.Length) * stain
}
