//go:build draw && darwin && cgo

package movie

import (
	"context"
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// the colour tags every movie must carry, in AVFoundation's words
const bt709Name = "ITU_R_709_2"

// highProfile is H.264's profile_idc for High. Defined by ITU-T H.264,
// annex A.2.4.
const highProfile = 100

// quarters is how the tests cut a sound to look at its parts. Defined.
const quarters = 4

// blueStep is how far blue moves from one frame to the next. Picked by
// the test author, 2026-11-12, so the frames differ plainly.
const blueStep = 8

// gradient is a frame that moves with its index, so no two are alike.
func gradient(size image.Point, index int) *image.RGBA {
	picture := image.NewRGBA(image.Rectangle{Max: size})
	for y := range size.Y {
		for x := range size.X {
			at := picture.PixOffset(x, y)
			picture.Pix[at] = uint8(x + index)
			picture.Pix[at+1] = uint8(y)
			picture.Pix[at+2] = uint8(index * blueStep)
			picture.Pix[at+3] = math.MaxUint8
		}
	}
	return picture
}

// opened is a writer for the request, or the test stops.
func opened(t *testing.T, request Request) *Writer {
	t.Helper()
	writer, err := New(context.Background(), mustSettings(t, request))
	if err != nil {
		t.Fatal(err)
	}
	return writer
}

// feed hands the writer frames from the first up to the last, not
// including it, or the test stops.
func feed(t *testing.T, writer *Writer, first, last int) {
	t.Helper()
	size := writer.Settings().Picture()
	for i := first; i < last; i++ {
		if err := writer.Frame(i, gradient(size, i)); err != nil {
			t.Fatal(err)
		}
	}
}

// written writes the whole request, closes it, and reads it back.
func written(t *testing.T, request Request) Facts {
	t.Helper()
	writer := opened(t, request)
	feed(t, writer, 0, request.Frames)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	facts, err := Probe(request.Path)
	if err != nil {
		t.Fatal(err)
	}
	return facts
}

// a written file probes back with the size, frame count, rate, codecs
// and both tracks that were asked for, and meets every rule.
func TestAWrittenMovieProbesBack(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 1)
	facts := written(t, request)
	if facts.Width() != request.Width || facts.Height() != request.Height ||
		facts.Frames() != request.Frames ||
		facts.Rate() != float64(request.Rate) ||
		facts.Duration() != time.Second {
		t.Errorf("%dx%d, %d frames at %v, %v; asked %+v", facts.Width(),
			facts.Height(), facts.Frames(), facts.Rate(), facts.Duration(),
			request)
	}
	if facts.VideoCodec() != "h264" || facts.Profile() != highProfile ||
		facts.Level() > highestLevel || !facts.FastStart() {
		t.Errorf("%s profile %d level %d, fast start %v",
			facts.VideoCodec(), facts.Profile(), facts.Level(),
			facts.FastStart())
	}
	tags := []string{facts.Primaries(), facts.Transfer(), facts.Matrix()}
	for _, tag := range tags {
		if tag != bt709Name {
			t.Errorf("colour tagged %v, want %s throughout", tags, bt709Name)
		}
	}
	if facts.AudioCodec() != "aac" || facts.AudioRate() != readRate ||
		facts.AudioChannels() != channels ||
		!near(facts.AudioDuration(), time.Second, readRate) {
		t.Errorf("audio %q at %v, %d channels, %v", facts.AudioCodec(),
			facts.AudioRate(), facts.AudioChannels(), facts.AudioDuration())
	}
}

// the anti-test: with no sound there is no audio track at all, and
// asking it for sound says so.
func TestNoSoundIsNoAudioTrack(t *testing.T) {
	request := smallRequest(t)
	facts := written(t, request)
	if facts.AudioCodec() != "" || facts.AudioChannels() != 0 {
		t.Errorf("a silent request has audio %q, %d channels",
			facts.AudioCodec(), facts.AudioChannels())
	}
	_, err := ReadSound(request.Path)
	if err == nil || !strings.Contains(err.Error(), "no sound") {
		t.Errorf("reading sound from a silent movie: %v", err)
	}
}

// the keyframe rule holds past two runs of 48, and frames between are
// not all keyframes, so the count is of something.
func TestKeyframesComeEvery48AtMost(t *testing.T) {
	request := smallRequest(t)
	request.Frames = picked("long-frames")
	facts := written(t, request)
	if facts.KeyframeGap() > keyframesWithin || facts.KeyframeGap() < pair {
		t.Errorf("keyframes %d frames apart, want 2 to %d",
			facts.KeyframeGap(), keyframesWithin)
	}
}

// sound longer than the picture is cut at the picture's end.
func TestLongerSoundIsCut(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 2)
	facts := written(t, request)
	if !near(facts.AudioDuration(), time.Second, readRate) {
		t.Errorf("two seconds of sound under one of picture: %v",
			facts.AudioDuration())
	}
}

// shorter sound is followed by silence to the picture's end.
func TestShorterSoundIsFollowedBySilence(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 0.5)
	facts := written(t, request)
	if !near(facts.AudioDuration(), time.Second, readRate) {
		t.Errorf("half a second of sound under one of picture: %v",
			facts.AudioDuration())
	}
	read, err := ReadSound(request.Path)
	if err != nil {
		t.Fatal(err)
	}
	quarter := readRate / quarters * channels
	head := loudness(read.samples[:quarter])
	tail := loudness(read.samples[len(read.samples)-quarter:])
	silence := pickedValue("silence-loudness")
	if tail > silence || head <= silence {
		t.Errorf("first quarter %v loud, last quarter %v", head, tail)
	}
}

// loudness is the root mean square of samples.
func loudness(samples []float32) float64 {
	var sum float64
	for _, sample := range samples {
		sum += float64(sample) * float64(sample)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

// worstAligned is the largest difference between the written sound and
// the read one over the middle half, at the shift that matches best.
// AAC delays and pads the sound, so the read one may be shifted.
func worstAligned(written, read []float32, reach int) (int, float64) {
	first := len(written) / quarters
	last := len(written) - first
	best, bestShift := math.Inf(1), 0
	for shift := -reach; shift <= reach; shift++ {
		var sum float64
		for i := first; i < last; i += channels {
			at := i + shift*channels
			if at < 0 || at+1 >= len(read) {
				sum = math.Inf(1)
				break
			}
			sum += math.Abs(float64(written[i] - read[at]))
		}
		if sum < best {
			best, bestShift = sum, shift
		}
	}
	var worst float64
	for i := first; i < last; i++ {
		difference := float64(written[i] - read[i+bestShift*channels])
		worst = math.Max(worst, math.Abs(difference))
	}
	return bestShift, worst
}

// ReadSound gives back what a written file's audio holds, on the
// channel it was written on, within what AAC's loss allows, and at the
// instant it was written: AAC's priming is not left at the front, so
// the sound keeps time with the pictures.
func TestReadSoundGivesBackWhatWasWritten(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 1)
	written(t, request)
	read, err := ReadSound(request.Path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Rate() != readRate ||
		!near(read.Duration(), time.Second, readRate) {
		t.Errorf("read %v at %d", read.Duration(), read.Rate())
	}
	shift, worst := worstAligned(request.Sound.samples, read.samples,
		picked("aac-delay"))
	t.Logf("aligned %d samples on, the worst difference is %v",
		shift, worst)
	if shift != 0 || worst > pickedValue("aac-error") {
		t.Errorf("read back %d samples late and %v away from what was "+
			"written", shift, worst)
	}
}

// the anti-test: a sound with the channels swapped does not match, so
// the comparison can tell the channels apart.
func TestSwappedChannelsDoNotMatch(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 1)
	written(t, request)
	read, err := ReadSound(request.Path)
	if err != nil {
		t.Fatal(err)
	}
	swapped := make([]float32, len(read.samples))
	for i := 0; i+1 < len(swapped); i += channels {
		swapped[i], swapped[i+1] = read.samples[i+1], read.samples[i]
	}
	_, worst := worstAligned(request.Sound.samples, swapped,
		picked("aac-delay"))
	if worst <= pickedValue("aac-error") {
		t.Errorf("swapped channels match within %v", worst)
	}
}

// with padding a 320x180 picture is written 320x192, and says so.
func TestAPaddedPictureIsWrittenTaller(t *testing.T) {
	request := smallRequest(t)
	request.Height = picked("short-height")
	request.Pad = true
	facts := written(t, request)
	if facts.Height() != picked("small-height") ||
		facts.Width() != request.Width {
		t.Errorf("padded to %dx%d", facts.Width(), facts.Height())
	}
}

// a written movie is counted, and Replace writes over a file there.
func TestStatsCountAndReplaceReplaces(t *testing.T) {
	request := smallRequest(t)
	if err := os.WriteFile(request.Path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	request.Replace = true
	writer := opened(t, request)
	feed(t, writer, 0, request.Frames)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(request.Path)
	if err != nil {
		t.Fatal(err)
	}
	stats := writer.Stats()
	if stats.Frames() != request.Frames || stats.Bytes() != info.Size() ||
		stats.Spent() <= 0 {
		t.Errorf("stats %+v for a file of %d", stats, info.Size())
	}
	if names := leftIn(t, request.Path); len(names) != 1 {
		t.Errorf("left beside the movie: %v", names)
	}
	if err := writer.Close(); err != nil || writer.Abort() != nil {
		t.Errorf("a second Close or an Abort after it: %v", err)
	}
}

// each refused frame is refused by name, fails the writer and leaves
// nothing, and Close then gives the same refusal.
func TestFramesAreRefusedByName(t *testing.T) {
	size := image.Pt(picked("small-width"), picked("small-height"))
	tall := image.Pt(size.X, size.Y+1)
	short := gradient(size, 0)
	short.Pix = short.Pix[:len(short.Pix)-1]
	cases := []struct {
		index   int
		picture *image.RGBA
		says    string
	}{
		{index: 0, picture: gradient(tall, 0), says: "the wrong size"},
		{index: 0, picture: nil, says: "no picture"},
		{index: 0, picture: short, says: "too few"},
		{index: 1, picture: gradient(size, 1), says: "out of order"},
		{index: picked("small-frames"), picture: gradient(size, 0),
			says: "more frames than the count"},
	}
	for _, tried := range cases {
		request := smallRequest(t)
		writer := opened(t, request)
		err := writer.Frame(tried.index, tried.picture)
		if err == nil || !strings.Contains(err.Error(), tried.says) {
			t.Errorf("frame %d: %v, want %q", tried.index, err, tried.says)
		}
		nothingLeft(t, request.Path)
		again := writer.Frame(0, gradient(size, 0))
		if !errors.Is(writer.Close(), err) || !errors.Is(again, err) {
			t.Errorf("after %q the writer took more or closed", tried.says)
		}
		nothingLeft(t, request.Path)
	}
}

// a frame after Close is refused by name.
func TestAFrameAfterCloseIsRefused(t *testing.T) {
	request := smallRequest(t)
	writer := opened(t, request)
	feed(t, writer, 0, request.Frames)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	err := writer.Frame(request.Frames, gradient(image.Pt(request.Width,
		request.Height), 0))
	if err == nil || !strings.Contains(err.Error(), "after Close") {
		t.Errorf("a frame after Close: %v", err)
	}
}

// Abort leaves no file and no temporary file, and Close after it says
// it was aborted.
func TestAbortLeavesNothing(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 1)
	writer := opened(t, request)
	feed(t, writer, 0, picked("early-frames"))
	if err := writer.Abort(); err != nil {
		t.Fatal(err)
	}
	nothingLeft(t, request.Path)
	if err := writer.Close(); !errors.Is(err, errAborted) {
		t.Errorf("Close after Abort: %v", err)
	}
}

// a cancelled context stops the writer: the next frame is refused,
// Close fails, and nothing is left.
func TestACancelledContextLeavesNothing(t *testing.T) {
	request := smallRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	writer, err := New(ctx, mustSettings(t, request))
	if err != nil {
		t.Fatal(err)
	}
	early := picked("early-frames")
	feed(t, writer, 0, early)
	cancel()
	size := writer.Settings().Picture()
	err = writer.Frame(early, gradient(size, early))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a frame after cancelling: %v", err)
	}
	if err := writer.Close(); !errors.Is(err, context.Canceled) {
		t.Errorf("Close after cancelling: %v", err)
	}
	nothingLeft(t, request.Path)
}

// the near miss: cancelled after the last frame and before Close, the
// movie is still not published.
func TestCancelledBeforeCloseLeavesNothing(t *testing.T) {
	request := smallRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	writer, err := New(ctx, mustSettings(t, request))
	if err != nil {
		t.Fatal(err)
	}
	feed(t, writer, 0, request.Frames)
	cancel()
	if err := writer.Close(); !errors.Is(err, context.Canceled) {
		t.Errorf("Close after cancelling: %v", err)
	}
	nothingLeft(t, request.Path)
}

// a context cancelled before New starts nothing.
func TestACancelledContextStartsNothing(t *testing.T) {
	request := smallRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(ctx, mustSettings(t, request)); err == nil {
		t.Errorf("New with a cancelled context started")
	}
	var none context.Context
	if _, err := New(none, mustSettings(t, request)); err == nil {
		t.Errorf("New with no context started")
	}
	if _, err := New(context.Background(), Settings{}); err == nil {
		t.Errorf("New with unchecked settings started")
	}
	nothingLeft(t, request.Path)
}

// Close with fewer frames than the count is an error and leaves nothing.
func TestClosingEarlyLeavesNothing(t *testing.T) {
	request := smallRequest(t)
	writer := opened(t, request)
	feed(t, writer, 0, picked("early-frames"))
	err := writer.Close()
	if err == nil || !strings.Contains(err.Error(), "nothing was published") {
		t.Errorf("closing early: %v", err)
	}
	nothingLeft(t, request.Path)
}

// a file that appears at the path while the movie is written, without
// Replace, is not written over: Close says so and leaves it alone.
func TestAFileThatAppearedIsNotLost(t *testing.T) {
	request := smallRequest(t)
	writer := opened(t, request)
	feed(t, writer, 0, request.Frames)
	kept := []byte("kept")
	if err := os.WriteFile(request.Path, kept, 0o600); err != nil {
		t.Fatal(err)
	}
	err := writer.Close()
	if err == nil || !strings.Contains(err.Error(), "publishing") {
		t.Errorf("publishing over a file that appeared: %v", err)
	}
	held, readErr := os.ReadFile(request.Path)
	if readErr != nil || string(held) != string(kept) {
		t.Errorf("the file that appeared now holds %q, %v", held, readErr)
	}
	if names := leftIn(t, request.Path); len(names) != 1 {
		t.Errorf("left beside it: %v", names)
	}
}

// a directory that goes away after the settings are checked is said
// so by New, which starts nothing.
func TestADirectoryThatWentIsRefusedByNew(t *testing.T) {
	request := smallRequest(t)
	request.Path = filepath.Join(t.TempDir(), "going", movieName)
	if err := os.Mkdir(filepath.Dir(request.Path), 0o700); err != nil {
		t.Fatal(err)
	}
	settings := mustSettings(t, request)
	if err := os.Remove(filepath.Dir(request.Path)); err != nil {
		t.Fatal(err)
	}
	_, err := New(context.Background(), settings)
	if err == nil || !strings.Contains(err.Error(), "temporary file") {
		t.Errorf("New into a directory that went: %v", err)
	}
}

// an encoder that has stopped writing refuses the next frame, and the
// writer fails, says why and leaves nothing.
func TestAStoppedEncoderFailsTheFrame(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 1)
	writer := opened(t, request)
	feed(t, writer, 0, picked("early-frames"))
	writer.pipe.cancel()
	size := writer.Settings().Picture()
	early := picked("early-frames")
	err := writer.Frame(early, gradient(size, early))
	if err == nil || !strings.Contains(err.Error(), "stopped writing") {
		t.Errorf("a frame to a stopped encoder: %v", err)
	}
	if !errors.Is(writer.Close(), err) {
		t.Errorf("Close did not give the encoder's failure")
	}
	nothingLeft(t, request.Path)
}

// an encoder that has stopped writing fails Close, which publishes
// nothing.
func TestAStoppedEncoderFailsClose(t *testing.T) {
	request := smallRequest(t)
	writer := opened(t, request)
	feed(t, writer, 0, request.Frames)
	writer.pipe.cancel()
	if err := writer.Close(); err == nil {
		t.Errorf("Close published from a stopped encoder")
	}
	nothingLeft(t, request.Path)
}

// a short, full size and with sound, finishes and probes back whole.
// Handed over in step from one thread, sound and pictures stalled here
// at frame 57; the small movies never showed it.
func TestAShortWithSoundFinishes(t *testing.T) {
	request := smallRequest(t)
	request.Width = picked("shorts-width")
	request.Height = picked("shorts-height")
	request.Frames = picked("shorts-frames")
	seconds := float64(request.Frames) / float64(request.Rate)
	request.Sound = tone(t, readRate, seconds)
	writer := opened(t, request)
	size := writer.Settings().Picture()
	pictures := []*image.RGBA{gradient(size, 0), gradient(size, 1)}
	for i := range request.Frames {
		if err := writer.Frame(i, pictures[i%len(pictures)]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	facts, err := Probe(request.Path)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Frames() != request.Frames || facts.AudioCodec() != "aac" ||
		facts.Level() != writer.Settings().level.idc {
		t.Errorf("%d frames, audio %q, level %d", facts.Frames(),
			facts.AudioCodec(), facts.Level())
	}
}
