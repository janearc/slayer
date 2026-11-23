package rig

// The numbers the show's geometry works in, moved from lightshow with
// their meanings. "Lightshow's" means picked for lightshow,
// 2026-11-09 and 10, by eye on the film, unless a line
// says otherwise.

// Defined: a quarter and a half turn, in degrees; a half.
const (
	quarter  = 90.0
	halfTurn = 180.0
	half     = 0.5
)

// azimuth is the bearing from the eye to the far tree, and the moon's,
// in degrees. From sets/moonlake.yaml, scene.moon.azimuth.
const azimuth = 109.65

// The eye's place on the map and the lake's height: the west rise
// (sets/moonlake.yaml things) and the fantasy block's lake, metres.
const (
	eyeEast    = 263104.5
	eyeNorth   = 2148423.5
	lakeHeight = 1076.0
)

// The far tree from the eye, east and north, and its ground's height:
// sets/moonlake.yaml things and the lidar under it.
const (
	farTreeEast   = 1727.5
	farTreeNorth  = -617.5
	farTreeGround = 1195.5
)

// The rim heads: 120 m either side of the far tree, 6 m over its ground
// before the towers stood. Lightshow's, matching the rim towers in
// sets/moonlake.yaml.
const (
	rimApart  = 120.0
	rimHeadUp = 6.0
)

// The shore installations: sets/moonlake.yaml's rig shore left and
// right, 8 m over the water, their fans lifted 7 degrees. Lightshow's.
const (
	shoreLeftEast   = 263652.4
	shoreLeftNorth  = 2148337.2
	shoreRightEast  = 263578.1
	shoreRightNorth = 2148129.0
	shoreHeadUp     = 8.0
	shoreLift       = 7.0
)

// rigGround is the ground under each tower, rim left and right, shore
// left and right, in metres: measured by the lidar, through
// improv/moonlake/rig-heights.yaml.
var rigGround = [4]float64{1180.2, 1191.3, 1076.1, 1079.4}

// The strobes, behind the far tree: the white 160 m behind and 20 m up;
// the magenta 220 m behind, 260 m left, 30 m up. Lightshow's.
const (
	whiteBehind   = 160.0
	whiteUp       = 20.0
	magentaBehind = 220.0
	magentaLeft   = 260.0
	magentaUp     = 30.0
)

// The searchlights: 200 m past the far tree, 10 m up, 140 m apart.
// Lightshow's.
const (
	searchBehind = 200.0
	searchUp     = 10.0
	searchApart  = 140.0
)

// The searchlights' moves, in degrees: a fan opening from 8 and 14 a
// rank out, from 70 up to 30; a sweep 20 out, 8 more a rank, swinging
// 18, tilted 42 and nodding 13; and the viewer's beams 0.017 of a
// radian apart a rank. Lightshow's.
const (
	fanFirst    = 8.0
	fanStep     = 14.0
	fanTop      = 70.0
	fanDrop     = 40.0
	viewerApart = 0.017
	sweepOut    = 20.0
	sweepRank   = 8.0
	sweepSwing  = 18.0
	sweepTilt   = 42.0
	sweepNod    = 13.0
)

// The backlights behind the close spruces, from the eye, and their
// ground's height. Lightshow's.
const (
	backLeftEast    = 46.4
	backLeftNorth   = -3.8
	backRightEast   = 38.3
	backRightNorth  = -26.4
	backlightGround = 1126.5
)

// The figures: how bright a beam is in each, the pyramid's eight by
// eight beams rising 6 degrees and 1.3 spreads wide, the scan's 48
// sweeps 1.4 spreads wide, the cone's 40 beams 7 degrees out, the
// wave's 16 beams rising 3 degrees one and a half times along the fan,
// and the five beams' light a sheet or cone shares. Lightshow's.
const (
	beamWeight    = 1.8
	pairWeight    = 1.4
	pyramidSide   = 8
	pyramidRise   = 6.0
	pyramidWide   = 1.3
	pyramidWeight = 0.45
	scanSweeps    = 48
	scanWide      = 1.4
	sheetLight    = 5.0
	coneRound     = 40
	coneHalf      = 7.0
	waveBeams     = 16
	waveRise      = 3.0
	waveRun       = 1.5
	waveWeight    = 0.9
)

// The mirrors: a swing of 45 degrees either way, a tilt of 4.5 down
// nodding 2.2, a spread of 22 opening 14, and their rates on the clock
// in radians a second. Lightshow's.
const (
	swingMost    = 45.0
	swingRate    = 1.7
	tiltMiddle   = -4.5
	tiltNod      = 2.2
	tiltRate     = 1.1
	tiltLead     = 0.6
	spreadMiddle = 22.0
	spreadOpen   = 14.0
	spreadRate   = 0.9
)

// strobeLuck is the share of frames a free-running strobe fires, and
// strobeRate its clock's frames a second. Lightshow's.
const (
	strobeLuck = 0.28
	strobeRate = 12.0
)

// The lattice hash's odd multipliers, shift and mix, and the 53 bits of
// a float64's fraction it keeps. Lightshow's: the multipliers are the
// golden ratio's and two of xxhash's primes, the mix murmur3's.
const (
	latticeX     = 0x9E3779B97F4A7C15
	latticeY     = 0xC2B2AE3D27D4EB4F
	latticeSeed  = 0x165667B19E3779F9
	latticeMix   = 0xff51afd7ed558ccd
	latticeShift = 33
	latticeDrop  = 11
	latticeBits  = 53
)
