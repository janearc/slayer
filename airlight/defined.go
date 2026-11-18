package airlight

import "math"

// The small numbers the reading is built on.
const (
	none     = 0           // defined: nothing; the first index
	one      = 1           // defined: a whole; the next index
	two      = 2           // defined: a pair; twice
	three    = 3           // defined: one more than two
	half     = 0.5         // defined: one of two
	fullTurn = 2 * math.Pi // defined: a circle's angle, in radians
)

// The wavelengths the fields are worked at.
//
// The span is the CIE 1931 observer's visible range as its tables give
// it: at both ends the eye's sensitivity is a few hundred-thousandths
// of its peak. Each wavelength stands for the band of everyNanometres
// about it, the ends' too, as moonlight's sky's ends do.
//
// moonlight's sky steps 16 nm from 384 to 768, 25 wavelengths of single
// scattering summed in microseconds. Here a wavelength is a walk of
// seconds, so this takes 21, and reaches both of the observer's ends,
// where Venus's light, which rises to the red, should not be cut short.
const (
	firstNanometre  = 380 // picked: see above
	lastNanometre   = 780 // picked: see above
	everyNanometres = 20  // picked: see above
)

// The field's grid: one kept height, the ground, and its bins.
//
// The sun's zenith is in 36 bins of five degrees, read linearly between
// their centres, so the sun may stand anywhere. It is the same for every
// air; the view's and the azimuth's bins are each air's own.
const zenithBins = 36 // picked: see above

// Venus's view and azimuth bins. The view's cosine from up is in 16
// bins of an eighth: the lowest above the horizon spans its first seven
// degrees. The azimuth from the sun is in 8 bins of 22.5 degrees,
// folded, since the sky is the same either side of the sun.
//
// Under Venus's cloud the sky changes little with azimuth, and slowly
// with height above the horizon: what changes from bin to bin there is
// mostly the walk's noise, a few hundred counts a bin, and finer bins
// would hold more of it.
const (
	venusViews    = 16 // picked: see above
	venusAzimuths = 8  // picked: see above
)

// Earth's view and azimuth bins. Earth's clear sky changes fastest near
// the horizon, where the light scattered once grows as one over the
// view's cosine until the air along the look is no longer thin: below
// about six degrees at 550 nm, where the column's depth is about 0.2.
//
// So the view's cosine is in 32 bins of a sixteenth, the lowest above
// the horizon its first 3.6 degrees. The haze leans forward, its
// asymmetry 0.63, so the sky about the sun changes faster round from it
// than Venus's: 16 azimuth bins of 11.25 degrees.
const (
	earthViews    = 32 // picked: see above
	earthAzimuths = 16 // picked: see above
)

// How the photons are walked.
const (
	// seed is the walk's seed, fixed so the same plan gives the same
	// fields; it is in the key.
	seed = 1 // picked: any fixed seed does
	// workers is how many walk at once. The same seed and workers give
	// the same fields, so it is in the key too.
	workers = 4 // picked: the four the runs here are given
	// venusPhotons is how many photons each of Venus's wavelengths is
	// walked with. The solver's report found a mean of a hundred counts
	// a downward bin, over every sun zenith, at about 2.2 million; at a
	// sun as high as the sets' the bins hold more than that mean.
	venusPhotons = 2_200_000 // picked: see above
	// earthPhotons is how many photons each of Earth's wavelengths is
	// walked with. Earth's thin air scatters few of them, so it takes
	// more than Venus's for the same counts. TestEarthAtSeaLevelIsReported
	// found a mean of 136 to 668 counts a downward bin, sun 30 degrees off.
	earthPhotons = 4_000_000 // picked: see above
)

// solverVersion names the code that turns an air into fields: skylight's
// walk and this package's reading of it. Bump it when either changes
// what a field holds. The air's own changes need no bump: its columns
// at every wavelength are in the key, number for number.
const solverVersion = "skylight 1, airlight 1" // defined: the first of each

// transmittanceSteps is how many steps of the sun's zenith, from the
// zenith to the nadir, the direct sun's share is kept at, and read
// linearly between.
const transmittanceSteps = 360 // picked: half a degree a step

// grey is the reflectance of the card an incident meter is made to
// render at its own reflectance: the photographer's mid grey.
const grey = 0.18 // defined: the grey card's eighteen percent

// mostBins is the most any count in a cache entry's header may be: far
// past any grid, so a header that is not a count is refused before
// anything is made for it.
const mostBins = 1 << 20 // picked: see above

// float64Bytes is the bytes a float64 takes, and float64Bits its bits.
const (
	float64Bytes = 8  // defined: sixty-four bits
	float64Bits  = 64 // defined: the same, counted in bits
)

// shortest is the precision that asks strconv for the fewest digits
// that read back as the same float64.
const shortest = -1 // defined: strconv's own word for it

// decimal is the base a key writes whole numbers in.
const decimal = 10 // defined: base ten
