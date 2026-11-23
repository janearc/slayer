package bodies

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Body is a body as a whole: a checked core - its name, kind, size and
// mass - and the parts hung on it, each there or not. NewBody is the
// door; Place and PlaceOn put it in a world.
type Body struct {
	// made is set only by NewBody: a Body{} is no body.
	made bool
	name string
	kind Kind
	// radius in kilometres, mass in kilograms, day the time it turns
	// once, tilt its axis off its orbit's north in radians. spin is a
	// hole's, its angular momentum per unit mass as a share of its
	// mass, zero for schwarzschild and under one for kerr.
	radius float64
	mass   float64
	day    time.Duration
	tilt   float64
	spin   float64
	// seed grows the noise of the parts that have no seed of their
	// own, the disc, the jets and the coma, so two bodies do not share
	// one noise. Clouds, Cities, Aurora and a Tail carry their own.
	seed uint64
	// complaint is the first bad argument a verb that cannot return an
	// error was given, without the package prefix; Place refuses it.
	complaint error
	// glow is what a star's surface gives off, and radiant what it
	// gives off and passes outside the visible.
	glow    maths.Light
	radiant thing.Radiant
	// the parts, each there or not.
	ground *Ground
	gas    *Gas
	air    *Air
	clouds *Clouds
	cities *Cities
	aurora *Aurora
	rings  *Rings
	disc   *Disc
	jets   *Jets
	flare  *Flare
	edge   Horizon
	coma   *Coma
	tails  []Tail
	plumes []Plume
	moons  []Moon
}

// NewBody is the door: a named body of a kind, its radius and its
// mass, both finite and above zero, wearing the parts. It refuses a
// nil part, one its constructor did not make, and one holding a bad
// argument a method was given; Place checks what needs a scale.
func NewBody(name string, kind Kind, radius Span,
	mass measure.Mass, parts ...Part) (Body, error) {
	if name == "" {
		return Body{}, fmt.Errorf("bodies: a body needs a name")
	}
	if kind < Rock || kind > Star {
		return Body{}, fmt.Errorf("bodies: %s is no kind of body",
			name)
	}
	around := radius.InKilometres()
	heavy := mass.InKilograms()
	if !(around > nought) || math.IsInf(around, infinitePositive) {
		return Body{}, fmt.Errorf("bodies: %s's radius is %v km: "+
			"above zero, finite", name, around)
	}
	if !(heavy > nought) || math.IsInf(heavy, infinitePositive) {
		return Body{}, fmt.Errorf("bodies: %s's mass is %v kg: "+
			"above zero, finite", name, heavy)
	}
	made := Body{made: true, name: name, kind: kind, radius: around, mass: heavy}
	if err := made.wear(parts); err != nil {
		return Body{}, err
	}
	return made, nil
}

// refusal is why a part cannot be worn: it was not made by its
// constructor, which it names, or it holds the complaint a method
// recorded. It is nil for a part that is made and holds none.
func refusal(made bool, constructor string, complaint error) error {
	if !made {
		return fmt.Errorf("was not made by %s", constructor)
	}
	return complaint
}

// wear hangs every part on the body, having checked all of them first:
// a nil part, one that its constructor did not make, or one holding a
// complaint is refused, by the body's name and the part's type, and
// nothing is hung. A part whose methods panic, as one embedding a nil
// part does, is refused the same way.
func (body *Body) wear(parts []Part) (err error) {
	trying := &attempt{name: body.name}
	defer trying.refuse(&err)
	for _, part := range parts {
		trying.part = part
		if part == nil {
			return fmt.Errorf("bodies: %s cannot wear a nil part",
				body.name)
		}
		if value := reflect.ValueOf(part); value.Kind() == reflect.Pointer &&
			value.IsNil() {
			return fmt.Errorf("bodies: %s cannot wear a nil %T",
				body.name, part)
		}
		if err := part.refusal(); err != nil {
			return fmt.Errorf("bodies: %s cannot wear a %T: %w",
				body.name, part, err)
		}
	}
	for _, part := range parts {
		trying.part = part
		part.hang(body)
	}
	return nil
}

// attempt is the part a body is trying to wear, held so that a part
// whose methods panic can be named in the refusal.
type attempt struct {
	name string
	part Part
}

// refuse turns a panic from the part's methods into the error it
// points at. It is deferred, and recovers only when it is called by the
// defer itself.
func (trying *attempt) refuse(err *error) {
	if lost := recover(); lost != nil {
		*err = fmt.Errorf("bodies: %s cannot wear a %T: its methods "+
			"panic (%v), as a type embedding a nil part's do",
			trying.name, trying.part, lost)
	}
}

// Turning is the body turning once a day, its axis leaning off its
// orbit's north. A negative day turns it backward. A tilt that is not a
// finite number is held, and Place refuses it.
func (body Body) Turning(day time.Duration, tilt measure.Angle) Body {
	body.day, body.tilt = day, tilt.InRadians()
	if body.complaint == nil && !finite(body.tilt) {
		body.complaint = fmt.Errorf("the tilt given to Turning is %v radians: a "+
			"finite number is expected", body.tilt)
	}
	return body
}

// Tilting is the body with its axis leaning off its orbit's north by a
// tilt and its day as it was. A tilt that is not a finite number is
// held, and Place refuses it.
func (body Body) Tilting(tilt measure.Angle) Body {
	return body.Turning(body.day, tilt)
}

// Resized is the body at another radius, a look a shot chooses: the
// mass and every part stay as they were. Place refuses a radius that
// is no size.
func (body Body) Resized(radius Span) Body {
	body.radius = radius.InKilometres()
	return body
}

// Spinning is a hole's spin: its angular momentum per unit mass as a
// share of its mass, negative for a hole turning the other way. Spin is
// signed, so it is a plain number and not a maths.Portion. Nothing
// is checked here: Place and Binary refuse a hole whose spin is, in
// magnitude, one or past, which has no horizon.
func (body Body) Spinning(spin float64) Body {
	body.spin = spin
	return body
}

// Seeded grows the noise of the parts that have no seed of their own
// from this number, so two bodies do not share one noise.
func (body Body) Seeded(seed uint64) Body {
	body.seed = seed
	return body
}

// Glowing is what a star's surface gives off, and what it gives off
// and passes outside the visible.
func (body Body) Glowing(glow maths.Light, radiant thing.Radiant) Body {
	hold(&body.complaint, lightComplaint("Glowing's glow", glow))
	if radiant != nil {
		hold(&body.complaint, radiantComplaint("Glowing's radiant", radiant))
	}
	body.glow, body.radiant = glow, radiant
	return body
}

// With is the body wearing more parts: how a shot adjusts a template
// without rebuilding it. It refuses what NewBody refuses.
func (body Body) With(parts ...Part) (Body, error) {
	// the slices are copied before anything is hung on them, so no
	// other body made from this one shares what is hung here
	body.moons = append([]Moon(nil), body.moons...)
	body.tails = append([]Tail(nil), body.tails...)
	body.plumes = append([]Plume(nil), body.plumes...)
	if err := body.wear(parts); err != nil {
		return Body{}, err
	}
	return body, nil
}

// WithoutAurora is the body with its aurora taken off, as a set that
// says so asks.
func (body Body) WithoutAurora() Body {
	body.aurora = nil
	return body
}

// WithoutClouds is the body under a clear sky, its cloud layer taken
// off: a shot about the ground wants nothing over it.
func (body Body) WithoutClouds() Body {
	body.clouds = nil
	return body
}

// WithoutCities is the body dark at night, its lights taken off.
func (body Body) WithoutCities() Body {
	body.cities = nil
	return body
}

// WithoutJets is the body with its jets taken off.
func (body Body) WithoutJets() Body {
	body.jets = nil
	return body
}

// WithoutFlare is the star quiet, its fireball taken off.
func (body Body) WithoutFlare() Body {
	body.flare = nil
	return body
}

// WithoutMoons is the body alone, its moons taken off: a near shot
// that never sees them places less.
func (body Body) WithoutMoons() Body {
	body.moons = nil
	return body
}

// Name is the body's name.
func (body Body) Name() string { return body.name }

// Kind is what the body mostly is.
func (body Body) Kind() Kind { return body.kind }

// Radius is the body's radius in kilometres.
func (body Body) Radius() float64 { return body.radius }

// Mass is the body's mass in kilograms.
func (body Body) Mass() float64 { return body.mass }

// Day is the time the body turns once, zero for one that does not.
func (body Body) Day() time.Duration { return body.day }

// Tilt is the body's axis off its orbit's north, in radians.
func (body Body) Tilt() float64 { return body.tilt }

// Spin is a hole's spin, zero for schwarzschild.
func (body Body) Spin() float64 { return body.spin }

// Seed grows the noise of the parts that have no seed of their own.
func (body Body) Seed() uint64 { return body.seed }

// Glow is what a star's surface gives off.
func (body Body) Glow() maths.Light { return body.glow }

// Radiant is what the body gives off and passes outside the visible.
func (body Body) Radiant() thing.Radiant { return body.radiant }

// Ground is the rock's surface, when it has one.
func (body Body) Ground() (Ground, bool) {
	if body.ground == nil {
		return Ground{}, false
	}
	return *body.ground, true
}

// Gas is the giant's body, when it has one.
func (body Body) Gas() (Gas, bool) {
	if body.gas == nil {
		return Gas{}, false
	}
	return *body.gas, true
}

// Air is the body's atmosphere, when it has one.
func (body Body) Air() (Air, bool) {
	if body.air == nil {
		return Air{}, false
	}
	return *body.air, true
}

// Clouds is the body's cloud layer, when it has one.
func (body Body) Clouds() (Clouds, bool) {
	if body.clouds == nil {
		return Clouds{}, false
	}
	return *body.clouds, true
}

// Cities is the body's lights at night, when it has them.
func (body Body) Cities() (Cities, bool) {
	if body.cities == nil {
		return Cities{}, false
	}
	return *body.cities, true
}

// Aurora is the body's aurora, when it has one.
func (body Body) Aurora() (Aurora, bool) {
	if body.aurora == nil {
		return Aurora{}, false
	}
	return *body.aurora, true
}

// Rings is the body's rings, when it has them.
func (body Body) Rings() (Rings, bool) {
	if body.rings == nil {
		return Rings{}, false
	}
	return *body.rings, true
}

// Disc is the hole's glowing disc, when it has one.
func (body Body) Disc() (Disc, bool) {
	if body.disc == nil {
		return Disc{}, false
	}
	return *body.disc, true
}

// Jets is the body's jets, when it has them.
func (body Body) Jets() (Jets, bool) {
	if body.jets == nil {
		return Jets{}, false
	}
	return *body.jets, true
}

// Flare is the star's fireball, when it has one.
func (body Body) Flare() (Flare, bool) {
	if body.flare == nil {
		return Flare{}, false
	}
	return *body.flare, true
}

// Edge is the dressing on a hole's horizon.
func (body Body) Edge() Horizon { return body.edge }

// Coma is the comet's head, when it has one.
func (body Body) Coma() (Coma, bool) {
	if body.coma == nil {
		return Coma{}, false
	}
	return *body.coma, true
}

// Tails is the body's tails, its own copy.
func (body Body) Tails() []Tail {
	return append([]Tail(nil), body.tails...)
}

// Plumes is the body's plumes, its own copy.
func (body Body) Plumes() []Plume {
	return append([]Plume(nil), body.plumes...)
}

// Moons is the body's moons, its own copy.
func (body Body) Moons() []Moon {
	return append([]Moon(nil), body.moons...)
}

// Aimed is the body with every tail pointed away from a sun at a point
// in the world, bending back along the way the body moves. A vector that
// is not finite is held, and Place refuses it. The body it
// is called on keeps its tails as they were.
func (body Body) Aimed(sun, trail maths.Vec) Body {
	sunComplaint := vecComplaint("Aimed's sun", sun)
	trailComplaint := vecComplaint("Aimed's trail", trail)
	hold(&body.complaint, firstComplaint(sunComplaint, trailComplaint))
	body.tails = append([]Tail(nil), body.tails...)
	for place := range body.tails {
		body.tails[place].sun = sun
		body.tails[place].trail = trail
	}
	return body
}
