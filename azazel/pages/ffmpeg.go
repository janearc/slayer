package pages

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// errNoFfmpeg is the failure of an encoder that has no ffmpeg to run.
var errNoFfmpeg = errors.New("ffmpeg was not found on the path, " +
	"so no partial film is made")

// errNoFfprobe is the failure of an encoder that cannot check its film.
var errNoFfprobe = errors.New("ffprobe was not found on the path, so the " +
	"film's sound cannot be checked and no partial film is made")

// framePattern is ffmpeg's name for the numbered frames.
const framePattern = "frame-%06d.png" // defined: slayer's frame names

// firstFrame is the number of a film's first frame.
const firstFrame = "0" // defined: a film begins at frame-000000.png

// padding makes the width even and the height a multiple of 16, padded
// with black, in BT.709 at television range, as QuickTime plays it. The
// frames are tagged with BT.709's primaries and transfer too, because the
// output flags alone leave both untagged, which ffprobe shows.
const padding = "pad=ceil(iw/2)*2:ceil(ih/16)*16:0:(oh-ih)/2:black," +
	"scale=iw:ih:out_color_matrix=bt709:out_range=tv,format=yuv420p," +
	"setparams=color_primaries=bt709:color_trc=bt709:colorspace=bt709:" +
	"range=tv"

// The rates the film is encoded at.
// picked: the omnipositor encoder's, for films up to 2048 wide.
const (
	bitrate    = "20M"
	maxrate    = "30M"
	bufferSize = "40M"
	keyEvery   = "48"
)

// audioRate is the rate the music is encoded at, in aac.
const audioRate = "192k" // picked: transparent for a review copy of music

// fadeSeconds is how long the music fades out where the partial film cuts
// it. picked: half a second ends the cut without a click, and is short
// enough that what plays up to the last frame is heard as it is.
const fadeSeconds = 0.5

// Ffmpeg makes partial films with the ffmpeg at Path, in hevc on
// VideoToolbox, tagged hvc1, with the music in aac, and checks each with
// the ffprobe at Probe before it is published.
type Ffmpeg struct {
	Path  string
	Probe string
}

// FoundFfmpeg is the encoder of the ffmpeg and ffprobe on the path. One
// not found is left empty, and Encode then says so.
func FoundFfmpeg() Ffmpeg {
	ffmpeg, _ := exec.LookPath("ffmpeg")
	ffprobe, _ := exec.LookPath("ffprobe")
	return Ffmpeg{Path: ffmpeg, Probe: ffprobe}
}

// Encode runs ffmpeg once over the frames and the music, then refuses a
// film whose sound ffprobe does not find whole.
func (encoder Ffmpeg) Encode(ctx context.Context, film Film) error {
	if encoder.Path == "" {
		return errNoFfmpeg
	}
	if encoder.Probe == "" {
		return errNoFfprobe
	}
	told, err := exec.CommandContext(ctx, encoder.Path,
		ffmpegArguments(film)...).CombinedOutput()
	if err != nil {
		toldText := strings.TrimSpace(string(told))
		return fmt.Errorf("ffmpeg: %w: %s", err,
			toldText)
	}
	return checkSound(ctx, encoder.Probe, film)
}

// filmLength is how long a film of count frames at a rate lasts, in
// seconds.
func filmLength(count int, rate float64) float64 {
	return float64(count) / rate
}

// number is a number as ffmpeg reads it.
func number(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// ffmpegArguments are ffmpeg's arguments for one partial film: the first
// frames from frame-000000.png at the film's rate, and the music from its
// start, cut to the same length with a fade, into an mp4.
func ffmpegArguments(film Film) []string {
	long := filmLength(film.Count, film.Rate)
	fade := fmt.Sprintf("afade=t=out:st=%s:d=%s",
		number(math.Max(0, long-fadeSeconds)), number(fadeSeconds))
	return []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-framerate", number(film.Rate), "-start_number", firstFrame,
		"-i", filepath.Join(film.Frames, framePattern), "-i", film.Music,
		"-map", "0:v:0", "-map", "1:a:0", "-frames:v",
		strconv.Itoa(film.Count), "-t", number(long), "-vf", padding,
		"-af", fade,
		"-c:v", "hevc_videotoolbox", "-profile:v", "main", "-tag:v", "hvc1",
		"-color_primaries", "bt709", "-color_trc", "bt709",
		"-colorspace", "bt709", "-color_range", "tv",
		"-b:v", bitrate, "-maxrate", maxrate, "-bufsize", bufferSize,
		"-g", keyEvery, "-c:a", "aac", "-b:a", audioRate,
		"-movflags", "+faststart", "-f", "mp4", film.Output,
	}
}

// streams is what ffprobe says of a film's streams: their kinds and how
// long each lasts.
type streams struct {
	Streams []struct {
		Kind     string `json:"codec_type"`
		Duration string `json:"duration"`
	} `json:"streams"`
}

// checkSound refuses a film without exactly one audio stream, or whose
// sound and picture differ in length by more than one frame.
func checkSound(ctx context.Context, probe string, film Film) error {
	told, err := exec.CommandContext(ctx, probe, "-v", "error",
		"-show_entries", "stream=codec_type,duration", "-of", "json",
		film.Output).Output()
	var found streams
	if err == nil {
		err = json.Unmarshal(told, &found)
	}
	if err != nil {
		return fmt.Errorf("ffprobe could not read the partial film: %w", err)
	}
	lengths := map[string][]float64{}
	for _, stream := range found.Streams {
		length, _ := strconv.ParseFloat(stream.Duration, 64)
		lengths[stream.Kind] = append(lengths[stream.Kind], length)
	}
	sound, picture := lengths["audio"], lengths["video"]
	if len(sound) != 1 || len(picture) != 1 ||
		math.Abs(sound[0]-picture[0]) > 1/film.Rate {
		return fmt.Errorf("the partial film's sound is not whole (audio %v, "+
			"video %v seconds), so it is not published", sound, picture)
	}
	return nil
}
