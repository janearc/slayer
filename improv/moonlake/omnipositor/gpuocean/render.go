package gpuocean

import (
	"context"
	"errors"
	"fmt"

	"github.com/janearc/slayer/improv/moonlake/ocean"
	"github.com/janearc/slayer/oceantex"
)

// Render draws the ocean for config as a texture, from the sky in image and
// the optional moonReference. It marches a ray for each pixel against the
// waves, finds the sky each one sees directly or by reflection, and shades the
// reflected ones, in batches of MaxSamples pixels. It checks ctx between
// stages and refuses a nil ctx, a bad size, or an image of another size.
func Render(ctx context.Context, config ocean.Config, image ocean.RGB, moonReference *ocean.RGB) (*oceantex.Texture, error) {
	if ctx == nil {
		return nil, errors.New("gpuocean: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > MaxRayPixels/config.Height {
		return nil, errors.New("gpuocean: invalid frame dimensions")
	}
	if image.Width != config.Width || image.Height != config.Height {
		return nil, errors.New("gpuocean: sky and output dimensions differ")
	}
	scene, camera, err := ocean.Setup(config, image, moonReference)
	if err != nil {
		return nil, fmt.Errorf("gpuocean: setup: %w", err)
	}
	sky, err := ocean.NewSky(config, image, moonReference)
	if err != nil {
		return nil, fmt.Errorf("gpuocean: sky: %w", err)
	}
	projection, err := sky.Projection()
	if err != nil {
		return nil, err
	}
	constants, err := scene.LightingConstants()
	if err != nil {
		return nil, fmt.Errorf("gpuocean: lighting: %w", err)
	}
	layers, err := Waves(scene.Sea, camera.Seconds, camera.Seed)
	if err != nil {
		return nil, err
	}
	marchLayers, err := MarchLayers(layers)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rayCamera := RayCamera{Width: camera.Width, Height: camera.Height, Focal: config.Focal, GaugeWidth: ocean.GaugeWidth(), Eye: camera.Eye}
	rays, err := Rays(rayCamera)
	if err != nil {
		return nil, fmt.Errorf("gpuocean: camera rays: %w", err)
	}
	result := &oceantex.Texture{Width: camera.Width, Height: camera.Height, Light: make([][3]float64, len(rays)), Exposure: 1}
	for start := 0; start < len(rays); start += MaxSamples {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(start+MaxSamples, len(rays))
		marchRays := make([]MarchRay, end-start)
		for index := start; index < end; index++ {
			column := index % camera.Width
			beside := column + 1
			if beside >= camera.Width {
				beside = column - 1
			}
			adjacent := rays[index]
			if camera.Width > 1 {
				adjacent = rays[index-column+beside]
			}
			angle, err := adjacentAngle(camera.Width, rays[index], adjacent)
			if err != nil {
				return nil, err
			}
			marchRays[index-start] = MarchRay{Ray: rays[index], PixelAngle: angle}
		}
		hits, err := March(marchLayers, marchRays)
		if err != nil {
			return nil, fmt.Errorf("gpuocean: march: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		slopes, err := Slopes(layers, marchRays, hits)
		if err != nil {
			return nil, fmt.Errorf("gpuocean: slopes: %w", err)
		}
		reflections := make([]ReflectionSample, 0, len(hits))
		reflectedIndex := make([]int, 0, len(hits))
		skyDirections := make([]oceantex.Vec, 0, len(hits))
		skyIndex := make([]int, 0, len(hits))
		for index, hit := range hits {
			ray := marchRays[index].Ray
			if hit.Sky {
				skyDirections = append(skyDirections, ray.Dir)
				skyIndex = append(skyIndex, index)
				continue
			}
			reflections = append(reflections, ReflectionSample{Eye: oceantex.Vec{X: -ray.Dir.X, Y: -ray.Dir.Y, Z: -ray.Dir.Z}, SlopeX: slopes[index].X, SlopeY: slopes[index].Y, Index: float32(constants.Index)})
			reflectedIndex = append(reflectedIndex, index)
		}
		reflected, err := Reflections(reflections)
		if err != nil {
			return nil, fmt.Errorf("gpuocean: reflection: %w", err)
		}
		for index, value := range reflected {
			if value.Direction == (oceantex.Vec{}) {
				continue
			}
			skyDirections = append(skyDirections, value.Direction)
			skyIndex = append(skyIndex, reflectedIndex[index])
		}
		colours := make([][3]float32, len(hits))
		for offset := 0; offset < len(skyDirections); offset += MaxSamples {
			end := min(offset+MaxSamples, len(skyDirections))
			skyValues, err := SkySamples(image, projection, skyDirections[offset:end])
			if err != nil {
				return nil, fmt.Errorf("gpuocean: sky samples: %w", err)
			}
			for index, value := range skyValues {
				colours[skyIndex[offset+index]] = value
			}
		}
		shadeSamples := make([]ShadeSample, 0, len(reflectedIndex))
		for _, index := range reflectedIndex {
			ray := marchRays[index].Ray
			shadeSamples = append(shadeSamples, ShadeSample{Eye: oceantex.Vec{X: -ray.Dir.X, Y: -ray.Dir.Y, Z: -ray.Dir.Z}, Slope: slopes[index], Sky: colours[index]})
		}
		shaded, err := Shade(constants, shadeSamples)
		if err != nil {
			return nil, fmt.Errorf("gpuocean: shade: %w", err)
		}
		for index, value := range shaded {
			colours[reflectedIndex[index]] = value
		}
		for index, value := range colours {
			result.Light[start+index] = [3]float64{float64(value[0]), float64(value[1]), float64(value[2])}
		}
	}
	return result, nil
}
