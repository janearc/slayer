package show

// The numbers the package works in. Each says whether it is defined,
// measured (and where), or picked (by whom, and when). The cue
// defaults were picked for moonlake, 2026-11-09 and
// 10, and moved here from cueshow unchanged.

// beatsPerBar is how many beats make a bar. Defined by the song: it is
// in four, bar 1 beat 1 the first drum beat (2026-11-09).
const beatsPerBar = 4

// stepsPerBeat is how many sixteenths a beat holds. Defined.
const stepsPerBeat = 4

// eighthsPerBeat is how many eighths a beat holds. Defined.
const eighthsPerBeat = 2

// onesPerBeat is how many bars' ones fall in a beat: a quarter. Defined.
const onesPerBeat = 1.0 / beatsPerBar

// floodsLookBack is how many of the floods' last hits are looked at for
// a flash still showing. Picked in cueshow: a hit three steps back has
// fallen past seeing.
const floodsLookBack = 3

// leadLookBack is how far back, in seconds, a loud lead note can have
// fired a flood that still shows. Picked in cueshow.
const leadLookBack = 0.5

// floodFlash is how long a flood's flash takes to fall to a third, in
// seconds: a strobe's afterglow as a camera sees it. Picked in cueshow.
const floodFlash = 0.08

// burstFrames and burstRate say how long after a downbeat its fire
// burst still shows: the fire is rendered from ignition for 82 frames
// at 24 a second, two bars at the song's tempo. Measured from the fire
// pass, in cueshow.
const (
	burstFrames = 82.0
	burstRate   = 24.0
	burstSpan   = burstFrames / burstRate
)

// setMoonAlt is the moon's altitude in sets/moonlake.yaml, where it
// stands until a cue moves it. Defined by the set.
const setMoonAlt = 2.3

// moonBeat is how much of a beat the moon's pulse takes to fall to a
// third. Picked in cueshow.
const moonBeat = 0.3

// searchStrike is how much of a beat the searchlights' strike takes to
// fall to a third. Picked in cueshow.
const searchStrike = 0.35

// sweepTurn is the aim a flank spot sweeps across and back through,
// from 0 out to 2 and home. Defined by lightshow's spots.
const sweepTurn = 2.0

// noGrid is the place on the grid of a moment before it, and the time of
// a place before it. Defined.
const noGrid = -1.0

// the state before any cue: a dark rig over the resting mist. hot and
// hotter rest at the lead's own loudness, its 90th and 95th centiles,
// so a tenth of its notes strike a strobe. Picked in cueshow.
const (
	restBeams      = 10
	restSweepBars  = 8.0
	restDuty       = 0.25
	restMist       = 0.12
	restHot        = 78
	restHotter     = 88
	restPulse      = "x...x...x...x..."
	restDecay      = 1.5
	restSpotBars   = 2.0
	restSearchBars = 4.0
	restPalette    = "green"
	restFloods     = "off"
	restStrobes    = "off"
	restFigure     = "fan"
	restBanks      = "unison"
	restMove       = "sweep"
)

// the strobe modes and the banks the fold tells apart. Defined by
// lightshow.
const (
	strobesLattice = "lattice"
	strobesGrid16  = "grid16"
	strobesGrid8   = "grid8"
	strobesLead    = "lead"
	banksOffset    = "offset"
	floodsGrid16   = "grid16"
	floodsGrid8    = "grid8"
	floodsOnes     = "ones"
	floodsLead     = "lead"
)

// latticeFires is what a strobe says when lightshow's own lattice drives
// it, not the grid or the lead. Defined by lightshow.
const latticeFires = -1

// rimMoon is the altitude, in degrees, at which the moon's disc is hidden
// behind the rim: the rim at 1.65 and the disc 1.55 in radius. Defined by
// the set; the cue sheet says so.
const rimMoon = 0.0

// pulseHit is the character a pulse marks a hit with. Defined by the
// cue sheet.
const pulseHit = 'x'

// paletteColours is each palette name as linear srgb, near enough. green
// is lightshow's own 532 nm green. Picked in cueshow.
var paletteColours = map[string][3]float64{
	"green": {0.08, 1, 0.12}, "red": {1, 0.04, 0.04},
	"blue": {0.08, 0.2, 1}, "cyan": {0.05, 0.85, 1},
	"amber": {1, 0.55, 0.08}, "magenta": {1, 0.25, 0.85},
	"white": {1, 1, 1},
}

// figureNames, bankNames and moveNames are the figures, banks and
// searchlight moves lightshow plays. Defined by lightshow.
var (
	figureNames = []string{"beam", "pair", "fan", "fan64", "scan", "cone",
		"wave"}
	bankNames = []string{"unison", banksOffset}
	moveNames = []string{"sweep", "chase", "fan", "viewer"}
)
