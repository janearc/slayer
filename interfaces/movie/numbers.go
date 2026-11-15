package movie

import "time"

// The numbers the package works in. Each says whether it is defined,
// measured (and where), or picked (by whom, and when).

// macroblock is the side of the square H.264 codes a picture in, in
// pixels. Defined by ITU-T H.264, clause 6.3.
const macroblock = 16

// keyframesWithin is the most frames between one keyframe and the next.
// Measured: a teammate's QuickTime trials on this machine, November
// 2026, found 48 plays where longer runs did not.
const keyframesWithin = 48

// highestLevel is the highest H.264 level written, as level_idc, ten
// times the level. Measured: a 4K file at 6.0 showed as blocks in
// QuickTime on this machine, November 2026; 5.2 played.
const highestLevel = 52

// levelTenths is how level_idc counts: ten to a level. Defined by
// ITU-T H.264, clause 7.4.2.1.1.
const levelTenths = 10

// highBitRate is the High profile's allowance over table A-1's MaxBR,
// in thousandths. Defined by ITU-T H.264 table A-2, cpbBrVclFactor 1250
// against Baseline's 1000.
const highBitRate = 1250

// thousand is the thousandths the allowance and kbit/s are counted in.
// Defined.
const thousand = 1000

// frameAreaRoot is the factor under the square root that bounds each
// side of a picture in macroblocks: no side exceeds sqrt(8 * MaxFS).
// Defined by ITU-T H.264, clause A.3.1 f and g.
const frameAreaRoot = 8

// bitsPerPixel is the video bit rate asked of the encoder, in bits for
// each pixel of each frame. Picked by the package author, 2026-11-12:
// 0.2 makes 1080x1920 at 24 a second about 10 Mbit/s, above what
// YouTube asks for 1080p uploads at that rate (8 Mbit/s).
const bitsPerPixel = 0.2

// audioBitRate is the AAC bit rate, in bits a second for both channels.
// Picked by the package author, 2026-11-12: YouTube asks for 128 kbit/s
// stereo or more; 192 leaves room.
const audioBitRate = 192000

// channels is the channels a Sound has: left and right. Defined.
const channels = 2

// mono is one channel, which ReadSound puts on both. Defined.
const mono = 1

// readRate is the sample rate ReadSound returns. Defined: 44100 is what
// Slayer's synth makes, the compact disc's rate.
const readRate = 44100

// soundRates are the sample rates a Sound may have. Defined: the two
// AAC-LC rates in common use, the compact disc's and video's.
var soundRates = []int{readRate, 48000}

// fullScale is the largest a sample may be either side of silence.
// Defined.
const fullScale = 1.0

// bytesPerPixel is the bytes of one RGBA pixel. Defined by image.RGBA.
const bytesPerPixel = 4

// waitPause is how long the writer sleeps between looks at whether the
// encoder will take more, so a wait never spins a core. Picked by the
// package author, 2026-11-12: two milliseconds is a tenth of a frame at
// 48 a second.
const waitPause = 2 * time.Millisecond

// waitLimit is the longest the writer waits for the encoder to take one
// more frame, or for the file to finish, before it gives up and says
// so. Picked by the package author, 2026-11-12: a 4K frame encodes in
// tens of milliseconds here; a minute is a stuck encoder.
const waitLimit = time.Minute

// whyLength is the room for the bridge's reason a call failed, in bytes.
// Picked by the package author, 2026-11-12, as the Metal bridge's.
const whyLength = 1024

// pair is what an even number divides by. Defined: H.264's 4:2:0
// colour is one sample for each two pixels across and down.
const pair = 2

// mostRate is the most frames a second offered. Picked by the package
// author, 2026-11-12, as the compositor's encoder has it: past 240 is a
// typo, and a frame would hold less than one sample of sound.
const mostRate = 240

// mostSide is the longest side, in pixels, a film's pictures are read
// at. Picked by the reader's author, 2026-11-12: twice the widest film
// read here (4096), so a larger ask is a typo, not a film.
const mostSide = 8192

// seekSlack is how far before a picture's own time, in seconds, a moment
// asked for still names that picture: a moment given in whole
// nanoseconds is cut short by up to one, and a picture's time comes back
// as a fraction of the file's own clock. A microsecond is far above both
// and far below any frame.
const seekSlack = 1e-6
