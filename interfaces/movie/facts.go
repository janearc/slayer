package movie

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// Facts is what Probe read back from a finished file.
//
//ninja:vow practice Probe makes one; tests and callers read it
type Facts struct {
	width, height int
	frames        int
	rate          float64
	duration      time.Duration
	video         string
	profile       int
	level         int
	primaries     string
	transfer      string
	matrix        string
	keyframeGap   int
	fastStart     bool
	audio         string
	audioRate     float64
	audioChannels int
	audioDuration time.Duration
}

// Width is the coded picture's width, in pixels.
func (facts Facts) Width() int { return facts.width }

// Height is the coded picture's height, padding and all, in pixels.
func (facts Facts) Height() int { return facts.height }

// Frames is how many pictures the video track holds, counted one by one.
func (facts Facts) Frames() int { return facts.frames }

// Rate is the video track's frames a second, as the file states it.
func (facts Facts) Rate() float64 { return facts.rate }

// Duration is how long the movie plays.
func (facts Facts) Duration() time.Duration { return facts.duration }

// VideoCodec is the video's codec: h264, or the file's own four letters.
func (facts Facts) VideoCodec() string { return facts.video }

// Profile is the H.264 profile_idc from the stream's configuration:
// 100 is High. Nought when the video is not H.264.
func (facts Facts) Profile() int { return facts.profile }

// Level is the H.264 level_idc: ten times the level, 40 for 4.0.
func (facts Facts) Level() int { return facts.level }

// Primaries is the colour primaries the video is tagged with, in
// AVFoundation's words, ITU_R_709_2 for BT.709; empty when untagged.
func (facts Facts) Primaries() string { return facts.primaries }

// Transfer is the transfer function the video is tagged with.
func (facts Facts) Transfer() string { return facts.transfer }

// Matrix is the colour matrix the video is tagged with.
func (facts Facts) Matrix() string { return facts.matrix }

// KeyframeGap is the most frames from one keyframe to the next, or to
// the end after the last.
func (facts Facts) KeyframeGap() int { return facts.keyframeGap }

// FastStart is whether the index comes before the media, so playback
// can start before the whole file has arrived.
func (facts Facts) FastStart() bool { return facts.fastStart }

// AudioCodec is the audio's codec, aac, or empty with no audio track.
func (facts Facts) AudioCodec() string { return facts.audio }

// AudioRate is the audio's samples a second; nought with no audio.
func (facts Facts) AudioRate() float64 { return facts.audioRate }

// AudioChannels is the audio's channels; nought with no audio.
func (facts Facts) AudioChannels() int { return facts.audioChannels }

// AudioDuration is how long the audio track plays.
func (facts Facts) AudioDuration() time.Duration {
	return facts.audioDuration
}

// codecNames are the codecs' common names by their four letters in the
// file. Defined by ISO/IEC 14496-15 (avc1) and Apple's Core Audio
// format identifiers ('aac ').
var codecNames = map[string]string{"avc1": "h264", "aac ": "aac"}

// codecName is a codec's common name, or its four letters as written.
func codecName(fourLetters string) string {
	if name, known := codecNames[fourLetters]; known {
		return name
	}
	return fourLetters
}

// The bytes before every box in an MPEG-4 file: a 32-bit size, then
// four letters naming the kind. Defined by ISO/IEC 14496-12, 4.2.
const (
	boxSize   = 4
	boxKind   = 4
	boxHeader = boxSize + boxKind
)

// boxWide is the 64-bit size that follows a header whose size is one.
// Defined by ISO/IEC 14496-12, clause 4.2.
const boxWide = 8

// Box sizes with a meaning of their own. Defined by ISO/IEC 14496-12,
// clause 4.2.
const (
	boxToEnd  = 0 // the box runs to the end of the file
	boxLarger = 1 // the real size follows, in 64 bits
)

// fastStart reads the boxes at the top of a file and says whether the
// index, moov, comes before the media, mdat.
func fastStart(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("movie: %w", err)
	}
	defer file.Close()
	for {
		kind, err := nextBox(file)
		if errors.Is(err, io.EOF) {
			return false, fmt.Errorf("movie: %s has no moov or no "+
				"mdat: it is not a finished mp4", path)
		}
		if err != nil {
			return false, fmt.Errorf("movie: %s: %w", path, err)
		}
		if kind == "moov" || kind == "mdat" {
			return kind == "moov", nil
		}
	}
}

// nextBox reads the header of the box the file is at, then moves past
// the box, and says what kind it was.
func nextBox(file io.ReadSeeker) (string, error) {
	var header [boxHeader]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return "", err
	}
	size := int64(binary.BigEndian.Uint32(header[:]))
	kind := string(header[boxSize:])
	read := int64(boxHeader)
	switch size {
	case boxToEnd:
		return kind, nil
	case boxLarger:
		var wide [boxWide]byte
		if _, err := io.ReadFull(file, wide[:]); err != nil {
			return "", err
		}
		size = int64(binary.BigEndian.Uint64(wide[:]))
		read += boxWide
	}
	if size < read {
		return "", fmt.Errorf("a %q box says it is %d bytes", kind, size)
	}
	_, err := file.Seek(size-read, io.SeekCurrent)
	return kind, err
}
