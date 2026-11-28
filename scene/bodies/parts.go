package bodies

import (
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// The parts a body wears, each behind its own constructor: the
// constructor checks what can be checked without a scale, and Place's
// sound check keeps the rest. A part is handed to NewBody, which hangs
// it; the readers answer what the constructors took, in the units
// bodies speak - kilometres, radians, shares.

// The numbers the parts' ground is made with, each picked.
var (
	nightGlowDefault = pick("night-glow-default")
	coastReach       = pick("coast-reach")
	radarRungs       = int(pick("radar-rungs"))
	radarVary        = pick("radar-vary")
	radarMix         = pick("radar-mix")
	bareGrey         = pick("bare-grey")
)

// Part is anything NewBody can hang on a body.
type Part interface {
	hang(body *Body)
	// refusal is why the part cannot be worn, or nil: it did not come
	// from its constructor, or a method was given a bad argument.
	refusal() error
}

// Shell is a band above a surface, from a low edge to a high one. Made
// inline, it carries any complaint privately; the constructor that
// receives it surfaces the complaint.
type Shell struct {
	made      bool    // set only by the constructor
	low, high float64 // kilometres
	complaint error
}

// NewShell is the band from low to high above the surface. A high edge
// not above the low one, or a low one below the ground, is no shell.
func NewShell(low, high Span) Shell {
	bottom, top := low.InKilometres(), high.InKilometres()
	if !finite(bottom) || !finite(top) || !(top > bottom) ||
		bottom < nought {
		return Shell{made: true, complaint: fmt.Errorf("a shell from "+
			"%v to %v km is no band: the high edge is above the "+
			"low one, and the low one is above the ground, both finite",
			bottom, top)}
	}
	return Shell{made: true, low: bottom, high: top}
}

// problem is why the shell is no band, without the package prefix, or
// nil.
func (shell Shell) problem() error {
	if shell.complaint != nil {
		return shell.complaint
	}
	if !shell.made {
		return errors.New("a Shell that was not made by NewShell is " +
			"no band")
	}
	return nil
}

// Low is the shell's lower edge above the surface, in kilometres.
func (shell Shell) Low() float64 { return shell.low }

// High is the shell's upper edge above the surface, in kilometres.
func (shell Shell) High() float64 { return shell.high }

// Coma is a comet's head: a cloud of gas and dust round the nucleus,
// scattering the sun in its colour and giving off a glow of its own,
// the green of carbon.
type Coma struct {
	made   bool    // set only by the constructor
	radius float64 // kilometres
	dense  float64 // extinction per kilometre at its heart
	colour maths.Light
	glow   maths.Light
}

// NewComa is a comet's head of a radius, losing light at its heart as
// the loss says, scattering the sun in a colour and glowing its own.
func NewComa(radius Span, loss Loss, colour,
	glow maths.Light) (Coma, error) {
	around := radius.InKilometres()
	if !positiveFinite(around) {
		return Coma{}, fmt.Errorf("bodies: a coma of %v km is no "+
			"cloud: kilometres, above zero", around)
	}
	dense, err := loss.problem()
	if err != nil {
		return Coma{}, fmt.Errorf("bodies: a coma's %w", err)
	}
	if err := firstComplaint(lightComplaint("the coma's colour", colour),
		lightComplaint("the coma's glow", glow)); err != nil {
		return Coma{}, fmt.Errorf("bodies: %w", err)
	}
	return Coma{made: true, radius: around, dense: dense, colour: colour,
		glow: glow}, nil
}

// hang is the part taking its place on the body.
func (coma Coma) hang(body *Body) { body.coma = &coma }

// Radius is the coma's radius, in kilometres.
func (coma Coma) Radius() float64 { return coma.radius }

// Dense is the extinction at the coma's heart, per kilometre.
func (coma Coma) Dense() float64 { return coma.dense }

// Colour is what the coma scatters, a Light.
func (coma Coma) Colour() maths.Light { return coma.colour }

// Glow is what the coma gives off, a Light.
func (coma Coma) Glow() maths.Light { return coma.glow }

// Tail is a stream from the body away from its sun, bending back along
// the body's trail. Aimed sets where the sun is and which way the body
// moves, once, where the body is at time zero: the tail is laid in the
// body's own turning, tilted space, so on a body that turns or leans it
// streams away from the sun only at zero and turns with the body after,
// kept as it was, on purpose.
type Tail struct {
	made         bool    // set only by the constructor
	length, base float64 // kilometres
	spread       float64
	curve        float64
	dense        float64 // extinction per kilometre at its heart
	colour, glow maths.Light
	radiant      thing.Radiant
	wisp         float64 // how ragged, a share; zero smooth
	seed         uint64
	sun, trail   maths.Vec
	complaint    error // the first bad argument a method was given
}

// NewTail is a stream of a length from a base width, spreading and
// curving as medium.Tail takes them, losing light at its heart as the
// loss says, scattering the sun in a colour and glowing its own.
func NewTail(length, base Span, spread, curve float64,
	loss Loss, colour, glow maths.Light) (Tail, error) {
	long, wide := length.InKilometres(), base.InKilometres()
	if !positiveFinite(long) || !positiveFinite(wide) {
		return Tail{}, fmt.Errorf("bodies: a tail %v km long from "+
			"a base %v km wide is no stream: both above zero, finite",
			long, wide)
	}
	dense, err := loss.problem()
	if err != nil {
		return Tail{}, fmt.Errorf("bodies: a tail's %w", err)
	}
	if err := firstComplaint(numberComplaint("a tail's spread", spread),
		numberComplaint("a tail's curve", curve),
		lightComplaint("the tail's colour", colour),
		lightComplaint("the tail's glow", glow)); err != nil {
		return Tail{}, fmt.Errorf("bodies: %w", err)
	}
	return Tail{made: true, length: long, base: wide, spread: spread, curve: curve,
		dense: dense, colour: colour, glow: glow}, nil
}

// Wisped is the tail ragged by a share, its wisps grown from a seed.
func (tail Tail) Wisped(wisp maths.Portion, seed uint64) (Tail, error) {
	ragged, err := wisp.Share()
	if err != nil {
		return tail, fmt.Errorf("bodies: a tail's wisp: %w", err)
	}
	tail.wisp, tail.seed = ragged, seed
	return tail, nil
}

// Radiating is the tail with what it gives off outside the visible.
func (tail Tail) Radiating(radiant thing.Radiant) Tail {
	hold(&tail.complaint, radiantComplaint("Tail.Radiating's radiant", radiant))
	tail.radiant = radiant
	return tail
}

// hang is the part taking its place on the body.
func (tail Tail) hang(body *Body) {
	body.tails = append(body.tails, tail)
}

// Length is the tail's length, in kilometres.
func (tail Tail) Length() float64 { return tail.length }

// Base is the tail's width at the base, in kilometres.
func (tail Tail) Base() float64 { return tail.base }

// Spread is how much the tail widens, as a share of the distance out,
// as medium.Tail takes it.
func (tail Tail) Spread() float64 { return tail.spread }

// Curve is the bend back along the trail, as a share of the distance
// squared over the length, as medium.Tail takes it.
func (tail Tail) Curve() float64 { return tail.curve }

// Dense is the extinction at the tail's heart, per kilometre.
func (tail Tail) Dense() float64 { return tail.dense }

// Colour is what the tail scatters, a Light.
func (tail Tail) Colour() maths.Light { return tail.colour }

// Glow is what the tail gives off, a Light.
func (tail Tail) Glow() maths.Light { return tail.glow }

// Radiant is what the tail gives off and passes outside the visible, as
// thing.Radiant says.
func (tail Tail) Radiant() thing.Radiant { return tail.radiant }

// Wisp is how ragged the tail is, a share from zero, smooth, to one.
func (tail Tail) Wisp() float64 { return tail.wisp }

// Seed grows the tail's noise; it is a number with no unit.
func (tail Tail) Seed() uint64 { return tail.seed }

// Sun is where Aimed said the sun stands, a point in the world's units.
func (tail Tail) Sun() maths.Vec { return tail.sun }

// Trail is which way Aimed said the body moves, a direction in the
// world.
func (tail Tail) Trail() maths.Vec { return tail.trail }

// Clouds is a world's cloud layer in a shell, covering a share of the
// sky, with its weather grown from a seed.
type Clouds struct {
	made      bool    // set only by the constructor
	low, high float64 // kilometres
	cover     float64 // share of the sky clouded
	dense     float64 // extinction per kilometre at a patch's heart
	seed      uint64
	turning   bool // use latitude-based weather even on a locked body
	soft      bool // leave the fine billows unturned by storms
	turn      float64
	hasTurn   bool
}

// NewClouds is a cloud layer filling a shell, covering a share of the
// sky, losing light at a patch's heart as the loss says, its weather
// grown from a seed.
func NewClouds(shell Shell, cover maths.Portion, loss Loss,
	seed uint64) (Clouds, error) {
	if err := shell.problem(); err != nil {
		return Clouds{}, fmt.Errorf("bodies: clouds: %w", err)
	}
	covered, err := cover.Share()
	if err != nil {
		return Clouds{}, fmt.Errorf("bodies: clouds' cover: %w", err)
	}
	dense, err := loss.problem()
	if err != nil {
		return Clouds{}, fmt.Errorf("bodies: clouds' %w", err)
	}
	return Clouds{made: true, low: shell.low, high: shell.high, cover: covered,
		dense: dense, seed: seed}, nil
}

// hang is the part taking its place on the body.
func (clouds Clouds) hang(body *Body) { body.clouds = &clouds }

// TurningWeather chooses latitude-based weather for this shell, even when
// the body keeps one face toward its star. It does not alter body motion.
func (clouds Clouds) TurningWeather() Clouds {
	clouds.turning = true
	return clouds
}

// SoftWeather keeps fine billows unturned by storms in turning weather.
// It chooses an artistic pattern; the broad clouds and storms stay.
func (clouds Clouds) SoftWeather() Clouds {
	clouds.turning, clouds.soft = true, true
	return clouds
}

// StormTurn sets the authored storm turn in radians at a storm's heart.
func (clouds Clouds) StormTurn(turn float64) Clouds {
	clouds.turn, clouds.hasTurn = turn, true
	return clouds
}

// WeatherChoice reports the authored pattern and optional storm turn.
func (clouds Clouds) WeatherChoice() (turning, soft bool, turn float64,
	hasTurn bool) {
	return clouds.turning, clouds.soft, clouds.turn, clouds.hasTurn
}

// Low is the layer's lower edge above the surface, in kilometres.
func (clouds Clouds) Low() float64 { return clouds.low }

// High is the layer's upper edge above the surface, in kilometres.
func (clouds Clouds) High() float64 { return clouds.high }

// Cover is the share of the sky clouded, from zero to one.
func (clouds Clouds) Cover() float64 { return clouds.cover }

// Dense is the extinction at a patch's heart, per kilometre.
func (clouds Clouds) Dense() float64 { return clouds.dense }

// Seed grows the weather's noise; it is a number with no unit.
func (clouds Clouds) Seed() uint64 { return clouds.seed }

// AuroraColours are an aurora's colours with height. The earth's are
// purple, green and red; a giant's blues stand in for the ultraviolet
// it really shines in.
type AuroraColours struct {
	fringe, bottom, top maths.Light
}

// NewAuroraColours is the fringe, the bottom and the top of the
// curtain.
func NewAuroraColours(fringe, bottom, top maths.Light) AuroraColours {
	return AuroraColours{fringe: fringe, bottom: bottom, top: top}
}

// Aurora is a glow in a shell, in an oval round each magnetic pole,
// the magnetic axis leaning off the spin.
type Aurora struct {
	made       bool    // set only by the constructor
	low, high  float64 // kilometres
	lat, width float64 // radians: the oval, from the pole's equator
	tilt       float64 // radians: the magnetic axis off the spin
	strength   float64 // the heart's glow as a share of the sun's
	curtains   float64 // the dressing; zero for a smooth band
	seed       uint64
	colours    AuroraColours
	radiant    thing.Radiant
	complaint  error // the first bad argument a method was given
}

// NewAurora is an aurora filling a shell, its oval at a latitude and
// width, the magnetic axis tilted off the spin, glowing a share of the
// sun's light at its heart - a night shot exposes for it - dressed in
// curtains grown from a seed, in its colours.
func NewAurora(shell Shell, lat, width, tilt measure.Angle,
	strength, curtains maths.Portion, seed uint64,
	colours AuroraColours) (Aurora, error) {
	if err := shell.problem(); err != nil {
		return Aurora{}, fmt.Errorf("bodies: an aurora: %w", err)
	}
	latitude, breadth, lean := lat.InRadians(), width.InRadians(),
		tilt.InRadians()
	if err := firstComplaint(numberComplaint("an aurora's latitude",
		latitude), numberComplaint("an aurora's width", breadth),
		numberComplaint("an aurora's tilt", lean),
		lightComplaint("the aurora's fringe", colours.fringe),
		lightComplaint("the aurora's bottom", colours.bottom),
		lightComplaint("the aurora's top", colours.top)); err != nil {
		return Aurora{}, fmt.Errorf("bodies: %w", err)
	}
	strong, err := strength.Share()
	if err != nil {
		return Aurora{}, fmt.Errorf("bodies: an aurora's "+
			"strength: %w", err)
	}
	dressed, err := curtains.Share()
	if err != nil {
		return Aurora{}, fmt.Errorf("bodies: an aurora's "+
			"curtains: %w", err)
	}
	return Aurora{made: true, low: shell.low, high: shell.high,
		lat: latitude, width: breadth,
		tilt: lean, strength: strong, curtains: dressed,
		seed: seed, colours: colours}, nil
}

// Radiating is the aurora with what it gives off outside the visible:
// an aurora driven by a magnetar is x-ray bright.
func (aurora Aurora) Radiating(radiant thing.Radiant) Aurora {
	hold(&aurora.complaint,
		radiantComplaint("Aurora.Radiating's radiant", radiant))
	aurora.radiant = radiant
	return aurora
}

// hang is the part taking its place on the body.
func (aurora Aurora) hang(body *Body) { body.aurora = &aurora }

// Low is the aurora's lower edge above the surface, in kilometres.
func (aurora Aurora) Low() float64 { return aurora.low }

// High is the aurora's upper edge above the surface, in kilometres.
func (aurora Aurora) High() float64 { return aurora.high }

// Lat is the oval's latitude from the pole's equator, in radians.
func (aurora Aurora) Lat() float64 { return aurora.lat }

// Width is the oval's width, in radians.
func (aurora Aurora) Width() float64 { return aurora.width }

// Tilt is the magnetic axis's lean off the spin, in radians.
func (aurora Aurora) Tilt() float64 { return aurora.tilt }

// Strength is the heart's glow as a share of the sun's light.
func (aurora Aurora) Strength() float64 { return aurora.strength }

// Curtains is the dressing, a share from zero, a smooth band, to one.
func (aurora Aurora) Curtains() float64 { return aurora.curtains }

// Seed grows the weather's noise; it is a number with no unit.
func (aurora Aurora) Seed() uint64 { return aurora.seed }

// Colours are the curtain's colours with height, each a Light.
func (aurora Aurora) Colours() AuroraColours { return aurora.colours }

// Radiant is what the aurora gives off and passes outside the visible,
// as thing.Radiant says.
func (aurora Aurora) Radiant() thing.Radiant { return aurora.radiant }

// Fringe is the curtain's fringe colour, a Light.
func (colours AuroraColours) Fringe() maths.Light { return colours.fringe }

// Bottom is the curtain's bottom colour, a Light.
func (colours AuroraColours) Bottom() maths.Light { return colours.bottom }

// Top is the curtain's top colour, a Light.
func (colours AuroraColours) Top() maths.Light { return colours.top }

// Cities is a world's lights at night, laid as the black marble lays
// the earth's: more on coasts and in the middle latitudes.
type Cities struct {
	made   bool    // set only by the constructor
	rungs  uint    // how many levels below the ground's tiles
	towns  float64 // share of the land tiles lit as towns
	metros float64 // share of the tiles two rungs up lit as metros
	glow   maths.Light
	seed   uint64
}

// NewCities lights a share of the land tiles as towns, rungs below the
// ground's own tiles, and a share of the tiles two rungs up as metros,
// each glowing, their places grown from a seed.
func NewCities(rungs uint, towns, metros maths.Portion, glow maths.Light,
	seed uint64) (Cities, error) {
	if err := rungsComplaint("the cities", rungs); err != nil {
		return Cities{}, err
	}
	if err := lightComplaint("the cities' glow", glow); err != nil {
		return Cities{}, fmt.Errorf("bodies: %w", err)
	}
	lit, err := towns.Share()
	if err != nil {
		return Cities{}, fmt.Errorf("bodies: the towns' share: %w",
			err)
	}
	big, err := metros.Share()
	if err != nil {
		return Cities{}, fmt.Errorf("bodies: the metros' share: %w",
			err)
	}
	return Cities{made: true, rungs: rungs, towns: lit, metros: big, glow: glow,
		seed: seed}, nil
}

// hang is the part taking its place on the body.
func (cities Cities) hang(body *Body) { body.cities = &cities }

// Rungs is how many levels below the ground's tiles the lights are
// laid, a count.
func (cities Cities) Rungs() int { return int(cities.rungs) }

// Share is the share of the land tiles lit as towns, from zero to one.
func (cities Cities) Share() float64 { return cities.towns }

// Metro is the share of the tiles two rungs up lit as metros, from zero
// to one.
func (cities Cities) Metro() float64 { return cities.metros }

// Glow is what the lights give off, a Light.
func (cities Cities) Glow() maths.Light { return cities.glow }

// Seed grows the places' noise; it is a number with no unit.
func (cities Cities) Seed() uint64 { return cities.seed }

// Jets is a pair of jets along the body's axis, leaning off it, with
// the knots riding out at the jets' own speed.
type Jets struct {
	made         bool    // set only by the constructor
	length, base float64 // kilometres
	spread       float64
	from         float64 // kilometres
	dense        float64 // extinction per kilometre at the base
	tilt         float64 // radians off the body's axis
	speed        float64 // share of the speed of light
	knots        float64 // the dressing; zero smooth
	glow         maths.Light
	radiant      thing.Radiant
	complaint    error // the first bad argument a method was given
}

// NewJets is a pair of jets of a length from a base width, spreading
// as medium.Jet takes it, born a distance out, losing light at the
// base as the loss says, leaning off the body's axis, moving at a
// share of the speed of light - which beams their light - and glowing.
func NewJets(length, base Span, spread float64,
	from Span, loss Loss, tilt measure.Angle,
	speed maths.Portion, glow maths.Light) (Jets, error) {
	long, wide := length.InKilometres(), base.InKilometres()
	if !positiveFinite(long) || !positiveFinite(wide) {
		return Jets{}, fmt.Errorf("bodies: jets %v km long from a "+
			"base %v km wide are no pair: both above zero, finite",
			long, wide)
	}
	dense, err := loss.problem()
	if err != nil {
		return Jets{}, fmt.Errorf("bodies: the jets' %w", err)
	}
	fast, err := speed.Share()
	if err != nil {
		return Jets{}, fmt.Errorf("bodies: the jets' speed: %w", err)
	}
	away := from.InKilometres()
	if !nonNegativeFinite(away) {
		return Jets{}, fmt.Errorf("bodies: jets born %v km out are "+
			"no distance: kilometres, zero or above, finite", away)
	}
	lean := tilt.InRadians()
	if err := firstComplaint(numberComplaint("the jets' spread", spread),
		numberComplaint("the jets' tilt", lean),
		lightComplaint("the jets' glow", glow)); err != nil {
		return Jets{}, fmt.Errorf("bodies: %w", err)
	}
	if flow := jetFlow(fast, wide); !finite(flow) {
		return Jets{}, fmt.Errorf("bodies: jets at %v of the speed of "+
			"light from a base %v km wide flow at %v base widths a "+
			"second: the flow is finite", fast, wide, flow)
	}
	return Jets{made: true, length: long, base: wide, spread: spread,
		from: away, dense: dense,
		tilt: lean, speed: fast, glow: glow}, nil
}

// Knotted is the jets dressed in lumps, as medium.Jet takes them: a
// share from zero, smooth, to one. A number outside that is held, and
// Place refuses it.
func (jets Jets) Knotted(knots float64) Jets {
	if !(knots >= nought && knots <= unity) {
		hold(&jets.complaint, fmt.Errorf("Jets.Knotted's knots is %v: "+
			"a share from zero to one is expected", knots))
	}
	jets.knots = knots
	return jets
}

// Radiating is the jets with what they give off outside the visible.
func (jets Jets) Radiating(radiant thing.Radiant) Jets {
	hold(&jets.complaint, radiantComplaint("Jets.Radiating's radiant", radiant))
	jets.radiant = radiant
	return jets
}

// hang is the part taking its place on the body.
func (jets Jets) hang(body *Body) { body.jets = &jets }

// Length is the jets' length, in kilometres.
func (jets Jets) Length() float64 { return jets.length }

// Base is the jets' width at the base, in kilometres.
func (jets Jets) Base() float64 { return jets.base }

// Spread is how much the jets widen, as a share of the distance out, as
// medium.Jet takes it.
func (jets Jets) Spread() float64 { return jets.spread }

// From is how far out the jets are born, in kilometres.
func (jets Jets) From() float64 { return jets.from }

// Dense is the extinction at the jets' base, per kilometre.
func (jets Jets) Dense() float64 { return jets.dense }

// Tilt is the jets' lean off the body's axis, in radians.
func (jets Jets) Tilt() float64 { return jets.tilt }

// Speed is the jets' speed as a share of the speed of light.
func (jets Jets) Speed() float64 { return jets.speed }

// Knots is the dressing, a share from zero, smooth, to one, all knots.
func (jets Jets) Knots() float64 { return jets.knots }

// Glow is what the jets give off, a Light.
func (jets Jets) Glow() maths.Light { return jets.glow }

// Radiant is what the jets give off and pass outside the visible, as
// thing.Radiant says.
func (jets Jets) Radiant() thing.Radiant { return jets.radiant }

// Flare is a fireball from a star's surface: born at a moment, growing
// at a speed, with an optical depth across its own radius however big
// it has grown. Before its moment there is nothing.
type Flare struct {
	made      bool // set only by the constructor
	at        time.Duration
	speed     float64 // kilometres a second
	opacity   float64 // optical depth across its radius, no unit
	glow      maths.Light
	radiant   thing.Radiant
	complaint error // the first bad argument a method was given
}

// NewFlare is a fireball born at a moment, growing at kilometres a
// second, of an optical depth across its own radius, glowing.
func NewFlare(at time.Duration, kmPerSecond, opacity float64,
	glow maths.Light) (Flare, error) {
	if !positiveFinite(kmPerSecond) {
		return Flare{}, fmt.Errorf("bodies: a flare growing at %v "+
			"km a second never shows: above zero", kmPerSecond)
	}
	if !nonNegativeFinite(opacity) {
		return Flare{}, fmt.Errorf("bodies: a flare of opacity %v "+
			"is no fireball: zero or above, finite", opacity)
	}
	if err := lightComplaint("the flare's glow", glow); err != nil {
		return Flare{}, fmt.Errorf("bodies: %w", err)
	}
	return Flare{made: true, at: at, speed: kmPerSecond, opacity: opacity,
		glow: glow}, nil
}

// Radiating is the flare with what it gives off outside the visible.
func (flare Flare) Radiating(radiant thing.Radiant) Flare {
	hold(&flare.complaint, radiantComplaint("Flare.Radiating's radiant", radiant))
	flare.radiant = radiant
	return flare
}

// hang is the part taking its place on the body.
func (flare Flare) hang(body *Body) { body.flare = &flare }

// At is the moment the fireball is born, as a duration.
func (flare Flare) At() time.Duration { return flare.at }

// Speed is how fast the fireball grows, in kilometres a second.
func (flare Flare) Speed() float64 { return flare.speed }

// Opacity is the optical depth across the fireball's radius, with no
// unit.
func (flare Flare) Opacity() float64 { return flare.opacity }

// Glow is what the fireball gives off, a Light.
func (flare Flare) Glow() maths.Light { return flare.glow }

// Radiant is what the fireball gives off and passes outside the
// visible, as thing.Radiant says.
func (flare Flare) Radiant() thing.Radiant { return flare.radiant }

// Plume is a volcanic plume: a column of gas and dust rising from a
// point on the ground and falling back in an umbrella, as io's do.
type Plume struct {
	made          bool      // set only by the constructor
	toward        maths.Vec // the point on the ground, from the centre
	height, width float64   // kilometres
	dense         float64   // extinction per kilometre at the heart
	colour        maths.Light
}

// NewPlume is a column rising toward a direction from the centre, of a
// height and an umbrella's width, losing light at its heart as the
// loss says, in a colour.
func NewPlume(toward maths.Vec, height, width Span,
	loss Loss, colour maths.Light) (Plume, error) {
	tall, wide := height.InKilometres(), width.InKilometres()
	if !finite(toward.X) || !finite(toward.Y) || !finite(toward.Z) ||
		toward == (maths.Vec{}) {
		return Plume{}, fmt.Errorf("bodies: a plume toward %v has no "+
			"point on the ground to rise from: a direction of finite "+
			"parts, not all zero", toward)
	}
	if !positiveFinite(tall) || !positiveFinite(wide) {
		return Plume{}, fmt.Errorf("bodies: a plume %v km tall and "+
			"%v km wide is no column: both above zero, finite", tall,
			wide)
	}
	dense, err := loss.problem()
	if err != nil {
		return Plume{}, fmt.Errorf("bodies: a plume's %w", err)
	}
	if err := lightComplaint("the plume's colour", colour); err != nil {
		return Plume{}, fmt.Errorf("bodies: %w", err)
	}
	return Plume{made: true, toward: toward, height: tall, width: wide,
		dense: dense, colour: colour}, nil
}

// hang is the part taking its place on the body.
func (plume Plume) hang(body *Body) {
	body.plumes = append(body.plumes, plume)
}

// Toward is the direction from the body's centre to the point the plume
// rises from; it has no unit.
func (plume Plume) Toward() maths.Vec { return plume.toward }

// Height is the column's height, in kilometres.
func (plume Plume) Height() float64 { return plume.height }

// Width is the umbrella's width, in kilometres.
func (plume Plume) Width() float64 { return plume.width }

// Dense is the extinction at the plume's heart, per kilometre.
func (plume Plume) Dense() float64 { return plume.dense }

// Colour is what the plume scatters, a Light.
func (plume Plume) Colour() maths.Light { return plume.colour }

// Fineness is detail painted finer than the ground's own hex tiles,
// for an eye near it. Each fine tile takes the land class of the
// coarse tile it sits in, its brightness varied and a neighbouring
// class let in, both by shares.
type Fineness struct {
	rungs  uint // levels finer; zero lets the picture choose
	vary   float64
	mix    float64
	ridges float64
}

// NewFineness paints the ground rungs levels below its own tiles,
// varying each fine tile's brightness by one share and letting a
// neighbouring class in by another. Zero rungs lets the picture choose
// from the angle a pixel covers; world.Fine says how.
func NewFineness(rungs uint, vary, mix maths.Portion) (Fineness, error) {
	if err := rungsComplaint("fineness", rungs); err != nil {
		return Fineness{}, err
	}
	varied, err := vary.Share()
	if err != nil {
		return Fineness{}, fmt.Errorf("bodies: fineness vary: %w",
			err)
	}
	mixed, err := mix.Share()
	if err != nil {
		return Fineness{}, fmt.Errorf("bodies: fineness mix: %w", err)
	}
	return Fineness{rungs: rungs, vary: varied, mix: mixed, ridges: 1}, nil
}

// Ridges sets an artistic multiplier on the painted ridge contribution.
func (fine Fineness) Ridges(share maths.Portion) (Fineness, error) {
	value, err := share.Share()
	if err != nil {
		return Fineness{}, fmt.Errorf("bodies: ridge contrast: %w", err)
	}
	// Twice the existing contrast at the top of the authored range is an
	// artistic choice. One half retains the original strength.
	fine.ridges = 2 * value
	return fine, nil
}

// Rungs is how many levels finer than the ground's tiles to paint, a
// count.
func (fine Fineness) Rungs() int { return int(fine.rungs) }

// Vary is the share by which each fine tile's brightness varies, from
// zero to one.
func (fine Fineness) Vary() float64 { return fine.vary }

// Mix is the share by which a neighbouring class is let in, from zero
// to one.
func (fine Fineness) Mix() float64 { return fine.mix }

// Ground is a rock world's surface: how its terrain is made, how it is
// painted, and what else it wears, each hung by a method.
type Ground struct {
	made    bool // set only by the constructor
	terrain world.Terrain
	look    world.Look
	// features places the features on a plain world, by asking it
	// where its land and water are; it refuses, with the reason, when
	// a feature has nowhere to go.
	features func(plain *world.World) ([]world.Feature, error)
	radiant  thing.Radiant
	// complaint is the first bad argument a method was given.
	complaint error
	rough     *world.Look
	fine      *Fineness
	// day and lights are pictures wrapped on the sphere in place of
	// the painted world and the cities, as nasa publishes the blue
	// and the black marble; turn moves their left edge round the
	// sphere; glow scales the lights against the sun's, zero read as
	// four thousandths.
	day, lights image.Image
	turn        float64 // radians
	glow        float64
}

// NewGround is a surface made from a terrain and painted in a look. It
// refuses a terrain or look holding a number that is not finite.
// The rest of what a ground wears - fineness, a radar look, wrapped
// maps, features, a radiant - hangs by its methods.
func NewGround(terrain world.Terrain, look world.Look) (Ground, error) {
	if err := firstComplaint(
		numberComplaint("the terrain's ocean", terrain.Ocean),
		numberComplaint("the terrain's cold", terrain.Cold),
		numberComplaint("the terrain's axis x", terrain.Axis.X),
		numberComplaint("the terrain's axis y", terrain.Axis.Y),
		numberComplaint("the terrain's axis z", terrain.Axis.Z),
		numberComplaint("the look's lines", look.Lines)); err != nil {
		return Ground{}, fmt.Errorf("bodies: %w", err)
	}
	for biome, paint := range look.Paints {
		err := lightComplaint(fmt.Sprintf("the look's paint %d", biome),
			paint)
		if err != nil {
			return Ground{}, fmt.Errorf("bodies: %w", err)
		}
	}
	return Ground{made: true, terrain: terrain, look: look}, nil
}

// lookComplaint is the complaint, without the package prefix, for a look
// holding a number that is not finite, or nil when it holds none.
func lookComplaint(what string, look world.Look) error {
	for biome, paint := range look.Paints {
		if err := lightComplaint(fmt.Sprintf("%s paint %d", what, biome),
			paint); err != nil {
			return err
		}
	}
	return numberComplaint(what+" lines", look.Lines)
}

// Fined is the ground painted finer than its own tiles.
func (ground Ground) Fined(fine Fineness) Ground {
	ground.fine = &fine
	return ground
}

// RadarRough is how rough each class is for a radar, as a look whose
// paints are greys from zero to one.
func (ground Ground) RadarRough(rough world.Look) Ground {
	hold(&ground.complaint, lookComplaint("Ground.RadarRough's look", rough))
	ground.rough = &rough
	return ground
}

// Mapped wraps an equirectangular picture of the ground by day round
// the sphere in place of the painted world, its left edge turned.
func (ground Ground) Mapped(day image.Image, turn measure.Angle) Ground {
	radians := turn.InRadians()
	hold(&ground.complaint, numberComplaint("Ground.Mapped's turn", radians))
	ground.day, ground.turn = day, radians
	return ground
}

// NightLit wraps a picture of the ground's lights at night, scaled by
// a share of the sun's light, which a night shot exposes for.
func (ground Ground) NightLit(lights image.Image,
	glow maths.Portion) (Ground, error) {
	shining, err := glow.Share()
	if err != nil {
		return ground, fmt.Errorf("bodies: the night lights' "+
			"glow: %w", err)
	}
	ground.lights, ground.glow = lights, shining
	return ground, nil
}

// Featured places features on the plain world, by asking it where its
// land and water are.
func (ground Ground) Featured(
	features func(plain *world.World) ([]world.Feature, error)) Ground {
	ground.features = features
	return ground
}

// Radiating is the ground with what it does outside the visible.
func (ground Ground) Radiating(radiant thing.Radiant) Ground {
	hold(&ground.complaint,
		radiantComplaint("Ground.Radiating's radiant", radiant))
	ground.radiant = radiant
	return ground
}

// hang is the part taking its place on the body.
func (ground Ground) hang(body *Body) { body.ground = &ground }

// Terrain is how the ground is made.
func (ground Ground) Terrain() world.Terrain { return ground.terrain }

// Look is how the ground is painted.
func (ground Ground) Look() world.Look { return ground.look }

// Radiant is what the ground gives off and passes outside the visible,
// as thing.Radiant says.
func (ground Ground) Radiant() thing.Radiant { return ground.radiant }

// Turn is how far the wrapped pictures' left edge is moved round the
// sphere, in radians.
func (ground Ground) Turn() float64 { return ground.turn }

// NightGlow scales the lights against the sun's, as a share of the
// sun's light; zero reads as four thousandths.
func (ground Ground) NightGlow() float64 { return ground.glow }

// Map is the picture worn by day, when one is.
func (ground Ground) Map() image.Image { return ground.day }

// Lights is the picture of the lights at night, when one is.
func (ground Ground) Lights() image.Image { return ground.lights }

// Fine is the finer paint, and whether it is worn.
func (ground Ground) Fine() (Fineness, bool) {
	if ground.fine == nil {
		return Fineness{}, false
	}
	return *ground.fine, true
}

// Rough is the radar look, and whether it is worn.
func (ground Ground) Rough() (world.Look, bool) {
	if ground.rough == nil {
		return world.Look{}, false
	}
	return *ground.rough, true
}

// Features places the features on a plain world, when there are any.
func (ground Ground) Features() func(plain *world.World) ([]world.Feature,
	error) {
	return ground.features
}

// groundMaterial is the body's surface as a material: its world, made
// on a sphere split detail times, painted in its look. A body with no
// ground is a mid-grey ball. It refuses when the world cannot be made.
func (body Body) groundMaterial(detail int) (*thing.Material, error) {
	if body.ground == nil {
		return &thing.Material{Name: body.name,
			Colour: maths.Grey(bareGrey)}, nil
	}
	ground := body.ground
	made, err := body.World(detail)
	if err != nil {
		return nil, err
	}
	material := &thing.Material{Name: body.name, Radiant: ground.radiant}
	switch {
	case ground.day != nil:
		material.Texture = texture.Picture{Image: ground.day,
			Turn: ground.turn}
	case ground.fine != nil:
		fine := ground.fine
		if ground.look.Lines <= nought {
			// no lines to draw tiles by: read from the world's tree at
			// the point, so no tile's shape shows, whatever the rung
			material.Texture = made.ScroogedRidges(ground.look, fine.vary,
				fine.mix, fine.ridges)
			break
		}
		material.Texture = made.Fine(ground.look, int(fine.rungs),
			fine.vary, fine.mix)
	default:
		material.Texture = made.Painted(ground.look)
	}
	if ground.lights != nil {
		glow := ground.glow
		if glow == nought {
			glow = nightGlowDefault
		}
		material.Shine = texture.Picture{Image: ground.lights,
			Turn: ground.turn, Gain: glow}
	}
	if cities := body.cities; cities != nil && ground.lights == nil {
		material.Shine = body.cityShine(made, ground, cities)
	}
	if ground.rough != nil {
		// a radar map is finer than the paint, varied and mixed
		material.Rough = made.Fine(*ground.rough, radarRungs, radarVary,
			radarMix)
	}
	return material, nil
}

// cityShine lays the cities' lights on the ground: on the made world's
// own land and coasts, or on the map's when the ground wears one.
func (body Body) cityShine(made *world.World, ground *Ground,
	cities *Cities) texture.Cities {
	shine := texture.Cities{Tiling: made.Tiling, Rungs: int(cities.rungs),
		Share: cities.towns, Metro: cities.metros, Glow: cities.glow,
		Seed: cities.seed}
	if ground.day != nil {
		// the map says where the land and the water are
		reader := mapShore{picture: texture.Picture{Image: ground.day,
			Turn: ground.turn}}
		shine.Land, shine.Coast = reader.land, reader.coast
		return shine
	}
	reader := tileShore{made: made}
	shine.Land, shine.Coast = reader.land, reader.coast
	return shine
}

// tileShore says where the land and the coasts are from a made world's
// own tiles.
type tileShore struct {
	made *world.World
}

// land is whether the tile nearest a point is land.
func (shore tileShore) land(point maths.Vec) bool {
	nearest := shore.made.Tiling.Nearest(point)
	return world.Land(shore.made.Biome(nearest))
}

// coast is whether the tile nearest a point has water beside it.
func (shore tileShore) coast(point maths.Vec) bool {
	tile := shore.made.Tiling.Nearest(point)
	for _, neighbour := range shore.made.Tiling.Neighbours(tile) {
		if world.Water(shore.made.Biome(neighbour)) {
			return true
		}
	}
	return false
}

// mapShore says where the land and the coasts are from a wrapped map.
type mapShore struct {
	picture texture.Picture
}

// land is whether a point is not water on the map.
func (shore mapShore) land(point maths.Vec) bool {
	return !shore.picture.Water(point)
}

// coast is whether water lies within a few degrees of a point on the
// map, north, south, east or west of it.
func (shore mapShore) coast(point maths.Vec) bool {
	upright := maths.Upright(point)
	for _, way := range []maths.Vec{upright.X, upright.X.Neg(), upright.Z,
		upright.Z.Neg()} {
		beyond := point.Unit().Add(way.Scale(coastReach))
		if shore.picture.Water(beyond) {
			return true
		}
	}
	return false
}

// Gas is a giant's body: how far below the cloud tops it goes opaque,
// the optical depth from the tops to there, and what it looks like.
type Gas struct {
	made    bool    // set only by the constructor
	depth   float64 // share of the radius
	opacity float64 // optical depth, no unit
	bands   texture.Banded
}

// NewGas is a giant's body going opaque a share of the radius below
// its cloud tops, of an optical depth from the tops to there, looking
// as its bands say.
func NewGas(depth maths.Portion, opacity float64,
	bands texture.Banded) (Gas, error) {
	deep, err := depth.Share()
	if err != nil {
		return Gas{}, fmt.Errorf("bodies: the gas's depth: %w", err)
	}
	if !(deep > nought) {
		return Gas{}, fmt.Errorf("bodies: a gas of depth %v is no "+
			"body: a share of the radius, above zero", deep)
	}
	if !nonNegativeFinite(opacity) {
		return Gas{}, fmt.Errorf("bodies: a gas of opacity %v is "+
			"no body: zero or above, finite", opacity)
	}
	if err := bandsComplaint(bands); err != nil {
		return Gas{}, fmt.Errorf("bodies: the gas's %w", err)
	}
	return Gas{made: true, depth: deep, opacity: opacity,
		bands: copyBands(bands)}, nil
}

// hang is the part taking its place on the body.
func (gas Gas) hang(body *Body) { body.gas = &gas }

// Depth is how far below the cloud tops the gas goes opaque, as a share
// of the radius.
func (gas Gas) Depth() float64 { return gas.depth }

// Opacity is the optical depth from the tops to there, with no unit.
func (gas Gas) Opacity() float64 { return gas.opacity }

// Bands is what the gas looks like: its own copy of the bands.
func (gas Gas) Bands() texture.Banded { return copyBands(gas.bands) }

// bandsComplaint is the complaint, without the package prefix, for a
// band texture holding a number that is not finite, or nil when it
// holds none.
func bandsComplaint(bands texture.Banded) error {
	for place, spot := range bands.Spots {
		if err := numberComplaint(fmt.Sprintf("bands' spot %d detail", place), spot.Detail); err != nil {
			return err
		}
		if spot.Detail < 0 || spot.Detail > 1 {
			return fmt.Errorf("bands' spot %d detail must be from nought to one", place)
		}
	}
	for place, paint := range bands.Paints {
		if err := lightComplaint(fmt.Sprintf("bands' paint %d", place),
			paint); err != nil {
			return err
		}
	}
	return firstComplaint(
		vecComplaint("bands' axis", bands.Axis),
		numberComplaint("bands' roil", bands.Roil),
		numberComplaint("bands' shear", bands.Shear),
		numberComplaint("bands' grain", bands.Grain),
		numberComplaint("bands' fine", bands.Fine),
		numberComplaint("bands' mottle", bands.Mottle))
}

// copyBands is the bands with their own paints, spots and streets, so
// no caller shares a slice with a part.
func copyBands(bands texture.Banded) texture.Banded {
	bands.Paints = append([]maths.Light(nil), bands.Paints...)
	bands.Spots = append([]texture.Spot(nil), bands.Spots...)
	bands.Streets = append([]texture.Street(nil), bands.Streets...)
	return bands
}

// Air is an atmosphere: its scale height, its extinction at the
// surface, and the colour of what it scatters.
type Air struct {
	made      bool    // set only by the constructor
	height    float64 // scale height in kilometres
	dense     float64 // extinction per kilometre at the surface
	colour    maths.Light
	radiant   thing.Radiant
	complaint error // the first bad argument a method was given
}

// NewAir is an atmosphere of a scale height, losing light at the
// surface as the loss says, scattering a colour.
func NewAir(scaleHeight Span, loss Loss,
	colour maths.Light) (Air, error) {
	tall := scaleHeight.InKilometres()
	if !positiveFinite(tall) {
		return Air{}, fmt.Errorf("bodies: an air of scale height "+
			"%v km is no atmosphere: kilometres, above zero",
			tall)
	}
	dense, err := loss.problem()
	if err != nil {
		return Air{}, fmt.Errorf("bodies: the air's %w", err)
	}
	if err := lightComplaint("the air's colour", colour); err != nil {
		return Air{}, fmt.Errorf("bodies: %w", err)
	}
	return Air{made: true, height: tall, dense: dense, colour: colour}, nil
}

// Radiating is the air with what it does outside the visible: venus's
// clouds pass radio, which is how its ground was mapped.
func (air Air) Radiating(radiant thing.Radiant) Air {
	hold(&air.complaint, radiantComplaint("Air.Radiating's radiant", radiant))
	air.radiant = radiant
	return air
}

// hang is the part taking its place on the body.
func (air Air) hang(body *Body) { body.air = &air }

// Height is the atmosphere's scale height, in kilometres.
func (air Air) Height() float64 { return air.height }

// Dense is the extinction at the surface, per kilometre.
func (air Air) Dense() float64 { return air.dense }

// Colour is what the air scatters, a Light.
func (air Air) Colour() maths.Light { return air.colour }

// Radiant is what the air gives off and passes outside the visible, as
// thing.Radiant says.
func (air Air) Radiant() thing.Radiant { return air.radiant }

// Rings are a disc round a giant, from an inner to an outer edge as
// multiples of the radius, of a thickness, and the rest as medium.Disc
// has it.
type Rings struct {
	made         bool    // set only by the constructor
	inner, outer float64 // times the radius
	thick        float64 // kilometres
	dense        float64 // extinction per kilometre
	profile      []float64
	grain        float64
	ringlets     float64
	colour       maths.Light
	complaint    error // the first bad argument a method was given
}

// NewRings is a disc from an inner to an outer edge, both as times the
// body's radius, of a thickness, losing light as the loss says, in a
// colour. Grain, ringlets and a radial profile hang by its methods.
func NewRings(innerTimes, outerTimes float64, thick Span,
	loss Loss, colour maths.Light) (Rings, error) {
	if !positiveFinite(innerTimes) || !(outerTimes > innerTimes) ||
		!positiveFinite(outerTimes) {
		return Rings{}, fmt.Errorf("bodies: rings from %v to %v "+
			"times the radius are no disc: the outer edge is "+
			"past the inner, both above zero", innerTimes,
			outerTimes)
	}
	dense, err := loss.problem()
	if err != nil {
		return Rings{}, fmt.Errorf("bodies: the rings' %w", err)
	}
	deep := thick.InKilometres()
	if !nonNegativeFinite(deep) {
		return Rings{}, fmt.Errorf("bodies: rings %v km thick are no "+
			"disc: kilometres, zero or above, finite", deep)
	}
	if err := lightComplaint("the rings' colour", colour); err != nil {
		return Rings{}, fmt.Errorf("bodies: %w", err)
	}
	return Rings{made: true, inner: innerTimes, outer: outerTimes,
		thick: deep, dense: dense,
		colour: colour}, nil
}

// Profiled is the rings with a radial profile, as medium.Disc takes it;
// the rings keep a copy of the slice given.
func (rings Rings) Profiled(profile []float64) Rings {
	for place, share := range profile {
		what := fmt.Sprintf("Rings.Profiled's element %d", place)
		hold(&rings.complaint, numberComplaint(what, share))
	}
	rings.profile = append([]float64(nil), profile...)
	return rings
}

// Grained is the rings dressed in grain and ringlets, as medium.Disc
// takes them.
func (rings Rings) Grained(grain, ringlets float64) Rings {
	if rings.complaint == nil {
		rings.complaint = firstComplaint(
			numberComplaint("Rings.Grained's grain", grain),
			numberComplaint("Rings.Grained's ringlets", ringlets))
	}
	rings.grain, rings.ringlets = grain, ringlets
	return rings
}

// hang is the part taking its place on the body.
func (rings Rings) hang(body *Body) { body.rings = &rings }

// Inner is the inner edge, in times the body's radius.
func (rings Rings) Inner() float64 { return rings.inner }

// Outer is the outer edge, in times the body's radius.
func (rings Rings) Outer() float64 { return rings.outer }

// Thick is the thickness, in kilometres.
func (rings Rings) Thick() float64 { return rings.thick }

// Dense is the extinction at the densest, per kilometre.
func (rings Rings) Dense() float64 { return rings.dense }

// Profile is the radial profile, as medium.Disc takes it: the density
// from the inner edge to the outer, as a multiple of Dense. It is the
// rings' own copy.
func (rings Rings) Profile() []float64 {
	return append([]float64(nil), rings.profile...)
}

// Grain is how much ringlets of noise thin the rings, a share from
// zero, smooth, to one.
func (rings Rings) Grain() float64 { return rings.grain }

// Ringlets is how many ringlets there are across the rings, a count.
func (rings Rings) Ringlets() float64 { return rings.ringlets }

// Colour is what the rings scatter, a Light.
func (rings Rings) Colour() maths.Light { return rings.colour }

// Horizon is a hole's edge; what is on it is a dressing, since nothing
// on a horizon can be seen: hex lines of a brightness drawn on a black
// ball just outside it, in a colour, white when unset.
//
// A spinning hole's horizon is flattened at the poles, so the ball is
// as wide as its equator and a fiftieth more: outside the horizon at
// the equator, and a hair further outside at the poles. The ball
// blocks a camera's way out from inside, where no horizon would; a
// fall through the horizon wants the hexes off.
type Horizon struct {
	made  bool // set only by the constructor
	hexes float64
	light maths.Light
}

// NewHorizon dresses a hole's edge in hex lines of a brightness,
// zero for none, in a colour, white when unset.
func NewHorizon(hexes float64, light maths.Light) (Horizon, error) {
	if !nonNegativeFinite(hexes) {
		return Horizon{}, fmt.Errorf("bodies: hexes of %v are no "+
			"dressing: a brightness, zero or above, finite", hexes)
	}
	if err := lightComplaint("the hexes' light", light); err != nil {
		return Horizon{}, fmt.Errorf("bodies: %w", err)
	}
	return Horizon{made: true, hexes: hexes, light: light}, nil
}

// hang is the part taking its place on the body.
func (edge Horizon) hang(body *Body) { body.edge = edge }

// Hexes is the brightness of the hex lines, with no unit; zero is none.
func (edge Horizon) Hexes() float64 { return edge.hexes }

// Light is the lines' colour, a Light, white when unset.
func (edge Horizon) Light() maths.Light { return edge.light }

// Disc is the glowing disc round a hole: its edges and thickness in
// kilometres, its colour falling from the hottest ring's by
// texture.Thin's profile.
type Disc struct {
	made         bool    // set only by the constructor
	inner, outer float64 // kilometres
	thick, dense float64 // kilometres; extinction per kilometre
	hot          float64 // the hottest ring, kelvin
	bright       float64 // the hottest ring's light
	speed        float64 // orbital speed at the inner edge, of light
	// streaks and shear are a dressing, as texture.Thin takes them: a
	// toggle for a film, zero honest.
	streaks, shear float64
	complaint      error // the first bad argument a method was given
}

// NewDisc is a glowing disc from an inner to an outer edge, of a
// thickness, losing light as the loss says, its hottest ring at a
// temperature shining a brightness: the disc's colour is a black
// body's by radius, none typed. Beaming and dressing hang by its
// methods.
func NewDisc(inner, outer, thick Span, loss Loss,
	hot measure.Temperature, bright float64) (Disc, error) {
	near, far := inner.InKilometres(), outer.InKilometres()
	if !positiveFinite(near) || !(far > near) || !positiveFinite(far) {
		return Disc{}, fmt.Errorf("bodies: a disc from %v to %v km "+
			"is no ring: the outer edge is past the inner, both "+
			"above zero, finite", near, far)
	}
	dense, err := loss.problem()
	if err != nil {
		return Disc{}, fmt.Errorf("bodies: the disc's %w", err)
	}
	heat := hot.InKelvin()
	if !nonNegativeFinite(heat) {
		return Disc{}, fmt.Errorf("bodies: a disc at %v kelvin is "+
			"no temperature: zero or above, finite", heat)
	}
	deep := thick.InKilometres()
	if !nonNegativeFinite(deep) {
		return Disc{}, fmt.Errorf("bodies: a disc %v km thick is "+
			"no ring: kilometres, zero or above, finite", deep)
	}
	if err := numberComplaint("a disc's brightness", bright); err != nil {
		return Disc{}, fmt.Errorf("bodies: %w", err)
	}
	return Disc{made: true, inner: near, outer: far, thick: deep,
		dense: dense, hot: heat, bright: bright}, nil
}

// Beamed is the disc with its gas's orbital speed at the inner edge as
// a share of the speed of light, which beams the side coming toward
// the eye.
func (disc Disc) Beamed(speed maths.Portion) (Disc, error) {
	fast, err := speed.Share()
	if err != nil {
		return disc, fmt.Errorf("bodies: the disc's speed: %w", err)
	}
	disc.speed = fast
	return disc, nil
}

// Dressed is the disc with streaks and shear, as texture.Thin takes
// them: a toggle for a film, zero honest.
func (disc Disc) Dressed(streaks, shear float64) Disc {
	if disc.complaint == nil {
		disc.complaint = firstComplaint(
			numberComplaint("Disc.Dressed's streaks", streaks),
			numberComplaint("Disc.Dressed's shear", shear))
	}
	disc.streaks, disc.shear = streaks, shear
	return disc
}

// hang is the part taking its place on the body.
func (disc Disc) hang(body *Body) { body.disc = &disc }

// Inner is the inner edge, in kilometres.
func (disc Disc) Inner() float64 { return disc.inner }

// Outer is the outer edge, in kilometres.
func (disc Disc) Outer() float64 { return disc.outer }

// Thick is the thickness, in kilometres.
func (disc Disc) Thick() float64 { return disc.thick }

// Dense is the extinction at the densest, per kilometre.
func (disc Disc) Dense() float64 { return disc.dense }

// Hot is the hottest ring's temperature, in kelvin.
func (disc Disc) Hot() float64 { return disc.hot }

// Bright is the hottest ring's light, a brightness with no unit.
func (disc Disc) Bright() float64 { return disc.bright }

// Speed is the gas's orbital speed at the inner edge, as a share of the
// speed of light.
func (disc Disc) Speed() float64 { return disc.speed }

// Streaks is the dressing, as texture.Thin takes it; zero is none, and
// it has no unit.
func (disc Disc) Streaks() float64 { return disc.streaks }

// Shear is the dressing, as texture.Thin takes it; it has no unit.
func (disc Disc) Shear() float64 { return disc.shear }

// Moon is a body on an orbit round another: its own body, the orbit's
// radius, and where on it the moon starts.
type Moon struct {
	made  bool // set only by the constructor
	body  Body
	orbit float64 // kilometres
	phase float64 // radians round the orbit
}

// NewMoon puts a body on an orbit of a radius round the one it hangs
// on, starting a phase round it. An orbit of zero is tolerated and
// never timed, as the worked orbits say; below zero, or endless, is
// no orbit.
func NewMoon(body Body, orbit Span,
	phase measure.Angle) (Moon, error) {
	if !body.made {
		return Moon{}, errors.New("bodies: a moon needs a body made " +
			"by NewBody, and was given a Body{}")
	}
	around := orbit.InKilometres()
	if !nonNegativeFinite(around) {
		return Moon{}, fmt.Errorf("bodies: %s on an orbit of %v km "+
			"goes nowhere: kilometres, zero or above, finite",
			body.name, around)
	}
	if err := numberComplaint("a moon's phase", phase.InRadians()); err != nil {
		return Moon{}, fmt.Errorf("bodies: %s: %w", body.name, err)
	}
	if body.complaint != nil {
		return Moon{}, fmt.Errorf("bodies: %s as a moon: %w", body.name,
			body.complaint)
	}
	return Moon{made: true, body: body, orbit: around,
		phase: phase.InRadians()}, nil
}

// hang is the part taking its place on the body.
func (moon Moon) hang(body *Body) {
	body.moons = append(body.moons, moon)
}

// Body is the moon's own body.
func (moon Moon) Body() Body { return moon.body }

// Orbit is the orbit's radius, in kilometres.
func (moon Moon) Orbit() float64 { return moon.orbit }

// Phase is where on the orbit the moon starts, in radians.
func (moon Moon) Phase() float64 { return moon.phase }

// refusal is why the part cannot be worn, or nil.
func (part Coma) refusal() error {
	return refusal(part.made, "NewComa", nil)
}

// refusal is why the part cannot be worn, or nil.
func (part Tail) refusal() error {
	return refusal(part.made, "NewTail", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Clouds) refusal() error {
	if part.hasTurn && !(part.turn >= 0 && part.turn <= 10) {
		return fmt.Errorf("bodies: cloud storm turn must be from 0 to 10 radians")
	}
	return refusal(part.made, "NewClouds", nil)
}

// refusal is why the part cannot be worn, or nil.
func (part Aurora) refusal() error {
	return refusal(part.made, "NewAurora", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Cities) refusal() error {
	return refusal(part.made, "NewCities", nil)
}

// refusal is why the part cannot be worn, or nil.
func (part Jets) refusal() error {
	return refusal(part.made, "NewJets", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Flare) refusal() error {
	return refusal(part.made, "NewFlare", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Plume) refusal() error {
	return refusal(part.made, "NewPlume", nil)
}

// refusal is why the part cannot be worn, or nil.
func (part Ground) refusal() error {
	return refusal(part.made, "NewGround", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Gas) refusal() error {
	return refusal(part.made, "NewGas", nil)
}

// refusal is why the part cannot be worn, or nil.
func (part Air) refusal() error {
	return refusal(part.made, "NewAir", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Rings) refusal() error {
	return refusal(part.made, "NewRings", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Horizon) refusal() error {
	return refusal(part.made, "NewHorizon", nil)
}

// refusal is why the part cannot be worn, or nil.
func (part Disc) refusal() error {
	return refusal(part.made, "NewDisc", part.complaint)
}

// refusal is why the part cannot be worn, or nil.
func (part Moon) refusal() error {
	return refusal(part.made, "NewMoon", nil)
}
