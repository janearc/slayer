// Package ocean adapts oceantex's CPU wave synthesis to Slayer scene shapes.
// It creates finite, one-instant height fields; it does not reproduce the
// complete oceantex view or ocean optics.
package ocean

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
)

// MaxHeightFieldSamples is the maximum number of samples on either side of
// an authored or generated finite height field.
const MaxHeightFieldSamples = 1024

// SmoothNormalBilinearVertexSlopes names the CPU shading-normal
// approximation that bilinearly interpolates finite-difference grid slopes.
const SmoothNormalBilinearVertexSlopes = "bilinear-vertex-slopes"

// SpectralSlopeNormals names CPU shading normals interpolated from the
// wave cascades' spectral slopes, independently of the sampled geometry.
const SpectralSlopeNormals = "spectral-slopes"

// HeightFieldConfig describes one finite sample of a synthesized sea.
// Extent is the square's side in metres, Samples includes both edges, and
// Seconds is the explicit time at which the wave fields are evaluated.
// SpectralShadingNormals opts into interpolated cascade slopes for shading;
// it leaves the same sampled height geometry and its distance bounds intact.
type HeightFieldConfig struct {
	Sea                    oceantex.Sea
	Extent                 float64
	Samples                int
	Seed                   uint64
	Seconds                float64
	SpectralShadingNormals bool
	SurfaceWake            *SurfaceWake
}

// SurfaceWake is a bounded, shot-authored height perturbation. Origin is the
// craft reference at Seconds=0 in east/north metres; HeadingRadians is the
// craft's forward direction clockwise from north. The wake translates at the
// authored speed and trails aft. It is a surface-following envelope, not a
// fluid simulation or a measured Alfa wake.
type SurfaceWake struct {
	OriginEastMetres, OriginNorthMetres                              float64
	HeadingRadians, SpeedMetresPerSecond                             float64
	AmplitudeMetres, HalfWidthMetres, LengthMetres, WavelengthMetres float64
	AddToSpectralShading                                             bool
}

// NewHeightField samples every oceantex wave cascade into one finite,
// bilinear Slayer height field. The authored grid is capped at 1024 by 1024
// samples. Extent may not exceed the largest cascade's period. When
// SpectralShadingNormals is set, the same cascade samples also contribute
// an explicitly approximate, shading-only spectral slope field; geometry
// and distance bounds are unchanged.
//
// The adapter maps oceantex east/north/up to Slayer east/up/south: east is
// Slayer x, up is y, and increasing Slayer z samples decreasing ocean north.
// It samples vertical height only. Oceantex's horizontal choppy displacement,
// footprint filtering, and shore treatment are not represented by
// shape.HeightField.
func NewHeightField(config HeightFieldConfig) (*shape.HeightField, error) {
	if !finite(config.Seconds) || config.Seconds < 0 {
		return nil, errors.New("scene/ocean: seconds must be finite and nonnegative")
	}
	if !finite(config.Extent) || config.Extent <= 0 {
		return nil, errors.New("scene/ocean: extent must be finite and positive")
	}
	if config.Samples < 2 || config.Samples > MaxHeightFieldSamples {
		return nil, fmt.Errorf("scene/ocean: samples must be from 2 to %d", MaxHeightFieldSamples)
	}
	if wake := config.SurfaceWake; wake != nil {
		if !finite(wake.OriginEastMetres) || !finite(wake.OriginNorthMetres) ||
			!finite(wake.HeadingRadians) || !finite(wake.SpeedMetresPerSecond) ||
			!finite(wake.AmplitudeMetres) || !finite(wake.HalfWidthMetres) ||
			!finite(wake.LengthMetres) || !finite(wake.WavelengthMetres) ||
			wake.SpeedMetresPerSecond < 0 || wake.AmplitudeMetres <= 0 ||
			wake.HalfWidthMetres <= 0 || wake.LengthMetres <= 0 ||
			wake.WavelengthMetres <= 0 || wake.WavelengthMetres > wake.LengthMetres {
			return nil, errors.New("scene/ocean: surface wake needs finite coordinates and positive amplitude, width, length and wavelength no longer than its length")
		}
		if config.SpectralShadingNormals && !wake.AddToSpectralShading {
			return nil, errors.New("scene/ocean: spectral shading with a surface wake requires wake slopes to be included")
		}
		if !config.SpectralShadingNormals && wake.AddToSpectralShading {
			return nil, errors.New("scene/ocean: surface wake slopes may be added only when spectral shading is enabled")
		}
	}
	step := config.Extent / float64(config.Samples-1)
	if !finite(step) || step <= 0 {
		return nil, errors.New("scene/ocean: extent and samples produce an invalid sample spacing")
	}

	cascades, err := oceantex.WaveCascades(config.Sea, config.Seconds, config.Seed)
	if err != nil {
		return nil, fmt.Errorf("scene/ocean: synthesize cascades: %w", err)
	}
	if len(cascades) == 0 {
		return nil, errors.New("scene/ocean: wave synthesis returned no cascades")
	}
	var largestPeriod float64
	for index, cascade := range cascades {
		if cascade.Field == nil || !finite(cascade.Field.Extent) || cascade.Field.Extent <= 0 {
			return nil, fmt.Errorf("scene/ocean: cascade %d has invalid extent", index)
		}
		largestPeriod = math.Max(largestPeriod, cascade.Field.Extent)
	}
	if config.Extent > largestPeriod {
		return nil, fmt.Errorf("scene/ocean: extent %.6g m exceeds largest cascade period %.6g m", config.Extent, largestPeriod)
	}

	heights := make([]float64, config.Samples*config.Samples)
	var slopeX, slopeZ []float64
	if config.SpectralShadingNormals {
		slopeX = make([]float64, len(heights))
		slopeZ = make([]float64, len(heights))
	}
	for row := 0; row < config.Samples; row++ {
		// HeightField row order moves south; oceantex rows move north.
		south := -config.Extent/2 + float64(row)*step
		for column := 0; column < config.Samples; column++ {
			east := -config.Extent/2 + float64(column)*step
			var height, slopeEast, slopeNorth float64
			for _, cascade := range cascades {
				field := cascade.Field
				// Both patches share the same scene origin. Center the finite
				// requested square on each patch's periodic tile.
				x := field.Extent/2 + east
				north := field.Extent/2 - south
				height += periodicSample(field, field.Height, x, north)
				if config.SpectralShadingNormals {
					slopeEast += periodicSample(field, field.SlopeX, x, north)
					slopeNorth += periodicSample(field, field.SlopeY, x, north)
				}
			}
			if wake := config.SurfaceWake; wake != nil {
				displacement, eastSlope, northSlope := wake.sample(east, -south, config.Seconds)
				height += displacement
				if config.SpectralShadingNormals {
					if !wake.AddToSpectralShading {
						return nil, errors.New("scene/ocean: surface wake with spectral shading must opt into wake slopes")
					}
					slopeEast += eastSlope
					slopeNorth += northSlope
				}
			}
			if !finite(height) {
				return nil, errors.New("scene/ocean: combined wave height is not finite")
			}
			at := row*config.Samples + column
			heights[at] = height
			if config.SpectralShadingNormals {
				if !finite(slopeEast) || !finite(slopeNorth) {
					return nil, errors.New("scene/ocean: combined spectral slope is not finite")
				}
				// Slayer z grows south while oceantex y grows north. HeightField
				// normalizes (-dH/dx, 1, -dH/dz), so store dH/dz=-dH/dnorth.
				slopeX[at], slopeZ[at] = slopeEast, -slopeNorth
			}
		}
	}
	result, err := shape.NewHeightField(heights, config.Samples, config.Extent)
	if err != nil {
		return nil, fmt.Errorf("scene/ocean: make height field: %w", err)
	}
	if config.SpectralShadingNormals {
		result, err = result.WithApproximateSpectralNormals(slopeX, slopeZ)
		if err != nil {
			return nil, fmt.Errorf("scene/ocean: make spectral shading normals: %w", err)
		}
	}
	return result, nil
}

// sample returns the wake height and its east and north slopes at a point and
// time.
func (wake SurfaceWake) sample(east, north, seconds float64) (height, eastSlope, northSlope float64) {
	forwardEast := math.Sin(wake.HeadingRadians)
	forwardNorth := math.Cos(wake.HeadingRadians)
	aftEast, aftNorth := -forwardEast, -forwardNorth
	centerEast := wake.OriginEastMetres + forwardEast*wake.SpeedMetresPerSecond*seconds
	centerNorth := wake.OriginNorthMetres + forwardNorth*wake.SpeedMetresPerSecond*seconds
	deltaEast, deltaNorth := east-centerEast, north-centerNorth
	aft := deltaEast*aftEast + deltaNorth*aftNorth
	cross := deltaEast*forwardNorth - deltaNorth*forwardEast
	if aft <= 0 || aft >= wake.LengthMetres {
		return 0, 0, 0
	}
	longitudinal := math.Sin(math.Pi * aft / wake.LengthMetres)
	transverse := math.Exp(-0.5 * math.Pow(cross/wake.HalfWidthMetres, 2))
	phase := 2 * math.Pi * aft / wake.WavelengthMetres
	envelope := wake.AmplitudeMetres * longitudinal * transverse
	height = envelope * math.Sin(phase)
	dAft := wake.AmplitudeMetres * transverse * ((math.Pi/wake.LengthMetres)*
		math.Cos(math.Pi*aft/wake.LengthMetres)*math.Sin(phase) +
		longitudinal*(2*math.Pi/wake.WavelengthMetres)*math.Cos(phase))
	dCross := -cross / (wake.HalfWidthMetres * wake.HalfWidthMetres) * height
	eastSlope = dAft*aftEast + dCross*forwardNorth
	northSlope = dAft*aftNorth - dCross*forwardEast
	return height, eastSlope, northSlope
}

// periodicSample matches oceantex.Field's bilinear, periodic sampling for a
// coordinate within one field period.
func periodicSample(field *oceantex.Field, values []float64, east, north float64) float64 {
	gx := east / field.Extent * float64(field.N)
	gy := north / field.Extent * float64(field.N)
	xFloor, yFloor := math.Floor(gx), math.Floor(gy)
	tx, ty := gx-xFloor, gy-yFloor
	x0 := ((int(xFloor) % field.N) + field.N) % field.N
	y0 := ((int(yFloor) % field.N) + field.N) % field.N
	x1, y1 := (x0+1)%field.N, (y0+1)%field.N
	top := values[y0*field.N+x0]*(1-tx) + values[y0*field.N+x1]*tx
	bottom := values[y1*field.N+x0]*(1-tx) + values[y1*field.N+x1]*tx
	return top*(1-ty) + bottom*ty
}

// finite reports whether value is a real number, not NaN or infinite.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
