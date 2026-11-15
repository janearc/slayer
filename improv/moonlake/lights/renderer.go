package lights

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"

	"github.com/janearc/slayer/improv/moonlake/cues"
)

type Config struct {
	LookDown, Focal, GaugeHeight, EyeOverLake              float64
	Fixtures                                               FixtureConfig
	Overhead                                               OverheadConfig
	Fire                                                   FireConfig
	Beam                                                   UFOConfig
	MoonDiscAltitude, MoonAltitude, MoonScale, SkyGlowGain float64
	Search                                                 SearchConfig
	HazeGain, RigDeck                                      float64
	MaxFireFrames                                          int
	MaxFireBytes                                           int64
}

type Assets struct {
	Plate, Mask        Image
	Disc, Water, Trees *Image
	Passes             map[string]Image
	Fire               *[3]FireFrames
}

type CompleteCues cues.Frame

type BaseAccelerator interface {
	Base(Image, *Image, []bool, float64, float64, float64) (Image, error)
	Fixtures(Image, []FixtureLayer, map[string]Image) (Image, error)
	Fire(FireConfig, FireInput) (Image, error)
	SkyGlow(Image, SkyGlowInput) (Image, error)
	Strobes(Image, StrobeInput) (Image, float64, error)
	Searchlights(Image, Image, SearchConfig, SearchInput) (Image, Image, error)
	Lasers(Image, Image, LaserConfig, LaserInput) (Image, Image, error)
	BackHaze(Image, []bool, BackHazeInput) (Image, error)
	Beam(Image, Image, UFOConfig, UFOInput, []bool, []bool) (Image, Image, error)
}

type Request struct {
	Seconds     float64
	Cues        *CompleteCues
	Stops       *float64
	Water       *Image
	FireFrames  *[3]FireFrames
	Accelerator BaseAccelerator
}

type Result struct {
	Sky, Front     Image
	Strobe         float64
	FireWorkingSet FireWorkingSet
}

var ErrFireResources = errors.New("fire resources refused")

const DefaultMaxFireBytes int64 = 1 << 30

// FireByteBudget is the byte limit for fire frames: configured, except that
// when configured is the default limit and cache is larger, it is cache.
func FireByteBudget(configured, cache int64) int64 {
	if configured == DefaultMaxFireBytes && cache > configured {
		return cache
	}
	return configured
}

// CheckFireBudget refuses, with ErrFireResources, a budget of zero or less or
// a working set whose selected frames need more bytes than the budget.
func CheckFireBudget(workingSet FireWorkingSet, budget int64) error {
	if budget <= 0 || workingSet.SelectedBytes > budget {
		return fmt.Errorf("%w: selected frames require %d bytes, maximum %d", ErrFireResources, workingSet.SelectedBytes, budget)
	}
	return nil
}

type FireWorkingSet struct {
	Frames         int
	SelectedBytes  int64
	TransientBytes int64
}

// MeasureFireWorkingSet counts the distinct frames in layers and their bytes
// at width by height, 24 bytes a pixel. Transient bytes are twice the
// selected ones plus one frame, or 0 with no frames. It refuses bad sizes, a
// maximum of zero or less, more distinct frames than maximum, and overflow.
func MeasureFireWorkingSet(layers []FireLayer, width, height, maximum int) (FireWorkingSet, error) {
	if maximum <= 0 || !dimensions(width, height) {
		return FireWorkingSet{}, fmt.Errorf("fire working set: invalid dimensions or frame maximum")
	}
	selected := make(map[[2]int]struct{})
	for _, layer := range layers {
		key := [2]int{int(layer.Sequence), layer.Index}
		if _, exists := selected[key]; exists {
			continue
		}
		if len(selected) >= maximum {
			return FireWorkingSet{}, fmt.Errorf("%w: selected distinct frames exceed maximum %d", ErrFireResources, maximum)
		}
		selected[key] = struct{}{}
	}
	frameBytes := int64(width) * int64(height) * 24
	if frameBytes <= 0 || frameBytes > math.MaxInt64/(2*int64(len(selected))+1) {
		return FireWorkingSet{}, fmt.Errorf("fire working set: byte count overflow")
	}
	bytes := int64(len(selected)) * frameBytes
	transient := int64(0)
	if len(selected) != 0 {
		transient = 2*bytes + frameBytes
	}
	return FireWorkingSet{Frames: len(selected), SelectedBytes: bytes, TransientBytes: transient}, nil
}

type StaticAssets struct {
	mu     sync.Mutex
	images map[string]Image
}

type StaticAssetNames struct {
	Plate, Mask, Disc, Water, Trees string
	Passes                          map[string]string
}

// Decode reads a PFM of exactly width by height from reader and stores it
// under name, replacing any earlier image of that name. It refuses a value
// that is not finite or lies outside float32 range. It takes the lock.
func (s *StaticAssets) Decode(name string, reader io.Reader, width, height int) error {
	image, _, err := DecodePFMExpected(reader, width, height)
	if err != nil {
		return err
	}
	for i, value := range image.Light {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return fmt.Errorf("%s channel %d: outside finite float32 range", name, i)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.images == nil {
		s.images = make(map[string]Image)
	}
	s.images[name] = image
	return nil
}

// Snapshot returns a copy of the image stored under name, or an empty image if
// there is none or s is nil.
func (s *StaticAssets) Snapshot(name string) Image {
	if s == nil {
		return Image{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneRendererImage(s.images[name])
}

// NewStaticRenderer builds a Renderer from the images stored under names. The
// plate and mask are always taken; the disc, water and trees only when their
// names are not empty. The renderer shares the stored pixels rather than
// copying them. It holds stored's lock while it works and refuses a nil stored.
func NewStaticRenderer(config Config, stored *StaticAssets, names StaticAssetNames) (*Renderer, error) {
	if stored == nil {
		return nil, fmt.Errorf("static assets: uninitialized")
	}
	stored.mu.Lock()
	defer stored.mu.Unlock()
	assets := Assets{Plate: stored.images[names.Plate], Mask: stored.images[names.Mask], Passes: make(map[string]Image)}
	for _, item := range []struct {
		name   string
		target **Image
	}{{names.Disc, &assets.Disc}, {names.Water, &assets.Water}, {names.Trees, &assets.Trees}} {
		if item.name != "" {
			image := stored.images[item.name]
			*item.target = &image
		}
	}
	for pass, name := range names.Passes {
		assets.Passes[pass] = stored.images[name]
	}
	return newRenderer(config, assets, true)
}

type Renderer struct {
	requestMu                    sync.Mutex
	config                       Config
	assets                       Assets
	camera                       Camera
	open, near, closeTrees       []bool
	beamTrees, beamWater         []bool
	overrideWater                Image
	overrideBeamWater            []bool
	passErrors                   map[string]error
	hazeEnabled                  bool
	rim                          []int
	moon                         [3]float64
	top, bottom, shore, shoreRow float64
	laser                        LaserConfig
}

// DefaultConfig returns the usual renderer settings: a camera looking 0.5
// degrees down, eight searchlights, and fire limits of 4096 frames and 1 GiB.
func DefaultConfig() Config {
	return Config{
		LookDown: -0.5, Focal: 150, GaugeHeight: 39.61, EyeOverLake: 1129.6 - 1076,
		Fixtures:         FixtureConfig{FloodPasses: [2]string{"flood-left", "flood-right"}, FloodGain: 400, SpotGain: 150},
		Overhead:         OverheadConfig{Gain: 1, Colour: [3]float64{0.62, 0.8, 1}},
		Fire:             DefaultFireConfig(),
		Beam:             UFOConfig{Out: 900, Radius: 90, Apex: 450, Gain: 40, Halo: 0.1},
		MoonDiscAltitude: 5.45, MoonAltitude: 3, MoonScale: 6, SkyGlowGain: 0.02,
		Search: SearchConfig{Gain: 0.05, Count: 8}, HazeGain: 0.6, MaxFireFrames: 4096, MaxFireBytes: DefaultMaxFireBytes,
	}
}

// NewRenderer builds a Renderer from config and assets. It keeps its own copy
// of every image, rounded to float32, so later changes to assets do not reach
// it.
func NewRenderer(config Config, assets Assets) (*Renderer, error) {
	return newRenderer(config, assets, false)
}

// newRenderer checks config, plate and every other image, and the fire frame
// count and bytes, then works out the sky mask, rim, near and close trees,
// moon colour, camera and laser positions. When immutable is true, images are
// checked and shared, not copied. A bad pass is noted and only refused when
// that pass is used.
func newRenderer(config Config, assets Assets, immutable bool) (*Renderer, error) {
	imageForRenderer := rendererImage
	if immutable {
		imageForRenderer = checkedRendererImage
	}
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("lighting config: %w", err)
	}
	if err := assets.Plate.validate(); err != nil {
		return nil, fmt.Errorf("plate: %w", err)
	}
	if assets.Fire != nil {
		layers := make([]FireLayer, 0)
		for sequence, source := range assets.Fire {
			if len(source.Images) > config.MaxFireFrames-len(layers) {
				return nil, fmt.Errorf("%w: loaded frames exceed maximum %d", ErrFireResources, config.MaxFireFrames)
			}
			for index := range source.Images {
				layers = append(layers, FireLayer{Sequence: FireSequence(sequence), Index: index})
			}
		}
		workingSet, err := MeasureFireWorkingSet(layers, assets.Plate.Width, assets.Plate.Height, config.MaxFireFrames)
		if err != nil {
			return nil, err
		}
		if err := CheckFireBudget(workingSet, config.MaxFireBytes); err != nil {
			return nil, err
		}
	}
	plate, err := imageForRenderer("plate", assets.Plate, 0, 0)
	if err != nil {
		return nil, err
	}
	mask, err := imageForRenderer("mask", assets.Mask, plate.Width, plate.Height)
	if err != nil {
		return nil, err
	}
	r := &Renderer{config: config, assets: Assets{Plate: plate, Mask: mask}, passErrors: make(map[string]error)}
	for _, item := range []struct {
		name   string
		input  *Image
		output **Image
	}{
		{"moon disc", assets.Disc, &r.assets.Disc},
		{"water mask", assets.Water, &r.assets.Water},
		{"tree mask", assets.Trees, &r.assets.Trees},
	} {
		if item.input == nil {
			continue
		}
		image, err := imageForRenderer(item.name, *item.input, plate.Width, plate.Height)
		if err != nil {
			return nil, err
		}
		*item.output = &image
	}
	if assets.Passes != nil {
		r.assets.Passes = make(map[string]Image, len(assets.Passes))
		for name, pass := range assets.Passes {
			image, err := imageForRenderer("pass "+strconv.Quote(name), pass, plate.Width, plate.Height)
			if err != nil {
				r.passErrors[name] = err
			}
			r.assets.Passes[name] = image
		}
	}
	if assets.Fire != nil {
		frames := new([3]FireFrames)
		loaded := 0
		for sequence, source := range assets.Fire {
			if source.Count < 0 {
				return nil, fmt.Errorf("fire %s: negative frame count", fireSequenceName(FireSequence(sequence)))
			}
			loaded += len(source.Images)
			if loaded > config.MaxFireFrames {
				return nil, fmt.Errorf("fire: loaded frames exceed maximum %d", config.MaxFireFrames)
			}
			frames[sequence] = FireFrames{Count: source.Count, Images: make(map[int]Image, len(source.Images))}
			for index, image := range source.Images {
				if index < 0 || index >= source.Count {
					return nil, fmt.Errorf("fire %s frame %d: outside count %d", fireSequenceName(FireSequence(sequence)), index, source.Count)
				}
				frames[sequence].Images[index] = cloneRendererImage(image)
			}
		}
		r.assets.Fire = frames
	}
	r.open, err = SkyOf(mask)
	if err != nil {
		return nil, fmt.Errorf("sky mask: %w", err)
	}
	rim, err := RimOf(r.open, plate.Width, plate.Height)
	if err != nil {
		return nil, fmt.Errorf("rim: %w", err)
	}
	r.rim = make([]int, plate.Width)
	for x := range r.rim {
		r.rim[x] = rim(x)
	}
	r.near, err = NearOf(r.open, rim, plate.Width, plate.Height)
	if err != nil {
		return nil, fmt.Errorf("near trees: %w", err)
	}
	if r.assets.Water != nil {
		r.beamWater = classifyBeamMask(*r.assets.Water)
	}
	if r.assets.Trees != nil {
		r.beamTrees = classifyBeamMask(*r.assets.Trees)
		for _, value := range r.assets.Trees.Light {
			if value > 0 {
				r.hazeEnabled = true
				break
			}
		}
		r.closeTrees, err = CoveredBy(*r.assets.Trees)
		if err != nil {
			return nil, fmt.Errorf("close trees: %w", err)
		}
	}
	reference := plate
	if r.assets.Disc != nil {
		reference = *r.assets.Disc
	}
	r.moon, err = Brightest(reference)
	if err != nil {
		return nil, fmt.Errorf("moon reference: %w", err)
	}
	r.camera = NewCamera(plate.Width, plate.Height, CameraSettings{Azimuth: 109.65, LookDown: config.LookDown, GaugeWidth: 70.41, Focal: config.Focal})
	half := math.Atan(config.GaugeHeight/2/config.Focal) * 180 / math.Pi
	r.top, r.bottom = config.LookDown+half, config.LookDown-half
	r.shore = math.Atan(-config.EyeOverLake/1660) * 180 / math.Pi
	r.shoreRow = (r.top - r.shore) / (r.top - r.bottom) * float64(plate.Height)
	if err := r.validateGeometry(); err != nil {
		return nil, err
	}
	r.laser = LaserConfig{Azimuth: 109.65, EyeOverLake: config.EyeOverLake, RigDeck: config.RigDeck,
		WaterZ: -config.EyeOverLake, Reach: 4000,
		FarTree:    Vec{1727.5, -617.5, 1195.5 - (1076 + config.EyeOverLake)},
		ShoreLeft:  Vec{263652.4 - 263104.5, 2148337.2 - 2148423.5, -config.EyeOverLake + 8},
		ShoreRight: Vec{263578.1 - 263104.5, 2148129.0 - 2148423.5, -config.EyeOverLake + 8}}
	return r, nil
}

// Render lights one frame at request.Seconds. Only one request runs at a time;
// a second is refused with ErrFireResources. It starts from the plate times
// the master level, adds each light in turn, with Accelerator doing the work
// where one is given, then checks every channel is finite and in float32 range
// and rounds it to float32.
func (r *Renderer) Render(request Request) (Result, error) {
	if r == nil {
		return Result{}, fmt.Errorf("lighting renderer: nil renderer")
	}
	if !r.requestMu.TryLock() {
		return Result{}, fmt.Errorf("%w: renderer request already active", ErrFireResources)
	}
	defer r.requestMu.Unlock()
	if !finite(request.Seconds) || math.Abs(request.Seconds*12) >= float64(int(^uint(0)>>1)/2) {
		return Result{}, fmt.Errorf("lighting seconds: invalid timestamp")
	}
	show := r.defaultCues()
	hasColour := request.Cues != nil
	if hasColour {
		show = cues.Frame(*request.Cues).Quantized()
	}
	if err := validateRendererCues(show); err != nil {
		return Result{}, err
	}
	workingSet := FireWorkingSet{}
	fireConfig := r.config.Fire
	if request.Stops != nil {
		fireConfig.Stops = *request.Stops
	}
	if !finite(fireConfig.Stops) || !finite(math.Exp2(fireConfig.Stops)) || math.Exp2(fireConfig.Stops) == 0 {
		return Result{}, fmt.Errorf("fire stops: outside finite exposure range")
	}
	if (r.assets.Fire != nil || request.FireFrames != nil) && show.Fire > 1e-4 {
		source := r.assets.Fire
		if request.FireFrames != nil {
			source = request.FireFrames
		}
		counts := [3]int{}
		for sequence, frames := range source {
			counts[sequence] = frames.Count
		}
		layers, err := PlanFire(fireConfig, FireCues{Level: show.Fire, Lit: show.FireLit, Since: show.FireSince}, counts)
		if err != nil {
			return Result{}, fmt.Errorf("fire plan: %w", err)
		}
		workingSet, err = MeasureFireWorkingSet(layers, r.assets.Plate.Width, r.assets.Plate.Height, r.config.MaxFireFrames)
		if err != nil {
			return Result{}, err
		}
		if err := CheckFireBudget(workingSet, r.config.MaxFireBytes); err != nil {
			return Result{}, err
		}
	}
	waterMask := r.beamWater
	water := Image{}
	if r.assets.Water != nil {
		water = *r.assets.Water
	}
	if request.Water != nil {
		var err error
		if sameRendererImage(*request.Water, water) {
			waterMask = r.beamWater
		} else if sameRendererImage(*request.Water, r.overrideWater) {
			water, waterMask = r.overrideWater, r.overrideBeamWater
		} else {
			water, err = rendererImage("water mask", *request.Water, r.assets.Plate.Width, r.assets.Plate.Height)
			if err != nil {
				return Result{}, err
			}
			waterMask = classifyBeamMask(water)
			r.overrideWater, r.overrideBeamWater = water, waterMask
		}
	}
	shift := 0.0
	if r.assets.Disc != nil {
		_, from, _ := r.camera.Project(Toward(109.65, r.config.MoonDiscAltitude).Scale(1e6))
		_, to, _ := r.camera.Project(Toward(109.65, show.MoonAlt).Scale(1e6))
		shift = to - from
	}
	var sky Image
	if request.Accelerator != nil {
		var err error
		sky, err = request.Accelerator.Base(r.assets.Plate, r.assets.Disc, r.open, show.Master, shift, show.Moon)
		if err != nil {
			return Result{}, fmt.Errorf("lighting base: %w", err)
		}
		if err := sky.validate(); err != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height {
			return Result{}, fmt.Errorf("lighting base: invalid output")
		}
	} else {
		sky = cloneRendererImage(r.assets.Plate)
		for i := range sky.Light {
			sky.Light[i] *= show.Master
		}
		if r.assets.Disc != nil {
			if err := AddMoonDisc(sky, *r.assets.Disc, r.open, MoonDiscConfig{Shift: shift, Level: show.Moon}); err != nil {
				return Result{}, fmt.Errorf("moon disc: %w", err)
			}
		}
	}
	if r.assets.Passes != nil && show.Overhead > 1e-4 {
		pass, err := r.pass("overhead")
		if err != nil {
			return Result{}, err
		}
		if request.Accelerator != nil {
			sky, err = request.Accelerator.Fixtures(sky, []FixtureLayer{{Name: "overhead", Amount: r.config.Overhead.Gain * show.Overhead, Colour: r.config.Overhead.Colour}}, map[string]Image{"overhead": pass})
		} else {
			err = AddOverhead(sky, pass, r.config.Overhead, show.Overhead)
		}
		if err != nil {
			return Result{}, fmt.Errorf("overhead: %w", err)
		}
		if err := sky.validate(); err != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height {
			return Result{}, fmt.Errorf("overhead: invalid output")
		}
	}
	front := Image{Width: sky.Width, Height: sky.Height, Light: make([]float64, len(sky.Light))}
	if r.assets.Passes != nil {
		layers, err := PlanFixtures(r.config.Fixtures, FixtureCues{FloodLeft: show.FloodLeft, FloodRight: show.FloodRight, Spots: show.Spots, SpotAim: show.SpotAim, HasColour: hasColour, Colour: show.Colour})
		if err != nil {
			return Result{}, fmt.Errorf("fixtures: %w", err)
		}
		passes := make(map[string]Image, len(layers))
		for _, layer := range layers {
			pass, err := r.pass(layer.Name)
			if err != nil {
				return Result{}, err
			}
			passes[layer.Name] = pass
		}
		if request.Accelerator != nil && len(layers) > 0 {
			sky, err = request.Accelerator.Fixtures(sky, layers, passes)
			if err == nil && (sky.validate() != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height) {
				err = fmt.Errorf("invalid output")
			}
		} else {
			err = AddFixtures(sky, layers, passes)
		}
		if err != nil {
			return Result{}, fmt.Errorf("fixtures: %w", err)
		}
	}
	if (r.assets.Fire != nil || request.FireFrames != nil) && show.Fire > 1e-4 {
		fire, err := r.renderFireAccelerated(fireConfig, FireCues{Level: show.Fire, Lit: show.FireLit, Since: show.FireSince}, request.Accelerator, request.FireFrames)
		if err != nil {
			return Result{}, err
		}
		for i := range front.Light {
			front.Light[i] += fire.Light[i]
		}
	}
	glow := SkyGlowInput{Open: r.open, Rim: r.rim, Camera: r.camera, Moon: r.moon,
		Gain: r.config.SkyGlowGain, Azimuth: 109.65, MoonAltitude: show.MoonAlt,
		MoonScale: r.config.MoonScale, MoonLevel: show.Moon, Master: show.Master}
	if request.Accelerator != nil {
		var err error
		sky, err = request.Accelerator.SkyGlow(sky, glow)
		if err != nil {
			return Result{}, fmt.Errorf("sky glow: %w", err)
		}
		if err := sky.validate(); err != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height {
			return Result{}, fmt.Errorf("sky glow: invalid output")
		}
	} else {
		AddSkyGlow(sky, glow)
	}
	strobeInput := StrobeInput{Open: r.open, Near: r.near, Rim: r.rim, Camera: r.camera,
		Moon: r.moon, FarTree: r.laser.FarTree, Azimuth: 109.65, Top: r.top, Bottom: r.bottom,
		Shore: r.shore, Seconds: request.Seconds, White: show.White, Magenta: show.Magenta}
	level := 0.0
	if request.Accelerator != nil {
		var err error
		sky, level, err = request.Accelerator.Strobes(sky, strobeInput)
		if err != nil {
			return Result{}, fmt.Errorf("strobes: %w", err)
		}
		if err := sky.validate(); err != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height || !finite(level) {
			return Result{}, fmt.Errorf("strobes: invalid output")
		}
	} else {
		level = AddStrobes(sky, strobeInput)
	}
	searchInput := SearchInput{Open: r.open, CloseTrees: r.closeTrees,
		Camera: r.camera, Moon: r.moon, FarTree: r.laser.FarTree, Azimuth: 109.65, ShoreRow: r.shoreRow,
		Level: show.Search, Gate: show.SearchGate, Phase: show.SearchPhase,
		MovePhase: show.SearchMovePhase, Move: show.SearchMove}
	if request.Accelerator != nil {
		var err error
		sky, front, err = request.Accelerator.Searchlights(sky, front, r.config.Search, searchInput)
		if err != nil {
			return Result{}, fmt.Errorf("searchlights: %w", err)
		}
		if sky.validate() != nil || front.validate() != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height || front.Width != sky.Width || front.Height != sky.Height {
			return Result{}, fmt.Errorf("searchlights: invalid output")
		}
	} else {
		AddSearchlights(sky, front, r.config.Search, searchInput)
	}
	laserInput := LaserInput{Open: r.open, CloseTrees: r.closeTrees, Rim: r.rim,
		Camera: r.camera, Moon: r.moon, Seconds: request.Seconds, ShoreRow: r.shoreRow,
		Cues: LaserCues{Rig: show.Rig, Sweep: show.Sweep, Gate: show.Gate, GateLeft: show.GateLeft,
			GateRight: show.GateRight, FigurePhase: show.FigurePhase, Beams: show.Beams, Parity: show.Parity,
			Colour: show.Colour, HasColour: hasColour, Figure: show.Figure}}
	if request.Accelerator != nil {
		var err error
		sky, front, err = request.Accelerator.Lasers(sky, front, r.laser, laserInput)
		if err != nil {
			return Result{}, fmt.Errorf("lasers: %w", err)
		}
		if sky.validate() != nil || front.validate() != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height || front.Width != sky.Width || front.Height != sky.Height {
			return Result{}, fmt.Errorf("lasers: invalid output")
		}
	} else {
		AddLasers(sky, front, r.laser, laserInput)
	}
	trees := Image{}
	if r.assets.Trees != nil {
		trees = *r.assets.Trees
		if r.hazeEnabled {
			input := BackHazeInput{Camera: r.camera, Moon: r.moon,
				Positions: Backlights(r.config.EyeOverLake), Levels: [2]float64{show.FloodLeft, show.FloodRight},
				Gain: r.config.HazeGain, Seconds: request.Seconds}
			if request.Accelerator != nil {
				var err error
				front, err = request.Accelerator.BackHaze(front, r.closeTrees, input)
				if err != nil {
					return Result{}, fmt.Errorf("back haze: %w", err)
				}
				if front.validate() != nil || front.Width != sky.Width || front.Height != sky.Height {
					return Result{}, fmt.Errorf("back haze: invalid output")
				}
			} else {
				addBackHaze(front, r.closeTrees, input)
			}
		}
	}
	if show.Overhead > 1e-4 {
		beam := r.config.Beam
		beam.Azimuth, beam.WaterZ = 109.65, -r.config.EyeOverLake
		input := UFOInput{Trees: trees, Water: water, Camera: r.camera, Moon: r.moon, Seconds: request.Seconds, Cues: UFOCues{Overhead: show.Overhead}}
		if request.Accelerator != nil {
			var err error
			sky, front, err = request.Accelerator.Beam(sky, front, beam, input, r.beamTrees, waterMask)
			if err != nil {
				return Result{}, fmt.Errorf("beam: %w", err)
			}
			if sky.validate() != nil || front.validate() != nil || sky.Width != r.assets.Plate.Width || sky.Height != r.assets.Plate.Height || front.Width != sky.Width || front.Height != sky.Height {
				return Result{}, fmt.Errorf("beam: invalid output")
			}
		} else {
			addUFOBeam(sky, front, beam, input, r.beamTrees, waterMask)
		}
	}
	for _, item := range []struct {
		name  string
		image Image
	}{{"sky", sky}, {"front", front}} {
		for i, value := range item.image.Light {
			if !finite(value) || math.Abs(value) > math.MaxFloat32 {
				return Result{}, fmt.Errorf("%s channel %d: outside finite float32 range", item.name, i)
			}
			item.image.Light[i] = float64(float32(value))
		}
	}
	return Result{Sky: sky, Front: front, Strobe: level, FireWorkingSet: workingSet}, nil
}

// defaultCues returns the cues used when a request carries none: master and
// moon full, ten beams in the fan figure, gates open, and -1 for the rig,
// sweep, strobes and per-side gates, which the light code reads as not set.
func (r *Renderer) defaultCues() cues.Frame {
	return cues.Frame{Rig: -1, Sweep: -1, White: -1, Magenta: -1, Beams: 10, Gate: 1,
		Parity: -1, Master: 1, Moon: 1, MoonAlt: r.config.MoonAltitude, Figure: "fan",
		GateLeft: -1, GateRight: -1, SearchMove: "sweep", SearchGate: 1}
}

// pass returns the pass image named name. It refuses a pass that is missing or
// that failed its checks when the renderer was built.
func (r *Renderer) pass(name string) (Image, error) {
	pass, found := r.assets.Passes[name]
	if !found {
		return Image{}, fmt.Errorf("pass %q: missing", name)
	}
	if err := r.passErrors[name]; err != nil {
		return Image{}, err
	}
	return pass, nil
}

// renderFire is renderFireAccelerated with no accelerator.
func (r *Renderer) renderFire(config FireConfig, cue FireCues, requested ...*[3]FireFrames) (Image, error) {
	return r.renderFireAccelerated(config, cue, nil, requested...)
}

// renderFireAccelerated renders the fire layer for cue from the frames in
// requested, if one is given and not nil, otherwise the renderer's own. It
// plans the layers, checks them against the fire limits, checks each frame it
// needs, then draws with accelerator if there is one, otherwise RenderFire.
func (r *Renderer) renderFireAccelerated(config FireConfig, cue FireCues, accelerator BaseAccelerator, requested ...*[3]FireFrames) (Image, error) {
	source := r.assets.Fire
	if len(requested) != 0 && requested[0] != nil {
		source = requested[0]
	}
	if source == nil {
		source = new([3]FireFrames)
	}
	input := FireInput{Width: r.assets.Plate.Width, Height: r.assets.Plate.Height, Cues: cue, CloseTrees: r.closeTrees}
	counts := [3]int{}
	for sequence, frames := range source {
		counts[sequence] = frames.Count
		input.Frames[sequence] = FireFrames{Count: frames.Count, Images: make(map[int]Image)}
	}
	layers, err := PlanFire(config, cue, counts)
	if err != nil {
		return Image{}, fmt.Errorf("fire plan: %w", err)
	}
	workingSet, err := MeasureFireWorkingSet(layers, input.Width, input.Height, r.config.MaxFireFrames)
	if err != nil {
		return Image{}, err
	}
	if err := CheckFireBudget(workingSet, r.config.MaxFireBytes); err != nil {
		return Image{}, err
	}
	for _, layer := range layers {
		frames := input.Frames[layer.Sequence].Images
		if _, found := frames[layer.Index]; found {
			continue
		}
		name := fmt.Sprintf("fire %s frame %d", fireSequenceName(layer.Sequence), layer.Index)
		pass, found := source[layer.Sequence].Images[layer.Index]
		if !found {
			return Image{}, fmt.Errorf("%s: missing", name)
		}
		frames[layer.Index], err = rendererFireImage(name, pass, input.Width, input.Height)
		if err != nil {
			return Image{}, err
		}
	}
	var fire Image
	if accelerator != nil {
		fire, err = accelerator.Fire(config, input)
	} else {
		fire, err = RenderFire(config, input)
	}
	if err != nil {
		return Image{}, fmt.Errorf("fire: %w", err)
	}
	if err := fire.validate(); err != nil || fire.Width != input.Width || fire.Height != input.Height {
		return Image{}, fmt.Errorf("fire: invalid output")
	}
	return fire, nil
}

// fireSequenceName is the name of a fire sequence: steady, swell or burst.
func fireSequenceName(sequence FireSequence) string {
	return [3]string{"steady", "swell", "burst"}[sequence]
}

// cloneRendererImage returns image with a Light slice of its own.
func cloneRendererImage(image Image) Image {
	image.Light = append([]float64(nil), image.Light...)
	return image
}

// sameRendererImage reports whether a and b are the same size, not empty, and
// every value of a, rounded to float32, equals the one in b.
func sameRendererImage(a, b Image) bool {
	if a.Width != b.Width || a.Height != b.Height || len(a.Light) == 0 || len(a.Light) != len(b.Light) {
		return false
	}
	for i, value := range a.Light {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 || float64(float32(value)) != b.Light[i] {
			return false
		}
	}
	return true
}

// checkedRendererImage returns image as given after checking it is valid and,
// when width is not 0, is width by height. It does not copy it or look at its
// values.
func checkedRendererImage(name string, image Image, width, height int) (Image, error) {
	if err := image.validate(); err != nil {
		return Image{}, fmt.Errorf("%s: %w", name, err)
	}
	if width != 0 && (image.Width != width || image.Height != height) {
		return Image{}, fmt.Errorf("%s: is %dx%d, the frame %dx%d", name, image.Width, image.Height, width, height)
	}
	return image, nil
}

// rendererImage is checkedRendererImage that also returns a copy, with every
// value rounded to float32, and refuses a value that is not finite or lies
// outside float32 range.
func rendererImage(name string, image Image, width, height int) (Image, error) {
	if err := image.validate(); err != nil {
		return Image{}, fmt.Errorf("%s: %w", name, err)
	}
	if width != 0 && (image.Width != width || image.Height != height) {
		return Image{}, fmt.Errorf("%s: is %dx%d, the frame %dx%d", name, image.Width, image.Height, width, height)
	}
	out := cloneRendererImage(image)
	for i, value := range out.Light {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return Image{}, fmt.Errorf("%s channel %d: outside finite float32 range", name, i)
		}
		out.Light[i] = float64(float32(value))
	}
	return out, nil
}

// rendererFireImage checks a fire frame is valid, width by height, finite and
// in float32 range. It returns image itself unless a value changes when rounded
// to float32, in which case it returns a rounded copy.
func rendererFireImage(name string, image Image, width, height int) (Image, error) {
	if err := image.validate(); err != nil {
		return Image{}, fmt.Errorf("%s: %w", name, err)
	}
	if image.Width != width || image.Height != height {
		return Image{}, fmt.Errorf("%s: is %dx%d, the frame %dx%d", name, image.Width, image.Height, width, height)
	}
	needsCopy := false
	for i, value := range image.Light {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return Image{}, fmt.Errorf("%s channel %d: outside finite float32 range", name, i)
		}
		if value != float64(float32(value)) {
			needsCopy = true
		}
	}
	if needsCopy {
		image = cloneRendererImage(image)
		for i, value := range image.Light {
			image.Light[i] = float64(float32(value))
		}
	}
	return image, nil
}

// validateGeometry refuses a camera whose size is not the plate's, whose focal
// length is not positive, whose axes are not unit length, or whose top, bottom
// or shore row is not valid.
func (r *Renderer) validateGeometry() error {
	cam := r.camera
	if cam.Width != float64(r.assets.Plate.Width) || cam.Height != float64(r.assets.Plate.Height) || !finite(cam.Focal) || cam.Focal <= 0 {
		return fmt.Errorf("lighting camera: invalid dimensions or focal length")
	}
	for _, axis := range []Vec{cam.Forward, cam.Right, cam.Up} {
		if !finite(axis.X) || !finite(axis.Y) || !finite(axis.Z) || math.Abs(axis.Dot(axis)-1) > 1e-10 {
			return fmt.Errorf("lighting camera: invalid projection basis")
		}
	}
	if !finite(r.top) || !finite(r.bottom) || r.top <= r.bottom || !finite(r.shoreRow) {
		return fmt.Errorf("lighting camera: invalid frame or shore projection")
	}
	return nil
}

// validate refuses a config that has a value not finite, a bad camera, a
// search count outside 1 to 4096, a fire or beam setting that is not positive,
// no fire frame or byte limit, or an empty flood pass name.
func (c Config) validate() error {
	for _, value := range []float64{c.LookDown, c.Focal, c.GaugeHeight, c.EyeOverLake,
		c.Fixtures.FloodGain, c.Fixtures.SpotGain, c.Overhead.Gain,
		c.Overhead.Colour[0], c.Overhead.Colour[1], c.Overhead.Colour[2],
		c.Fire.Gain, c.Fire.Decay, c.Fire.FPS, c.Fire.Pulse, c.Fire.Flash, c.Fire.Snap,
		c.Fire.SwellFall, c.Fire.BurstGain, c.Fire.Top, c.Fire.Stops,
		c.Beam.Out, c.Beam.Radius, c.Beam.Apex, c.Beam.Gain, c.Beam.Halo,
		c.MoonDiscAltitude, c.MoonAltitude, c.MoonScale, c.SkyGlowGain, c.Search.Gain, c.HazeGain, c.RigDeck} {
		if !finite(value) {
			return fmt.Errorf("nonfinite setting")
		}
	}
	if c.Focal <= 0 || c.GaugeHeight <= 0 || math.Abs(c.LookDown) >= 90 || c.EyeOverLake <= 0 {
		return fmt.Errorf("invalid camera geometry")
	}
	if c.Search.Count <= 0 || c.Search.Count > 4096 {
		return fmt.Errorf("search count: expected 1 through 4096")
	}
	if c.MaxFireFrames <= 0 {
		return fmt.Errorf("fire frame maximum: expected positive bound")
	}
	if c.MaxFireBytes <= 0 {
		return fmt.Errorf("fire byte maximum: expected positive bound")
	}
	if c.Fire.FPS <= 0 || c.Fire.Decay <= 0 || c.Fire.Snap <= 0 || c.Fire.SwellFall <= 0 || c.Fire.Top <= 0 {
		return fmt.Errorf("invalid fire timing or shoulder")
	}
	if c.Beam.Out <= 0 || c.Beam.Radius <= 0 || c.Beam.Apex <= 0 || c.MoonScale <= 0 {
		return fmt.Errorf("invalid beam or moon geometry")
	}
	if c.Fixtures.FloodPasses[0] == "" || c.Fixtures.FloodPasses[1] == "" {
		return fmt.Errorf("flood passes: expected two names")
	}
	return nil
}

// validateRendererCues refuses cues with a value that is not finite, Beams
// outside 0 to 4096, or a search move phase over 1e9 in size.
func validateRendererCues(c cues.Frame) error {
	for _, value := range []float64{c.Rig, c.Sweep, c.Colour[0], c.Colour[1], c.Colour[2], c.Gate,
		c.FigurePhase, c.GateLeft, c.GateRight, c.FloodLeft, c.FloodRight, c.Spots, c.SpotAim,
		c.Search, c.SearchPhase, c.SearchMovePhase, c.SearchGate, c.Master, c.Overhead,
		c.Fire, c.FireLit, c.Moon, c.MoonAlt} {
		if !finite(value) {
			return fmt.Errorf("lighting cues: nonfinite value")
		}
	}
	if c.Beams < 0 || c.Beams > 4096 {
		return fmt.Errorf("lighting cues: beams outside 0 through 4096")
	}
	if math.Abs(c.SearchMovePhase) > 1e9 {
		return fmt.Errorf("lighting cues: search move phase outside supported range")
	}
	return nil
}
