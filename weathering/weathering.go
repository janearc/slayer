// Package weathering lays what happens to a place over every surface in
// a set's world, by rules, and not on any one building.
//
// An agent is a cause: rain running from a sill, soot, dust settling,
// moss, ivy, a hand with a spray can, rust, a mine upwind, a factory's
// glitter, nanomachines spreading from a seed, a festival's splatter, or
// a dwelling left empty.
//
// Each has a rule for where it can land, written as a function of a site:
// a point in the world, which way the surface there faces, the material it
// is, and what stands above it. Each has an age, which only ever lays more
// (a festival's splatter fades). The rule decides where; the age and the
// agent's own numbers decide how much.
//
// A layer is a list of agents and the setting they act in. Apply puts a
// layer over a world: every surface it can reach gets a texture that
// asks the layer at each point it is drawn. Nothing here is a number a
// design chose; each is named in the set file that asks for the layer.
//
// Limits, said once. What an agent lays is colour, so leaves, crust and
// cracks are drawn flat, not raised, and a glitter fleck is a bright
// spot, not a glint that follows the eye. The texture is read only by
// the CPU tracer, which hands it the surface's way; the Metal path reads
// the surface as it was before weathering. Shade is the face turned from
// the sun or sheltered from above, not shadow cast by a neighbour.
package weathering

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Kind is which agent an Agent is.
type Kind string

// The agents.
const (
	// Rain streaks run down walls from sills and ledges.
	Rain Kind = "rain"
	// Soot darkens low and sheltered surfaces.
	Soot Kind = "soot"
	// Dust settles on upward faces that nothing stands over.
	Dust Kind = "dust"
	// Moss grows in shade and damp, low, and on the north.
	Moss Kind = "moss"
	// Ivy climbs a wall from the ground up.
	Ivy Kind = "ivy"
	// Graffiti is marks at arm's reach on the faces of walls, over the
	// street and where a writer climbs to: tags, throw-ups and pieces,
	// drawn as shapes in paint and never as letters.
	Graffiti Kind = "graffiti"
	// Rust is on steel, and runs in stains below it.
	Rust Kind = "rust"
	// Spall is cracks and the bare face where brick has flaked.
	Spall Kind = "spall"
	// Vacancy is a state of a dwelling: boarded or broken windows and
	// dark rooms.
	Vacancy Kind = "vacancy"
	// Mineral is dust from a point, thinning with distance and downwind.
	Mineral Kind = "mineral"
	// Glitter is flecks that settle downwind of a point.
	Glitter Kind = "glitter"
	// Nano is contamination spreading along a surface from a seed.
	Nano Kind = "nano"
	// Festival is splatter round a plaza, fading with age.
	Festival Kind = "festival"
	// Spout is a dark streak down a wall below each spout.
	Spout Kind = "spout"
	// Wet is damp low on walls, where a wet street splashes them, and on
	// upward faces the rain lands on.
	Wet Kind = "wet"
)

// Kinds are the agents, in the order the set file lists them.
var Kinds = []Kind{Rain, Soot, Dust, Moss, Ivy, Graffiti, Rust, Spall,
	Vacancy, Mineral, Glitter, Nano, Festival, Spout, Wet}

// Agent is one cause and its numbers. Each number is read by the kinds
// its comment names, and a kind ignores the rest; a number a kind needs
// and is not given is nought, which that kind's rule treats as none.
type Agent struct {
	Kind Kind
	Name string
	// On names what the agent may land on: a surface lands only if its
	// material's name holds one of these words. No words is every surface.
	// For rust they name the steel, for vacancy the panes.
	On []string
	// Mark is the one kind of mark a graffiti agent writes: a tag, a
	// throw-up or a piece, each as big as its agent's Grain makes it.
	// An agent with none writes all three.
	Mark string
	// Seek and Shun name, by the words of a material's name, what a
	// writer goes for and what a writer leaves alone: graffiti is Sought
	// times more again on the one, and keeps only Shunned of its share
	// on the other. No words seek or shun nothing.
	Seek, Shun []string
	// Colour is what the agent lays, and Other a second colour for the
	// kinds that lay two: graffiti's other paint, the bare brick under a
	// spall, a festival's other splatter, the dark behind a broken pane.
	// Line is graffiti's outline, and the ink of a tag not in a fill.
	Colour, Other, Line maths.Light
	// At is a place in the world: a mine, a factory, a nanomachine seed,
	// a plaza, the middle of a run of empty homes. Facing is the way a
	// nanomachine seed's surface faces, and the way the fronts of the
	// homes face that vacancy empties, which are level.
	At, Facing maths.Vec

	// Strength is the most of a surface the agent covers, nought to one.
	Strength float64
	// Settle is the age in years by which the agent has laid about
	// two thirds of what it will. Nought is all at once.
	Settle float64
	// Grain is how wide a patch, a mark, a cell or a fleck is, in metres.
	Grain float64
	// Share is how much of the surface may be touched, nought to one.
	Share float64
	// Tilt is the least that a face points up to count as an upward
	// face, as the share of its normal along up. Level is the most it
	// may point up or down and still be a wall.
	Tilt, Level float64
	// Shelter is how far above a point, in metres, something must stand
	// for the point to count as sheltered from the sky.
	Shelter float64
	// Reach is how far the agent acts: a height, a distance, a radius.
	Reach float64
	// Length is how long a streak runs below what feeds it, in metres.
	Length float64
	// Spacing is how far apart streaks and ivy roots are along a wall.
	Spacing float64
	// Climb is the most height ivy gains, in metres, once fully grown.
	Climb float64
	// Low and High are the heights above ground, in metres, between
	// which graffiti is put and a festival's splatter climbs a wall; for
	// graffiti they are also the reach above anything a writer stands on.
	Low, High float64
	// Heaven is the share of graffiti's street share that lands where a
	// writer climbs to: a roof, a ledge, a catwalk, with sky over it.
	// Behind is how far behind a wall's face, in metres, a writer on a
	// roof stands to reach over a parapet: more than the parapet is thick.
	Heaven, Behind float64
	// Sought and Shunned are how much more graffiti is on what a writer
	// seeks, and the share of it left on what a writer shuns.
	Sought, Shunned float64
	// Shade is the most a face may point toward the sun and count as in
	// shade, as the share of its normal along the sun; North how much
	// moss prefers faces toward north, nought to one.
	Shade, North float64
	// Catch is how much more of an agent lodges in a mortar joint, and
	// how much more graffiti there is on an empty dwelling.
	Catch float64
	// Spread is how wide a plume opens with distance downwind, and
	// Upwind how much of its strength reaches upwind of its source.
	Spread, Upwind float64
	// Rate is how fast a nanomachine front moves, in metres a year, and
	// Turn how much it slows going round a bend.
	Rate, Turn float64
	// Fleck is how big a glitter fleck is, in metres.
	Fleck float64
	// Cell and Storey are how wide and how tall a dwelling's windows
	// are taken together, in metres; no storey is a whole column.
	Cell, Storey float64
	// Boarded and Broken are the shares of empty dwellings' windows that
	// are boarded and that are broken; the rest are dark.
	Boarded, Broken float64
	// Proud is how far boards and shards stand off the glass, Plank how
	// tall a board is, Slant how much a board's ends differ in height,
	// all in metres.
	Proud, Plank, Slant float64
	// Crack is how thin a crack is, in metres.
	Crack float64
	// Spalled is the share of brick that has flaked to its bare face.
	Spalled float64
}

// Setting is what the agents act in: the world's directions, its sun
// and wind, how high its ground is, and what stands over each place.
type Setting struct {
	// Seed makes every patch, cell and fleck; the same seed lays the same.
	Seed uint64
	// Age is the layer's age in years, which every agent lays by.
	Age float64
	// Up, North, Sun and Wind are directions in the world's space. Sun
	// is toward the sun, and nought where there is none, which leaves
	// every face in shade. Wind is the way the wind blows.
	Up, North, Sun, Wind maths.Vec
	// Ground is the height of the ground, in metres.
	Ground float64
	// Probe is how far off a surface, in metres, a look overhead starts,
	// so that a wall does not shelter itself.
	Probe float64
	// Block is how wide, in metres, the cells are that what stands over
	// a place is kept by.
	Block float64
	// Over is what stands over each place, built by Apply.
	Over *Shelter
	// Spouts are the places in the world where water runs from, found by
	// Apply in the things of material SpoutMaterial.
	Spouts []maths.Vec
}

// Layer is the agents and the setting they act in.
type Layer struct {
	Setting
	Agents []Agent
}

// Site is a place on a surface that is being asked about.
type Site struct {
	// Point is where, in the world; Normal is which way the surface
	// faces there, a unit vector out of the surface.
	Point, Normal maths.Vec
	// Name is the surface's material.
	Name string
	// Joint is whether the point is in a mortar joint.
	Joint bool
}

// grown is how much of itself an agent has laid by an age: nought at no
// age, rising and never falling, and one at last. A settle of nought is
// all at once.
func grown(age, settle float64) float64 {
	if !(settle > 0) { // defined: no settling time is all at once
		return 1 // defined: all of it
	}
	return 1 - math.Exp(-math.Max(age, 0)/settle)
}
