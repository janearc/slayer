package movie

import (
	_ "embed"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
)

// picksRaw is picks.yaml, the numbers the package chose.
//
//go:embed picks.yaml
var picksRaw []byte

// book is the picked numbers, read once when the package starts.
var book = readBook()

// readBook reads picks.yaml. A file that does not read is a fault in the
// package, so it stops the program at its start and says why.
func readBook() numbers.Numbers {
	read, err := numbers.Read("picks.yaml", picksRaw)
	if err != nil {
		panic(err)
	}
	return read
}

// choice is the value of a picked number. A name the file lacks is a
// fault in the package, so it stops the program and says which.
func choice(name string) float64 {
	number, err := book.Get(name)
	if err != nil {
		panic(err)
	}
	return number.Value
}

// whole is the value of a picked number that is a count.
func whole(name string) int { return int(choice(name)) }

// keyframesWithin is the most frames between one keyframe and the next.
var keyframesWithin = whole("keyframes-within")

// highestLevel is the highest H.264 level written, as level_idc, ten
// times the level.
var highestLevel = whole("highest-level")

// bitsPerPixel is the video bit rate asked of the encoder, in bits for
// each pixel of each frame.
var bitsPerPixel = choice("bits-per-pixel")

// audioBitRate is the AAC bit rate, in bits a second for both channels.
var audioBitRate = whole("audio-bit-rate")

// waitPause is how long the writer sleeps between looks at whether the
// encoder will take more, so a wait never spins a core.
var waitPause = time.Duration(whole("wait-pause-milliseconds")) *
	time.Millisecond

// waitLimit is the longest the writer waits for the encoder to take one
// more frame, or for the file to finish, before it gives up and says so.
var waitLimit = time.Duration(whole("wait-limit-seconds")) * time.Second

// whyLength is the room for the bridge's reason a call failed, in bytes.
var whyLength = whole("why-length")

// mostRate is the most frames a second offered.
var mostRate = whole("most-rate")

// mostSide is the longest side, in pixels, a film's pictures are read at.
var mostSide = whole("most-side")

// mostPictureSide is the longest side, in pixels, a request may ask for.
var mostPictureSide = whole("most-picture-side")

// seekSlack is how far before a picture's own time, in seconds, a moment
// asked for still names that picture.
var seekSlack = choice("seek-slack")

// ticksPerFrame is how finely the video track counts time, in ticks a
// frame.
var ticksPerFrame = whole("ticks-per-frame")

// soundChunk is how many stereo pairs are handed to the writer at a time.
var soundChunk = whole("sound-chunk")
