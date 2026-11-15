package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/develop"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/ocean"
	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/encoder"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

type stretchOptions struct {
	from, to                      int
	fps                           float64
	settings, out, encode, ffmpeg string
	trace                         string
	png, redraw, experimentalGPU  bool
	gpuOcean                      bool
}

const stretchPage = `# stretch

stretch --from FRAME --to FRAME --settings YAML --out DIR --png

Composites a half-open range of film frames. The default path is CPU
and runs cues, lighting, ocean, matte, mist, and develop in that order.
Use --experimental-gpu for GPU plate, moon, overhead, fixtures, fire,
sky glow, strobes, searchlight and laser splats, back haze, UFO beam,
matte, mist and develop; searchlight and laser planning stays on the CPU.
--experimental-gpu-ocean additionally selects the unvalidated Metal ocean;
it requires --experimental-gpu. The GPU paths support --encode only and
refuse --png until GPU identity and resume support exist.
FPS defaults to settings. Relative input paths use the settings directory.
Ocean moon altitude stays at camera.moon_alt throughout the stretch.
Lighting cues retain cueshow's constant 2.3-degree moon-altitude default;
only explicit cues change it. No synthetic cue is inserted.
--encode PATH also streams the complete stretch to HEVC; it requires
--ffmpeg with an explicit absolute executable path. Enabled song inputs
are refused for encoding; explicitly disable song for a silent movie.
At least one of --png and --encode is required. PNG names are film frame
numbers, such as 03370.png; encoder indices start at zero.

PNG resume verifies complete execution and input identities before reuse.
Conflicting or corrupt frames require --redraw; only store-owned entries
may be replaced. Arbitrary existing files are never replaced. --redraw
recomputes every requested frame, including frames sent to the encoder.
Existing movies also require --redraw. PNG publication is directory-anchored
and checks cancellation immediately before commit. Keep the output directory
separate from all inputs. Every movie frame is streamed in film order,
including verified PNGs reused without rendering.
Darwin host admission waits cancellably for the one-minute load average
and used swap bytes to be at or below their configured limits. Host query
failures refuse work. Disk floors are checked on both output volumes.
Score and cues must be regular files of at most 16 MiB each.
--trace PATH writes a new Chrome trace of stage wall times for the
experimental GPU path. These spans include CPU work and waits; they are
not Metal GPU execution times. The trace is published after the movie.
--dry-run validates arguments without loading settings or writing files.
`

// declareStretch adds the stretch verb and its flags to t. Its facts say it
// writes frame PNGs and an optional movie, and that it uses the GPU.
func declareStretch(t *pzm.Tool) {
	facts := verb.NewFacts(verb.WritesTo("film-frame PNGs and an optional HEVC movie"), verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	t.Verb("stretch", verb.NoShort, "composite a short stretch", stretchPage, "Example_stretch", facts, stretch,
		pzm.Value("from", verb.NoShort, "first film frame, inclusive"), pzm.Value("to", verb.NoShort, "last film frame, exclusive"),
		pzm.Value("fps", verb.NoShort, "override the settings frame rate"), pzm.Value("settings", verb.NoShort, "settings YAML file"),
		pzm.Value("out", verb.NoShort, "output directory"), pzm.Switch("png", verb.NoShort, "write film-frame PNGs"),
		pzm.Switch("experimental-gpu", verb.NoShort, "use the experimental hybrid GPU path; PNG is not yet supported"),
		pzm.Switch("experimental-gpu-ocean", verb.NoShort, "use the unvalidated GPU ocean in the experimental GPU path"),
		pzm.Value("trace", verb.NoShort, "write experimental GPU stage wall-time trace"),
		pzm.Value("encode", verb.NoShort, "output HEVC movie"), pzm.Value("ffmpeg", verb.NoShort, "absolute ffmpeg executable path"),
		pzm.Switch("redraw", verb.NoShort, "recompute and replace requested outputs"))
}

// stretchArgs lets the stretch verb be left out: if the first argument is a
// stretch flag, it puts stretch in front. Known flags are rewritten with two
// dashes. The word after a flag that takes a value, and everything after a
// bare double dash, is passed on untouched. Other arguments are returned as
// they came.
func stretchArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	known := map[string]bool{"from": true, "to": true, "fps": true, "settings": true, "out": true, "png": true, "experimental-gpu": true, "experimental-gpu-ocean": true, "trace": true, "encode": true, "redraw": true, "ffmpeg": true}
	first := strings.SplitN(strings.TrimLeft(args[0], "-"), "=", 2)[0]
	implicit := strings.HasPrefix(args[0], "-") && known[first]
	explicit := args[0] == "stretch" || args[0] == "--stretch"
	if !implicit && !explicit {
		return args
	}
	result := []string{}
	start := 0
	if implicit {
		result = append(result, "stretch")
	} else {
		result = append(result, args[0])
		start = 1
	}
	value := false
	for position, arg := range args[start:] {
		if value {
			result = append(result, arg)
			value = false
			continue
		}
		if arg == "--" {
			result = append(result, arg)
			result = append(result, args[start+position+1:]...)
			return result
		}
		name := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)[0]
		if strings.HasPrefix(arg, "-") && known[name] {
			arg = "--" + strings.TrimLeft(arg, "-")
			value = !strings.Contains(arg, "=") && name != "png" && name != "redraw" && name != "experimental-gpu" && name != "experimental-gpu-ocean"
		}
		result = append(result, arg)
	}
	return result
}

// options reads the stretch flags into stretchOptions. From defaults to 0, to
// to 72, and fps to 0, which means use the settings. It refuses a range that
// is empty, negative or over encoder.MaxFrames, positional arguments, a
// missing settings, out or output kind, png with the GPU path, trace without
// the GPU path or over 149796 frames, and a movie without an absolute ffmpeg.
func options(call pzm.Call) (stretchOptions, error) {
	o := stretchOptions{from: 0, to: 72, settings: call.Value("settings"), out: call.Value("out"), encode: call.Value("encode"), ffmpeg: call.Value("ffmpeg"), trace: call.Value("trace"), png: call.On("png"), redraw: call.On("redraw"), experimentalGPU: call.On("experimental-gpu"), gpuOcean: call.On("experimental-gpu-ocean")}
	var err error
	if call.Given("from") {
		o.from, err = strconv.Atoi(call.Value("from"))
		if err != nil {
			return o, fmt.Errorf("from: %w", err)
		}
	}
	if call.Given("to") {
		o.to, err = strconv.Atoi(call.Value("to"))
		if err != nil {
			return o, fmt.Errorf("to: %w", err)
		}
	}
	if call.Given("fps") {
		o.fps, err = strconv.ParseFloat(call.Value("fps"), 64)
		if err != nil || math.IsNaN(o.fps) || math.IsInf(o.fps, 0) || o.fps <= 0 {
			return o, errors.New("fps must be finite and positive")
		}
	}
	if len(call.Args()) != 0 || o.from < 0 || o.to <= o.from || o.to-o.from > encoder.MaxFrames {
		return o, errors.New("expected a nonnegative half-open frame range of at most 1048576 frames, with no positional arguments")
	}
	if strings.TrimSpace(o.settings) == "" || strings.TrimSpace(o.out) == "" || (!o.png && o.encode == "") {
		return o, errors.New("settings, out, and at least one of png or encode are required")
	}
	if o.experimentalGPU && o.png {
		return o, errors.New("experimental GPU path currently refuses --png; GPU identity and resume are unavailable")
	}
	if o.gpuOcean && !o.experimentalGPU {
		return o, errors.New("experimental GPU ocean requires --experimental-gpu")
	}
	if call.Given("trace") && (strings.TrimSpace(o.trace) == "" || !o.experimentalGPU || o.to-o.from > (1<<20)/7) {
		return o, errors.New("trace requires experimental GPU, a nonempty path and at most 149796 frames")
	}
	if call.Given("encode") && strings.TrimSpace(o.encode) == "" {
		return o, errors.New("encode path must not be empty")
	}
	if o.encode != "" && !filepath.IsAbs(o.ffmpeg) {
		return o, errors.New("encode requires an explicit absolute --ffmpeg executable path")
	}
	return o, nil
}

// stretch is the stretch verb. A dry run only prints its plan. Otherwise it
// reads the settings, refuses ones it cannot honour, checks that outputs do
// not clash with inputs, waits at the host gate and disk floor, reads the
// score and cues, and renders by one of three routes: the experimental GPU,
// the CPU with verified PNG resume, or the CPU straight to the movie. It
// stops on interrupt or SIGTERM. The trace is written after rendering.
func stretch(call pzm.Call) error {
	o, err := options(call)
	if err != nil {
		return err
	}
	if call.DryRun() {
		if o.experimentalGPU {
			_, err = fmt.Fprintf(call.Out(), "would composite with the experimental hybrid GPU path; PNG: %t; movie: %q; trace: %q; PNG resume unavailable\n", o.png, o.encode, o.trace)
			return err
		}
		_, err = fmt.Fprintf(call.Out(), "would composite CPU film frames [%d,%d); PNG: %t; movie: %q; verified PNG resume\n", o.from, o.to, o.png, o.encode)
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	settingsBytes, err := readInput(ctx, o.settings, 1<<20)
	if err != nil {
		return err
	}
	s, err := settings.Parse(strings.NewReader(string(settingsBytes)))
	if err != nil {
		return err
	}
	resolvePaths(&s, filepath.Dir(o.settings))
	if o.fps == 0 {
		o.fps = s.FPS
	}
	if o.encode != "" && s.Song != nil && !s.Song.Off {
		return errors.New("song: audio muxing is unsupported by the stretch encoder; explicitly disable song for a silent movie")
	}
	if o.encode != "" && (o.fps > 240 || s.Frame.Width%2 != 0) {
		return errors.New("encode requires an even frame width and fps at most 240")
	}
	if err := protectInputs(o, s); err != nil {
		return err
	}
	if s.DebugMask != nil && !s.DebugMask.Off {
		return errors.New("debug_mask is unsupported by the CPU stretch command")
	}
	if s.Camera.ShoreDistance != 1660 {
		return errors.New("camera.shore_distance: CPU lighting currently requires 1660")
	}
	check := func() error {
		if err := waitGate(ctx, s.Gates, readHostUsage, gatePause, call.Out()); err != nil {
			return err
		}
		return outputDiskSpace(o, s.DiskFloorBytes, diskSpace)
	}
	if err := check(); err != nil {
		return err
	}
	if err := os.MkdirAll(o.out, 0700); err != nil {
		return err
	}
	if err := preflightOutputs(o); err != nil {
		return err
	}
	if err := outputDiskSpace(o, s.DiskFloorBytes, diskSpace); err != nil {
		return err
	}
	score, err := readInput(ctx, s.Score, 16<<20)
	if err != nil {
		return err
	}
	sheet, err := readInput(ctx, s.Cues, 16<<20)
	if err != nil {
		return err
	}
	if o.experimentalGPU {
		show, err := parseShow(score, sheet)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		assets, err := compositor.LoadStaticAssets(s)
		if err != nil {
			return err
		}
		c, l := configs(s, o)
		stage, err := assets.NewLightStage(l)
		if err != nil {
			return err
		}
		if err := stage.EnableExperimentalGPU(); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		c.Show, c.Light, c.MoonReference, c.WaterMask, c.TreeMask = show, admittedLight{stage: stage, check: check}, assets.MoonReference, *assets.Lighting.Water, assets.Lighting.Trees
		runner := compositor.RunExperimentalGPU
		var profile *compositor.Profile
		if o.trace != "" {
			profile, err = compositor.NewProfile((o.to - o.from) * 7)
			if err != nil {
				return err
			}
		}
		if o.gpuOcean {
			runner = func(ctx context.Context, config compositor.Config, consume func(compositor.Result) error) error {
				return compositor.RunExperimentalGPUWithOcean(ctx, config, consume, profile)
			}
		} else if profile != nil {
			runner = func(ctx context.Context, config compositor.Config, consume func(compositor.Result) error) error {
				return compositor.RunExperimentalGPUProfiled(ctx, config, consume, profile)
			}
		}
		deps := stretchDeps{run: runner, encode: func(ctx context.Context, c encoder.Config) (frameEncoder, error) { return encoder.New(ctx, c) }, check: check}
		if err := renderStretch(ctx, o, c, deps); err != nil {
			return err
		}
		if profile != nil {
			return writeTrace(o.trace, profile)
		}
		return nil
	}
	if o.png {
		c, l := configs(s, o)
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		executable, err = filepath.EvalSymlinks(executable)
		if err != nil {
			return err
		}
		hash, err := identity.HashFile(executable)
		if err != nil {
			return err
		}
		execution, err := compositor.NewExecutionSnapshot(ctx, s, c, l, score, sheet, hash, "cpu", "cpu/"+runtime.GOOS+"/"+runtime.GOARCH)
		if err != nil {
			return err
		}
		deps := stretchDeps{prepare: func(ctx context.Context, index int) (preparedFrame, error) { return execution.PrepareFrame(ctx, index) }, encode: func(ctx context.Context, c encoder.Config) (frameEncoder, error) { return encoder.New(ctx, c) }, check: check}
		return renderStretch(ctx, o, c, deps)
	}
	show, err := parseShow(score, sheet)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	assets, err := compositor.LoadStaticAssets(s)
	if err != nil {
		return err
	}
	c, l := configs(s, o)
	stage, err := assets.NewLightStage(l)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.Show, c.Light, c.MoonReference, c.WaterMask, c.TreeMask = show, admittedLight{stage: stage, check: check}, assets.MoonReference, *assets.Lighting.Water, assets.Lighting.Trees
	deps := stretchDeps{run: compositor.Run, encode: func(ctx context.Context, c encoder.Config) (frameEncoder, error) { return encoder.New(ctx, c) }, check: check}
	return renderStretch(ctx, o, c, deps)
}

// resolvePaths makes each relative file path in s absolute by joining it to
// base, the directory of the settings file. Empty paths, absolute paths and
// optional paths that are switched off are left alone.
func resolvePaths(s *settings.Settings, base string) {
	resolve := func(p *string) {
		if *p != "" && !filepath.IsAbs(*p) {
			*p = filepath.Join(base, *p)
		}
	}
	for _, p := range []*string{&s.Score, &s.Cues, &s.Passes.Nodisc, &s.Passes.Disc, &s.Passes.Mask, &s.Passes.Water, &s.Passes.Trees, &s.Passes.BackLeft, &s.Passes.BackRight, &s.Passes.Overhead} {
		resolve(p)
	}
	for _, p := range []*settings.OptionalPath{s.Fire.Frames, s.Fire.Swell, s.Fire.Burst, s.Song, s.MoonRef, s.DebugMask, s.Passes.SpotLeft0, s.Passes.SpotLeft1, s.Passes.SpotLeft2, s.Passes.SpotRight0, s.Passes.SpotRight1, s.Passes.SpotRight2} {
		if p != nil && !p.Off {
			resolve(&p.Path)
		}
	}
}

// parseShow parses the score and the cue sheet into a show with cues.Parse.
func parseShow(score, sheet []byte) (*cues.Show, error) {
	return cues.Parse(score, sheet)
}

// configs builds the compositor and lighting configurations from the settings
// s and the frame range and fps in o. The top, bottom and shore angles go
// through historicalAngle.
func configs(s settings.Settings, o stretchOptions) (compositor.Config, lights.Config) {
	l := lights.DefaultConfig()
	l.LookDown, l.Focal, l.GaugeHeight, l.EyeOverLake = s.Camera.Look, s.Camera.Focal, s.Camera.FrameHeight, s.Camera.Eye
	l.MoonDiscAltitude, l.MoonAltitude, l.MoonScale = s.Camera.MoonDiscAlt, s.Camera.MoonAlt, s.Camera.MoonScale
	l.Fixtures.FloodPasses, l.Fixtures.FloodGain, l.Fixtures.SpotGain = s.Night.FloodPasses, s.Night.FloodGain, s.Night.SpotGain
	l.Overhead.Gain = s.Night.OverheadGain
	l.Search.Gain, l.Search.Count = s.Night.SearchGain, s.Night.SearchCount
	l.HazeGain, l.RigDeck, l.SkyGlowGain = s.Night.HazeGain, s.Night.RigDeck, s.Night.SkyGlow
	l.Beam = lights.UFOConfig{Out: s.Beam.Out, Radius: s.Beam.Radius, Apex: s.Beam.Apex, Gain: s.Beam.Gain, Halo: s.Beam.Halo}
	l.Fire = lights.FireConfig{Gain: s.Fire.Gain, Decay: s.Fire.Decay, FPS: s.Fire.FPS,
		Percussive: s.Fire.Swell != nil && !s.Fire.Swell.Off, Burst: s.Fire.Burst != nil && !s.Fire.Burst.Off,
		SteadyFrom: float64(s.Fire.SteadyFrom), Pulse: s.Fire.Pulse, Flash: s.Fire.Flash,
		Snap: s.Fire.Snap, SwellFall: s.Fire.SwellFall, BurstGain: s.Fire.BurstGain, Top: s.Fire.Top, Stops: s.Night.Exposure}
	half := math.Atan(s.Camera.FrameHeight/2/s.Camera.Focal) * 180 / math.Pi
	d := develop.DefaultOptions()
	d.Exposure = s.Night.Exposure
	d.Bloom = s.Night.Bloom
	c := compositor.Config{From: o.from, To: o.to, FPS: o.fps, Width: s.Frame.Width, Height: s.Frame.Height,
		Ocean: ocean.Config{FrameHeight: s.Camera.FrameHeight, EyeOverLake: s.Camera.Eye, LookDown: s.Camera.Look, Focal: s.Camera.Focal, MoonAltitude: s.Camera.MoonAlt, MoonScale: s.Camera.MoonScale, Wind: s.Night.Wind},
		Top:   historicalAngle(s.Camera.Look + half), Bottom: historicalAngle(s.Camera.Look - half), Shore: historicalAngle(math.Atan(-s.Camera.Eye/s.Camera.ShoreDistance) * 180 / math.Pi), MoonRadius: s.Camera.MoonRadius,
		MistSeed: s.Night.MistSeed, MistLayers: s.Night.MistThick, MistColour: s.Night.MistColour, OverheadMist: s.Night.OverheadMist, StrobeMist: s.Night.StrobeMist, Develop: d}
	return c, l
}

// historicalAngle rounds value to three decimal places by way of its text.
func historicalAngle(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.3f", value), 64)
	return value
}

type frameEncoder interface {
	WriteFrame(int, *image.RGBA) (time.Duration, error)
	Close() error
	Abort() error
}
type preparedFrame interface {
	Key() identity.Key
	Render(context.Context, func(compositor.Result) error) error
}

type stretchDeps struct {
	prepare func(context.Context, int) (preparedFrame, error)
	run     func(context.Context, compositor.Config, func(compositor.Result) error) error
	encode  func(context.Context, encoder.Config) (frameEncoder, error)
	check   func() error
}

// protectInputs refuses outputs that would damage an input. An output may not
// be the same file as an input or this program, nor inside a fire input
// directory. With PNG output, the output directory may not contain any input,
// fire directory or this program. Paths are compared with symbolic links
// resolved.
func protectInputs(o stretchOptions, s settings.Settings) error {
	inputs := []string{o.settings, s.Score, s.Cues, s.Passes.Nodisc, s.Passes.Disc, s.Passes.Mask, s.Passes.Water, s.Passes.Trees, s.Passes.BackLeft, s.Passes.BackRight, s.Passes.Overhead, o.ffmpeg}
	for _, p := range []*settings.OptionalPath{s.Song, s.MoonRef, s.DebugMask, s.Passes.SpotLeft0, s.Passes.SpotLeft1, s.Passes.SpotLeft2, s.Passes.SpotRight0, s.Passes.SpotRight1, s.Passes.SpotRight2} {
		if p != nil && !p.Off {
			inputs = append(inputs, p.Path)
		}
	}
	if o.png {
		root, err := canonicalOutput(filepath.Join(o.out, ".store-protection"))
		if err != nil {
			return err
		}
		root = filepath.Dir(root)
		protected := append([]string(nil), inputs...)
		for _, dir := range []*settings.OptionalPath{s.Fire.Frames, s.Fire.Swell, s.Fire.Burst} {
			if dir != nil && !dir.Off {
				protected = append(protected, dir.Path)
			}
		}
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		protected = append(protected, executable)
		for _, input := range protected {
			if input == "" {
				continue
			}
			path, err := canonicalOutput(input)
			if err != nil {
				return err
			}
			if resolved, e := filepath.EvalSymlinks(path); e == nil {
				path = resolved
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
				return fmt.Errorf("output directory contains input %s", input)
			}
		}
	}
	outputs := []string{}
	if o.png {
		for i := o.from; i < o.to; i++ {
			outputs = append(outputs, filepath.Join(o.out, fmt.Sprintf("%05d.png", i)))
		}
	}
	if o.encode != "" {
		outputs = append(outputs, o.encode)
	}
	if o.trace != "" {
		outputs = append(outputs, o.trace)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	inputs = append(inputs, executable)
	for _, output := range outputs {
		absolute, err := canonicalOutput(output)
		if err != nil {
			return err
		}
		for _, input := range inputs {
			if input == "" {
				continue
			}
			path, err := filepath.Abs(input)
			if err != nil {
				return err
			}
			resolved, err := filepath.EvalSymlinks(path)
			if err == nil {
				path = resolved
			}
			if path == absolute {
				return fmt.Errorf("output would replace input %s", input)
			}
		}
		for _, dir := range []*settings.OptionalPath{s.Fire.Frames, s.Fire.Swell, s.Fire.Burst} {
			if dir == nil || dir.Off {
				continue
			}
			path, err := filepath.Abs(dir.Path)
			if err != nil {
				return err
			}
			if resolved, e := filepath.EvalSymlinks(path); e == nil {
				path = resolved
			}
			rel, err := filepath.Rel(path, absolute)
			if err != nil {
				return err
			}
			if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
				return fmt.Errorf("output is inside fire input directory %s", dir.Path)
			}
		}
	}
	return nil
}

// canonicalOutput returns path as an absolute path with the symbolic links in
// its directory resolved, though the path need not exist: the nearest existing
// ancestor is resolved and the missing names are added back. The last name is
// not resolved. It fails on errors other than a missing directory.
func canonicalOutput(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(absolute)
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(parent)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Join(resolved, filepath.Base(absolute)), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if _, e := os.Lstat(parent); e == nil {
			return "", err
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", err
		}
		missing = append(missing, filepath.Base(parent))
		parent = next
	}
}

// preflightOutputs checks the outputs before anything is written. With PNG and
// a movie, the movie may not sit in the identity store. The trace may not
// exist. No two outputs may share a path or a file. An existing output must be
// a regular file, or a store link for a PNG, and an existing movie needs
// redraw.
func preflightOutputs(o stretchOptions) error {
	if o.png && o.encode != "" {
		root, err := canonicalOutput(filepath.Join(o.out, ".store-protection"))
		if err != nil {
			return err
		}
		movie, err := canonicalOutput(o.encode)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(filepath.Dir(root), movie)
		if err != nil {
			return err
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && (strings.ContainsRune(rel, filepath.Separator) || strings.HasPrefix(rel, ".generation-"))) {
			return errors.New("movie output conflicts with identity store storage")
		}
	}
	paths := []string{}
	if o.png {
		for i := o.from; i < o.to; i++ {
			paths = append(paths, filepath.Join(o.out, fmt.Sprintf("%05d.png", i)))
		}
	}
	if o.encode != "" {
		paths = append(paths, o.encode)
	}
	if o.trace != "" {
		if _, err := os.Lstat(o.trace); err == nil {
			return fmt.Errorf("trace output exists: %s", o.trace)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		paths = append(paths, o.trace)
	}
	seen := map[string]bool{}
	var identities []os.FileInfo
	for _, path := range paths {
		absolute, err := canonicalOutput(path)
		if err != nil {
			return err
		}
		if seen[absolute] {
			return fmt.Errorf("conflicting output paths: %s", path)
		}
		seen[absolute] = true
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		isPNG := o.png && path != o.encode
		if !info.Mode().IsRegular() && !(isPNG && info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("output %s is not a regular file or store link", path)
		}
		for _, previous := range identities {
			if os.SameFile(previous, info) {
				return fmt.Errorf("conflicting output file identities: %s", path)
			}
		}
		identities = append(identities, info)
		if !isPNG && !o.redraw {
			return fmt.Errorf("output %s exists: use --redraw to replace the movie", path)
		}
	}
	return nil
}

// renderStretch renders film frames c.From up to c.To in order, and gives each
// to the movie encoder when o.encode is set. With PNG output each frame comes
// from deps.prepare through verifiedPNG, reusing the store; otherwise from
// deps.run. It calls deps.check before and between frames, and refuses a
// result that is out of order, the wrong size, or missing. The encoder is
// closed on success and aborted on failure.
func renderStretch(ctx context.Context, o stretchOptions, c compositor.Config, deps stretchDeps) (err error) {
	if err := preflightOutputs(o); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if deps.check != nil {
		if err = deps.check(); err != nil {
			return err
		}
	}
	var store *identity.Store
	if o.png {
		if deps.prepare == nil {
			return errors.New("PNG output requires an authoritative execution snapshot")
		}
		store, err = identity.Open(o.out)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, store.Close()) }()
	}
	var enc frameEncoder
	if o.encode != "" {
		enc, err = deps.encode(ctx, encoder.Config{FFmpeg: o.ffmpeg, Output: o.encode, Width: c.Width, Height: c.Height, FPS: c.FPS, FrameCount: c.To - c.From, Overwrite: o.redraw})
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				err = errors.Join(err, enc.Abort())
			}
		}()
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if deps.check != nil {
		if err = deps.check(); err != nil {
			return err
		}
	}
	next := c.From
	consume := func(r compositor.Result) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if next >= c.To || r.Index != next || r.Image == nil || r.Image.Bounds().Dx() != c.Width || r.Image.Bounds().Dy() != c.Height {
			return fmt.Errorf("frame %d: invalid compositor result", next)
		}
		if deps.check != nil {
			if err := deps.check(); err != nil {
				return err
			}
		}
		if enc != nil {
			if _, err := enc.WriteFrame(next-c.From, r.Image); err != nil {
				return err
			}
		}
		next++
		return nil
	}
	if o.png {
		for index := c.From; index < c.To; index++ {
			if err = ctx.Err(); err != nil {
				return err
			}
			if deps.check != nil {
				if err = deps.check(); err != nil {
					return err
				}
			}
			prepared, prepareErr := deps.prepare(ctx, index)
			if prepareErr != nil {
				return prepareErr
			}
			if prepared == nil {
				return errors.New("execution returned no prepared frame")
			}
			im, frameErr := verifiedPNG(ctx, store, prepared.Key(), o.redraw, func() (*image.RGBA, error) {
				if deps.check != nil {
					if err := deps.check(); err != nil {
						return nil, err
					}
				}
				var im *image.RGBA
				err := prepared.Render(ctx, func(r compositor.Result) error {
					if im != nil || r.Index != index || r.Image == nil || r.Image.Bounds().Dx() != c.Width || r.Image.Bounds().Dy() != c.Height {
						return fmt.Errorf("frame %d: invalid prepared result", index)
					}
					im = r.Image
					return nil
				})
				if err != nil {
					return nil, err
				}
				if im == nil {
					return nil, fmt.Errorf("frame %d: missing prepared result", index)
				}
				if deps.check != nil {
					if err := deps.check(); err != nil {
						return nil, err
					}
				}
				return im, nil
			})
			if frameErr != nil {
				return frameErr
			}
			if err = consume(compositor.Result{Index: index, Image: im}); err != nil {
				return err
			}
		}
		if err = store.Close(); err != nil {
			return err
		}
	} else {
		err = deps.run(ctx, c, consume)
	}
	if err != nil {
		return err
	}
	if next != c.To {
		return fmt.Errorf("frame %d: compositor produced no frame", next)
	}
	if enc != nil {
		return enc.Close()
	}
	return nil
}

// atomicPNG writes img as a PNG to a temporary file beside path and syncs it,
// then puts it at path: renamed over it if overwrite is set, which refuses an
// existing file that is not regular, or else hard linked, which fails if path
// exists. The temporary file is removed, and ctx is checked before publishing.
func atomicPNG(ctx context.Context, path string, img image.Image, overwrite bool) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".omnipositor-*.png")
	if err != nil {
		return err
	}
	temporary := f.Name()
	defer func() {
		if removeErr := os.Remove(temporary); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	if err = png.Encode(f, img); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Sync(); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Close(); err != nil {
		return err
	}
	if overwrite {
		if info, e := os.Lstat(path); e == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("output %s is not a regular file", path)
		} else if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return os.Rename(temporary, path)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Link(temporary, path)
}

// diskSpace refuses if the volume holding path has fewer than floor bytes
// available. The path need not exist; its nearest existing ancestor is used.
func diskSpace(path string, floor int64) error {
	var stat syscall.Statfs_t
	probe, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		err = syscall.Statfs(probe, &stat)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return err
		}
		probe = parent
	}
	if float64(stat.Bavail)*float64(stat.Bsize) < float64(floor) {
		return fmt.Errorf("disk resource refused: %s has less than %d available bytes", path, floor)
	}
	return nil
}
