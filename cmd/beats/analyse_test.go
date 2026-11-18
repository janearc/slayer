package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/beats"
)

// writeClicks writes a stereo sixteen-bit recording of 44100 samples a
// second to a path: a click every half second from the first second on,
// seconds long, with the second channel opposite to the first. A stem
// of the same length is made by asking for the same seconds.
func writeClicks(t *testing.T, path string, seconds int) {
	t.Helper()
	const rate = 44100
	var data bytes.Buffer
	for i := 0; i < rate*seconds; i++ {
		var sample int16
		since := i % (rate / 2)
		if i >= rate && since < 662 {
			sample = 26000
		}
		binary.Write(&data, binary.LittleEndian, sample)
		binary.Write(&data, binary.LittleEndian, -sample)
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(36+data.Len()))
	out.WriteString("WAVEfmt ")
	for _, field := range []any{uint32(16), uint16(1), uint16(2),
		uint32(rate), uint32(rate * 4), uint16(4), uint16(16)} {
		binary.Write(&out, binary.LittleEndian, field)
	}
	out.WriteString("data")
	binary.Write(&out, binary.LittleEndian, uint32(data.Len()))
	out.Write(data.Bytes())
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// analyseRun runs the analyse verb on arguments and gives the exit code
// and what it wrote to its output and error streams.
func analyseRun(args ...string) (int, string, string) {
	var out, errs bytes.Buffer
	code := run(append([]string{"analyse"}, args...), strings.NewReader(""),
		&out, &errs)
	return code, out.String(), errs.String()
}

// A click track at 120 beats a minute is analysed to a score that keeps
// the audio's path, its hash and its length, marks itself estimated, and
// holds beats half a second apart.
func TestAnalyseWritesAnEstimatedScoreOfTheClickTrack(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "track.wav")
	writeClicks(t, audio, 8)
	out := filepath.Join(dir, "score.yaml")
	code, said, errs := analyseRun(audio, "--out", out, "--tempo", "120",
		"--meter", "3")
	if code != 0 {
		t.Fatalf("analyse exited %d: %s", code, errs)
	}
	if !strings.Contains(said, "estimated beats over 8.000 seconds") ||
		!strings.HasSuffix(strings.TrimSpace(said), out) {
		t.Errorf("analyse said %q, want the beat count, 8.000 seconds and "+
			"the score's path", said)
	}
	grid, err := beats.Read(out)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(audio)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%x", sha256.Sum256(raw))
	if grid.Score.SHA256 != want || grid.Score.Audio != audio {
		t.Errorf("the score names %s with hash %s, want %s with %s",
			grid.Score.Audio, grid.Score.SHA256, audio, want)
	}
	if math.Abs(grid.Score.Duration-8) > 0.001 {
		t.Errorf("the score's duration is %v, want 8", grid.Score.Duration)
	}
	if grid.Score.Status != "estimated" || grid.Score.Analysis != nil {
		t.Errorf("the score is %q with analysis %v, want estimated and "+
			"none", grid.Score.Status, grid.Score.Analysis)
	}
	times := grid.Score.Beats.Times
	if len(times) < 10 {
		t.Fatalf("the score holds %d beats, want at least 10", len(times))
	}
	for i := 1; i < len(times); i++ {
		if gap := times[i] - times[i-1]; math.Abs(gap-0.5) > 0.02 {
			t.Errorf("beats %d and %d are %v apart, want 0.5", i-1, i, gap)
		}
	}
}

// A drum stem of the audio's length is analysed in its place and kept in
// the score with its own hash; the method says so.
func TestAnalyseUsesADrumStemAndRecordsIt(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "track.wav")
	stem := filepath.Join(dir, "drums.wav")
	writeClicks(t, audio, 6)
	writeClicks(t, stem, 6)
	out := filepath.Join(dir, "score.yaml")
	code, _, errs := analyseRun(audio, "--out", out, "--drums", stem,
		"--tempo", "120", "--first", "1")
	if code != 0 {
		t.Fatalf("analyse exited %d: %s", code, errs)
	}
	grid, err := beats.Read(out)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(stem)
	if err != nil {
		t.Fatal(err)
	}
	if grid.Score.Analysis == nil || grid.Score.Analysis.File != stem ||
		grid.Score.Analysis.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Errorf("the score's analysis is %+v, want the stem and its hash",
			grid.Score.Analysis)
	}
	if grid.Score.Method != "drum-stem-energy-dp-v1" {
		t.Errorf("the method is %q, want drum-stem-energy-dp-v1",
			grid.Score.Method)
	}
}

// A stem that is not the audio's length, or is not there, is refused
// and no score is written.
func TestAnalyseRefusesAStemThatDoesNotMatch(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "track.wav")
	short := filepath.Join(dir, "short.wav")
	writeClicks(t, audio, 6)
	writeClicks(t, short, 5)
	cases := []struct {
		name, stem, want string
	}{
		{"a shorter stem", short, "drum stem must retain"},
		{"a missing stem", filepath.Join(dir, "none.wav"), "none.wav"},
	}
	for _, c := range cases {
		out := filepath.Join(dir, "score-"+strings.ReplaceAll(c.name, " ", "-"))
		code, _, errs := analyseRun(audio, "--out", out, "--drums", c.stem)
		if code == 0 || !strings.Contains(errs, c.want) {
			t.Errorf("%s: exit %d, error %q, want a refusal containing %q",
				c.name, code, errs, c.want)
		}
		if _, err := os.Stat(out); err == nil {
			t.Errorf("%s: a score was written", c.name)
		}
	}
}

// Arguments the verb cannot use are refused before any audio is opened,
// each with its own reason.
func TestAnalyseRefusesBadArguments(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "score.yaml")
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no audio", []string{"--out", out}, "analyse needs one audio file"},
		{"no out", []string{"track.wav"}, "analyse needs one audio file"},
		{"tempo not a number", []string{"track.wav", "--out", out,
			"--tempo", "fast"}, "--tempo needs a number"},
		{"meter not a number", []string{"track.wav", "--out", out,
			"--meter", "x"}, "--meter needs a number"},
		{"first not a number", []string{"track.wav", "--out", out,
			"--first", "y"}, "--first needs a number"},
		{"meter of zero", []string{"track.wav", "--out", out,
			"--meter", "0"}, "meter is a whole count in 1..32"},
		{"meter past 32", []string{"track.wav", "--out", out,
			"--meter", "33"}, "meter is a whole count in 1..32"},
		{"meter with a fraction", []string{"track.wav", "--out", out,
			"--meter", "2.5"}, "meter is a whole count in 1..32"},
		{"meter of NaN", []string{"track.wav", "--out", out,
			"--meter", "NaN"}, "meter is a whole count in 1..32"},
		{"negative tempo", []string{"track.wav", "--out", out,
			"--tempo", "-5"}, ""},
	}
	for _, c := range cases {
		code, _, errs := analyseRun(c.args...)
		if code == 0 || !strings.Contains(errs, c.want) {
			t.Errorf("%s: exit %d, error %q, want a refusal containing %q",
				c.name, code, errs, c.want)
		}
	}
}

// An audio file that is not there is refused, and a score already at the
// path is left as it was.
func TestAnalyseRefusesMissingAudioAndKeepsAnExistingScore(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "score.yaml")
	code, _, errs := analyseRun(filepath.Join(dir, "none.wav"), "--out", out)
	if code == 0 || !strings.Contains(errs, "none.wav") {
		t.Errorf("a missing audio file: exit %d, error %q", code, errs)
	}
	audio := filepath.Join(dir, "track.wav")
	writeClicks(t, audio, 6)
	if err := os.WriteFile(out, []byte("corrected"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errs = analyseRun(audio, "--out", out, "--tempo", "120")
	if code == 0 || !strings.Contains(errs, "exists") {
		t.Errorf("an existing score: exit %d, error %q, want a refusal "+
			"saying it exists", code, errs)
	}
	if raw, err := os.ReadFile(out); err != nil || string(raw) != "corrected" {
		t.Errorf("the existing score is now %q (%v), want it kept", raw, err)
	}
}

// Silence has no rhythm to estimate, so no score is written for it.
func TestAnalyseRefusesAudioWithNoBeat(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "quiet.wav")
	writeClicks(t, audio, 1)
	out := filepath.Join(dir, "score.yaml")
	code, _, errs := analyseRun(audio, "--out", out, "--tempo", "120")
	if code == 0 || errs == "" {
		t.Errorf("a second of silence: exit %d, error %q, want a refusal",
			code, errs)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a score was written for silence")
	}
}

// A file that is not audio, given as the track or as the stem, is
// refused with the reader's reason and no score is written.
func TestAnalyseRefusesAFileThatIsNotAudio(t *testing.T) {
	dir := t.TempDir()
	text := filepath.Join(dir, "notes.wav")
	if err := os.WriteFile(text, []byte("these are not samples"), 0o644); err != nil {
		t.Fatal(err)
	}
	audio := filepath.Join(dir, "track.wav")
	writeClicks(t, audio, 6)
	for _, args := range [][]string{{text}, {audio, "--drums", text}} {
		out := filepath.Join(dir, fmt.Sprintf("score-%d.yaml", len(args)))
		code, _, errs := analyseRun(append(args, "--out", out)...)
		if code == 0 || !strings.Contains(errs, "movie:") {
			t.Errorf("%v: exit %d, error %q, want the movie reader's "+
				"refusal", args, code, errs)
		}
		if _, err := os.Stat(out); err == nil {
			t.Errorf("%v: a score was written", args)
		}
	}
}
