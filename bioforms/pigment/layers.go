package pigment

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// Mask gives the coverage of a mark in the bioform's rest space. Footprint
// is the width sampled by a pixel, in the same units as the rest points.
type Mask interface {
	Coverage(point maths.Vec, footprint float64) float64
}

// MaskFunc lets an anatomy supply its own region or paint chart.
type MaskFunc func(point maths.Vec, footprint float64) float64

func (mask MaskFunc) Coverage(point maths.Vec, footprint float64) float64 {
	return mask(point, footprint)
}

// Style determines how a mark combines with the existing skin reflectance.
type Style uint8

const (
	Freckle Style = iota + 1
	Scar
	Blemish
	Tattoo
)

// Layer applies a rest-space mask with the given strength. Colour is linear
// RGB. Freckles and tattoos use it as a transmission filter. Scars and
// blemishes use it as the local diffuse reflectance of the marked tissue.
type Layer struct {
	Style   Style
	Mask    Mask
	Colour  maths.Light
	Opacity float64
}

// Paint holds an ordered, immutable list of marks. Its zero value is clear.
type Paint struct {
	layers []Layer
}

// NewPaint validates the marks and keeps its own copy of the layer list.
// Supplied masks must be safe for concurrent reads during rendering.
func NewPaint(layers []Layer) (Paint, error) {
	for i, layer := range layers {
		if layer.Style < Freckle || layer.Style > Tattoo || layer.Mask == nil ||
			!share(layer.Opacity) || !share(layer.Colour.R) ||
			!share(layer.Colour.G) || !share(layer.Colour.B) {
			return Paint{}, fmt.Errorf("pigment: invalid paint layer %d", i)
		}
	}
	return Paint{layers: slices.Clone(layers)}, nil
}

// At mixes the marks in their authored order with the unmarked reflectance.
func (paint Paint) At(base maths.Light, point maths.Vec, footprint float64) maths.Light {
	for _, layer := range paint.layers {
		coverage := layer.Mask.Coverage(point, footprint)
		if !(coverage > 0) {
			continue
		}
		alpha := layer.Opacity * min(coverage, 1)
		marked := layer.Colour
		if layer.Style == Freckle || layer.Style == Tattoo {
			marked = base.Times(marked)
		}
		base = base.Scale(1 - alpha).Add(marked.Scale(alpha))
	}
	return base
}

func share(value float64) bool { return value >= 0 && value <= 1 }

func positive(value float64) bool { return value > 0 && !math.IsInf(value, 0) }

func finite(point maths.Vec) bool {
	return math.Abs(point.X) < math.Inf(1) && math.Abs(point.Y) < math.Inf(1) &&
		math.Abs(point.Z) < math.Inf(1)
}
