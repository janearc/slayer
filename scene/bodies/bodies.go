// Package bodies assembles planets, stars, and black holes from scene parts. A
// rocky planet combines ground, atmosphere, and moons; a gas giant adds rings;
// a black-hole scene combines a dark body with an accretion disc.
//
// NewBody is the door: it checks the core - name, kind, size and mass -
// and hangs the parts, each made by its own constructor from typed
// quantities, each checking its own arguments. Place adds the parts to
// a world and times satellite orbits from the primary's mass; the
// caller supplies a scale to suit the shot.
//
// Lengths are in kilometres, masses in kilograms, angles in radians and
// times are a time.Duration. A physical figure names its published
// source beside its declaration; one that does not is not yet verified.
package bodies

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/si"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// Kind is what a body mostly is.
type Kind int

// The kinds.
const (
	Rock  Kind = iota // a surface you could stand on
	Giant             // gas, no surface
	Hole              // a black hole and its disc
	Star              // a ball that shines: a sun, a neutron star
)

// Sun is the sun's mass in kilograms, the unit the big masses come in.
const Sun = 1.989e30

// lightSpeed is the speed of light, in metres a second, as the SI
// defines it and celestial carries it.
const lightSpeed = si.Speed

// metresPerKilometre is defined: a kilometre is a thousand metres.
const metresPerKilometre = 1000

// horizonPerMass is defined: a hole's horizon radius is twice its mass
// as a length, 2GM over c squared.
const horizonPerMass = 2

// reverse is defined: the factor that turns a direction round.
const reverse = -1

// fullTurn is defined: a circle, in radians.
const fullTurn = 2 * math.Pi

// secondsPerJulianYear is defined: the Julian year is 365.25 days of
// 86400 seconds.
const secondsPerJulianYear = 365.25 * 86400

// jetsOffset keeps a body's jets from growing from the same noise as
// its disc: the jets' seed is the body's plus this.
var jetsOffset = uint64(pick("jets-seed-offset"))

// moonDetailDivisor and moonDetailLeast set a moon's tiling from its
// primary's: the detail over the divisor, and no fewer than the least.
var (
	moonDetailDivisor = int(pick("moon-detail-divisor"))
	moonDetailLeast   = int(pick("moon-detail-least"))
)

// The dressing of a hole's horizon and field, each a number picked.
var (
	holeReach         = pick("hole-reach")
	horizonBallMargin = pick("horizon-ball-margin")
	horizonHexSplits  = int(pick("horizon-hex-splits"))
)

// The wisps of a coma and a tail, each a number picked.
var (
	comaWispDivisor = pick("coma-wisp-divisor")
	comaWispAmount  = pick("coma-wisp-amount")
	tailWispWidths  = pick("tail-wisp-widths")
)

// orbitYearsLimit is the years of a period at and past which an orbit
// is refused, a little short of what a time.Duration holds.
var orbitYearsLimit = pick("orbit-years-limit")

// Omega is how fast a circular orbit goes round the body at a radius,
// in radians a second, by kepler. It is zero for a body with no mass or
// less, and for a radius that is zero, less, or not a number.
func (body *Body) Omega(radius Span) float64 {
	around := radius.InKilometres()
	if body.mass <= nought || !(around > nought) {
		return nought
	}
	metres := around * metresPerKilometre
	return math.Sqrt(thing.G * body.mass / (metres * metres * metres))
}

// Schwarzschild is the radius of a hole of a mass, in kilometres: worked
// out in metres, where G lives, and brought to kilometres at the end.
func Schwarzschild(mass float64) float64 {
	return horizonPerMass * thing.G * mass / (lightSpeed * lightSpeed) /
		metresPerKilometre
}

// Place puts the body's parts into the world at a point, with scale
// turning kilometres into the world's units and detail how finely its
// ground is tiled. It refuses, with the reason, when a part cannot be
// made: a ground whose feature has nowhere to go.
//
// The parts: the ground, the gas, the hole and its disc, the air, the
// rings, the plumes, and the moons on their orbits with their own
// parts, each a thing or a volume that moves as the body does.
func (body Body) Place(into *thing.World, at maths.Vec, scale float64,
	detail int) error {
	if err := vecComplaint(body.name+" placed at a point that", at); err != nil {
		return fmt.Errorf("bodies: %w", err)
	}
	lean := maths.Rotation(maths.XAxis, body.tilt)
	space := maths.Subspace{Origin: at, Orientation: lean}
	var motion thing.Motion = thing.Still(space)
	if body.day != nought {
		// the body turns on its own axis once a day
		motion = thing.Spin{In: space, Axis: lean.Out(maths.Up),
			Period: body.day}
	}
	if err := body.build(into, motion, scale, detail); err != nil {
		return err
	}
	for place, moon := range body.moons {
		err := orbitable(body.name+"'s "+moon.body.name, body.mass,
			moon.orbit*metresPerKilometre)
		if err != nil {
			return err
		}
		orbit, err := body.Orbit(place, at, scale)
		if err != nil {
			return err
		}
		err = moon.body.build(into, orbit, scale,
			max(moonDetailLeast, detail/moonDetailDivisor))
		if err != nil {
			return err
		}
	}
	return nil
}

// PlaceOn puts the body and its parts into the world carried by a
// motion of the caller's, at a scale: the body's own turning is not
// added, so a planet on a track keeps one face along it, as a moon
// does. Its moons are not placed. It refuses as Place does.
func (body Body) PlaceOn(into *thing.World, motion thing.Motion,
	scale float64, detail int) error {
	return body.build(into, motion, scale, detail)
}

// Orbit is the orbit of the body's moon at a place when the body is
// placed at a point at a scale: round the body's own north, kepler's
// period from the body's mass, locked so the moon keeps one face to
// the body.
//
// It is an error when the body has no moon at the place.
func (body Body) Orbit(place int, at maths.Vec,
	scale float64) (thing.Orbit, error) {
	if place < nought || place >= len(body.moons) {
		return thing.Orbit{}, fmt.Errorf("bodies: %s has %d moons: "+
			"there is none at place %d", body.name, len(body.moons),
			place)
	}
	moon := body.moons[place]
	lean := maths.Rotation(maths.XAxis, body.tilt)
	return thing.Orbit{Centre: at, Radius: moon.orbit * scale,
		Axis:   lean.Out(maths.Up),
		Period: thing.Period(body.mass, moon.orbit*metresPerKilometre),
		Phase:  moon.phase, Locked: true}, nil
}

// Stand is where someone stands on a body's ground: the body's space
// at the moment (its scale one), the ground's radius in the world's
// units, which way from the body's centre in its own space, and how
// high above the ground.
//
// It gives the point and the local up, both in the world.
func Stand(space maths.Subspace, radius float64, toward maths.Vec,
	height float64) (at, up maths.Vec) {
	toward = toward.Unit()
	return space.Out(toward.Scale(radius + height)),
		space.Orientation.Out(toward).Unit()
}

// build makes every part the body has, by its kind and then each
// optional part, as things and volumes in the world, all carried by
// one motion. Place and PlaceOn decide where and how it moves; build
// decides what it is made of, and refuses when a part cannot be made.
func (body Body) build(into *thing.World, motion thing.Motion, scale float64,
	detail int) error {
	if into == nil {
		return fmt.Errorf("bodies: %s cannot be placed in a nil world",
			body.name)
	}
	if err := body.sound(scale); err != nil {
		return err
	}
	radius := body.radius * scale
	switch body.kind {
	case Rock:
		ground, err := body.groundMaterial(detail)
		if err != nil {
			return err
		}
		into.Add(thing.Thing{
			Shape:    shape.Sphere{Radius: radius},
			Material: ground,
			Motion:   motion,
		})
	case Giant:
		if body.gas != nil {
			bands := body.gas.bands
			bands.Axis = maths.Up
			into.AddVolume(thing.Volume{
				Medium: medium.Giant{Radius: radius,
					Depth: body.gas.depth * radius,
					Thick: body.gas.opacity /
						(body.gas.depth * radius)},
				Material: &thing.Material{Name: body.name,
					Texture: bands},
				Motion: motion,
			})
		}
	case Star:
		into.Add(thing.Thing{
			Shape: shape.Sphere{Radius: radius},
			Material: &thing.Material{Name: body.name,
				Glow: body.glow, Radiant: body.radiant},
			Motion: motion,
			// a star is the light: it casts no shadow of its own
			Attrs: thing.Attrs{Clear: true},
		})
	case Hole:
		body.buildHole(into, motion, scale)
	}
	return body.buildAtmosphere(into, motion, scale, radius)
}

// buildAtmosphere puts the body's air, aurora and clouds into the world,
// each round the radius, as volumes carried by the body's motion.
func (body Body) buildAtmosphere(into *thing.World, motion thing.Motion,
	scale, radius float64) error {
	if air := body.air; air != nil {
		into.AddVolume(thing.Volume{
			Medium: medium.Air{Radius: radius,
				Height: air.height * scale,
				Thick:  air.dense / scale},
			Material: &thing.Material{Name: body.name + " air",
				Colour: air.colour, Radiant: air.radiant},
			Motion: motion,
		})
	}
	if aurora := body.aurora; aurora != nil {
		axis := maths.Rotation(maths.XAxis, aurora.tilt).Out(maths.Up)
		into.AddVolume(thing.Volume{
			Medium: medium.Aurora{Radius: radius,
				Low:  aurora.low * scale,
				High: aurora.high * scale, Lat: aurora.lat,
				Width: aurora.width, Axis: axis,
				Thick:    unity / (aurora.high - aurora.low) / scale,
				Curtains: aurora.curtains, Seed: aurora.seed},
			Material: &thing.Material{Name: body.name + " aurora",
				Shine: texture.Aurora{Radius: radius,
					Low:  aurora.low * scale,
					High: aurora.high * scale,
					Fringe: aurora.colours.fringe.Scale(
						aurora.strength),
					Bottom: aurora.colours.bottom.Scale(
						aurora.strength),
					Top: aurora.colours.top.Scale(
						aurora.strength)},
				Radiant: aurora.radiant},
			Motion: motion,
		})
	}
	if clouds := body.clouds; clouds != nil {
		into.AddVolume(thing.Volume{
			Medium: medium.Clouds{Radius: radius,
				Low:   clouds.low * scale,
				High:  clouds.high * scale,
				Cover: clouds.cover,
				Thick: clouds.dense / scale, Seed: clouds.seed,
				Locked: body.locked && !clouds.turning, Toward: body.toward,
				Soft: clouds.soft, Turn: clouds.turn, HasTurn: clouds.hasTurn},
			Material: &thing.Material{Name: body.name + " clouds",
				Colour: maths.Grey(unity)},
			Motion: motion,
		})
	}
	return body.buildMatter(into, motion, scale, radius)
}

// buildMatter puts the body's rings, jets, flare and coma into the world,
// as volumes carried by the body's motion.
func (body Body) buildMatter(into *thing.World, motion thing.Motion,
	scale, radius float64) error {
	if rings := body.rings; rings != nil {
		into.AddVolume(thing.Volume{
			Medium: medium.Disc{Inner: rings.inner * radius,
				Outer:   rings.outer * radius,
				Thick:   rings.thick * scale,
				Dense:   rings.dense / scale,
				Profile: rings.profile, Grain: rings.grain,
				Ringlets: rings.ringlets},
			Material: &thing.Material{Name: body.name + " rings",
				Colour: rings.colour},
			Motion: motion,
		})
	}
	if jets := body.jets; jets != nil {
		body.buildJets(into, motion, jets, scale)
	}
	if flare := body.flare; flare != nil {
		into.AddVolume(thing.Volume{
			Medium: medium.Puff{Radius: unity, Thick: flare.opacity},
			Material: &thing.Material{Name: body.name + " flare",
				Glow: flare.glow, Radiant: flare.radiant},
			Motion: thing.Growing{Of: motion, From: flare.at,
				Rate: flare.speed * scale},
		})
	}
	if coma := body.coma; coma != nil {
		into.AddVolume(thing.Volume{
			Medium: medium.Wispy{Of: medium.Puff{
				Radius: coma.radius * scale,
				Thick:  coma.dense / scale}, Seed: body.seed,
				Scale:  coma.radius * scale / comaWispDivisor,
				Amount: comaWispAmount},
			Material: &thing.Material{Name: body.name + " coma",
				Colour: coma.colour, Glow: coma.glow},
			Motion: motion,
		})
	}
	return body.buildStreams(into, motion, scale, radius)
}

// buildStreams puts the body's tails and plumes into the world, as
// volumes carried by the body's motion. It refuses a tail that cannot
// stream.
func (body Body) buildStreams(into *thing.World, motion thing.Motion,
	scale, radius float64) error {
	if err := body.buildTails(into, motion, scale); err != nil {
		return err
	}
	for _, plume := range body.plumes {
		up := plume.toward.Unit()
		into.AddVolume(thing.Volume{
			Medium: medium.Plume{Height: plume.height * scale,
				Width: plume.width * scale,
				Thick: plume.dense / scale},
			Material: &thing.Material{Name: body.name + " plume",
				Colour: plume.colour},
			Motion: thing.Nested{Of: motion, In: maths.Subspace{
				Origin:      up.Scale(radius),
				Orientation: maths.Upright(up)}},
		})
	}
	return nil
}

// buildHole puts a hole into the world: its field, the dressed ball
// just outside its horizon, and its disc, beamed when the gas moves.
func (body Body) buildHole(into *thing.World, motion thing.Motion,
	scale float64) {
	schwarzschild := Schwarzschild(body.mass) * scale
	// the hole's field: its mass as a length is half the
	// schwarzschild radius; it bends rays within twenty radii, and
	// the black is the field's own, rays that came from the horizon
	//
	// the hexes, a dressing, want a ball to be drawn on, just
	// outside the horizon, since nothing on it can be seen
	massLength := schwarzschild / horizonPerMass
	spinLength := math.Abs(body.spin) * massLength
	into.AddField(thing.Field{Mass: massLength, Spin: spinLength,
		Reach:  holeReach * schwarzschild,
		Motion: turnedFor(body.spin, motion)})
	body.buildHorizon(into, motion, scale, massLength, spinLength)
}

// buildHorizon puts the dressed ball just outside a hole's horizon into
// the world, when it has hexes, and then the hole's disc. The hole's mass
// and spin are lengths in the world's units.
func (body Body) buildHorizon(into *thing.World, motion thing.Motion,
	scale, massLength, spinLength float64) {
	if body.edge.hexes > nought {
		light := body.edge.light
		if light == (maths.Light{}) {
			light = maths.Grey(unity)
		}
		// the horizon's kerr-schild radius, as
		// holemath.Hole.Horizon has it, which bodies does not
		// import on purpose: its reach at the poles; at the
		// equator it reaches root(rplus^2 + spin^2)
		rplus := massLength + math.Sqrt(math.Max(nought,
			massLength*massLength-spinLength*spinLength))
		into.Add(thing.Thing{
			Shape: shape.Sphere{
				Radius: horizonBallMargin * math.Sqrt(rplus*rplus+
					spinLength*spinLength)},
			Material: &thing.Material{Name: body.name + " horizon",
				Shine: texture.Wire{
					Tiling: hexes.Sphere{Splits: horizonHexSplits},
					Light:  light.Scale(body.edge.hexes)}},
			Motion: motion,
		})
	}
	body.buildDisc(into, motion, scale)
}

// buildDisc puts a hole's disc into the world, beamed when the gas moves.
func (body Body) buildDisc(into *thing.World, motion thing.Motion,
	scale float64) {
	if disc := body.disc; disc != nil {
		volume := thing.Volume{
			Medium: medium.Disc{Inner: disc.inner * scale,
				Outer: disc.outer * scale,
				Thick: disc.thick * scale,
				Dense: disc.dense / scale},
			// the streaks go round at the orbital rate the
			// hole's mass sets, in the disc's own units
			Material: &thing.Material{Name: body.name + " disc",
				Shine: texture.Thin{Inner: disc.inner * scale,
					Outer:   disc.outer * scale,
					Hot:     disc.hot,
					Bright:  disc.bright,
					Streaks: disc.streaks,
					Shear:   disc.shear, Seed: body.seed,
					Omega: body.Omega(Kilometres(unity / scale))}},
			Motion: motion,
		}
		if disc.speed > nought {
			volume.Beam = &thing.Beam{Axis: maths.Up,
				Speed: disc.speed, Round: true,
				At: disc.inner * scale}
		}
		into.AddVolume(volume)
	}
}

// buildTails streams every tail from the body: away from the sun,
// bending back along the trail, each a volume carried by the body's
// motion in its own leaning frame.
func (body Body) buildTails(into *thing.World, motion thing.Motion,
	scale float64) error {
	for _, tail := range body.tails {
		// away from the sun, bending back along the trail
		at := motion.At(nought).Origin
		if at == tail.sun {
			return fmt.Errorf("bodies: %s is at its sun, so its "+
				"tail has no way to stream: call Aimed with a sun "+
				"somewhere else", body.name)
		}
		away := at.Add(tail.sun.Scale(reverse)).Unit()
		back := tail.trail.Scale(reverse)
		along := away.Dot(back)
		back = back.Add(away.Scale(-along)).Unit()
		if back == (maths.Vec{}) || math.IsNaN(back.X) {
			back = maths.Upright(away).X
		}
		frame := maths.Orientation{X: back, Y: away,
			Z: back.Cross(away)}
		var made medium.Medium = medium.Tail{
			Length: tail.length * scale,
			Base:   tail.base * scale, Spread: tail.spread,
			Curve: tail.curve, Thick: tail.dense / scale}
		if tail.wisp > nought {
			// wisps a few base widths long, so they are several
			// pixels from any distance the tail is seen at
			made = medium.Wispy{Of: made, Seed: tail.seed,
				Scale:  tailWispWidths * tail.base * scale,
				Amount: tail.wisp}
		}
		into.AddVolume(thing.Volume{
			Medium: made,
			Material: &thing.Material{Name: body.name + " tail",
				Colour: tail.colour, Glow: tail.glow,
				Radiant: tail.radiant},
			Motion: thing.Nested{Of: motion,
				In: maths.Subspace{Orientation: frame}},
		})
	}
	return nil
}

// jetFlow is the speed at which the knots ride out, in base widths a
// second, of jets moving at a share of the speed of light from a base a
// width in kilometres wide.
func jetFlow(speed, base float64) float64 {
	return speed * lightSpeed / metresPerKilometre / base
}

// buildJets puts the body's jets into the world: a cone of glowing gas
// each way along an axis leaning off the body's, carried by the body's
// motion, with the knots riding out at the jets' own speed and the bulk
// flow beaming their light toward the eye.
func (body Body) buildJets(into *thing.World, motion thing.Motion, jets *Jets,
	scale float64) {
	// the knots ride out at the jet's own speed, in base widths a
	// second, so a jet seen over time is alive
	flow := jetFlow(jets.speed, jets.base)
	// the gas: lengths to the world's units, the density per unit
	cone := medium.Jet{Length: jets.length * scale,
		Base: jets.base * scale, Spread: jets.spread,
		Thick: jets.dense / scale, From: jets.from * scale,
		Flow: flow, Knots: jets.knots, Seed: body.seed + jetsOffset}
	// what it glows and gives off in bands
	gas := &thing.Material{Name: body.name + " jets", Glow: jets.glow,
		Radiant: jets.radiant}
	// leaning off the body's axis, carried by its motion
	leaning := thing.Nested{Of: motion,
		In: maths.Subspace{
			Orientation: maths.Rotation(maths.XAxis, jets.tilt)}}
	// the bulk flow, which beams the light toward the eye
	beam := &thing.Beam{Axis: maths.Up, Speed: jets.speed}
	into.AddVolume(thing.Volume{Medium: cone, Material: gas, Motion: leaning,
		Beam: beam})
}

// groundTiling is the sphere a ground is made on, split detail times.
func (Body) groundTiling(detail int) hexes.Tiling {
	return hexes.Sphere{Splits: max(unity, detail)}
}

// World is the world a rock body's ground is made from: its terrain on
// a sphere split detail times, its features placed by asking the plain
// world where they go. It refuses a body with no ground, and a feature
// with nowhere to go, with the reason.
//
// With features the terrain is made twice, plain and then with them,
// since world has no way to carve a made world. That cost is a thread.
func (body Body) World(detail int) (*world.World, error) {
	ground := body.ground
	if ground == nil {
		return nil, fmt.Errorf("bodies: %s has no ground to make a "+
			"world from", body.name)
	}
	terrain := ground.terrain
	terrain.Tiling = body.groundTiling(detail)
	if ground.features != nil {
		features, err := ground.features(world.New(terrain))
		if err != nil {
			return nil, fmt.Errorf("bodies: %s: %w", body.name, err)
		}
		terrain.Features = features
	}
	return world.New(terrain), nil
}

// Binary places two bodies going round their common centre, `at`, apart
// from each other, each on its own orbit timed by the two masses
// together, as pluto and charon are. It runs the checks Place runs on
// each body at the scale, and refuses a distance that is negative or
// not a finite number.
//
// Each is drawn as a ball of its ground, whatever its kind, or a grey
// one when it has none: a pair of stars comes out two grey balls with no
// glow. Day, tilt, air and moons are not read, and the two do not keep a
// face to each other.
func Binary(into *thing.World, one, other Body, at maths.Vec,
	apart Span, scale float64, detail int) error {
	if into == nil {
		return fmt.Errorf("bodies: %s and %s cannot be placed in a "+
			"nil world", one.name, other.name)
	}
	if err := vecComplaint(one.name+" and "+other.name+
		" placed at a point that", at); err != nil {
		return fmt.Errorf("bodies: %w", err)
	}
	distance, total, err := pairable(one, other, apart, scale)
	if err != nil {
		return err
	}
	oneRadius := distance * other.mass / total
	otherRadius := distance * one.mass / total
	period := thing.Period(total, distance*metresPerKilometre)
	for _, pairing := range []struct {
		body   Body
		radius float64
		phase  float64
	}{{one, oneRadius, nought}, {other, otherRadius, halfTurn}} {
		ground, err := pairing.body.groundMaterial(detail)
		if err != nil {
			return err
		}
		into.Add(thing.Thing{
			Shape: shape.Sphere{
				Radius: pairing.body.radius * scale},
			Material: ground,
			Motion: thing.Orbit{Centre: at,
				Radius: pairing.radius * scale,
				Period: period, Phase: pairing.phase},
		})
	}
	return nil
}

// pairable is the distance in kilometres and the mass of two bodies
// together, when they can go round each other at a scale: each sound,
// the distance finite and not below zero, some mass between them, and an
// orbit that can be timed. It refuses otherwise, with the names.
func pairable(one, other Body, apart Span,
	scale float64) (distance, total float64, err error) {
	for _, body := range []Body{one, other} {
		if err := body.sound(scale); err != nil {
			return nought, nought, err
		}
	}
	distance = apart.InKilometres()
	if !nonNegativeFinite(distance) {
		return nought, nought, fmt.Errorf("bodies: %s and %s are %v km "+
			"apart: kilometres, zero or above, finite", one.name,
			other.name, distance)
	}
	total = one.mass + other.mass
	if !(total > nought) {
		return nought, nought, fmt.Errorf("bodies: %s and %s have no "+
			"mass to go round each other by", one.name, other.name)
	}
	err = orbitable(one.name+" and "+other.name, total,
		distance*metresPerKilometre)
	return distance, total, err
}

// halfTurn is the turn that takes a space's up to its down: half a
// circle, defined as the angle that reverses a direction.
const halfTurn = math.Pi

// turnedFor is motion with its space turned about x by a half turn when
// spin is below zero, and motion itself otherwise. celestial takes a
// spin of zero or more only: a hole turning the other way is the same
// hole seen from below, so its axis is turned and its spin kept
// positive (hole.NewHole in celestial).
func turnedFor(spin float64, motion thing.Motion) thing.Motion {
	if !retrograde(spin) {
		return motion
	}
	over := maths.Local()
	over.Orientation = maths.Rotation(maths.XAxis, halfTurn)
	return thing.Nested{Of: motion, In: over}
}

// noSpin is the spin of a body that does not turn, defined as zero.
const noSpin = 0

// retrograde is whether a spin turns the other way from the body's
// north: it is below noSpin.
func retrograde(spin float64) bool { return spin < noSpin }

// sound is what is wrong with the body at a scale, if anything, by
// name: a body not made by NewBody, a complaint a verb recorded, a
// scale that is not a size, a radius that is not a size (Resized can
// set one), or a hole spinning at its mass or past, which has no
// horizon to draw. The checks that need no scale are made by the part
// constructors.
func (body Body) sound(scale float64) error {
	if !body.made {
		return errors.New("bodies: a Body{} was not made by NewBody")
	}
	if body.complaint != nil {
		return fmt.Errorf("bodies: %s: %w", body.name, body.complaint)
	}
	if !(scale > nought) || math.IsInf(scale, infinitePositive) {
		return fmt.Errorf("bodies: %s at a scale of %v: a scale is "+
			"kilometres to the world's units, above zero",
			body.name, scale)
	}
	if !(body.radius > nought) ||
		math.IsInf(body.radius, infinitePositive) {
		return fmt.Errorf("bodies: %s resized to %v km: kilometres, "+
			"above zero", body.name, body.radius)
	}
	if body.kind != Hole && (math.IsNaN(body.spin) ||
		math.IsInf(body.spin, anySign)) {
		return fmt.Errorf("bodies: %s spins at %v: the spin must be a "+
			"finite number", body.name, body.spin)
	}
	if body.kind == Hole && !(math.Abs(body.spin) < unity) {
		return fmt.Errorf("bodies: %s spins at %v of its mass: at "+
			"one or past, in either direction, a hole has no "+
			"horizon", body.name, body.spin)
	}
	return nil
}

// orbitable is whether an orbit round mass at a radius in metres can be
// timed: thing.Period refuses one past what a Duration holds, 292.27
// years, and this refuses one of 292 years or more first, a little on
// the safe side, as an error with the names.
func orbitable(names string, mass, metres float64) error {
	if mass <= nought || metres <= nought {
		return nil
	}
	years := fullTurn *
		math.Sqrt(metres*metres*metres/(thing.G*mass)) /
		secondsPerJulianYear
	if !(years < orbitYearsLimit) {
		return fmt.Errorf("bodies: %s go round in %v years: %v or "+
			"more is past what a time.Duration holds", names, years,
			orbitYearsLimit)
	}
	return nil
}
