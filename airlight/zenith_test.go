package airlight

import (
	"math"
	"sync"
	"testing"

	"github.com/janearc/slayer/airs"
	"github.com/janearc/slayer/skylight"
)

// Earth's noon zenith worked out apart from the solver: the light the air
// scatters once, summed exactly over the column's own shells taken as
// flat, with a bound on what it scatters more than once; set beside the
// solver's field at the same geometry, the noon set's ground and sun.
const (
	// noonGround is the noon set's ground above the mean radius, in
	// metres, as the cache's Earth entry for it keeps it.
	noonGround = 1052.3131103515625 // picked: the entry's own number
	// noonDegrees is the noon set's sun from the zenith, in degrees.
	noonDegrees = 30 // defined: as FIX-REPORT read it from the set
	// checkNanometres is the wavelength the field is checked at.
	checkNanometres = 550 // defined: the brief's
	// sunSteps is how many steps of the sun's zenith the sum takes
	// across each of the field's zenith bins.
	sunSteps = 8 // picked: the light changes little across five degrees
	// capSteps is how many steps of the view's cosine the sum takes
	// across the field's top view bin.
	capSteps = 16 // picked: see above
	// roundSteps is how many steps of the azimuth from the sun every sum
	// takes over half a turn; the other half is the same.
	roundSteps = 32 // picked: under six degrees a step
	// heightSteps is how many heights up each shell the second order is
	// summed at.
	heightSteps = 4 // picked: shells are thin to the light
	// sphereSteps is how many steps of the cosine each half of the sphere
	// is summed in, for the second order and the sky's light on the ground.
	sphereSteps = 64 // picked: under two degrees a step about the sun
	// escapeSteps is how many steps of the cosine the share of an even
	// light that crosses the column unscattered is summed in.
	escapeSteps = 1024 // picked: see above
	// noiseSigmas is how many standard deviations of the field's noise
	// the check allows each way.
	noiseSigmas = 3 // picked: see above
	// rayleighScale is Rayleigh's phase's: three quarters of one and the
	// turn's cosine squared, which sums to four pi over the sphere.
	rayleighScale = 0.75 // defined: see above
	// solidTurns is the whole sphere's solid angle.
	solidTurns = 4 * math.Pi // defined: four pi steradians
	// henyeyPower is the power of the Henyey-Greenstein phase's
	// denominator.
	henyeyPower = 1.5 // defined: three halves
)

// a slab is a column taken as flat shells: at a sun 30 degrees from the
// zenith and a look near it the planet's curve changes the paths by parts
// in ten thousand. It keeps the shells, the optical depth from the ground
// to each shell's bottom and last to the top, and the ground's albedo.
type slab struct {
	shells []skylight.Shell
	depths []float64
	ground float64
}

// a point is a height in a slab, in metres, and the optical depth from
// the ground up to it.
type point struct{ height, depth float64 }

// slabOf is a column as a slab.
func slabOf(column skylight.Column) slab {
	made := slab{shells: column.Shells, depths: []float64{none},
		ground: column.GroundAlbedo}
	for index, shell := range column.Shells {
		made.depths = append(made.depths, made.depths[index]+
			extinctionOf(shell)*(shell.Top-shell.Bottom))
	}
	return made
}

// extinctionOf is all a shell takes from a beam, a metre.
func extinctionOf(shell skylight.Shell) float64 {
	return shell.GasScattering + shell.GasAbsorption + shell.ParticleExtinction
}

// rayleigh is Rayleigh's phase at a turn's cosine.
func rayleigh(turn float64) float64 { return rayleighScale * (one + turn*turn) }

// henyey is the Henyey-Greenstein phase of an asymmetry at a turn's
// cosine.
func henyey(lean, turn float64) float64 {
	return (one - lean*lean) / math.Pow(one+lean*lean-two*lean*turn,
		henyeyPower)
}

// phaseOf is a shell's scattering a metre into a turn's cosine, its gas's
// and its particles', each by its phase.
func phaseOf(shell skylight.Shell, turn float64) float64 {
	return shell.GasScattering*rayleigh(turn) + shell.ParticleExtinction*
		shell.ParticleAlbedo*henyey(shell.ParticleAsymmetry, turn)
}

// relative is (e to the x, less one) over x, which is one at nought.
func relative(x float64) float64 {
	if x == none {
		return one
	}
	return math.Expm1(x) / x
}

// once is the light the air has scattered once that arrives at a point,
// looking at a cosine from up and an azimuth from the sun, the sun at a
// cosine from up, per unit sunlight square to it, per steradian.
//
// Each shell's part above the point, for a look up, or below it, for a
// look down, adds its scattering times the integral along the look of
// the sun's light reaching each place and the look's light leaving it.
// Inside a shell both depths are straight in the height, so the integral
// is exact: e to the depths at the part's foot, times its length, times
// relative of how fast they change along it, over the look's cosine.
func (s slab) once(at point, look, round, sun float64) float64 {
	turn := look*sun + sineOf(look)*sineOf(sun)*math.Cos(round)
	slant, upward := math.Abs(look), math.Copysign(one, look)
	top := s.depths[len(s.shells)]
	var sum float64
	for index, shell := range s.shells {
		low, high := shell.Bottom, shell.Top
		if look > none {
			low = max(low, at.height)
		} else {
			high = min(high, at.height)
		}
		if !(high > low) {
			continue
		}
		rate := extinctionOf(shell)
		foot := s.depths[index] + rate*(low-shell.Bottom)
		along := math.Exp(-(top-foot)/sun - math.Abs(foot-at.depth)/slant)
		length := high - low
		sum += phaseOf(shell, turn) / solidTurns * along * length *
			relative(rate*length*(one/sun-upward/slant)) / slant
	}
	return sum
}

// direct is the sun's own light landing on level ground, per unit
// sunlight square to it, the sun at a cosine from up.
func (s slab) direct(sun float64) float64 {
	return sun * math.Exp(-s.depths[len(s.shells)]/sun)
}

// skyOnce is the light the air has scattered once landing on level
// ground: once's light every way above, times its cosine from up.
func (s slab) skyOnce(sun float64) float64 {
	step, turn := one/float64(sphereSteps), math.Pi/roundSteps
	var sum float64
	for band := range sphereSteps {
		look := (float64(band) + half) * step
		for round := range roundSteps {
			sum += s.once(point{}, look, (float64(round)+half)*turn, sun) *
				look * step * two * turn
		}
	}
	return sum
}

// sources is what a point in a shell scatters toward straight down, a
// steradian a metre: of the light there the air has scattered once, and
// of the light the ground sends up, lit by landing.
//
// Light arriving along a look of cosine c turns into straight down by a
// turn whose cosine is c, whatever its azimuth, so each band of the
// sphere's cosines is one phase times the light round it.
func (s slab) sources(at point, shell skylight.Shell, sun,
	landing float64) (air, ground float64) {
	step, turn := one/float64(sphereSteps), math.Pi/roundSteps
	for band := range two * sphereSteps {
		look := -one + (float64(band)+half)*step
		phase := phaseOf(shell, look) / solidTurns
		for round := range roundSteps {
			air += phase * s.once(at, look, (float64(round)+half)*turn, sun) *
				step * two * turn
		}
		if look < none {
			sent := s.ground * landing / math.Pi *
				math.Exp(-at.depth/math.Abs(look))
			ground += phase * sent * step * fullTurn
		}
	}
	return air, ground
}

// second is the light at the ground straight up that the air scattered
// twice, and that the ground sent up and the air scattered once: each
// point's sources toward straight down, at heightSteps heights up each
// shell, seen through the depth to the ground.
func (s slab) second(sun, landing float64) (air, ground float64) {
	for index, shell := range s.shells {
		thick := (shell.Top - shell.Bottom) / heightSteps
		for step := range heightSteps {
			height := shell.Bottom + (float64(step)+half)*thick
			at := point{height, s.depths[index] +
				extinctionOf(shell)*(height-shell.Bottom)}
			fromAir, fromGround := s.sources(at, shell, sun, landing)
			seen := math.Exp(-at.depth) * thick
			air, ground = air+fromAir*seen, ground+fromGround*seen
		}
	}
	return air, ground
}

// again is the share of each order of the light past the second that
// goes on to another: at most what an even light loses crossing the
// whole column, one less twice the third exponential integral of its
// depth, and the ground's albedo besides.
//
// Taking the orders past the second as falling by it, with the zenith's
// share of each as the second's, is an assumption, not a theorem: the
// part of the bound that rests on it is said apart.
func (s slab) again() float64 {
	depth := s.depths[len(s.shells)]
	var crossing float64
	for band := range escapeSteps {
		cosine := (float64(band) + half) / escapeSteps
		crossing += two * cosine * math.Exp(-depth/cosine) / escapeSteps
	}
	return one - crossing + s.ground
}

// a zenith is the light straight up at the ground with the sun at a
// cosine, worked by the slab: what the air scattered once, the second
// order and the ground's light scattered once, the rest past them, and
// the light landing on level ground from the sun and the sky.
type zenith struct {
	once, second, ground, rest float64
	direct, sky, skyRest       float64
}

// zenithOf is the slab's zenith with the sun at a cosine from up.
func (s slab) zenithOf(sun float64) zenith {
	made := zenith{once: s.once(point{}, one, none, sun),
		direct: s.direct(sun), sky: s.skyOnce(sun)}
	made.second, made.ground = s.second(sun, made.direct+made.sky)
	share := s.again()
	made.rest = (made.second + made.ground) * share / (one - share)
	made.skyRest = made.sky * share / (one - share)
	return made
}

// bound is the most the zenith's light past the first order can be.
func (z zenith) bound() float64 { return z.second + z.ground + z.rest }

// binned is the light scattered once into the field's top view bin with
// the sun 30 degrees from the zenith, as the field reads it there: the
// mean over the bin's cosines and over every azimuth, and over each of
// the two bins of the sun's zenith either side of 30 degrees by its area
// on the sphere, then the mean of the two.
func (s slab) binned() float64 {
	width := math.Pi / zenithBins
	upper := int(math.Round(noonDegrees * degree / width))
	spread, turn := two/float64(earthViews), math.Pi/roundSteps
	var mean float64
	for _, bin := range []int{upper - one, upper} {
		var sum, area float64
		for step := range sunSteps {
			angle := (float64(bin) + (float64(step)+half)/sunSteps) * width
			for band := range capSteps {
				look := one - (float64(band)+half)/capSteps*spread
				for round := range roundSteps {
					sum += math.Sin(angle) * s.once(point{}, look,
						(float64(round)+half)*turn, math.Cos(angle))
				}
			}
			area += math.Sin(angle) * capSteps * roundSteps
		}
		mean += sum / area / two
	}
	return mean
}

// earthColumn is Earth's air at a wavelength cut at the noon set's
// ground, as the solver takes it and as a slab.
func earthColumn(nanometres float64) (skylight.Column, slab, error) {
	column, err := fromAirs{name: earthName,
		column: airs.Earth}.Column(nanometres)
	if err == nil {
		column, err = grounded(column, noonGround)
	}
	return column, slabOf(column), err
}

// earthSlab is earthColumn's slab, or the test stopped.
func earthSlab(t *testing.T, nanometres float64) slab {
	t.Helper()
	_, flat, err := earthColumn(nanometres)
	if err != nil {
		t.Fatal(err)
	}
	return flat
}

// a check is the solver's field beside the exact sum at the check's
// wavelength: the field's top view bin with the sun at 30 degrees, the
// crossings behind it, and the slab's light scattered once into the same
// bin, with the slab's zenith.
type check struct {
	field, counts, binned float64
	zenith                zenith
}

// earthChecked is the check, worked once for every test that asks: a
// walk of Earth's photons at the check's wavelength.
var earthChecked = sync.OnceValues(work)

// checked is the check, the walk done once for the tests that share it.
func checked(t *testing.T) check {
	t.Helper()
	made, err := earthChecked()
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// work walks Earth's air at the check's wavelength on Earth's own grid
// and photons, and sums the slab beside it.
func work() (check, error) {
	column, flat, err := earthColumn(checkNanometres)
	if err != nil {
		return check{}, err
	}
	grid := skylight.Grid{Heights: []float64{none}, Zeniths: zenithBins,
		Views: earthViews, Azimuths: earthAzimuths}
	field, _, err := skylight.Solve(column, grid, earthPhotons, seed, workers)
	if err != nil {
		return check{}, err
	}
	sun := noonDegrees * degree
	top := centre(earthViews-one, earthViews)*two - one
	made := check{binned: flat.binned(),
		zenith: flat.zenithOf(math.Cos(sun))}
	for round := range earthAzimuths {
		azimuth := centre(round, earthAzimuths) * math.Pi
		made.field += field.Radiance(none, sun, top, azimuth) / earthAzimuths
		for _, side := range []float64{-half, half} {
			made.counts += float64(field.Count(none,
				sun+side*math.Pi/zenithBins, top, azimuth))
		}
	}
	return made, nil
}

// within is whether a value of the field's bin lies where the air puts
// it: no less than the light scattered once, and no more than that and
// the bound past it, each by the field's noise.
func (c check) within(value float64) bool {
	noise := noiseSigmas * c.field / math.Sqrt(c.counts)
	more := c.zenith.bound() / c.zenith.once * c.binned
	return value >= c.binned-noise && value <= c.binned+more+noise
}

// the solver's top view bin at noon holds Earth's light at 550 nm: at
// least what the air scatters once into it, summed exactly over its
// shells, and no more than that and the bound on the light it scatters
// more than once, within the field's noise.
func TestEarthsZenithIsTheAirs(t *testing.T) {
	c := checked(t)
	z := c.zenith
	t.Logf("550 nm, sun %d degrees: the field's top bin %.5g from %.0f "+
		"crossings; the exact once %.5g, %.4g of it", noonDegrees, c.field,
		c.counts, c.binned, c.field/c.binned)
	t.Logf("straight up: once %.5g; second %.4g, ground %.4g, rest %.4g "+
		"(on the assumption); bound %.4g of once", z.once, z.second,
		z.ground, z.rest, z.bound()/z.once)
	t.Logf("on level ground: sun %.5g, sky once %.5g, sky past once at "+
		"most %.4g", z.direct, z.sky, z.skyRest)
	if !c.within(c.field) {
		t.Errorf("the field's %.5g is not between the air's %.5g and %.5g",
			c.field, c.binned, c.binned*(one+z.bound()/z.once))
	}
}

// the anti-test: a field half or twice the solver's, which a fault of
// the azimuth's folding or of a phase's sampling would give, is outside
// what the air allows, and the check sees it.
func TestAFieldHalfOrTwiceTheAirsIsSeen(t *testing.T) {
	c := checked(t)
	for _, factor := range []float64{half, two} {
		if c.within(c.field * factor) {
			t.Errorf("%g times the field, %.5g, is within the air's", factor,
				c.field*factor)
		}
	}
}

// the exact sum is whole where the look's cosine meets the sun's, where
// how fast the two depths change along a shell is the same and relative
// is read at nought: the light there is the light a hair either side.
func TestTheExactSumIsWholeWhereTheLooksMeet(t *testing.T) {
	flat := earthSlab(t, checkNanometres)
	sun := math.Cos(noonDegrees * degree)
	at := flat.once(point{}, sun, none, sun)
	beside := flat.once(point{}, sun*(one+meetShare), none, sun)
	if !(math.Abs(at-beside) <= meetShare*at*meetSlope) {
		t.Errorf("at the sun's cosine %.9g, beside it %.9g", at, beside)
	}
}

// the anti-test: relative is not the plain quotient, which at nought is
// no number.
func TestRelativeAtNoughtIsANumber(t *testing.T) {
	if plain := math.Expm1(none) / none; !math.IsNaN(plain) {
		t.Fatalf("the plain quotient at nought is %v", plain)
	}
	if got := relative(none); got != one {
		t.Errorf("relative at nought is %v", got)
	}
}

// How near the light at the sun's cosine must be to the light beside it.
const (
	// meetShare is how far beside the sun's cosine the look is, as a
	// share of it.
	meetShare = 1e-7 // picked: see above
	// meetSlope is how much the light may change for a share's change of
	// the cosine, as a share of itself: the light near the sun changes
	// with the cosine by a few times itself, no more.
	meetSlope = 100 // picked: well past that
)

// Earth's noon zenith over a grey card, reported, under the shelf's sun:
// the exact sum's luminance at the zenith over that of a level grey card
// at every wavelength of the fields, with the light scattered once only,
// and with the bound past it, the card's sky taken at its least and most.
func TestEarthsNoonZenithOverAGreyCardIsReported(t *testing.T) {
	if testing.Short() {
		t.Skip("21 wavelengths of second orders take a minute")
	}
	sun := shelfSun(t)
	air := fromAirs{name: earthName, column: airs.Earth, sun: sun,
		sunAt: airs.Sun.Band}
	plan := wavelengths()
	fields := &Fields{wavelengths: plan, band: everyNanometres}
	zeniths := make([]zenith, len(plan))
	for wave, nanometres := range plan {
		power, err := air.Sun(float64(nanometres))
		if err != nil {
			t.Fatal(err)
		}
		fields.sun = append(fields.sun, power)
		flat := earthSlab(t, float64(nanometres))
		zeniths[wave] = flat.zenithOf(math.Cos(noonDegrees * degree))
	}
	luminance := func(power func(z zenith) float64) float64 {
		return fields.seen(func(wave int) float64 {
			return power(zeniths[wave])
		}).Picture(one).Luminance()
	}
	card := func(z zenith) float64 { return grey * (z.direct + z.sky) / math.Pi }
	brightest := func(z zenith) float64 {
		return grey * (z.direct + z.sky + z.skyRest) / math.Pi
	}
	once := luminance(func(z zenith) float64 { return z.once })
	most := luminance(func(z zenith) float64 { return z.once + z.bound() })
	t.Logf("zenith over a level grey card, luminance: once alone %.4g; "+
		"between %.4g and %.4g with the bound", once/luminance(card),
		once/luminance(brightest), most/luminance(card))
}
