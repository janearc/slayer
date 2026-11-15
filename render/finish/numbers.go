package finish

// what the package counts in, each defined
const (
	// none is no count at all. It is defined as zero.
	none = 0
	// one is a single thing. It is defined as one.
	one = 1
	// pair is two things. It is defined as two.
	pair = 2
	// whole is the whole of a share. It is defined as one.
	whole = 1
	// half is half of a share. It is defined as one over two.
	half = 0.5
	// stopBase is what one stop multiplies light by. It is defined: a
	// stop doubles the light.
	stopBase = 2.0
)

// what the titles and the plates are picked from
const (
	// defaultRows is how many rows of cells a picture of titles is laid
	// out in when the job says none. It is picked: it is the number a
	// set's hud uses (set/hud.go), so a title and a set's hud read at
	// the same size.
	defaultRows = 24
	// seedMost is the largest seed a roll takes. It is defined by the
	// 64 bit seed ninjafilm.Roll takes and the 32 bit bound the job
	// puts on it.
	seedMost = 1<<32 - 1
)

// the exposures a print is developed with, by how dense the stock is.
// Each is recalled from ninjafilm/doc/filming.md, "Exposure, and the
// screen": about 2 for the single silver records, 4 for dusk and
// lantern, 16 for pollen. ninjafilm's authors measured them.
const (
	exposureSilver = 2.0
	exposureDye    = 4.0
	exposureSpeck  = 16.0
)

// shoulderShare is where a stock's screen starts its shoulder: this
// share of the stops its clearest print holds above white, that many
// stops below white. It is picked (the fable-team stock slice,
// 2026-11-12) from the night fire through lantern; see the README.
const shoulderShare = 0.4

// transmission is, by a stock's name, the exposure that puts a white
// card filmed at the stock's rating on the screen's white.
var transmission = map[string]float64{
	"soot": exposureSilver, "ash": exposureSilver,
	"smoke": exposureSilver, "dusk": exposureDye,
	"lantern": exposureDye, "pollen": exposureSpeck,
}
