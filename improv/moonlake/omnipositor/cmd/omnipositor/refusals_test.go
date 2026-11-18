package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/encoder"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

// failAfter is a writer that accepts a set number of writes and then refuses
// every later one.
type failAfter struct{ left int }

// Write accepts the write while any are left and refuses it after that.
func (w *failAfter) Write(p []byte) (int, error) {
	if w.left <= 0 {
		return 0, errors.New("write refused")
	}
	w.left--
	return len(p), nil
}

// runInto runs the tool with args, sending standard output to out, and
// returns the exit status and what was written to standard error.
func runInto(out *failAfter, args ...string) (int, string) {
	var errs bytes.Buffer
	code := tool().Run(args, out, &errs)
	return code, errs.String()
}

// writeFile writes content to name in dir and returns the path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestCompareRefusals checks that compare reports a missing file for each
// of the two image kinds, and an empty difference path.
func TestCompareRefusals(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.png")
	pngFixture(t, good, 10)
	for _, args := range [][]string{
		{"compare", good, filepath.Join(dir, "missing.png")},
		{"compare", filepath.Join(dir, "missing.pfm"), filepath.Join(dir, "other.pfm")},
		{"compare", good, good, "--difference", ""},
	} {
		code, out := run(args...)
		if code == 0 {
			t.Errorf("accepted %v: %s", args, out)
		}
	}
	_, out := run("compare", good, good, "--difference", "")
	if !strings.Contains(out, "difference path must not be empty") {
		t.Errorf("empty difference not named: %s", out)
	}
}

// TestCompareCheckPasses checks that identical PNG files pass --check and
// report a fraction of one.
func TestCompareCheckPasses(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.png"), filepath.Join(dir, "b.png")
	pngFixture(t, a, 77)
	pngFixture(t, b, 77)
	code, out := run("compare", a, b, "--check")
	if code != 0 || !strings.Contains(out, "fraction within one: 1.000000000 (100.000000%)") {
		t.Fatalf("identical files failed check: %d %s", code, out)
	}
}

// TestCompareReportsWriteFailures checks that a standard output that stops
// accepting writes fails the compare verb at each of its three kinds of
// output, for PNG and for PFM, and the dry run.
func TestCompareReportsWriteFailures(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.png"), filepath.Join(dir, "b.png")
	pngFixture(t, a, 1)
	pngFixture(t, b, 2)
	pfm := filepath.Join(dir, "a.pfm")
	var data bytes.Buffer
	data.WriteString("PF\n1 1\n-1\n")
	data.Write(make([]byte, 12))
	if err := os.WriteFile(pfm, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"compare", a, b}, {"compare", pfm, pfm}} {
		for allowed := 0; allowed < 3; allowed++ {
			code, errs := runInto(&failAfter{left: allowed}, args...)
			if code == 0 || !strings.Contains(errs, "write refused") {
				t.Errorf("%v with %d writes allowed: %d %q", args, allowed, code, errs)
			}
		}
	}
	code, errs := runInto(&failAfter{}, "compare", a, b, "--dry-run")
	if code == 0 || !strings.Contains(errs, "write refused") {
		t.Errorf("dry run: %d %q", code, errs)
	}
}

// TestWriteMetricsFormat checks the text of the metrics for a hand-worked
// fraction and PSNR, and that each of its writes can fail.
func TestWriteMetricsFormat(t *testing.T) {
	var histogram [256]uint64
	histogram[0], histogram[255] = 5, 7
	var out bytes.Buffer
	if err := writeMetrics(&out, 0.5, 30, histogram); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"fraction within one: 0.500000000 (50.000000%)\n", "PSNR: 30 dB\n", "\n0: 5\n", "\n255: 7\n", "\n1: 0\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	if got := strings.Count(text, "\n"); got != 259 {
		t.Errorf("expected 259 lines, got %d", got)
	}
	for allowed := 0; allowed < 2; allowed++ {
		if err := writeMetrics(&failAfter{left: allowed}, 0.5, 30, histogram); err == nil {
			t.Errorf("writes allowed %d: no error", allowed)
		}
	}
}

// TestWriteDifferenceRemovesFailedFile checks that an image that cannot be
// encoded leaves no file behind, and that an existing file is refused.
func TestWriteDifferenceRemovesFailedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "difference.png")
	err := writeDifference(path, image.NewNRGBA(image.Rect(0, 0, 0, 0)))
	if err == nil || !strings.Contains(err.Error(), "write difference PNG") {
		t.Fatalf("empty image: %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed write left a file: %v", statErr)
	}
	if err := writeDifference(filepath.Join(dir, "no-such-dir", "d.png"), image.NewNRGBA(image.Rect(0, 0, 1, 1))); err == nil || !strings.Contains(err.Error(), "create difference PNG") {
		t.Fatalf("missing directory: %v", err)
	}
}

// TestParseHostUsageValues checks the load and swap figures for hand-worked
// text, one unit at a time, and the refusals.
func TestParseHostUsageValues(t *testing.T) {
	const load = "{ 7.5 2.00 3.00 }"
	swapText := func(used string) string { return "total = 0.00M  used = " + used + "  free = 0.00M" }
	for _, c := range []struct {
		used string
		want int64
	}{
		{"0.00M", 0}, {"1B", 1}, {"1K", 1024}, {"1.5K", 1536}, {"2M", 2 << 20}, {"2G", 2 << 30}, {"1T", 1 << 40}, {"0.0001K", 1},
	} {
		got, err := parseHostUsage(load, swapText(c.used))
		if err != nil || got.swap != c.want || got.load != 7.5 {
			t.Errorf("used %s: %+v %v", c.used, got, err)
		}
	}
	for _, c := range []struct{ name, load, swap string }{
		{"two loads", "{ 1 2 }", swapText("1M")},
		{"word for load", "{ x 2 3 }", swapText("1M")},
		{"negative load", "{ -1 2 3 }", swapText("1M")},
		{"not a number load", "{ NaN 2 3 }", swapText("1M")},
		{"infinite load", "{ Inf 2 3 }", swapText("1M")},
		{"no used figure", load, "total = 1M free = 1M"},
		{"empty swap", load, ""},
		{"short figure", load, swapText("M")},
		{"unknown unit", load, swapText("5X")},
		{"not a number swap", load, swapText("abcM")},
		{"negative swap", load, swapText("-1M")},
		{"not a number swap value", load, swapText("NaNM")},
		{"swap too large", load, swapText("1e30T")},
	} {
		if got, err := parseHostUsage(c.load, c.swap); err == nil {
			t.Errorf("%s: accepted, got %+v", c.name, got)
		}
	}
}

// TestHostOutputLimit checks that the captured host output takes 4096 bytes
// and refuses a write that would pass that, keeping what it held.
func TestHostOutputLimit(t *testing.T) {
	var out hostOutput
	if n, err := out.Write(make([]byte, 4000)); n != 4000 || err != nil {
		t.Fatalf("first write: %d %v", n, err)
	}
	if n, err := out.Write(make([]byte, 97)); n != 0 || err == nil {
		t.Fatalf("write past the limit: %d %v", n, err)
	}
	if out.Len() != 4000 {
		t.Fatalf("refused write changed the buffer: %d", out.Len())
	}
	if n, err := out.Write(make([]byte, 96)); n != 96 || err != nil {
		t.Fatalf("write up to the limit: %d %v", n, err)
	}
}

// TestReadHostUsage checks the host reading: on Darwin it is a plausible
// load and swap figure, and elsewhere it is refused. A context that has ended
// is refused either way.
func TestReadHostUsage(t *testing.T) {
	usage, err := readHostUsage(context.Background())
	if runtime.GOOS != "darwin" {
		if err == nil || !strings.Contains(err.Error(), "Darwin") {
			t.Fatalf("not refused off Darwin: %v", err)
		}
		return
	}
	if err != nil || usage.load < 0 || usage.swap < 0 {
		t.Fatalf("host reading: %+v %v", usage, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = readHostUsage(ctx)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "vm.loadavg") {
		t.Fatalf("ended context: %v", err)
	}
}

// TestGatePauseStopsWhenContextEnds checks that the pause returns the error
// of a context that has ended, without waiting the second.
func TestGatePauseStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := gatePause(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("pause: %v", err)
	}
}

// TestWaitGateRefusals checks the limits and readings the gate will not
// work with, and that a pause error and an ended context stop it.
func TestWaitGateRefusals(t *testing.T) {
	fine := func(context.Context) (hostUsage, error) { return hostUsage{}, nil }
	for _, g := range []settings.Gates{
		{LoadLimit: 0}, {LoadLimit: -1}, {LoadLimit: nan()}, {LoadLimit: inf()}, {LoadLimit: 1, SwapLimitBytes: -1},
	} {
		if err := waitGate(context.Background(), g, fine, unpaused, &strings.Builder{}); err == nil || !strings.Contains(err.Error(), "invalid load or swap limit") {
			t.Errorf("%+v: %v", g, err)
		}
	}
	good := settings.Gates{LoadLimit: 1, SwapLimitBytes: 10}
	if err := waitGate(context.Background(), good, fine, unpaused, nil); err == nil || !strings.Contains(err.Error(), "nowhere to say") {
		t.Errorf("nil writer: %v", err)
	}
	for _, usage := range []hostUsage{{load: nan()}, {load: inf()}, {load: -1}, {swap: -1}} {
		read := func(context.Context) (hostUsage, error) { return usage, nil }
		if err := waitGate(context.Background(), good, read, unpaused, &strings.Builder{}); err == nil || !strings.Contains(err.Error(), "invalid host statistics") {
			t.Errorf("%+v: %v", usage, err)
		}
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitGate(ended, good, fine, unpaused, &strings.Builder{}); !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
	stop := errors.New("pause stopped")
	busy := func(context.Context) (hostUsage, error) { return hostUsage{load: 5}, nil }
	if err := waitGate(context.Background(), good, busy, func(context.Context) error { return stop }, &strings.Builder{}); !errors.Is(err, stop) {
		t.Errorf("pause error: %v", err)
	}
}

// nan returns a value that is not a number.
func nan() float64 { zero := 0.0; return zero / zero }

// inf returns positive infinity.
func inf() float64 { one, zero := 1.0, 0.0; return one / zero }

// TestReadInputShapes checks an input that is missing, an empty one, a limit
// that is not positive, and one that takes several reads.
func TestReadInputShapes(t *testing.T) {
	dir := t.TempDir()
	if _, err := readInput(context.Background(), filepath.Join(dir, "missing"), 10); err == nil || !strings.Contains(err.Error(), "input ") {
		t.Errorf("missing file: %v", err)
	}
	empty := writeFile(t, dir, "empty", "")
	if got, err := readInput(context.Background(), empty, 10); err != nil || len(got) != 0 {
		t.Errorf("empty file: %q %v", got, err)
	}
	if _, err := readInput(context.Background(), empty, 0); err == nil || !strings.Contains(err.Error(), "exceeds 0 bytes") {
		t.Errorf("zero limit: %v", err)
	}
	big := bytes.Repeat([]byte("ab"), 35000)
	path := writeFile(t, dir, "big", string(big))
	got, err := readInput(context.Background(), path, int64(len(big)))
	if err != nil || !bytes.Equal(got, big) {
		t.Errorf("multi-read file: %d bytes, %v", len(got), err)
	}
}

// TestOutputDiskSpaceChecksEachVolume checks that the first failing volume
// is the one reported, and that the trace directory is checked.
func TestOutputDiskSpaceChecksEachVolume(t *testing.T) {
	o := stretchOptions{out: "film", encode: filepath.Join("movies", "a.mp4"), trace: filepath.Join("traces", "t.json")}
	var seen []string
	err := outputDiskSpace(o, 5, func(path string, floor int64) error {
		seen = append(seen, path)
		return nil
	})
	if err != nil || !reflect.DeepEqual(seen, []string{"film", "movies", "traces"}) {
		t.Fatalf("%v %v", seen, err)
	}
	for _, failing := range []string{"film", "traces"} {
		sentinel := errors.New("full: " + failing)
		seen = nil
		err := outputDiskSpace(o, 5, func(path string, _ int64) error {
			seen = append(seen, path)
			if path == failing {
				return sentinel
			}
			return nil
		})
		if !errors.Is(err, sentinel) {
			t.Errorf("%s: %v", failing, err)
		}
		if failing == "film" && len(seen) != 1 {
			t.Errorf("went on after the first failure: %v", seen)
		}
	}
}

// recordingStage is a light stage that keeps the request it was given.
type recordingStage struct {
	got    []compositor.LightRequest
	result compositor.LightResult
}

// Light records r and returns the stage's fixed result.
func (s *recordingStage) Light(_ context.Context, r compositor.LightRequest) (compositor.LightResult, error) {
	s.got = append(s.got, r)
	return s.result, nil
}

// TestAdmittedLightChecksFirst checks that a refused check lights nothing and
// returns its error, and that an allowed one passes the request through.
func TestAdmittedLightChecksFirst(t *testing.T) {
	stage := &recordingStage{result: compositor.LightResult{Strobe: 0.25}}
	refuse := errors.New("host busy")
	refused := admittedLight{stage: stage, check: func() error { return refuse }}
	if _, err := refused.Light(context.Background(), compositor.LightRequest{Index: 3}); !errors.Is(err, refuse) || len(stage.got) != 0 {
		t.Fatalf("refused: %v, lit %d", err, len(stage.got))
	}
	allowed := admittedLight{stage: stage, check: func() error { return nil }}
	result, err := allowed.Light(context.Background(), compositor.LightRequest{Index: 4})
	if err != nil || result.Strobe != 0.25 || len(stage.got) != 1 || stage.got[0].Index != 4 {
		t.Fatalf("allowed: %+v %v %v", result, err, stage.got)
	}
}

// TestDiskSpaceRefusals checks a floor no volume can meet, and a path whose
// parent is a file.
func TestDiskSpaceRefusals(t *testing.T) {
	dir := t.TempDir()
	err := diskSpace(dir, 1<<62)
	if err == nil || !strings.Contains(err.Error(), "disk resource refused") {
		t.Fatalf("huge floor: %v", err)
	}
	file := writeFile(t, dir, "file", "x")
	if err := diskSpace(filepath.Join(file, "child"), 1); err == nil || strings.Contains(err.Error(), "disk resource refused") {
		t.Fatalf("path below a file: %v", err)
	}
}

// TestCanonicalOutputFailures checks that a path below a file and a path
// below a dangling link are refused, and that a missing tail is added back.
func TestCanonicalOutputFailures(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := writeFile(t, dir, "file", "x")
	if _, err := canonicalOutput(filepath.Join(file, "a", "b")); err == nil {
		t.Error("accepted a path below a file")
	}
	dangling := filepath.Join(dir, "dangling")
	if err := os.Symlink(filepath.Join(dir, "nowhere"), dangling); err != nil {
		t.Fatal(err)
	}
	if _, err := canonicalOutput(filepath.Join(dangling, "x")); err == nil {
		t.Error("accepted a path below a dangling link")
	}
	got, err := canonicalOutput(filepath.Join(dir, "a", "b", "c.png"))
	if err != nil || got != filepath.Join(dir, "a", "b", "c.png") {
		t.Errorf("missing tail: %q %v", got, err)
	}
}

// TestProtectInputsRefusals checks the ways an output can replace an input:
// the movie or trace is an input file, and a path that cannot be resolved.
func TestProtectInputsRefusals(t *testing.T) {
	dir := t.TempDir()
	score := writeFile(t, dir, "score.yaml", "x")
	song := writeFile(t, dir, "song.wav", "x")
	s := settings.Settings{Score: score, Song: &settings.OptionalPath{Path: song}}
	out := filepath.Join(dir, "film")
	for _, o := range []stretchOptions{
		{out: out, encode: score, ffmpeg: "/x/ffmpeg"},
		{out: out, encode: song, ffmpeg: "/x/ffmpeg"},
		{out: out, trace: score},
	} {
		if err := protectInputs(o, s); err == nil || !strings.Contains(err.Error(), "output would replace input") {
			t.Errorf("%+v: %v", o, err)
		}
	}
	if err := protectInputs(stretchOptions{out: out, encode: filepath.Join(dir, "new.mp4")}, s); err != nil {
		t.Errorf("independent movie refused: %v", err)
	}
	s.Song.Off, s.Song.Path = true, ""
	if err := protectInputs(stretchOptions{out: out, encode: song}, s); err != nil {
		t.Errorf("switched-off input protected: %v", err)
	}
	file := writeFile(t, dir, "file", "x")
	if err := protectInputs(stretchOptions{out: out, encode: filepath.Join(file, "x", "m.mp4")}, s); err == nil {
		t.Error("accepted a movie below a file")
	}
	if err := protectInputs(stretchOptions{out: filepath.Join(file, "x"), png: true, from: 1, to: 2}, s); err == nil {
		t.Error("accepted an output directory below a file")
	}
	below := settings.Settings{Score: filepath.Join(file, "x", "score.yaml")}
	if err := protectInputs(stretchOptions{out: out, png: true, from: 1, to: 2}, below); err == nil {
		t.Error("accepted an input below a file")
	}
}

// TestPreflightOutputsRefusals checks each way the planned outputs can
// conflict with each other or with what is on disk.
func TestPreflightOutputsRefusals(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "film")
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	base := stretchOptions{out: out, from: 1, to: 3, png: true}
	cases := []struct {
		name string
		edit func(o *stretchOptions)
		want string
	}{
		{"movie in a store subdirectory", func(o *stretchOptions) { o.encode = filepath.Join(out, "sub", "m.mp4") }, "identity store"},
		{"movie in a generation directory", func(o *stretchOptions) { o.encode = filepath.Join(out, ".generation-1") }, "identity store"},
		{"movie is a frame", func(o *stretchOptions) { o.encode = filepath.Join(out, "00001.png") }, "conflicting output paths"},
		{"trace is the movie", func(o *stretchOptions) {
			o.png = false
			o.encode = filepath.Join(dir, "m.mp4")
			o.trace = o.encode
		}, "trace output exists"},
	}
	writeFile(t, dir, "m.mp4", "movie")
	for _, c := range cases {
		o := base
		c.edit(&o)
		if err := preflightOutputs(o); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if err := preflightOutputs(stretchOptions{out: out, encode: filepath.Join(dir, "m.mp4")}); err == nil || !strings.Contains(err.Error(), "use --redraw") {
		t.Errorf("existing movie: %v", err)
	}
	if err := preflightOutputs(stretchOptions{out: out, encode: filepath.Join(dir, "m.mp4"), redraw: true}); err != nil {
		t.Errorf("existing movie with redraw: %v", err)
	}
	same := stretchOptions{out: out, encode: filepath.Join(dir, "t.json"), trace: filepath.Join(dir, "t.json"), redraw: true}
	if err := preflightOutputs(same); err == nil || !strings.Contains(err.Error(), "conflicting output paths") {
		t.Errorf("trace and movie on one path: %v", err)
	}
	if err := os.Mkdir(filepath.Join(out, "00001.png"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := preflightOutputs(base); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("directory as frame: %v", err)
	}
	if err := os.Remove(filepath.Join(out, "00001.png")); err != nil {
		t.Fatal(err)
	}
	first := writeFile(t, out, "00001.png", "frame")
	if err := os.Link(first, filepath.Join(out, "00002.png")); err != nil {
		t.Fatal(err)
	}
	if err := preflightOutputs(base); err == nil || !strings.Contains(err.Error(), "file identities") {
		t.Errorf("hard linked frames: %v", err)
	}
	file := writeFile(t, dir, "plain", "x")
	if err := preflightOutputs(stretchOptions{out: out, trace: filepath.Join(file, "t.json")}); err == nil {
		t.Error("accepted a trace below a file")
	}
	if err := preflightOutputs(stretchOptions{out: filepath.Join(file, "film"), png: true, encode: filepath.Join(dir, "x.mp4")}); err == nil {
		t.Error("accepted an output directory below a file")
	}
	if err := preflightOutputs(stretchOptions{out: out, encode: filepath.Join(file, "x", "m.mp4")}); err == nil {
		t.Error("accepted a movie below a file")
	}
}

// TestPreflightAllowsStoreLinks checks that a symbolic link in place of a
// frame is accepted for PNG output.
func TestPreflightAllowsStoreLinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "elsewhere"), filepath.Join(dir, "00001.png")); err != nil {
		t.Fatal(err)
	}
	if err := preflightOutputs(stretchOptions{out: dir, from: 1, to: 2, png: true}); err != nil {
		t.Fatalf("store link refused: %v", err)
	}
}

// TestAtomicPNGRefusals checks a missing directory, an image that cannot be
// encoded, and a replacement of something that is not a regular file.
func TestAtomicPNGRefusals(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if err := atomicPNG(context.Background(), filepath.Join(dir, "none", "f.png"), img, false); err == nil {
		t.Error("wrote into a missing directory")
	}
	if err := atomicPNG(context.Background(), filepath.Join(dir, "f.png"), image.NewRGBA(image.Rect(0, 0, 0, 0)), false); err == nil {
		t.Error("encoded an empty image")
	}
	target := filepath.Join(dir, "f.png")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := atomicPNG(context.Background(), target, img, true); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("replaced a directory: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Errorf("temporary files left: %v %v", entries, err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := atomicPNG(ended, filepath.Join(dir, "g.png"), img, false); !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
}

// TestWriteTraceRefusals checks a missing directory and a missing profile,
// and that a failed write leaves no temporary file.
func TestWriteTraceRefusals(t *testing.T) {
	dir := t.TempDir()
	if err := writeTrace(filepath.Join(dir, "none", "t.json"), nil); err == nil || !strings.Contains(err.Error(), "create temporary file") {
		t.Errorf("missing directory: %v", err)
	}
	if err := writeTrace(filepath.Join(dir, "t.json"), nil); err == nil {
		t.Error("wrote a trace for no profile")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Errorf("files left: %v %v", entries, err)
	}
}

// TestDecodeRGBARefusesNonPNG checks that bytes that are not a PNG are an
// error, and that a decoded image keeps its pixels.
func TestDecodeRGBARefusesNonPNG(t *testing.T) {
	if _, err := decodeRGBA([]byte("not a png")); err == nil {
		t.Error("decoded text")
	}
	want := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	got, err := decodeRGBA(pngBytes(t, resumeImageWith(want)))
	if err != nil || got.RGBAAt(0, 0) != want {
		t.Errorf("round trip: %v %v", got, err)
	}
}

// TestValidateRenderedRGBAShapes checks the frame shapes the validator
// refuses and one it accepts.
func TestValidateRenderedRGBAShapes(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	cases := []struct {
		name string
		im   *image.RGBA
		ok   bool
	}{
		{"empty", &image.RGBA{}, false},
		{"zero stride", &image.RGBA{Pix: make([]byte, 16), Rect: image.Rect(0, 0, 2, 2)}, false},
		{"stride below a row", &image.RGBA{Pix: make([]byte, 16), Stride: 4, Rect: image.Rect(0, 0, 2, 2)}, false},
		{"too few bytes for a row", &image.RGBA{Pix: make([]byte, 4), Stride: 8, Rect: image.Rect(0, 0, 2, 1)}, false},
		{"too few rows", &image.RGBA{Pix: make([]byte, 12), Stride: 8, Rect: image.Rect(0, 0, 2, 2)}, false},
		{"width past the limit", &image.RGBA{Stride: 1, Rect: image.Rect(0, 0, maxInt/4+1, 1)}, false},
		{"span past an int", &image.RGBA{Stride: 1, Rect: image.Rect(-2, 0, maxInt, 1)}, false},
		{"tall span past an int", &image.RGBA{Stride: 8, Rect: image.Rect(0, -2, 2, maxInt)}, false},
		{"exact fit", &image.RGBA{Pix: make([]byte, 24), Stride: 8, Rect: image.Rect(0, 0, 2, 3)}, true},
	}
	for _, c := range cases {
		err := validateRenderedRGBA(c.im)
		if c.ok != (err == nil) || (!c.ok && !errors.Is(err, errInvalidRenderedFrame)) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// TestSpanWithinInt checks the span for hand-worked bounds and the two ways
// it is refused.
func TestSpanWithinInt(t *testing.T) {
	for _, c := range []struct {
		min, max, limit int
		span            int
		ok              bool
	}{
		{0, 5, 100, 5, true},
		{-5, 3, 100, 8, true},
		{3, 2, 100, 0, false},
		{-10, 100, 100, 0, false},
		{-1, 99, 100, 100, true},
	} {
		span, ok := spanWithinInt(c.min, c.max, c.limit)
		if ok != c.ok || (ok && span != c.span) {
			t.Errorf("%+v: %d %v", c, span, ok)
		}
	}
}

// TestVerifiedPNGRenderFailures checks that a render error, a hook error and
// an unreadable stored frame each reach the caller, and that a store entry
// that is not a PNG is refused.
func TestVerifiedPNGRenderFailures(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := resumeKey(t, 10, "failures")
	boom := errors.New("render failed")
	if _, err := verifiedPNG(context.Background(), store, key, false, func() (*image.RGBA, error) { return nil, boom }); !errors.Is(err, boom) {
		t.Errorf("render error: %v", err)
	}
	hook := errors.New("hook failed")
	if _, err := verifiedPNGWithHook(context.Background(), store, key, false, func() error { return hook }, func() (*image.RGBA, error) { return resumeImage(), nil }); !errors.Is(err, hook) {
		t.Errorf("hook error: %v", err)
	}
	if _, err := verifiedPNG(context.Background(), store, key, false, func() (*image.RGBA, error) { return &image.RGBA{}, nil }); !errors.Is(err, errInvalidRenderedFrame) {
		t.Errorf("empty frame: %v", err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := verifiedPNG(ended, store, key, false, func() (*image.RGBA, error) { return resumeImage(), nil }); !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
	im, err := verifiedPNG(context.Background(), store, key, false, func() (*image.RGBA, error) { return resumeImage(), nil })
	if err != nil || im.RGBAAt(0, 0).R != 90 {
		t.Fatalf("publish: %v %v", im, err)
	}
	other := resumeKey(t, 11, "other recipe")
	if _, err := verifiedPNG(context.Background(), store, other, false, func() (*image.RGBA, error) { return resumeImage(), nil }); err != nil {
		t.Fatalf("second key: %v", err)
	}
}

// renderDeps builds stretch dependencies whose frames come from run, with an
// encoder that records its frames.
func renderDeps(run func(context.Context, compositor.Config, func(compositor.Result) error) error, enc *mockEncoding) stretchDeps {
	return stretchDeps{run: run, encode: func(context.Context, encoder.Config) (frameEncoder, error) { return enc, nil }}
}

// TestRenderStretchRunRoute checks the route that renders straight to the
// movie: frames in order are encoded from zero, and each way a result can be
// wrong, or a step can fail, is refused with the encoder aborted.
func TestRenderStretchRunRoute(t *testing.T) {
	dir := t.TempDir()
	c := compositor.Config{From: 4, To: 6, FPS: 24, Width: 2, Height: 2}
	o := stretchOptions{from: 4, to: 6, encode: filepath.Join(dir, "m.mp4"), ffmpeg: "/x/ffmpeg", out: dir}
	frame := func(i int) compositor.Result {
		return compositor.Result{Index: i, Image: image.NewRGBA(image.Rect(0, 0, 2, 2))}
	}
	enc := &mockEncoding{}
	good := func(_ context.Context, _ compositor.Config, consume func(compositor.Result) error) error {
		for i := 4; i < 6; i++ {
			if err := consume(frame(i)); err != nil {
				return err
			}
		}
		return nil
	}
	if err := renderStretch(context.Background(), o, c, renderDeps(good, enc)); err != nil || !enc.closed || !reflect.DeepEqual(enc.indices, []int{0, 1}) {
		t.Fatalf("good run: %v %+v", err, enc)
	}
	bad := map[string]func(context.Context, compositor.Config, func(compositor.Result) error) error{
		"out of order": func(_ context.Context, _ compositor.Config, consume func(compositor.Result) error) error {
			return consume(frame(5))
		},
		"no image": func(_ context.Context, _ compositor.Config, consume func(compositor.Result) error) error {
			return consume(compositor.Result{Index: 4})
		},
		"wrong size": func(_ context.Context, _ compositor.Config, consume func(compositor.Result) error) error {
			return consume(compositor.Result{Index: 4, Image: image.NewRGBA(image.Rect(0, 0, 3, 2))})
		},
		"too many": func(ctx context.Context, c compositor.Config, consume func(compositor.Result) error) error {
			if err := good(ctx, c, consume); err != nil {
				return err
			}
			return consume(frame(6))
		},
		"too few": func(_ context.Context, _ compositor.Config, consume func(compositor.Result) error) error {
			return consume(frame(4))
		},
	}
	for name, run := range bad {
		enc := &mockEncoding{}
		err := renderStretch(context.Background(), o, c, renderDeps(run, enc))
		if err == nil || enc.closed || !enc.aborted {
			t.Errorf("%s: %v %+v", name, err, enc)
		}
		if name == "too few" && (err == nil || !strings.Contains(err.Error(), "produced no frame")) {
			t.Errorf("too few: %v", err)
		}
	}
	runFails := errors.New("compositor failed")
	enc = &mockEncoding{}
	if err := renderStretch(context.Background(), o, c, renderDeps(func(context.Context, compositor.Config, func(compositor.Result) error) error { return runFails }, enc)); !errors.Is(err, runFails) || !enc.aborted {
		t.Errorf("run failure: %v %+v", err, enc)
	}
}

// TestRenderStretchSteps checks the refusals that come before any frame:
// a check that fails at each of its calls, an encoder that cannot start, an
// ended context, PNG output with nothing to prepare, a store that cannot be
// opened, and a movie sink that fails to write.
func TestRenderStretchSteps(t *testing.T) {
	dir := t.TempDir()
	c := compositor.Config{From: 4, To: 5, FPS: 24, Width: 2, Height: 2}
	o := stretchOptions{from: 4, to: 5, encode: filepath.Join(dir, "m.mp4"), ffmpeg: "/x/ffmpeg", out: dir}
	frames := func(_ context.Context, _ compositor.Config, consume func(compositor.Result) error) error {
		return consume(compositor.Result{Index: 4, Image: image.NewRGBA(image.Rect(0, 0, 2, 2))})
	}
	refused := errors.New("check refused")
	for failAt := 1; failAt <= 3; failAt++ {
		calls := 0
		deps := renderDeps(frames, &mockEncoding{})
		deps.check = func() error {
			calls++
			if calls == failAt {
				return refused
			}
			return nil
		}
		if err := renderStretch(context.Background(), o, c, deps); !errors.Is(err, refused) {
			t.Errorf("check call %d: %v", failAt, err)
		}
	}
	noEncoder := errors.New("no encoder")
	deps := renderDeps(frames, nil)
	deps.encode = func(context.Context, encoder.Config) (frameEncoder, error) { return nil, noEncoder }
	if err := renderStretch(context.Background(), o, c, deps); !errors.Is(err, noEncoder) {
		t.Errorf("encoder start: %v", err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := renderStretch(ended, o, c, renderDeps(frames, &mockEncoding{})); !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
	png := stretchOptions{from: 4, to: 5, png: true, out: dir}
	if err := renderStretch(context.Background(), png, c, stretchDeps{}); err == nil || !strings.Contains(err.Error(), "authoritative execution snapshot") {
		t.Errorf("PNG without prepare: %v", err)
	}
	png.out = filepath.Join(dir, "missing")
	if err := renderStretch(context.Background(), png, c, stretchDeps{prepare: func(context.Context, int) (preparedFrame, error) { return nil, nil }}); err == nil {
		t.Error("opened a store in a missing directory")
	}
	if err := renderStretch(context.Background(), stretchOptions{from: 4, to: 5, out: dir, encode: filepath.Join(dir, "x.mp4"), ffmpeg: "/x/ffmpeg", redraw: true}, c, stretchDeps{
		run: frames, encode: func(context.Context, encoder.Config) (frameEncoder, error) { return &mockEncoding{fail: true}, nil }}); err == nil || !strings.Contains(err.Error(), "pipe failed") {
		t.Errorf("encoder write: %v", err)
	}
}

// TestRenderStretchPreparedFrames checks the PNG route when the preparation
// or the render goes wrong in each way.
func TestRenderStretchPreparedFrames(t *testing.T) {
	c := compositor.Config{From: 4, To: 5, FPS: 24, Width: 2, Height: 2}
	good := compositor.Result{Index: 4, Image: image.NewRGBA(image.Rect(0, 0, 2, 2))}
	boom := errors.New("prepare failed")
	checkRefused := errors.New("check refused")
	cases := []struct {
		name    string
		prepare func(t *testing.T) func(context.Context, int) (preparedFrame, error)
		check   func(call int) error
		want    string
	}{
		{"prepare error", func(t *testing.T) func(context.Context, int) (preparedFrame, error) {
			return func(context.Context, int) (preparedFrame, error) { return nil, boom }
		}, nil, "prepare failed"},
		{"no frame", func(t *testing.T) func(context.Context, int) (preparedFrame, error) {
			return func(context.Context, int) (preparedFrame, error) { return nil, nil }
		}, nil, "no prepared frame"},
		{"render error", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error { return boom }), nil, "prepare failed"},
		{"render gives nothing", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error { return nil }), nil, "missing prepared result"},
		{"render gives wrong index", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error {
			return consume(compositor.Result{Index: 9, Image: good.Image})
		}), nil, "invalid prepared result"},
		{"render gives two", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error {
			if err := consume(good); err != nil {
				return err
			}
			return consume(good)
		}), nil, "invalid prepared result"},
		{"check refuses before render", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error { return consume(good) }), func(call int) error {
			if call == 4 {
				return checkRefused
			}
			return nil
		}, "check refused"},
		{"check refuses after render", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error { return consume(good) }), func(call int) error {
			if call == 5 {
				return checkRefused
			}
			return nil
		}, "check refused"},
		{"check refuses when the frame is consumed", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error { return consume(good) }), func(call int) error {
			if call == 6 {
				return checkRefused
			}
			return nil
		}, "check refused"},
		{"check refuses before prepare", renderWith(c, func(ctx context.Context, consume func(compositor.Result) error) error { return consume(good) }), func(call int) error {
			if call == 3 {
				return checkRefused
			}
			return nil
		}, "check refused"},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		calls := 0
		deps := stretchDeps{prepare: tc.prepare(t)}
		if tc.check != nil {
			deps.check = func() error { calls++; return tc.check(calls) }
		}
		o := stretchOptions{from: 4, to: 5, png: true, out: dir}
		err := renderStretch(context.Background(), o, c, deps)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}

// renderWith returns a prepare function whose frames render through render
// and carry a key that is the same for every frame.
func renderWith(c compositor.Config, render func(context.Context, func(compositor.Result) error) error) func(t *testing.T) func(context.Context, int) (preparedFrame, error) {
	return func(t *testing.T) func(context.Context, int) (preparedFrame, error) {
		return func(_ context.Context, index int) (preparedFrame, error) {
			return mockPreparedFrame{key: resumeKey(t, int64(index), "prepared"), render: render}, nil
		}
	}
}

// TestRenderStretchStopsWhenContextEnds checks that a context that ends after
// the encoder starts, during a frame, or between PNG frames stops the render
// and aborts the encoder.
func TestRenderStretchStopsWhenContextEnds(t *testing.T) {
	dir := t.TempDir()
	c := compositor.Config{From: 4, To: 6, FPS: 24, Width: 2, Height: 2}
	o := stretchOptions{from: 4, to: 6, encode: filepath.Join(dir, "m.mp4"), ffmpeg: "/x/ffmpeg", out: dir}
	frame := func(i int) compositor.Result {
		return compositor.Result{Index: i, Image: image.NewRGBA(image.Rect(0, 0, 2, 2))}
	}

	ctx, cancel := context.WithCancel(context.Background())
	enc := &mockEncoding{}
	deps := stretchDeps{encode: func(context.Context, encoder.Config) (frameEncoder, error) { cancel(); return enc, nil }}
	if err := renderStretch(ctx, o, c, deps); !errors.Is(err, context.Canceled) || !enc.aborted {
		t.Errorf("ended after the encoder started: %v %+v", err, enc)
	}

	ctx, cancel = context.WithCancel(context.Background())
	enc = &mockEncoding{}
	deps = renderDeps(func(_ context.Context, _ compositor.Config, consume func(compositor.Result) error) error {
		cancel()
		return consume(frame(4))
	}, enc)
	if err := renderStretch(ctx, o, c, deps); !errors.Is(err, context.Canceled) || !enc.aborted || len(enc.indices) != 0 {
		t.Errorf("ended during a frame: %v %+v", err, enc)
	}

	ctx, cancel = context.WithCancel(context.Background())
	pngOptions := stretchOptions{from: 4, to: 6, png: true, out: t.TempDir()}
	prepared := 0
	deps = stretchDeps{prepare: func(_ context.Context, index int) (preparedFrame, error) {
		prepared++
		return mockPreparedFrame{key: resumeKey(t, int64(index), "ending"), render: func(_ context.Context, consume func(compositor.Result) error) error {
			cancel()
			return consume(frame(index))
		}}, nil
	}}
	if err := renderStretch(ctx, pngOptions, c, deps); !errors.Is(err, context.Canceled) || prepared != 1 {
		t.Errorf("ended between PNG frames: %v after %d prepared", err, prepared)
	}
}

// TestRenderStretchPNGOnly checks that PNG output with no movie succeeds,
// renders each frame once, and reuses the stored frames on a second run.
func TestRenderStretchPNGOnly(t *testing.T) {
	dir := t.TempDir()
	c := compositor.Config{From: 4, To: 6, FPS: 24, Width: 2, Height: 2}
	o := stretchOptions{from: 4, to: 6, png: true, out: dir}
	rendered := 0
	deps := stretchDeps{prepare: func(_ context.Context, index int) (preparedFrame, error) {
		return mockPreparedFrame{key: resumeKey(t, int64(index), "png only"), render: func(_ context.Context, consume func(compositor.Result) error) error {
			rendered++
			return consume(compositor.Result{Index: index, Image: resumeImage()})
		}}, nil
	}}
	for run, want := range []int{2, 2} {
		if err := renderStretch(context.Background(), o, c, deps); err != nil || rendered != want {
			t.Fatalf("run %d: %v, rendered %d, want %d", run, err, rendered, want)
		}
	}
	for _, name := range []string{"00004.png", "00005.png"} {
		if _, err := os.Lstat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestRenderStretchPreflightFirst checks that the outputs are checked before
// anything is started.
func TestRenderStretchPreflightFirst(t *testing.T) {
	dir := t.TempDir()
	movie := writeFile(t, dir, "m.mp4", "old")
	started := false
	deps := stretchDeps{encode: func(context.Context, encoder.Config) (frameEncoder, error) {
		started = true
		return &mockEncoding{}, nil
	}}
	o := stretchOptions{from: 4, to: 5, encode: movie, ffmpeg: "/x/ffmpeg", out: dir}
	if err := renderStretch(context.Background(), o, compositor.Config{From: 4, To: 5, Width: 2, Height: 2}, deps); err == nil || !strings.Contains(err.Error(), "use --redraw") || started {
		t.Errorf("existing movie: %v, started %v", err, started)
	}
}

// TestPreflightOutputPathsBelowFiles checks the outputs whose own path cannot
// be inspected because a parent is a file.
func TestPreflightOutputPathsBelowFiles(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "film")
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	file := writeFile(t, dir, "plain", "x")
	if err := preflightOutputs(stretchOptions{out: out, png: true, from: 1, to: 2, encode: filepath.Join(file, "x", "m.mp4")}); err == nil {
		t.Error("accepted a movie below a file next to PNG output")
	}
	if err := preflightOutputs(stretchOptions{out: out, encode: filepath.Join(file, "x")}); err == nil {
		t.Error("accepted a movie whose path cannot be inspected")
	}
}

// TestVerifiedPNGStopsWhenRenderEndsContext checks that a context that ends
// while the frame renders stops the publish and leaves the store empty.
func TestVerifiedPNGStopsWhenRenderEndsContext(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(context.Background())
	key := resumeKey(t, 10, "ends during render")
	_, err = verifiedPNG(ctx, store, key, false, func() (*image.RGBA, error) { cancel(); return resumeImage(), nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ended during render: %v", err)
	}
	if _, err := store.Verify(key); !errors.Is(err, identity.ErrMissing) {
		t.Errorf("a stopped publish left an entry: %v", err)
	}
}
