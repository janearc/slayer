package omnipositor

import (
	"context"
	"fmt"
	"math"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

type RendererLightStage struct {
	renderer      *lights.Renderer
	fireLoader    *LoadedAssets
	accelerator   lights.BaseAccelerator
	maxFireFrames int
	maxFireBytes  int64
	width, height int
}

// NewLightStage makes a lighting stage from config and assets. It refuses a
// pass or fire frame that is not the plate's size or has a value that is not
// finite and fits in a float32.
func NewLightStage(config lights.Config, assets lights.Assets) (*RendererLightStage, error) {
	renderer, err := lights.NewRenderer(config, assets)
	if err != nil {
		return nil, err
	}
	width, height := assets.Plate.Width, assets.Plate.Height
	for name, pass := range assets.Passes {
		if err := validateLightImage(pass, width, height); err != nil {
			return nil, fmt.Errorf("pass %q: %w", name, err)
		}
	}
	if assets.Fire != nil {
		for sequence, frames := range assets.Fire {
			for index, pass := range frames.Images {
				if err := validateLightImage(pass, width, height); err != nil {
					return nil, fmt.Errorf("fire %s frame %d: %w", [3]string{"steady", "swell", "burst"}[sequence], index, err)
				}
			}
		}
	}
	return &RendererLightStage{renderer: renderer, width: width, height: height}, nil
}

// Light renders the lighting for request. It refuses a request that carries a
// fire snapshot, since only a prepared frame may bring one.
func (s *RendererLightStage) Light(ctx context.Context, request LightRequest) (LightResult, error) {
	return s.light(ctx, request, false)
}

// light renders the lighting for request. A request with a fire snapshot is
// refused unless owned is true. It refuses a nil ctx, a request of the wrong
// size, and a second fire request while one is running on the same loader.
func (s *RendererLightStage) light(ctx context.Context, request LightRequest, owned bool) (LightResult, error) {
	if ctx == nil {
		return LightResult{}, fmt.Errorf("lighting: nil context")
	}
	if err := ctx.Err(); err != nil {
		return LightResult{}, err
	}
	if s == nil || s.renderer == nil {
		return LightResult{}, fmt.Errorf("lighting: uninitialized stage")
	}
	if request.Width != s.width || request.Height != s.height {
		return LightResult{}, fmt.Errorf("lighting dimensions: request %dx%d, assets %dx%d", request.Width, request.Height, s.width, s.height)
	}
	if s.fireLoader != nil {
		if !s.fireLoader.fireRequestMu.TryLock() {
			return LightResult{}, fmt.Errorf("%w: asset fire request already active", lights.ErrFireResources)
		}
		defer s.fireLoader.fireRequestMu.Unlock()
	}
	complete := lights.CompleteCues(request.Cues)
	complete.FireSince = append([]float64(nil), request.Cues.FireSince...)
	var fireFrames *[3]lights.FireFrames
	if request.FireSnapshot != nil {
		if !owned {
			return LightResult{}, ErrUnboundExecutionSnapshot
		}
		fireFrames = &request.FireSnapshot.frames
	} else if s.fireLoader != nil {
		frames, err := s.fireLoader.loadFireBudget(ctx, request, false, s.maxFireFrames, s.maxFireBytes)
		if err != nil {
			return LightResult{}, err
		}
		fireFrames = &frames
	}
	if err := ctx.Err(); err != nil {
		return LightResult{}, err
	}
	result, err := s.renderer.Render(lights.Request{
		Seconds:     request.Seconds,
		Cues:        &complete,
		Stops:       &request.Stops,
		Water:       &request.WaterMask,
		FireFrames:  fireFrames,
		Accelerator: s.accelerator,
	})
	if err != nil {
		return LightResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return LightResult{}, err
	}
	return LightResult{Sky: result.Sky, Front: result.Front, Strobe: result.Strobe}, nil
}

// validateLightImage refuses a light image that is not width by height with
// three values a pixel, or that has a value that is not finite or does not
// fit in a float32.
func validateLightImage(input lights.Image, width, height int) error {
	if input.Width != width || input.Height != height || len(input.Light) != width*height*3 {
		return fmt.Errorf("invalid top-down light dimensions")
	}
	for _, value := range input.Light {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return fmt.Errorf("light outside finite float32 range")
		}
	}
	return nil
}
