package explain

import (
	"fmt"
	"image/color"
	"math"
)

// Explainer is what composes a show's explainer frames, from the parts
// the show supplied.
type Explainer struct {
	parts Parts
}

// New is an explainer over a show's parts. A clock, a script and one to
// mostLights lights are needed; a ground may be left out, and then the
// hud theme's is used.
func New(parts Parts) (*Explainer, error) {
	if parts.Clock == nil || parts.Script == nil {
		return nil, fmt.Errorf("%w: an explainer needs the show's clock "+
			"and its script", ErrExplain)
	}
	if len(parts.Lights) < 1 || len(parts.Lights) > mostLights {
		return nil, fmt.Errorf("%w: %d lights; the panel shows 1 to %d",
			ErrExplain, len(parts.Lights), mostLights)
	}
	if parts.Ground == nil {
		parts.Ground = func(float64) (color.RGBA, error) {
			tones, err := colours()
			return tones["ground"], err
		}
	}
	parts.Lights = append([]Light(nil), parts.Lights...)
	return &Explainer{parts: parts}, nil
}

// Script is the explainer's script.
func (of *Explainer) Script() *Script { return of.parts.Script }

// Lamp is one light at a moment: how bright it is, and how bright its
// ring.
type Lamp struct {
	light       Light
	level, ring float64
}

// Label is the lamp's name on the panel.
func (of Lamp) Label() string { return of.light.label }

// Level is how bright the lamp is, 0 to 1.
func (of Lamp) Level() float64 { return of.level }

// Ring is how bright the lamp's ring is, 0 to 1.
func (of Lamp) Ring() float64 { return of.ring }

// Panel is everything the panel shows at a moment.
type Panel struct {
	beat     Beat
	lamps    []Lamp
	ground   color.RGBA
	chant    string
	singing  bool
	sentence Sentence
	saying   bool
}

// Beat is the beat the moment is in.
func (of Panel) Beat() Beat { return of.beat }

// Lamps is the lamps, in order, a copy.
func (of Panel) Lamps() []Lamp { return append([]Lamp(nil), of.lamps...) }

// Ground is the panel's ground.
func (of Panel) Ground() color.RGBA { return of.ground }

// Chant is the chant's line, if one is being sung.
func (of Panel) Chant() (string, bool) { return of.chant, of.singing }

// Says is the sentence standing, if any.
func (of Panel) Says() (Sentence, bool) { return of.sentence, of.saying }

// PanelAt is the panel at a moment of the show.
func (of *Explainer) PanelAt(seconds float64) (Panel, error) {
	beat, err := of.parts.Clock(seconds)
	if err != nil {
		return Panel{}, err
	}
	made := Panel{beat: beat}
	if made.ground, err = of.parts.Ground(seconds); err != nil {
		return Panel{}, err
	}
	for _, light := range of.parts.Lights {
		lamp := Lamp{light: light}
		if lamp.level, err = light.level(seconds); err != nil {
			return Panel{}, err
		}
		if lamp.ring, err = light.ring(seconds); err != nil {
			return Panel{}, err
		}
		lamp.level, lamp.ring = clamp(lamp.level), clamp(lamp.ring)
		made.lamps = append(made.lamps, lamp)
	}
	made.chant, made.singing = of.parts.Script.Chant(seconds)
	made.sentence, made.saying = of.parts.Script.Says(seconds)
	return made, nil
}

// clamp is a level held to 0 to 1; not a number is nought.
func clamp(level float64) float64 {
	if math.IsNaN(level) {
		return 0
	}
	return math.Max(0, math.Min(1, level))
}
