package weathering

import "math"

// wet is the damp rain and a wet street leave: on a wall, low down, where
// the street splashes it, thinning with height over a reach; and on a face
// that points up and that nothing stands over, a sill or a ledge or a roof,
// where rain lands and stays.
//
// Its colour is the dark of damp stone. A wall beyond its reach of the
// ground is dry, and so is anything sheltered from the sky. Wet is the
// colour of the damp only: how a wet surface mirrors is its material's
// gloss, and no agent lays that.
//
// Wet needs a reach, which is how high the splash goes; without one a wall
// is dry. Share is how much of the splash zone is damp, and Grain how wide
// its patches are, as for soot.
func (layer Layer) wet(agent Agent, site *Site) float64 {
	if layer.wall(site, agent.Level) {
		return layer.splash(agent, site)
	}
	up := layer.upward(site, agent.Tilt)
	if up == 0 || layer.sheltered(agent, site) {
		return 0
	}
	return up * lodged(agent, site) *
		layer.patch(agent, site.Point, agent.Grain, agent.Share)
}

// splash is how damp a wall is at a site from the street's spray: all of
// it at the ground, falling off with height over the agent's reach.
func (layer Layer) splash(agent Agent, site *Site) float64 {
	if !(agent.Reach > 0) { // defined: no reach, no splash
		return 0
	}
	risen := math.Max(layer.height(site), 0)
	low := math.Exp(-risen / agent.Reach)
	return low * lodged(agent, site) *
		layer.patch(agent, site.Point, agent.Grain, agent.Share)
}
