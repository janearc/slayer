package medium

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// sedimentParabola is the four of the parabola 4*a*(1-a), which is one at its
// peak halfway along the plume.
const sedimentParabola = 4 // defined: the parabola's peak is one

// SedimentPlumeConfig describes an authored, bounded river outflow in metres.
// Extinction is the peak additional loss per metre. It is an optical parameter,
// not a measurement of sediment concentration or salinity.
type SedimentPlumeConfig struct {
	Mouth                     maths.Vec
	DirectionX, DirectionZ    float64
	Length, Depth             float64
	MouthWidth, OffshoreWidth float64
	Extinction                float64
}

// SedimentPlume is a compact, smoothly bounded approximation of suspended
// material spreading from a river mouth. Its midpoint is below Mouth.Y.
type SedimentPlume struct {
	config SedimentPlumeConfig
	dx, dz float64
	bounds maths.Box
}

// finiteSediment is whether every number of the config is finite.
func finiteSediment(config SedimentPlumeConfig) bool {
	values := []float64{config.Mouth.X, config.Mouth.Y, config.Mouth.Z,
		config.DirectionX, config.DirectionZ, config.Length, config.Depth,
		config.MouthWidth, config.OffshoreWidth, config.Extinction}
	for _, value := range values {
		if !weatherFinite(value) {
			return false
		}
	}
	return true
}

// NewSedimentPlume validates and constructs a stationary density field.
func NewSedimentPlume(config SedimentPlumeConfig) (*SedimentPlume, error) {
	if !finiteSediment(config) {
		return nil, errors.New("medium: sediment plume parameters must be finite")
	}
	length := math.Hypot(config.DirectionX, config.DirectionZ)
	if length == nought || !weatherFinite(length) || config.Length <= nought ||
		config.Depth/double == nought || config.Depth < nought ||
		config.MouthWidth/double <= nought ||
		config.OffshoreWidth/double <= nought || config.Extinction < nought {
		return nil, errors.New("medium: sediment plume needs a direction, positive dimensions and nonnegative extinction")
	}
	dx, dz := config.DirectionX/length, config.DirectionZ/length
	width := math.Max(config.MouthWidth, config.OffshoreWidth) / double
	endX := config.Mouth.X + dx*config.Length
	endZ := config.Mouth.Z + dz*config.Length
	bounds := corners(
		maths.V(math.Min(config.Mouth.X, endX)-width,
			config.Mouth.Y-config.Depth, math.Min(config.Mouth.Z, endZ)-width),
		maths.V(math.Max(config.Mouth.X, endX)+width, config.Mouth.Y,
			math.Max(config.Mouth.Z, endZ)+width))
	for _, value := range []float64{bounds.Min.X, bounds.Min.Y, bounds.Min.Z,
		bounds.Max.X, bounds.Max.Y, bounds.Max.Z} {
		if !weatherFinite(value) {
			return nil, errors.New("medium: sediment plume bounds exceed finite coordinates")
		}
	}
	if bounds.Min.X >= bounds.Max.X || bounds.Min.Y >= bounds.Max.Y ||
		bounds.Min.Z >= bounds.Max.Z {
		return nil, errors.New("medium: sediment plume bounds must have positive extent")
	}
	return &SedimentPlume{config: config, dx: dx, dz: dz, bounds: bounds}, nil
}

// Bounds encloses the full field, including the widening offshore end.
func (plume *SedimentPlume) Bounds() maths.Box { return plume.bounds }

// Density gives additional extinction per metre. The field fades smoothly at
// the source, outer end, lateral margins, surface and lower boundary.
func (plume *SedimentPlume) Density(point maths.Vec) float64 {
	config := plume.config
	x, z := point.X-config.Mouth.X, point.Z-config.Mouth.Z
	along := (x*plume.dx + z*plume.dz) / config.Length
	vertical := (point.Y - config.Mouth.Y + config.Depth/double) /
		(config.Depth / double)
	if math.IsNaN(along) || math.IsNaN(vertical) || along <= nought ||
		along >= unity || math.Abs(vertical) >= unity {
		return nought
	}
	width := config.MouthWidth + along*(config.OffshoreWidth-config.MouthWidth)
	across := (-x*plume.dz + z*plume.dx) / (width / double)
	if math.IsNaN(across) || math.Abs(across) >= unity {
		return nought
	}
	longitudinal := sedimentParabola * along * (unity - along)
	lateral := unity - across*across
	depth := unity - vertical*vertical
	return config.Extinction * longitudinal * lateral * lateral * depth * depth
}
