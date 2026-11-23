package main

import (
	"context"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/janearc/slayer/improv/moonlake/ocean"
	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/encoder"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
)

func Example_stretch() {
	tool().Run(stretchArgs([]string{"-from", "10", "-to", "12", "-settings", "night.yaml", "-out", "film", "-png", "--dry-run"}), os.Stdout, os.Stderr)
}

func TestStretchArguments(t *testing.T) {
	outDir := t.TempDir()
	code, out := run(stretchArgs([]string{"-from", "10", "-to", "12", "-fps", "24", "-settings", "missing.yaml", "-out", outDir, "-png", "--dry-run"})...)
	if code != 0 || !strings.Contains(out, "[10,12)") {
		t.Fatalf("%d %s", code, out)
	}
	for _, args := range [][]string{
		{"stretch", "--from", "-1", "--to", "3", "--settings", "x", "--out", "x", "--png"},
		{"stretch", "--from", "2", "--to", "2", "--settings", "x", "--out", "x", "--png"},
		{"stretch", "--settings", "x", "--out", "x"},
		{"stretch", "--settings", "x", "--out", "x", "--png", "--fps", "NaN"},
		{"stretch", "--settings", "x", "--out", "x", "--png", "extra"},
	} {
		if code, out := run(args...); code == 0 {
			t.Fatalf("accepted %v: %s", args, out)
		}
	}
}

func TestCuesLegacyDefault(t *testing.T) {
	show, err := parseShow([]byte("score: {beats: {times: [0, 1, 2]}}"), []byte("cues: [{at_sec: 1, moon_alt: 7}]"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ sec, alt float64 }{{-1, 2.3}, {0, 2.3}, {1, 7}} {
		cue, err := show.At(tc.sec, 24)
		if err != nil || cue.MoonAlt != tc.alt || cue.Master != 1 || cue.Mist != .12 {
			t.Fatalf("%+v %v", cue, err)
		}
	}
}

func TestCuesPreserveEarlierCueFade(t *testing.T) {
	show, err := parseShow([]byte("score: {}"), []byte("cues: [{at_sec: -1, moon_alt: 8, fade: 2}]"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ sec, alt float64 }{{-2, 2.3}, {-1, 2.3}, {0, 5.15}, {1, 8}} {
		cue, err := show.At(tc.sec, 24)
		if err != nil || cue.MoonAlt != tc.alt {
			t.Fatalf("at %g: %+v %v", tc.sec, cue, err)
		}
	}
}

func TestOutputAliases(t *testing.T) {
	dir := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}
	o := stretchOptions{from: 10, to: 11, out: dir, png: true, encode: filepath.Join(alias, "00010.png"), redraw: true}
	if err := preflightOutputs(o); err == nil || !strings.Contains(err.Error(), "conflicting output") {
		t.Fatalf("accepted parent alias: %v", err)
	}
	original := filepath.Join(dir, "00010.png")
	if err := os.WriteFile(original, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	o.encode = filepath.Join(dir, "clip.mp4")
	if err := os.Link(original, o.encode); err != nil {
		t.Fatal(err)
	}
	if err := preflightOutputs(o); err == nil || !strings.Contains(err.Error(), "conflicting output") {
		t.Fatalf("accepted hard-link alias: %v", err)
	}
	if err := os.Remove(o.encode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(o.encode, []byte("independent"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := preflightOutputs(o); err != nil {
		t.Fatalf("independent outputs refused: %v", err)
	}
}

// readings is a host whose usage is each of these in turn, the last one
// from then on, counting how often it was asked.
type readings struct {
	each  []hostUsage
	asked int
}

func (host *readings) read(context.Context) (hostUsage, error) {
	at := min(host.asked, len(host.each)-1)
	host.asked++
	return host.each[at], nil
}

// unpaused is a pause that does not wait, for the tests
func unpaused(context.Context) error { return nil }

// refusing is somewhere to say things that takes nothing
func refusing(t *testing.T) *mocks.MockWriter {
	writer := mocks.NewMockWriter(gomock.NewController(t))
	writer.EXPECT().Write(gomock.Any()).
		Return(0, errors.New("nowhere to write")).AnyTimes()
	return writer
}

// The regression for 2026-11-11: with swap over its limit the gate held a
// run for ten minutes and said nothing, so it read as a hang. A gate that
// holds says so at once: which limit is over, the reading, the setting
// that limits it, and how to go on.
func TestAHeldGateSaysWhyItWaits(t *testing.T) {
	g := settings.Gates{LoadLimit: 16, SwapLimitBytes: 8 << 30}
	// the night's own numbers: 11762.69 MB of swap in use, load 2.28
	night := hostUsage{load: 2.28, swap: 11762 << 20}
	host := &readings{each: []hostUsage{night, night, night, {load: 2, swap: 1 << 30}}}
	var told strings.Builder
	if err := waitGate(context.Background(), g, host.read, unpaused, &told); err != nil {
		t.Fatal(err)
	}
	said := told.String()
	for _, want := range []string{"waiting", "swap in use is 11.49 GiB", "gates.swap_limit_bytes of 8.00 GiB", "raise the limit", "interrupt"} {
		if !strings.Contains(said, want) {
			t.Errorf("the held gate does not say %q: %q", want, said)
		}
	}
	if strings.Contains(said, "load") {
		t.Errorf("the gate blames the load, which was under its limit: %q", said)
	}
	// held three readings running for the one reason, it says it once
	if lines := strings.Count(said, "\n"); lines != 1 {
		t.Errorf("held three times for one reason, the gate spoke %d times: %q", lines, said)
	}
}

// the anti-test of the one above: a gate that does not hold says nothing
// at all
func TestAGateThatDoesNotHoldSaysNothing(t *testing.T) {
	g := settings.Gates{LoadLimit: 16, SwapLimitBytes: 8 << 30}
	host := &readings{each: []hostUsage{{load: 16, swap: 8 << 30}}}
	var told strings.Builder
	if err := waitGate(context.Background(), g, host.read, unpaused, &told); err != nil {
		t.Fatal(err)
	}
	if told.Len() != 0 {
		t.Errorf("a gate at its limits, not over them, spoke: %q", told.String())
	}
	if host.asked != 1 {
		t.Errorf("a gate that did not hold asked the host %d times", host.asked)
	}
}

// What holds the gate may change while it holds: it says each reason as
// it comes, the load, then the swap, then both.
func TestAHeldGateSaysEachNewReason(t *testing.T) {
	g := settings.Gates{LoadLimit: 16, SwapLimitBytes: 8 << 30}
	host := &readings{each: []hostUsage{
		{load: 17, swap: 0}, {load: 18, swap: 0},
		{load: 1, swap: 9 << 30},
		{load: 17, swap: 9 << 30},
		{load: 1, swap: 0}}}
	var told strings.Builder
	if err := waitGate(context.Background(), g, host.read, unpaused, &told); err != nil {
		t.Fatal(err)
	}
	said := told.String()
	lines := strings.Split(strings.TrimSuffix(said, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("three reasons in turn, and the gate spoke %d times: %q", len(lines), told.String())
	}
	if !strings.Contains(lines[0], "the load is 17.00, over gates.load_limit of 16.00") || strings.Contains(lines[0], "swap") {
		t.Errorf("first it was the load alone: %q", lines[0])
	}
	if !strings.Contains(lines[1], "swap in use is 9.00 GiB") || strings.Contains(lines[1], "load") {
		t.Errorf("then it was the swap alone: %q", lines[1])
	}
	if !strings.Contains(lines[2], "the load is 17.00") || !strings.Contains(lines[2], "swap in use is 9.00 GiB") {
		t.Errorf("then it was both: %q", lines[2])
	}
}

// the anti-test of the one above: a gate with nowhere to speak, or whose
// words cannot be written, does not wait in silence. It refuses.
func TestAGateThatCannotSpeakRefusesToWait(t *testing.T) {
	g := settings.Gates{LoadLimit: 16, SwapLimitBytes: 8 << 30}
	over := &readings{each: []hostUsage{{load: 17}}}
	if err := waitGate(context.Background(), g, over.read, unpaused, nil); err == nil {
		t.Error("a gate with nowhere to speak waited")
	}
	if over.asked != 0 {
		t.Errorf("a gate with nowhere to speak asked the host %d times first", over.asked)
	}
	err := waitGate(context.Background(), g, over.read, unpaused, refusing(t))
	if err == nil || !strings.Contains(err.Error(), "saying why it waits") {
		t.Errorf("a gate whose words could not be written went on: %v", err)
	}
}

func TestHostGate(t *testing.T) {
	g := settings.Gates{LoadLimit: 16, SwapLimitBytes: 1024}
	calls := 0
	err := waitGate(context.Background(), g, func(context.Context) (hostUsage, error) {
		calls++
		if calls == 1 {
			return hostUsage{load: 17}, nil
		}
		if calls == 2 {
			return hostUsage{load: 1, swap: 1025}, nil
		}
		return hostUsage{load: 16, swap: 1024}, nil
	}, func(context.Context) error { return nil }, &strings.Builder{})
	if err != nil || calls != 3 {
		t.Fatalf("%d %v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	err = waitGate(ctx, g, func(context.Context) (hostUsage, error) { cancel(); return hostUsage{load: 17}, nil }, gatePause, &strings.Builder{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	sentinel := errors.New("host read failed")
	err = waitGate(context.Background(), g, func(context.Context) (hostUsage, error) { return hostUsage{}, sentinel }, gatePause, &strings.Builder{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("host failure hidden: %v", err)
	}
	usage, err := parseHostUsage("{ 1.25 2.00 3.00 }", "total = 1024.00M used = 1.50M free = 1022.50M (encrypted)")
	if err != nil || usage.load != 1.25 || usage.swap != 1572864 {
		t.Fatalf("%+v %v", usage, err)
	}
	if _, err := parseHostUsage("bad", "bad"); err == nil {
		t.Fatal("accepted malformed host statistics")
	}
}

func TestBoundedInputsAndDiskVolumes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "score.yaml")
	if err := os.WriteFile(path, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readInput(context.Background(), path, 4); err == nil {
		t.Fatal("accepted oversized input")
	}
	if _, err := readInput(context.Background(), dir, 10); err == nil {
		t.Fatal("accepted directory")
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readInput(context.Background(), fifo, 10); err == nil {
		t.Fatal("accepted FIFO")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readInput(ctx, path, 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	got, err := readInput(context.Background(), path, 5)
	if err != nil || string(got) != "12345" {
		t.Fatalf("%q %v", got, err)
	}
	o := stretchOptions{out: filepath.Join(dir, "not-yet", "frames"), encode: filepath.Join(t.TempDir(), "movie.mp4")}
	var checked []string
	err = outputDiskSpace(o, 100, func(path string, floor int64) error {
		checked = append(checked, path)
		if floor != 100 {
			t.Fatal(floor)
		}
		return nil
	})
	if err != nil || !reflect.DeepEqual(checked, []string{o.out, filepath.Dir(o.encode)}) {
		t.Fatalf("%v %v", checked, err)
	}
	o.from, o.to, o.png = 10, 11, true
	if err := preflightOutputs(o); err != nil {
		t.Fatalf("missing output directory: %v", err)
	}
	sentinel := errors.New("movie volume full")
	if err := outputDiskSpace(o, 100, func(path string, _ int64) error {
		if path == filepath.Dir(o.encode) {
			return sentinel
		}
		return nil
	}); !errors.Is(err, sentinel) {
		t.Fatalf("movie disk failure hidden: %v", err)
	}
	if err := diskSpace(o.out, 1); err != nil {
		t.Fatalf("missing output directory ancestor: %v", err)
	}
}

func TestHistoricalAngleRounding(t *testing.T) {
	s := settings.Settings{Camera: settings.Camera{Eye: 71, Look: -6, Focal: 85, FrameHeight: 40, ShoreDistance: 1660}}
	c, _ := configs(s, stretchOptions{})
	if c.Top != 7.241 || c.Bottom != -19.241 || c.Shore != -2.449 {
		t.Fatalf("angles must retain historical three-decimal formatting: %g %g %g", c.Top, c.Bottom, c.Shore)
	}
}

func TestConfigMapping(t *testing.T) {
	s := settings.Settings{Frame: settings.Frame{Width: 2, Height: 4}, Camera: settings.Camera{Eye: 71, Look: -6, Focal: 85, FrameHeight: 40, ShoreDistance: 1660, MoonRadius: 1.5, MoonAlt: 4, MoonScale: 6, MoonDiscAlt: 5}, Night: settings.Night{Exposure: 23, Bloom: 1.6, Wind: 8, MistThick: 12, SearchCount: 8, MistSeed: 3, OverheadMist: 3, StrobeMist: .25}}
	c, l := configs(s, stretchOptions{from: 10, to: 12, fps: 24})
	if c.From != 10 || c.To != 12 || c.FPS != 24 || c.Ocean.MoonAltitude != 4 || c.Ocean.Wind != 8 || c.Develop.Exposure != 23 || c.Develop.Knee != .7 || c.Develop.Top != 8 || c.MistLayers != 12 || l.EyeOverLake != 71 || l.MoonDiscAltitude != 5 || l.Search.Count != 8 {
		t.Fatalf("%+v %+v", c, l)
	}
	if c.Top <= c.Bottom || c.Shore >= c.Top || c.Shore <= c.Bottom {
		t.Fatalf("geometry %+v", c)
	}
}

type mockEncoding struct {
	indices         []int
	closed, aborted bool
	fail            bool
}

func (e *mockEncoding) WriteFrame(i int, p *image.RGBA) (time.Duration, error) {
	e.indices = append(e.indices, i)
	if e.fail {
		return 0, errors.New("pipe failed")
	}
	return 0, nil
}
func (e *mockEncoding) Close() error { e.closed = true; return nil }
func (e *mockEncoding) Abort() error { e.aborted = true; return nil }

func TestStretchOutputsAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		dir := t.TempDir()
		show, err := parseShow([]byte("score: {}"), []byte("cues: [{at_sec: 0.4, moon_alt: 7}, {at_sec: 0.45, moon_alt: 9}]"))
		if err != nil {
			t.Fatal(err)
		}
		c := compositor.Config{From: 10, To: 13, FPS: 24, Width: 2, Height: 2, Show: show, Ocean: ocean.Config{MoonAltitude: 4}}
		e := &mockEncoding{fail: fail}
		var film []int
		calls := 0
		deps := stretchDeps{prepare: func(ctx context.Context, index int) (preparedFrame, error) {
			return mockPreparedFrame{key: resumeKey(t, int64(index), "recipe"), render: func(ctx context.Context, consume func(compositor.Result) error) error {
				calls++
				if c.From != 10 || c.To != 13 || c.Ocean.MoonAltitude != 4 {
					t.Fatalf("stretch must retain static ocean altitude: %+v", c)
				}
				cue, err := c.Show.At(float64(index)/c.FPS, c.FPS)
				if err != nil {
					t.Fatal(err)
				}
				if cue.MoonAlt == c.Ocean.MoonAltitude {
					t.Fatal("fixture must distinguish static ocean altitude from moving cue")
				}
				film = append(film, index)
				blank := image.NewRGBA(image.Rect(0, 0, 2, 2))
				return consume(compositor.Result{Index: index, Image: blank})
			}}, nil
		}, encode: func(ctx context.Context, ec encoder.Config) (frameEncoder, error) {
			if ec.FrameCount != 3 || ec.FPS != 24 {
				t.Fatalf("%+v", ec)
			}
			return e, nil
		}}
		o := stretchOptions{from: 10, to: 13, fps: 24, out: dir, png: true, encode: filepath.Join(dir, "clip.mp4"), ffmpeg: "/explicit/ffmpeg"}
		err = renderStretch(context.Background(), o, c, deps)
		if fail {
			if err == nil || !e.aborted || e.closed {
				t.Fatalf("failure: %v %+v", err, e)
			}
			continue
		}
		if err != nil || !e.closed || calls != 3 || !reflect.DeepEqual(e.indices, []int{0, 1, 2}) || !reflect.DeepEqual(film, []int{10, 11, 12}) {
			t.Fatalf("%v %+v %v", err, e, film)
		}
		if _, err := os.Stat(filepath.Join(dir, "00010.png")); err != nil {
			t.Fatal(err)
		}
		if err := preflightOutputs(o); err != nil {
			t.Fatal(err)
		}
		e.indices, film, calls = nil, nil, 0
		if err := renderStretch(context.Background(), o, c, deps); err != nil {
			t.Fatal(err)
		}
		if calls != 0 || len(film) != 0 || !reflect.DeepEqual(e.indices, []int{0, 1, 2}) {
			t.Fatalf("resume rendered or omitted frames: %d %v %v", calls, film, e.indices)
		}
		o.redraw = true
		if err := preflightOutputs(o); err != nil {
			t.Fatal(err)
		}
		e.indices = nil
		film = nil
		calls = 0
		if err := renderStretch(context.Background(), o, c, deps); err != nil {
			t.Fatal(err)
		}
		if calls != 3 || !reflect.DeepEqual(e.indices, []int{0, 1, 2}) || !reflect.DeepEqual(film, []int{10, 11, 12}) {
			t.Fatalf("redraw omitted existing PNG frames: %v %v", e.indices, film)
		}
	}
}

func TestInputProtectionAndNormalization(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "00010.png")
	if err := os.WriteFile(input, []byte("input"), 0600); err != nil {
		t.Fatal(err)
	}
	s := settings.Settings{Score: input}
	o := stretchOptions{from: 10, to: 11, out: dir, png: true, redraw: true}
	if err := protectInputs(o, s); err == nil {
		t.Fatal("accepted input overwrite")
	}
	got := stretchArgs([]string{"-settings", "-night.yaml", "-from", "-1", "--", "-png"})
	want := []string{"stretch", "--settings", "-night.yaml", "--from", "-1", "--", "-png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	s = settings.Settings{Score: "score.yaml", Fire: settings.Fire{Frames: &settings.OptionalPath{Path: "fire"}}, Night: settings.Night{FloodPasses: [2]string{"left", "right"}}}
	resolvePaths(&s, dir)
	if s.Score != filepath.Join(dir, "score.yaml") || s.Fire.Frames.Path != filepath.Join(dir, "fire") || s.Night.FloodPasses[0] != "left" {
		t.Fatalf("%+v", s)
	}
}

func TestProtectInputsMissingSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	fire := filepath.Join(dir, "fire")
	if err := os.Mkdir(fire, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(fire, alias); err != nil {
		t.Fatal(err)
	}
	o := stretchOptions{from: 10, to: 11, png: true, out: filepath.Join(alias, "missing", "frames")}
	s := settings.Settings{Fire: settings.Fire{Frames: &settings.OptionalPath{Path: fire}}}
	if err := protectInputs(o, s); err == nil || !strings.Contains(err.Error(), "inside fire input") {
		t.Fatalf("accepted missing descendant via alias: %v", err)
	}
	s.Fire.Frames.Path = filepath.Join(dir, "independent")
	if err := protectInputs(o, s); err != nil {
		t.Fatalf("independent directory: %v", err)
	}
}

type mockPreparedFrame struct {
	key    identity.Key
	render func(context.Context, func(compositor.Result) error) error
}

func (p mockPreparedFrame) Key() identity.Key { return p.key }
func (p mockPreparedFrame) Render(ctx context.Context, consume func(compositor.Result) error) error {
	return p.render(ctx, consume)
}

type cancelImage struct {
	image.Image
	cancel context.CancelFunc
}

func (p cancelImage) At(x, y int) color.Color { p.cancel(); return p.Image.At(x, y) }

func TestAtomicPNGCancellation(t *testing.T) {
	for _, overwrite := range []bool{false, true} {
		dir := t.TempDir()
		path := filepath.Join(dir, "frame.png")
		before := []byte("existing frame")
		if overwrite {
			if err := os.WriteFile(path, before, 0600); err != nil {
				t.Fatal(err)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		img := cancelImage{Image: image.NewRGBA(image.Rect(0, 0, 2, 2)), cancel: cancel}
		err := atomicPNG(ctx, path, img, overwrite)
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation during PNG encode: %v", err)
		}
		got, err := os.ReadFile(path)
		if overwrite {
			if err != nil || !reflect.DeepEqual(got, before) {
				t.Fatalf("replaced canceled output: %q %v", got, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("published canceled PNG: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if overwrite {
			want = 1
		}
		if len(entries) != want {
			t.Fatalf("temporary leaked: %v", entries)
		}
	}
}

func TestAtomicPNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "00042.png")
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if err := atomicPNG(context.Background(), path, img, false); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicPNG(context.Background(), path, img, false); err == nil {
		t.Fatal("overwrote without redraw")
	}
	after, _ := os.ReadFile(path)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("changed existing file")
	}
	if err := atomicPNG(context.Background(), path, img, true); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temporary files: %v", entries)
	}
}
