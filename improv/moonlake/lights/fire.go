package lights

import (
	"fmt"
	"math"
)

type FireConfig struct {
	Gain, Decay, FPS                                     float64
	Percussive, Burst                                    bool
	SteadyFrom, Pulse, Flash, Snap, SwellFall, BurstGain float64
	Top, Stops                                           float64
}

type FireCues struct {
	Level, Lit float64
	Since      []float64
}

type FireSequence int

const (
	FireSteady FireSequence = iota
	FireSwell
	FireBurst
)

type FireLayer struct {
	Sequence FireSequence
	Index    int
	Amount   float64
}

type FireFrames struct {
	Count  int
	Images map[int]Image
}

type FireInput struct {
	Width, Height int
	Cues          FireCues
	Frames        [3]FireFrames
	CloseTrees    []bool
}

// DefaultFireConfig returns the fire settings the renderer starts with: 24
// frames a second and 23 stops of exposure.
func DefaultFireConfig() FireConfig {
	return FireConfig{Gain: 0.1, Decay: 1, FPS: 24, SteadyFrom: 42,
		Pulse: 3, Flash: 8, Snap: 0.12, SwellFall: 0.3, BurstGain: 1,
		Top: 0.6, Stops: 23}
}

// FireStrike is how hard the latest beat hits, as a multiplier. It is 1 unless
// config is percussive and a beat is lit. Otherwise it is 1 plus a spike that
// dies away with the time since the latest beat, using Flash rather than Pulse
// when that beat is the lit one.
func FireStrike(config FireConfig, cue FireCues) float64 {
	if !config.Percussive || cue.Lit < 0 || len(cue.Since) == 0 {
		return 1
	}
	beat := cue.Since[0]
	spike := config.Pulse
	if math.Abs(beat-cue.Lit) < 1e-3 {
		spike = config.Flash
	}
	return 1 + spike*math.Exp(-beat/config.Snap)
}

// FireContributions lists the fire frames to blend and how much of each. A
// steady config gives one layer for each age in cue.Since. A percussive one
// gives a steady and a swell layer from the latest beat, and a burst layer for
// each age when Burst is set. It gives nothing for a percussive config with no
// lit beat, and an error for bad frame timing or ages. Frame indexes are not
// yet limited to the frames that exist.
func FireContributions(config FireConfig, cue FireCues) ([]FireLayer, error) {
	limit := float64(int(^uint(0)>>1) / 2)
	if config.Percussive && (cue.Lit < 0 || len(cue.Since) == 0) {
		return nil, nil
	}
	if len(cue.Since) == 0 {
		return nil, nil
	}
	if !finite(config.FPS) || config.FPS <= 0 {
		return nil, fmt.Errorf("invalid fire frame timing")
	}
	for _, ago := range cue.Since {
		if !finite(ago) || ago < 0 || ago*config.FPS >= limit {
			return nil, fmt.Errorf("invalid fire age %v", ago)
		}
	}
	var layers []FireLayer
	if !config.Percussive {
		for _, ago := range cue.Since {
			layers = append(layers, FireLayer{FireSteady, int(math.Round(ago * config.FPS)),
				config.Gain * cue.Level * math.Exp(-ago/config.Decay)})
		}
		return layers, nil
	}
	beat := cue.Since[0]
	level := config.Gain * cue.Level * FireStrike(config, cue)
	swell := math.Exp(-beat / config.SwellFall)
	if amount := level * (1 - swell); !(amount <= 0) {
		if !finite(config.SteadyFrom) || config.SteadyFrom < 0 || config.SteadyFrom >= limit {
			return nil, fmt.Errorf("invalid fire frame timing")
		}
		layers = append(layers, FireLayer{FireSteady, int(config.SteadyFrom) + int(beat*config.FPS), amount})
	}
	if amount := level * swell; !(amount <= 0) {
		layers = append(layers, FireLayer{FireSwell, int(beat * config.FPS), amount})
	}
	if config.Burst {
		for _, ago := range cue.Since {
			thrown := config.Gain * cue.Level * config.BurstGain
			if math.Abs(ago-cue.Lit) < 1e-3 {
				thrown *= 2
			}
			layers = append(layers, FireLayer{FireBurst, int(math.Round(ago * config.FPS)), thrown})
		}
	}
	return layers, nil
}

// PlanFire turns FireContributions into the layers to draw, given how many
// frames each of the three sequences holds in counts. Steady and burst layers
// past the end are dropped, as are percussive ones with no amount. Other
// indexes are held to the last frame. It refuses a negative count, and a
// percussive layer whose sequence holds no frames.
func PlanFire(config FireConfig, cue FireCues, counts [3]int) ([]FireLayer, error) {
	for _, count := range counts {
		if count < 0 {
			return nil, fmt.Errorf("negative fire frame count")
		}
	}
	contributions, err := FireContributions(config, cue)
	if err != nil {
		return nil, err
	}
	var layers []FireLayer
	for _, layer := range contributions {
		count := counts[layer.Sequence]
		if !config.Percussive || layer.Sequence == FireBurst {
			if layer.Index >= count {
				continue
			}
		}
		if config.Percussive && layer.Amount <= 0 {
			continue
		}
		if count == 0 {
			return nil, fmt.Errorf("fire sequence %d holds no frames", layer.Sequence)
		}
		layer.Index = min(layer.Index, count-1)
		layers = append(layers, layer)
	}
	return layers, nil
}

// RenderFire adds up the planned fire frames, each times its layer amount,
// into a new image, leaving out pixels covered by CloseTrees, then runs
// FireShoulder on it. It refuses bad sizes, a wrong-sized mask or frame, and
// a frame that is missing.
func RenderFire(config FireConfig, input FireInput) (Image, error) {
	if !dimensions(input.Width, input.Height) {
		return Image{}, fmt.Errorf("invalid fire output dimensions")
	}
	if input.CloseTrees != nil {
		if err := validateMask(input.CloseTrees, input.Width, input.Height); err != nil {
			return Image{}, err
		}
	}
	counts := [3]int{input.Frames[0].Count, input.Frames[1].Count, input.Frames[2].Count}
	layers, err := PlanFire(config, input.Cues, counts)
	if err != nil {
		return Image{}, err
	}
	fire := Image{Width: input.Width, Height: input.Height, Light: make([]float64, input.Width*input.Height*3)}
	for _, layer := range layers {
		pass, ok := input.Frames[layer.Sequence].Images[layer.Index]
		if !ok {
			return Image{}, fmt.Errorf("missing fire sequence %d frame %d", layer.Sequence, layer.Index)
		}
		if err := pass.validate(); err != nil {
			return Image{}, err
		}
		if pass.Width != fire.Width || pass.Height != fire.Height {
			return Image{}, fmt.Errorf("fire sequence %d frame %d is %dx%d, the frame %dx%d",
				layer.Sequence, layer.Index, pass.Width, pass.Height, fire.Width, fire.Height)
		}
		for i := range fire.Light {
			if input.CloseTrees != nil && input.CloseTrees[i/3] {
				continue
			}
			fire.Light[i] += layer.Amount * pass.Light[i]
		}
	}
	FireShoulder(fire, config.Top*FireStrike(config, input.Cues), math.Exp2(config.Stops))
	return fire, nil
}

// FireShoulder softens bright pixels of fire in place. A pixel's luminance
// times gain is passed through a curve that tops out at top, and the colour is
// scaled to match, so nothing ends above top/gain and dim pixels barely move.
// An infinite luminance becomes top/gain in every channel. Pixels with
// luminance zero or less, or not a number, are left alone.
func FireShoulder(fire Image, top, gain float64) {
	for i := 0; i+2 < len(fire.Light); i += 3 {
		r, g, b := fire.Light[i], fire.Light[i+1], fire.Light[i+2]
		y := (0.2126*r + 0.7152*g + 0.0722*b) * gain
		if math.IsInf(y, 1) {
			for c := range 3 {
				fire.Light[i+c] = top / gain
			}
			continue
		}
		if !(y > 0) {
			continue
		}
		scale := -top * math.Expm1(-y/top) / y
		fire.Light[i] = r * scale
		fire.Light[i+1] = g * scale
		fire.Light[i+2] = b * scale
	}
}
