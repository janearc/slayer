package weathering

import (
	"hash/fnv"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the numbers a rule is drawn with
const (
	// patchOctaves is how many scales of noise a patch has. picked: a
	// patch, and the ragged edge of a patch.
	patchOctaves = 3
	// patchSoft is how far, in the noise's units, a patch's edge fades.
	// picked: soft enough not to show as a pixel's stair.
	patchSoft = 0.15
	// patchRange is about how far either side of nought three octaves of
	// gradient noise mostly stay. picked: as scene/texture's dust.
	patchRange = 0.6
	// saltStep keeps two agents' fields far apart in the noise's seeds,
	// which share fields a small step apart.
	saltStep = 1 << 32
	// frontRagged is how far a nanomachine front's reach may differ from
	// its mean, as a share of it, by noise. picked: a front that wanders.
	frontRagged = 0.4
	// frontEdge is how much of a front's reach, from its outer end, is
	// its soft edge. picked: a front with a visible edge, not a ring.
	frontEdge = 0.2
	// splatFill is how far from a cell's middle a splat or a tag reaches,
	// as a share of the cell. picked: marks that do not touch one another.
	splatFill = 0.45
	// tagSquash is how many times wider than tall a tag is. picked: a
	// tag is written along a wall, not down it.
	tagSquash = 2.2
	// tagLeast is the share of the largest tag that the smallest is.
	// picked: tags that differ, and none that is a speck.
	tagLeast = 0.5
	// cellMiddle is where in a cell, as a share of it, its middle is.
	cellMiddle = 0.5
	// tiny keeps a width from being nought where it is divided by.
	tiny = 1e-9
)

// rule is where an agent may land: nought where its rule says it cannot,
// up to one where it lands most. It knows nothing of age or strength.
type rule func(layer Layer, agent Agent, site *Site) float64

// rules is each kind's rule.
var rules = map[Kind]rule{
	Rain: Layer.rain, Soot: Layer.soot, Dust: Layer.dust, Moss: Layer.moss,
	Ivy: Layer.ivy, Graffiti: Layer.graffiti, Rust: Layer.rust,
	Spall: Layer.spall, Vacancy: Layer.none, Mineral: Layer.mineral,
	Glitter: Layer.glitter, Nano: Layer.nano, Festival: Layer.festival,
	Spout: Layer.spout,
	Wet:   Layer.wet,
}

// Amount is how much of the site the agent covers, nought to one. It is
// nought wherever the agent's rule says it cannot land, and otherwise
// rises with the layer's age and never falls with it, except for a
// festival's splatter, which only fades.
func (layer Layer) Amount(agent Agent, site *Site) float64 {
	each, known := rules[agent.Kind]
	if !known || !named(site.Name, layer.landing(agent)) {
		return 0
	}
	where := each(layer, agent, site)
	if !(where > 0) { // defined: a rule that says no, or a number it cannot
		return 0 // defined: none
	}
	return math.Min(1, where*agent.Strength*layer.pace(agent))
}

// landing is the words a surface's material must hold for an agent to
// land on it. Rust and vacancy name their own materials, and so land on
// any surface and let their rules ask.
func (layer Layer) landing(agent Agent) []string {
	if agent.Kind == Rust || agent.Kind == Vacancy {
		return nil
	}
	return agent.On
}

// pace is how much of its strength an agent has by the layer's age: what
// it has laid, or for a festival what has not faded, or for a front, which
// moves by its own rate, all of it.
func (layer Layer) pace(agent Agent) float64 {
	laid := grown(layer.Age, agent.Settle)
	switch agent.Kind {
	case Festival:
		return 1 - laid
	case Nano:
		return 1
	}
	return laid
}

// none is the rule of an agent that lays no colour of its own.
func (layer Layer) none(Agent, *Site) float64 { return 0 }

// salt is a number that tells one agent's noise from another's.
func salt(agent Agent) uint64 {
	hashed := fnv.New32a()
	_, _ = hashed.Write([]byte(agent.Name))
	return uint64(hashed.Sum32()) * saltStep
}

// ramp is nought below from, one above to, and smooth between.
func ramp(value, from, to float64) float64 {
	if !(to > from) { // defined: no width is a step
		if value >= to {
			return 1
		}
		return 0
	}
	through := math.Min(1, math.Max(0, (value-from)/(to-from)))
	return through * through * (3 - 2*through)
}

// patch is how much of a patch of a width grain lies at a point, where
// share of the surface is in patch: by smooth noise in the world's space.
// No share is none; no width is an even film of the share.
func (layer Layer) patch(agent Agent, point maths.Vec, grain,
	share float64) float64 {
	if !(share > 0) { // defined: none asked
		return 0
	}
	if !(grain > 0) { // defined: no width is an even film
		return math.Min(1, share)
	}
	at := point.Scale(1 / grain)
	wave := noise.Fractal(layer.Seed+salt(agent), at.X, at.Y, at.Z,
		patchOctaves)
	level := patchRange * (1 - 2*share)
	return ramp(wave, level, level+patchSoft)
}

// height is how far above the ground a site is, in metres.
func (layer Layer) height(site *Site) float64 {
	return site.Point.Dot(layer.Up) - layer.Ground
}

// upward is how much a face points up, from nought where it points
// across or down to one where it points straight up, once it points up
// at least as far as tilt.
func (layer Layer) upward(site *Site, tilt float64) float64 {
	along := site.Normal.Dot(layer.Up)
	if along < tilt || !(along > 0) {
		return 0
	}
	return along
}

// wall is whether a face is a wall: it points up or down by no more
// than level.
func (layer Layer) wall(site *Site, level float64) bool {
	return math.Abs(site.Normal.Dot(layer.Up)) <= level
}

// overhead is what stands over a site within reach metres, looking up
// from just off its surface: how far above, the material, and whether
// anything does.
func (layer Layer) overhead(site *Site, reach float64) (float64, string,
	bool) {
	from := site.Point.Add(site.Normal.Scale(layer.Probe))
	return layer.Over.Above(from, reach)
}

// sheltered is whether something stands over a site within the agent's
// shelter.
func (layer Layer) sheltered(agent Agent, site *Site) bool {
	_, _, covered := layer.overhead(site, agent.Shelter)
	return covered
}

// lodged is how much more of an agent stays in a mortar joint than on a
// brick's face: one, or one and the agent's catch.
func lodged(agent Agent, site *Site) float64 {
	if site.Joint {
		return 1 + agent.Catch
	}
	return 1
}

// column is how much of a wall a streak holds at a site: the wall is
// cut into columns agent.Spacing wide along it, a share of them hold a
// streak, and a streak is thickest at the middle of its column.
func (layer Layer) column(agent Agent, site *Site, share float64) float64 {
	if !(agent.Spacing > 0) { // defined: no columns
		return 0
	}
	along := site.Point.Dot(layer.Up.Cross(site.Normal).Unit()) / agent.Spacing
	column := math.Floor(along)
	if noise.Hash(layer.Seed+salt(agent), int64(column)) >= share {
		return 0
	}
	bump := math.Sin(math.Pi * (along - column))
	return bump * bump
}

// rain is streaks on walls below what feeds them: a sill, a ledge, an
// eave, anything standing over the wall within a streak's length. A
// streak fades as it runs down.
func (layer Layer) rain(agent Agent, site *Site) float64 {
	if !layer.wall(site, agent.Level) || !(agent.Length > 0) {
		return 0
	}
	rise, _, fed := layer.overhead(site, agent.Length)
	if !fed {
		return 0
	}
	return (1 - rise/agent.Length) * layer.column(agent, site, agent.Share)
}

// soot is dark on what is low and what is sheltered: thick under an
// eave, and only a share of that where rain reaches, falling off with
// height over a reach.
func (layer Layer) soot(agent Agent, site *Site) float64 {
	risen := math.Max(layer.height(site), 0)
	low := math.Exp(-risen / agent.Reach)
	if !(agent.Reach > 0) { // defined: no reach, no height to fall off over
		low = 1
	}
	kept := agent.Share
	if layer.sheltered(agent, site) {
		kept = 1
	}
	return low * kept * lodged(agent, site) *
		layer.patch(agent, site.Point, agent.Grain, 1-agent.Share)
}

// dust settles on faces that point up and that nothing stands over.
func (layer Layer) dust(agent Agent, site *Site) float64 {
	up := layer.upward(site, agent.Tilt)
	if up == 0 || layer.sheltered(agent, site) {
		return 0
	}
	return up * lodged(agent, site) *
		layer.patch(agent, site.Point, agent.Grain, agent.Share)
}

// moss grows where it is shaded and damp: on a face turned from the sun
// or sheltered from above, low down, and more on faces toward north.
func (layer Layer) moss(agent Agent, site *Site) float64 {
	shaded := site.Normal.Dot(layer.Sun) <= agent.Shade ||
		layer.sheltered(agent, site)
	damp := 1 - layer.height(site)/agent.Reach
	if !shaded || !(agent.Reach > 0) || !(damp > 0) {
		return 0
	}
	north := 1 - agent.North +
		agent.North*(1+site.Normal.Dot(layer.North))/2
	return damp * north * lodged(agent, site) *
		layer.patch(agent, site.Point, agent.Grain, agent.Share)
}

// ivy climbs a wall from the ground: from roots every agent.Spacing along
// it, a share of them, each to a height of its own up to the climb,
// with a ragged top, and holds to the joints.
func (layer Layer) ivy(agent Agent, site *Site) float64 {
	if !layer.wall(site, agent.Level) || !(agent.Spacing > 0) {
		return 0
	}
	sideways := layer.Up.Cross(site.Normal).Unit()
	root := math.Floor(site.Point.Dot(sideways) / agent.Spacing)
	if noise.Hash(layer.Seed+salt(agent), int64(root)) >= agent.Share {
		return 0
	}
	top := agent.Climb * grown(layer.Age, agent.Settle) *
		(1 - frontRagged*noise.Hash(layer.Seed+salt(agent)+1, int64(root)))
	reach := ramp(top-layer.height(site), 0, top*frontEdge)
	return reach * lodged(agent, site)
}

// rust is on steel, in patches, and runs below it: where the surface is
// not steel but steel stands over it within a run, a stain falls from it
// in streaks.
func (layer Layer) rust(agent Agent, site *Site) float64 {
	if named(site.Name, agent.On) && len(agent.On) > 0 {
		return layer.patch(agent, site.Point, agent.Grain, agent.Share)
	}
	if !layer.wall(site, agent.Level) || !(agent.Length > 0) {
		return 0
	}
	rise, above, fed := layer.overhead(site, agent.Length)
	if !fed || !named(above, agent.On) {
		return 0
	}
	return (1 - rise/agent.Length) * layer.column(agent, site, agent.Share)
}

// spall is cracks and flaked brick, on brick: in areas a share of the
// wall, hair cracks where noise crosses nought and patches that have
// flaked to the bare face. The crack's colour is Colour and the bare
// face Other; it is told apart in paint.
func (layer Layer) spall(agent Agent, site *Site) float64 {
	return math.Max(layer.crack(agent, site), layer.flaked(agent, site)) *
		lodged(agent, site)
}

// crack is how much of a hair crack lies at a site: a ridge of noise a
// crack's width across, where the area is one that cracks.
func (layer Layer) crack(agent Agent, site *Site) float64 {
	if !(agent.Grain > 0) || !(agent.Crack > 0) {
		return 0
	}
	at := site.Point.Scale(1 / agent.Grain)
	seed := layer.Seed + salt(agent) + 2
	wave := math.Abs(noise.Fractal(seed, at.X, at.Y, at.Z, patchOctaves))
	line := ramp(agent.Crack/agent.Grain-wave, 0, agent.Crack/agent.Grain)
	area := layer.patch(Agent{Name: agent.Name + " area"}, site.Point,
		agent.Grain*patchWide, agent.Share)
	return line * area
}

// patchWide is how many times wider an area that cracks is than a crack's
// own grain. picked: areas, and not every wall, crack.
const patchWide = 4

// flaked is how much flaked-off brick lies at a site: patches of the
// brick that spalled.
func (layer Layer) flaked(agent Agent, site *Site) float64 {
	return layer.patch(Agent{Name: agent.Name + " flake"}, site.Point,
		agent.Grain, agent.Spalled)
}

// plume is how much of a source's strength reaches a point: thinning
// with distance over agent.Reach, and by a lobe that opens downwind, as
// wide as Spread of the distance along the wind, with Upwind of it
// reaching the other way. With no wind it is a circle.
func (layer Layer) plume(agent Agent, point maths.Vec) float64 {
	if !(agent.Reach > 0) { // defined: no reach, no plume
		return 0
	}
	flat := func(vec maths.Vec) maths.Vec {
		alongUp := vec.Dot(layer.Up)
		return vec.Sub(layer.Up.Scale(alongUp))
	}
	away := flat(point.Sub(agent.At))
	thin := math.Exp(-away.Len() / agent.Reach)
	wind := flat(layer.Wind).Unit()
	along := away.Dot(wind)
	if wind == (maths.Vec{}) { // defined: no wind
		return thin
	}
	if along <= 0 {
		return agent.Upwind * thin
	}
	width := agent.Spread * math.Max(along, agent.Grain)
	if !(width > 0) { // defined: no spread is no lobe
		return thin
	}
	across := away.Sub(wind.Scale(along)).Len()
	return thin * math.Exp(-(across/width)*(across/width))
}

// mineral is dust from a point: it settles as dust does, on faces that
// point up with nothing over them, in the strength the plume brings.
func (layer Layer) mineral(agent Agent, site *Site) float64 {
	up := layer.upward(site, agent.Tilt)
	if up == 0 || layer.sheltered(agent, site) {
		return 0
	}
	return up * layer.plume(agent, site.Point) * lodged(agent, site)
}

// glitter is flecks, settled as dust does and carried by a plume: a cell
// agent.Fleck wide holds a fleck if the plume's strength there beats a
// number drawn for the cell, so the flecks thin out as the plume does.
func (layer Layer) glitter(agent Agent, site *Site) float64 {
	up := layer.upward(site, agent.Tilt)
	if up == 0 || !(agent.Fleck > 0) || layer.sheltered(agent, site) {
		return 0
	}
	at := site.Point.Scale(1 / agent.Fleck)
	cellX := int64(math.Floor(at.X))
	cellY := int64(math.Floor(at.Y))
	cellZ := int64(math.Floor(at.Z))
	drawn := noise.Hash3(layer.Seed+salt(agent), cellX, cellY, cellZ)
	if drawn >= layer.plume(agent, site.Point)*agent.Share {
		return 0
	}
	return up
}

// nano is contamination spreading along a surface from a seed: a front
// that has moved Rate a year for the layer's age, ragged by noise, slowed
// round a bend by how far the surface turns from the seed's, and soft at
// its edge. The distance is by the straight line, made longer by the turn.
func (layer Layer) nano(agent Agent, site *Site) float64 {
	front := agent.Rate * layer.Age
	if !(front > 0) {
		return 0
	}
	bend := 1 - site.Normal.Dot(agent.Facing.Unit())
	away := site.Point.Sub(agent.At).Len() * (1 + agent.Turn*bend)
	at := site.Point.Scale(1 / math.Max(agent.Grain, tiny))
	reach := front * (1 + frontRagged*noise.Fractal(layer.Seed+salt(agent),
		at.X, at.Y, at.Z, patchOctaves))
	return 1 - ramp(away, reach*(1-frontEdge), reach)
}

// festival is splatter round a plaza: in cells agent.Grain wide on the
// ground and on walls up to High, a share of them marked, fewer the
// farther from the plaza's middle, none past its reach.
func (layer Layer) festival(agent Agent, site *Site) float64 {
	flat := site.Point.Sub(agent.At)
	alongUp := flat.Dot(layer.Up)
	flat = flat.Sub(layer.Up.Scale(alongUp))
	fade := 1 - flat.Len()/agent.Reach
	standing := layer.upward(site, agent.Tilt) > 0 ||
		layer.wall(site, agent.Level) && layer.height(site) <= agent.High
	if !(agent.Reach > 0) || !(agent.Grain > 0) || !(fade > 0) || !standing {
		return 0
	}
	at := site.Point.Scale(1 / agent.Grain)
	cell := [3]int64{int64(math.Floor(at.X)), int64(math.Floor(at.Y)),
		int64(math.Floor(at.Z))}
	if noise.Hash3(layer.Seed+salt(agent), cell[0], cell[1], cell[2]) >=
		agent.Share*fade*fade {
		return 0
	}
	offset := math.Hypot(at.X-float64(cell[0])-cellMiddle,
		at.Z-float64(cell[2])-cellMiddle)
	return ramp(splatFill-offset, 0, splatFill*frontEdge)
}
