// Package bodies assembles planets, stars, and black holes from scene parts. A
// rocky planet combines ground, atmosphere, and moons; a gas giant adds rings;
// a black-hole scene combines a dark body with an accretion disc.
//
// Place adds those components to a world and times satellite orbits from the
// primary's mass. The caller supplies a scale to suit the shot.
//
// Sizes, masses, and durations were originally entered from memory and are not
// yet verified against published sources. Mark each verified value beside its
// declaration and cite its source; NASA planetary fact sheets are the intended
// starting point.
package bodies

import (
	"fmt"
	"image"
	"math"
	"time"

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

// lightSpeed is the speed of light, in metres a second.
const lightSpeed = 299792458.0

// jetsOffset keeps a body's jets from growing from the same noise as
// its disc: the jets' seed is the body's plus this.
const jetsOffset = 4

// Body is a body as a whole.
type Body struct {
	Name string
	Kind Kind
	// Radius in kilometres, Mass in kilograms, Day the time it turns once,
	// Tilt its axis off its orbit's north in radians. Spin is a hole's, its
	// angular momentum per unit mass as a share of its mass, zero for
	// schwarzschild and under one for kerr.
	Radius float64
	Mass   float64
	Day    time.Duration
	Tilt   float64
	Spin   float64
	// Seed grows the noise of the parts that have no seed of their
	// own, the disc, the jets and the coma, so two bodies do not share
	// one noise. Clouds, Cities, Aurora and a Tail carry their own.
	Seed uint64
	// Glow is what a star's surface gives off, and Radiant what it
	// gives off and passes outside the visible.
	Glow    maths.Light
	Radiant thing.Radiant
	// The parts, each there or not.
	Ground *Ground
	Gas    *Gas
	Air    *Air
	Clouds *Clouds
	Cities *Cities
	Aurora *Aurora
	Rings  *Rings
	Disc   *Disc
	Jets   *Jets
	Flare  *Flare
	Edge   Horizon
	Coma   *Coma
	Tails  []Tail
	Plumes []Plume
	Moons  []Moon
}

// Coma is a comet's head: a cloud of gas and dust round the nucleus,
// Radius in kilometres, Dense at its heart, scattering the sun in
// Colour and giving off Glow of its own, the green of carbon.
//
// Dense, here and on every part that has one, is an extinction per
// kilometre: how much of the light is lost in a kilometre of path
// where the part is densest, falling off by the medium's own profile,
// one number for every band.
//
// Thick is always a length in kilometres, and Opacity an optical depth
// with no unit.
type Coma struct {
	Radius float64
	Dense  float64
	Colour maths.Light
	Glow   maths.Light
}

// Tail is a stream from the body away from its sun: Length, Base, Spread and
// Curve as a medium.Tail takes them, in kilometres, Dense per kilometre at its
// heart, scattering the sun in Colour and glowing Glow; Wisp is how ragged it
// is, zero smooth.
//
// Sun is where the sun is in the world and Trail which way the body is moving,
// both set by the shot with Aim: the tail streams away from the sun and bends
// back along the trail, set once, where the body is at time zero.
//
// the tail is laid in the body's own turning, tilted space, so on a body that
// turns or leans it streams away from the sun only at zero and turns with the
// body after: kept as it was, as ruled.
type Tail struct {
	Length, Base, Spread float64
	Curve                float64
	Dense                float64
	Colour, Glow         maths.Light
	Radiant              thing.Radiant
	Wisp                 float64
	Seed                 uint64
	Sun, Trail           maths.Vec
}

// Aim points every tail of the body away from a sun at a point in the
// world, bending back along the way the body moves.
func (b *Body) Aim(sun, trail maths.Vec) {
	for i := range b.Tails {
		b.Tails[i].Sun, b.Tails[i].Trail = sun, trail
	}
}

// Clouds is a world's cloud layer: Low and High above the surface in
// kilometres, Cover the share of the sky clouded, Dense the extinction
// per kilometre at a patch's heart, Seed its weather.
type Clouds struct {
	Low, High float64
	Cover     float64
	Dense     float64
	Seed      uint64
}

// Aurora is a body's aurora: a glow in a shell from Low to High above the
// surface in kilometres, in an oval round each magnetic pole at Lat from the
// pole's equator and Width wide, both in radians, the magnetic axis Tilt off
// the spin, about x.
//
// Strength is the glow at the oval's heart as a share of the sun's light, which
// a night shot exposes for; Curtains the dressing, zero for a smooth band.
//
// Fringe, Bottom and Top are its colours with height; the earth's are purple,
// green and red, a giant's blues standing in for the ultraviolet it really
// shines in.
type Aurora struct {
	Low, High           float64
	Lat, Width          float64
	Tilt                float64
	Strength            float64
	Curtains            float64
	Seed                uint64
	Fringe, Bottom, Top maths.Light
	// Radiant, when there is one, is what the aurora gives off and
	// passes outside the visible: an aurora driven by a magnetar is
	// x-ray bright.
	Radiant thing.Radiant
}

// Cities is a world's lights at night, laid as the black marble lays
// the earth's: Rungs below the ground's tiles, Share of the land tiles
// lit as towns and Metro of the tiles two rungs up as metros, more on
// coasts and in the middle latitudes, each Glow, Seed their places.
type Cities struct {
	Rungs int
	Share float64
	Metro float64
	Glow  maths.Light
	Seed  uint64
}

// Jets is a pair of jets along the body's axis, Tilt off it: Length,
// Base, From and the spread of a medium.Jet in kilometres, Dense per
// kilometre at the base, and what they glow and give off in bands.
//
// Speed is the share of the speed of light they move at, which beams
// their light.
type Jets struct {
	Length, Base, Spread float64
	From                 float64
	Dense                float64
	Tilt                 float64
	Speed                float64
	// Knots is a dressing, as medium.Jet takes it: lumps along the jet,
	// zero for a smooth one.
	Knots   float64
	Glow    maths.Light
	Radiant thing.Radiant
}

// Flare is a fireball from a star's surface: born At, growing at Speed
// kilometres a second, Opacity the optical depth across its own radius
// however big it has grown, glowing and giving off in bands. Before At
// there is nothing.
type Flare struct {
	At      time.Duration
	Speed   float64
	Opacity float64
	Glow    maths.Light
	Radiant thing.Radiant
}

// Plume is a volcanic plume: a column of gas and dust rising from a
// point on the ground and falling back in an umbrella, as io's do.
//
// Toward is the point on the ground as a direction from the centre;
// Height and Width are in kilometres; Dense is the extinction per
// kilometre at the column's heart.
type Plume struct {
	Toward        maths.Vec
	Height, Width float64
	Dense         float64
	Colour        maths.Light
}

// Fineness is detail painted finer than the ground's own hex tiles, for an eye
// near it. Each fine tile takes the land class of the coarse tile it sits in,
// its brightness varied by Vary and a neighbouring class let in by Mix, both
// shares.
//
// Rungs is how many levels finer the hex tiling goes, zero for the picture to
// choose from the angle a pixel covers (world.Fine says how).
type Fineness struct {
	Rungs     int
	Vary, Mix float64
}

// Ground is a rock world's surface: how its terrain is made, how it is
// painted, and what features it has, placed by what the terrain has
// where.
type Ground struct {
	Terrain world.Terrain
	Look    world.Look
	// Features places the features on a plain world, by asking it
	// where its land and water are; it refuses, with the reason, when
	// a feature has nowhere to go.
	Features func(w *world.World) ([]world.Feature, error)
	// Radiant is what the ground does outside the visible, and Rough how
	// rough each class is for a radar, as a look whose paints are greys
	// from zero to one. Nil is opaque, dark, and rough alike.
	Radiant thing.Radiant
	Rough   *world.Look
	// Fine, when there is one, paints the ground below its own rung,
	// for an eye near it. Nil paints the tiles as they are.
	Fine *Fineness
	// Map and Lights, when given, are pictures wrapped on the sphere in
	// place of the painted world and the cities: an equirectangular map of
	// the ground by day, and of its lights at night, as nasa publishes the
	// blue and the black marble.
	//
	// Turn moves their left edge round the sphere, in radians.
	//
	// The lights are scaled by Glow, a city's light against the sun's,
	// which a night shot exposes for; zero is read as four thousandths.
	Map, Lights image.Image
	Turn        float64
	Glow        float64
}

// Gas is a giant's body: Depth, how far below the cloud tops it goes
// opaque, as a share of the radius; Opacity, the optical depth from the
// cloud tops to there, no unit; and Bands, what it looks like.
type Gas struct {
	Depth   float64
	Opacity float64
	Bands   texture.Banded
}

// Air is an atmosphere: Height its scale height in kilometres, Dense
// the extinction per kilometre at the surface, and the colour of what
// it scatters.
type Air struct {
	Height float64
	Dense  float64
	Colour maths.Light
	// Radiant is what the air does outside the visible: venus's clouds
	// pass radio, which is how its ground was mapped.
	Radiant thing.Radiant
}

// Rings are a disc round a giant: from Inner to Outer as shares of the
// radius, Thick its thickness in kilometres, and the rest as a
// medium.Disc has it.
type Rings struct {
	Inner, Outer float64
	Thick        float64
	Dense        float64
	Profile      []float64
	Grain        float64
	Ringlets     float64
	Colour       maths.Light
}

// Horizon is a hole's edge, a body's Edge; what is on it is a dressing, since
// nothing on a horizon can be seen. Hexes is the brightness of hex lines drawn
// on a black ball just outside it, zero for none, and Light their colour, white
// when unset.
//
// A spinning hole's horizon is flattened at the poles, so the ball is as wide
// as its equator and a fiftieth more: outside the horizon at the equator, and a
// hair further outside at the poles.
//
// The ball blocks a camera's way out from inside, where no horizon would; a
// fall through the horizon wants the hexes off.
type Horizon struct {
	Hexes float64
	Light maths.Light
}

// Disc is the glowing disc round a hole: from Inner to Outer in
// kilometres, Thick its thickness in kilometres, Dense the extinction
// per kilometre; its colour falls from Hot at the inner edge by
// texture.Thin's profile.
type Disc struct {
	Inner, Outer float64
	Thick, Dense float64
	// Hot is the hottest ring's temperature in kelvin and Bright its
	// light: the disc's colour is a black body's by radius, none typed.
	Hot, Bright float64
	// Speed is the gas's orbital speed at the inner edge as a share of the
	// speed of light, which beams the side coming toward the eye; zero is
	// none.
	Speed float64
	// Streaks and Shear are a dressing, as texture.Thin takes them: a
	// toggle for a film, zero honest.
	Streaks, Shear float64
}

// Moon is a body on an orbit round another: its own body, the orbit's
// radius in kilometres, and where on it the moon starts.
type Moon struct {
	Body  Body
	Orbit float64
	Phase float64
}

// Omega is how fast a circular orbit goes round the body at a radius,
// in radians a second, by kepler: the radius in kilometres.
func (b *Body) Omega(radius float64) float64 {
	if b.Mass <= 0 || radius <= 0 {
		return 0
	}
	r := radius * 1000
	return math.Sqrt(thing.G * b.Mass / (r * r * r))
}

// Schwarzschild is the radius of a hole of a mass, in kilometres: worked
// out in metres, where G lives, and brought to kilometres at the end.
func Schwarzschild(mass float64) float64 {
	return 2 * thing.G * mass / (lightSpeed * lightSpeed) / 1000
}

// Place puts the body's parts into the world at a point, with scale
// turning kilometres into the world's units and detail how finely its
// ground is tiled. It refuses, with the reason, when a part cannot be
// made: a ground whose feature has nowhere to go.
//
// The parts: the ground, the gas, the hole and its disc, the air, the
// rings, the plumes, and the moons on their orbits with their own
// parts, each a thing or a volume that moves as the body does.
func (b Body) Place(w *thing.World, at maths.Vec, scale float64,
	detail int) error {
	lean := maths.Rotation(maths.XAxis, b.Tilt)
	space := maths.Subspace{Origin: at, Orientation: lean}
	var motion thing.Motion = thing.Still(space)
	if b.Day != 0 {
		// the body turns on its own axis once a day
		motion = thing.Spin{In: space, Axis: lean.Out(maths.Up), Period: b.Day}
	}
	if err := b.build(w, motion, scale, detail); err != nil {
		return err
	}
	for i, m := range b.Moons {
		if err := orbitable(b.Name+"'s "+m.Body.Name, b.Mass, m.Orbit*1000); err != nil {
			return err
		}
		err := m.Body.build(w, b.Orbit(i, at, scale), scale, max(2, detail/2))
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
func (b Body) PlaceOn(w *thing.World, motion thing.Motion, scale float64,
	detail int) error {
	return b.build(w, motion, scale, detail)
}

// Orbit is the orbit of the body's i-th moon when the body is placed at
// a point at a scale: round the body's own north, kepler's period from
// the body's mass, locked so the moon keeps one face to the body.
func (b Body) Orbit(i int, at maths.Vec, scale float64) thing.Orbit {
	m := b.Moons[i]
	lean := maths.Rotation(maths.XAxis, b.Tilt)
	return thing.Orbit{Centre: at, Radius: m.Orbit * scale,
		Axis: lean.Out(maths.Up), Period: thing.Period(b.Mass, m.Orbit*1000),
		Phase: m.Phase, Locked: true}
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
func (b Body) build(w *thing.World, motion thing.Motion, scale float64,
	detail int) error {
	if err := b.sound(scale); err != nil {
		return err
	}
	r := b.Radius * scale
	switch b.Kind {
	case Rock:
		ground, err := b.groundMaterial(detail)
		if err != nil {
			return err
		}
		w.Add(thing.Thing{
			Shape:    shape.Sphere{Radius: r},
			Material: ground,
			Motion:   motion,
		})
	case Giant:
		if b.Gas != nil {
			bands := b.Gas.Bands
			bands.Axis = maths.Up
			w.AddVolume(thing.Volume{
				Medium: medium.Giant{Radius: r, Depth: b.Gas.Depth * r,
					Thick: b.Gas.Opacity / (b.Gas.Depth * r)},
				Material: &thing.Material{Name: b.Name, Texture: bands},
				Motion:   motion,
			})
		}
	case Star:
		w.Add(thing.Thing{
			Shape: shape.Sphere{Radius: r},
			Material: &thing.Material{Name: b.Name, Glow: b.Glow,
				Radiant: b.Radiant},
			Motion: motion,
			// a star is the light: it casts no shadow of its own
			Attrs: thing.Attrs{Clear: true},
		})
	case Hole:
		rs := Schwarzschild(b.Mass) * scale
		// the hole's field: its mass as a length is half the
		// schwarzschild radius; it bends rays within twenty radii, and
		// the black is the field's own, rays that came from the horizon
		//
		// the hexes, a dressing, want a ball to be drawn on, just
		// outside the horizon, since nothing on it can be seen
		m := rs / 2
		spin := b.Spin * m
		w.AddField(thing.Field{Mass: m, Spin: spin, Reach: 20 * rs,
			Motion: motion})
		if b.Edge.Hexes > 0 {
			light := b.Edge.Light
			if light == (maths.Light{}) {
				light = maths.Grey(1)
			}
			// the horizon's kerr-schild radius, as
			// holemath.Hole.Horizon has it, which bodies does not
			// import on purpose: its reach at the poles; at the
			// equator it reaches root(rplus^2 + spin^2)
			rplus := m + math.Sqrt(math.Max(0, m*m-spin*spin))
			w.Add(thing.Thing{
				Shape: shape.Sphere{
					Radius: 1.02 * math.Sqrt(rplus*rplus+spin*spin)},
				Material: &thing.Material{Name: b.Name + " horizon",
					Shine: texture.Wire{Tiling: hexes.Sphere{N: 6},
						Light: light.Scale(b.Edge.Hexes)}},
				Motion: motion,
			})
		}
		if b.Disc != nil {
			disc := b.Disc
			v := thing.Volume{
				Medium: medium.Disc{Inner: disc.Inner * scale, Outer: disc.Outer * scale,
					Thick: disc.Thick * scale, Dense: disc.Dense / scale},
				// the streaks go round at the orbital rate the
				// hole's mass sets, in the disc's own units
				Material: &thing.Material{Name: b.Name + " disc",
					Shine: texture.Thin{Inner: disc.Inner * scale,
						Outer: disc.Outer * scale, Hot: disc.Hot,
						Bright: disc.Bright, Streaks: disc.Streaks,
						Shear: disc.Shear, Seed: b.Seed,
						Omega: b.Omega(1 / scale)}},
				Motion: motion,
			}
			if disc.Speed > 0 {
				v.Beam = &thing.Beam{Axis: maths.Up, Speed: disc.Speed,
					Round: true, At: disc.Inner * scale}
			}
			w.AddVolume(v)
		}
	}
	if air := b.Air; air != nil {
		w.AddVolume(thing.Volume{
			Medium: medium.Air{Radius: r, Height: air.Height * scale,
				Thick: air.Dense / scale},
			Material: &thing.Material{Name: b.Name + " air", Colour: air.Colour,
				Radiant: air.Radiant},
			Motion: motion,
		})
	}
	if aurora := b.Aurora; aurora != nil {
		axis := maths.Rotation(maths.XAxis, aurora.Tilt).Out(maths.Up)
		w.AddVolume(thing.Volume{
			Medium: medium.Aurora{Radius: r, Low: aurora.Low * scale,
				High: aurora.High * scale, Lat: aurora.Lat, Width: aurora.Width,
				Axis: axis, Thick: 1 / (aurora.High - aurora.Low) / scale,
				Curtains: aurora.Curtains, Seed: aurora.Seed},
			Material: &thing.Material{Name: b.Name + " aurora",
				Shine: texture.Aurora{Radius: r, Low: aurora.Low * scale,
					High:   aurora.High * scale,
					Fringe: aurora.Fringe.Scale(aurora.Strength),
					Bottom: aurora.Bottom.Scale(aurora.Strength),
					Top:    aurora.Top.Scale(aurora.Strength)},
				Radiant: aurora.Radiant},
			Motion: motion,
		})
	}
	if clouds := b.Clouds; clouds != nil {
		w.AddVolume(thing.Volume{
			Medium: medium.Clouds{Radius: r, Low: clouds.Low * scale,
				High: clouds.High * scale, Cover: clouds.Cover,
				Thick: clouds.Dense / scale, Seed: clouds.Seed},
			Material: &thing.Material{Name: b.Name + " clouds",
				Colour: maths.Grey(1)},
			Motion: motion,
		})
	}
	if rings := b.Rings; rings != nil {
		w.AddVolume(thing.Volume{
			Medium: medium.Disc{Inner: rings.Inner * r, Outer: rings.Outer * r,
				Thick: rings.Thick * scale, Dense: rings.Dense / scale,
				Profile: rings.Profile, Grain: rings.Grain,
				Ringlets: rings.Ringlets},
			Material: &thing.Material{Name: b.Name + " rings",
				Colour: rings.Colour},
			Motion: motion,
		})
	}
	if jets := b.Jets; jets != nil {
		b.buildJets(w, motion, jets, scale)
	}
	if flare := b.Flare; flare != nil {
		w.AddVolume(thing.Volume{
			Medium: medium.Puff{Radius: 1, Thick: flare.Opacity},
			Material: &thing.Material{Name: b.Name + " flare", Glow: flare.Glow,
				Radiant: flare.Radiant},
			Motion: thing.Growing{Of: motion, From: flare.At,
				Rate: flare.Speed * scale},
		})
	}
	if coma := b.Coma; coma != nil {
		w.AddVolume(thing.Volume{
			Medium: medium.Wispy{Of: medium.Puff{Radius: coma.Radius * scale,
				Thick: coma.Dense / scale}, Seed: b.Seed,
				Scale: coma.Radius * scale / 3, Amount: 0.5},
			Material: &thing.Material{Name: b.Name + " coma", Colour: coma.Colour,
				Glow: coma.Glow},
			Motion: motion,
		})
	}
	for _, tail := range b.Tails {
		// away from the sun, bending back along the trail
		at := motion.At(0).Origin
		if at == tail.Sun {
			return fmt.Errorf("bodies: %s is at its sun, so its tail has "+
				"no way to stream: Aim it at a sun somewhere else", b.Name)
		}
		away := at.Add(tail.Sun.Scale(-1)).Unit()
		back := tail.Trail.Scale(-1)
		back = back.Add(away.Scale(-away.Dot(back))).Unit()
		if back == (maths.Vec{}) || math.IsNaN(back.X) {
			back = maths.Upright(away).X
		}
		o := maths.Orientation{X: back, Y: away, Z: back.Cross(away)}
		var m medium.Medium = medium.Tail{Length: tail.Length * scale,
			Base: tail.Base * scale, Spread: tail.Spread, Curve: tail.Curve,
			Thick: tail.Dense / scale}
		if tail.Wisp > 0 {
			// wisps a few base widths long, so they are several
			// pixels from any distance the tail is seen at
			m = medium.Wispy{Of: m, Seed: tail.Seed, Scale: 3 * tail.Base * scale,
				Amount: tail.Wisp}
		}
		w.AddVolume(thing.Volume{
			Medium: m,
			Material: &thing.Material{Name: b.Name + " tail", Colour: tail.Colour,
				Glow: tail.Glow, Radiant: tail.Radiant},
			Motion: thing.Nested{Of: motion, In: maths.Subspace{Orientation: o}},
		})
	}
	for _, plume := range b.Plumes {
		up := plume.Toward.Unit()
		w.AddVolume(thing.Volume{
			Medium: medium.Plume{Height: plume.Height * scale,
				Width: plume.Width * scale, Thick: plume.Dense / scale},
			Material: &thing.Material{Name: b.Name + " plume",
				Colour: plume.Colour},
			Motion: thing.Nested{Of: motion, In: maths.Subspace{
				Origin: up.Scale(r), Orientation: maths.Upright(up)}},
		})
	}
	return nil
}

// buildJets puts the body's jets into the world: a cone of glowing gas
// each way along an axis Tilt off the body's, carried by the body's
// motion, with the knots riding out at the jets' own speed and the bulk
// flow beaming their light toward the eye.
func (b Body) buildJets(w *thing.World, motion thing.Motion, jets *Jets,
	scale float64) {
	// the knots ride out at the jet's own speed, in base widths a
	// second, so a jet seen over time is alive
	flow := 0.0
	if jets.Base > 0 {
		flow = jets.Speed * lightSpeed / 1000 / jets.Base
	}
	// the gas: lengths to the world's units, the density per unit
	cone := medium.Jet{Length: jets.Length * scale, Base: jets.Base * scale,
		Spread: jets.Spread, Thick: jets.Dense / scale, From: jets.From * scale,
		Flow: flow, Knots: jets.Knots, Seed: b.Seed + jetsOffset}
	// what it glows and gives off in bands
	gas := &thing.Material{Name: b.Name + " jets", Glow: jets.Glow,
		Radiant: jets.Radiant}
	// leaning Tilt off the body's axis, carried by its motion
	leaning := thing.Nested{Of: motion,
		In: maths.Subspace{Orientation: maths.Rotation(maths.XAxis, jets.Tilt)}}
	// the bulk flow, which beams the light toward the eye
	beam := &thing.Beam{Axis: maths.Up, Speed: jets.Speed}
	w.AddVolume(thing.Volume{Medium: cone, Material: gas, Motion: leaning,
		Beam: beam})
}

// groundTiling is the sphere a ground is made on, split detail times.
func (Body) groundTiling(detail int) hexes.Tiling {
	return hexes.Sphere{N: max(1, detail)}
}

// groundMaterial is the body's surface as a material: its world, made
// on a sphere split detail times, painted in its look. A body with no
// Ground is a mid-grey ball. It refuses when the world cannot be made.
func (b Body) groundMaterial(detail int) (*thing.Material, error) {
	if b.Ground == nil {
		return &thing.Material{Name: b.Name, Colour: maths.Grey(0.5)}, nil
	}
	g := b.Ground
	made, err := b.World(detail)
	if err != nil {
		return nil, err
	}
	m := &thing.Material{Name: b.Name, Radiant: g.Radiant}
	switch {
	case g.Map != nil:
		m.Texture = texture.Picture{Image: g.Map, Turn: g.Turn}
	case g.Fine != nil:
		f := g.Fine
		m.Texture = made.Fine(g.Look, f.Rungs, f.Vary, f.Mix)
	default:
		m.Texture = made.Painted(g.Look)
	}
	if g.Lights != nil {
		glow := g.Glow
		if glow == 0 {
			glow = 0.004
		}
		m.Shine = texture.Picture{Image: g.Lights, Turn: g.Turn, Gain: glow}
	}
	if cities := b.Cities; cities != nil && g.Lights == nil {
		shine := texture.Cities{Tiling: made.Tiling, Rungs: cities.Rungs,
			Share: cities.Share, Metro: cities.Metro, Glow: cities.Glow,
			Seed: cities.Seed,
			Land: func(p maths.Vec) bool {
				return world.Land(made.Biome(made.Tiling.Nearest(p)))
			},
			// a coast is a land tile with water beside it
			Coast: func(p maths.Vec) bool {
				tile := made.Tiling.Nearest(p)
				for _, n := range made.Tiling.Neighbours(tile) {
					if world.Water(made.Biome(n)) {
						return true
					}
				}
				return false
			}}
		if g.Map != nil {
			// the map says where the land and the water are: a
			// coast is land with water within a few degrees
			pic := texture.Picture{Image: g.Map, Turn: g.Turn}
			shine.Land = func(p maths.Vec) bool { return !pic.Water(p) }
			shine.Coast = func(p maths.Vec) bool {
				o := maths.Upright(p)
				for _, d := range []maths.Vec{o.X, o.X.Neg(), o.Z, o.Z.Neg()} {
					if pic.Water(p.Unit().Add(d.Scale(0.04))) {
						return true
					}
				}
				return false
			}
		}
		m.Shine = shine
	}
	if g.Rough != nil {
		// a radar map is finer than the paint: three rungs below the
		// world's own, varied and mixed, as magellan's is in look; the
		// 3, 0.3 and 0.35 are chosen, not from its data
		m.Rough = made.Fine(*g.Rough, 3, 0.3, 0.35)
	}
	return m, nil
}

// World is the world a rock body's ground is made from: its terrain on
// a sphere split detail times, its features placed by asking the plain
// world where they go. It refuses a body with no ground, and a feature
// with nowhere to go, with the reason.
//
// With features the terrain is made twice, plain and then with them,
// since world has no way to carve a made world. That cost is a thread.
func (b Body) World(detail int) (*world.World, error) {
	g := b.Ground
	if g == nil {
		return nil, fmt.Errorf("bodies: %s has no ground to make a world from",
			b.Name)
	}
	t := g.Terrain
	t.Tiling = b.groundTiling(detail)
	if g.Features != nil {
		features, err := g.Features(world.New(t))
		if err != nil {
			return nil, fmt.Errorf("bodies: %s: %w", b.Name, err)
		}
		t.Features = features
	}
	return world.New(t), nil
}

// Binary places two bodies going round their common centre, `at`, apart
// kilometres from each other, each on its own orbit timed by the two
// masses together, as pluto and charon are. It refuses as Place does.
//
// each is drawn as a rock's grey ground, whatever its Kind: a pair of
// stars comes out two grey balls with no glow. Day, Tilt, Air and Moons
// are not read, and the two do not keep a face to each other: kept as it
// is, as ruled.
func Binary(w *thing.World, a, b Body, at maths.Vec, apart, scale float64,
	detail int) error {
	total := a.Mass + b.Mass
	if !(total > 0) {
		return fmt.Errorf("bodies: %s and %s have no mass to go round "+
			"each other by", a.Name, b.Name)
	}
	if err := orbitable(a.Name+" and "+b.Name, total, apart*1000); err != nil {
		return err
	}
	ra, rb := apart*b.Mass/total, apart*a.Mass/total
	period := thing.Period(total, apart*1000)
	for _, p := range []struct {
		body   Body
		radius float64
		phase  float64
	}{{a, ra, 0}, {b, rb, math.Pi}} {
		ground, err := p.body.groundMaterial(detail)
		if err != nil {
			return err
		}
		w.Add(thing.Thing{
			Shape:    shape.Sphere{Radius: p.body.Radius * scale},
			Material: ground,
			Motion: thing.Orbit{Centre: at, Radius: p.radius * scale,
				Period: period, Phase: p.phase},
		})
	}
	return nil
}

// sound is what is wrong with the body at a scale, if anything, by
// name: a scale that is not a size, a giant's gas of no depth, an aurora
// whose top is not above its bottom, or a hole spinning at its mass or
// past, which has no horizon to draw.
func (b Body) sound(scale float64) error {
	if !(scale > 0) || math.IsInf(scale, 1) {
		return fmt.Errorf("bodies: %s at a scale of %v: a scale is "+
			"kilometres to the world's units, above nought", b.Name, scale)
	}
	if b.Kind == Giant && b.Gas != nil && !(b.Gas.Depth > 0) {
		return fmt.Errorf("bodies: %s's gas has a depth of %v: it is a "+
			"share of the radius, above nought", b.Name, b.Gas.Depth)
	}
	if a := b.Aurora; a != nil && !(a.High > a.Low) {
		return fmt.Errorf("bodies: %s's aurora runs from %v to %v: its "+
			"High is above its Low", b.Name, a.Low, a.High)
	}
	if b.Kind == Hole && !(b.Spin < 1) {
		return fmt.Errorf("bodies: %s spins at %v of its mass: at one or "+
			"past, a hole has no horizon", b.Name, b.Spin)
	}
	return nil
}

// orbitable is whether an orbit round mass at a radius in metres can be
// timed: thing.Period refuses one past what a Duration holds, 292.27
// years, and this refuses one of 292 years or more first, a little on
// the safe side, as an error with the names.
func orbitable(names string, mass, metres float64) error {
	if mass <= 0 || metres <= 0 {
		return nil
	}
	years := 2 * math.Pi * math.Sqrt(metres*metres*metres/(thing.G*mass)) /
		(365.25 * 86400)
	if !(years < 292) {
		return fmt.Errorf("bodies: %s go round in %v years: 292 or more "+
			"is past what a time.Duration holds", names, years)
	}
	return nil
}
