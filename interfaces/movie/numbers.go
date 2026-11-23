package movie

import (
	"math"
	"time"
)

// The numbers the package works in that are true by definition. Each says
// where it is defined. A number that was chosen or measured is not here:
// it is in picks.yaml, and picks.go reads it.

// zero is no count, and the first place. Defined.
const zero = 0

// one is a single count, the step to the next place, and the numerator of
// a reciprocal. Defined.
const one = 1

// decimal is the base counts are written in. Defined.
const decimal = 10

// macroblock is the side of the square H.264 codes a picture in, in
// pixels. Defined by ITU-T H.264, clause 6.3.
const macroblock = 16

// levelTenths is how level_idc counts: ten to a level. Defined by
// ITU-T H.264, clause 7.4.2.1.1.
const levelTenths = 10

// highBitRate is the High profile's allowance over table A-1's MaxBR,
// in thousandths. Defined by ITU-T H.264 table A-2, cpbBrVclFactor 1250
// against Baseline's 1000.
const highBitRate = 1250

// frameAreaRoot is the factor under the square root that bounds each
// side of a picture in macroblocks: no side exceeds sqrt(8 * MaxFS).
// Defined by ITU-T H.264, clause A.3.1 f and g.
const frameAreaRoot = 8

// channels is the channels a Sound has: left and right. Defined.
const channels = 2

// mono is one channel, which ReadSound puts on both. Defined.
const mono = 1

// readRate is the sample rate ReadSound returns. Defined: 44100 is what
// Slayer's synth makes, the compact disc's rate.
const readRate = 44100

// videoRate is the sample rate of video, the second AAC-LC rate in
// common use. Defined.
const videoRate = 48000

// soundRates are the sample rates a Sound may have. Defined: the two
// AAC-LC rates in common use, the compact disc's and video's.
var soundRates = []int{readRate, videoRate}

// fullScale is the largest a sample may be either side of silence.
// Defined.
const fullScale = 1.0

// bytesPerPixel is the bytes of one RGBA pixel. Defined by image.RGBA.
const bytesPerPixel = 4

// pair is what an even number divides by. Defined: H.264's 4:2:0
// colour is one sample for each two pixels across and down.
const pair = 2

// mostFrames is the most frames a movie may hold: past it, the movie's
// length is more nanoseconds than a time.Duration counts. Defined by
// time.Duration, an int64 count of nanoseconds.
const mostFrames = math.MaxInt64 / int64(time.Second)

// headingRows is the rows at the top of levels.csv that name its
// columns. Defined by the file.
const headingRows = 1

// nameColumn is the column of levels.csv that holds a level's name.
// Defined by the file's heading.
const nameColumn = 0

// firstCount is the column of levels.csv that holds the first count, the
// one after the level's name. Defined by the file's heading.
const firstCount = 1

// positive is whether a count or a rate is above nought.
func positive[Number int | int64 | float64](value Number) bool {
	return value > zero
}
