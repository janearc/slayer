package omnipositor

import (
	"context"
	"fmt"
	"image"
	"math"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/gpudevelop"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/session"
)

// SceneTiming records host wall time and GPU trace time separately. Plate
// cost includes its first preparation; a reused plate reports zero cost.
type SceneTiming struct {
	Index            int     `json:"index"`
	SceneSeconds     float64 `json:"scene_seconds"`
	PlateFrame       int     `json:"plate_frame"`
	PlateSeconds     float64 `json:"plate_seconds"`
	TransportSeconds float64 `json:"transport_seconds"`
	CompositeSeconds float64 `json:"composite_seconds"`
	DevelopSeconds   float64 `json:"develop_seconds"`
	TraceGPUSeconds  float64 `json:"trace_gpu_seconds"`
}

// SceneResult contains one developed frame and its linear-light counterpart.
// Callers consume it before asking for another; no full film is held in RAM.
type SceneResult struct {
	Timing SceneTiming
	Linear *frame.Frame
	Image  *image.RGBA
}

// RenderScene connects the ordinary set session to Omnipositor. A reference
// draws every full frame; a composite draws a sharp plate and primary volume
// transport separately. Both use the same Metal developer and scene clock.
// Reuse and reduced-resolution transport are explicit visual approximations.
func RenderScene(ctx context.Context, plan ScenePlan, wanted job.Job, stamp session.Stamp, reference bool, consume func(SceneResult) error) error {
	if ctx == nil || consume == nil {
		return fmt.Errorf("composite scene: context and consumer are required")
	}
	if err := plan.Validate(wanted); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	composite, err := session.OpenComposite(ctx, wanted, stamp)
	if err != nil {
		return err
	}
	defer composite.Close()
	w, h := composite.Size()
	if w%plan.Scale != 0 || h%plan.Scale != 0 {
		return fmt.Errorf("composite scene: staged size does not divide by transport scale")
	}
	tone := composite.Tone()
	if curve, yes := wanted.Print(); yes {
		tone.Knee, tone.Top = curve.Knee(), curve.Top()
		if bloom, said := curve.Bloom(); said {
			tone.Bloom = bloom
		}
		if stops, said := curve.Stops(); said {
			if tone.Exposure == 0 {
				tone.Exposure = 1
			}
			tone.Exposure *= math.Exp2(stops)
		}
	}
	developer, err := gpudevelop.New(w, h, tone)
	if err != nil {
		return err
	}
	defer developer.Close()
	ask, err := session.NewAsk(w/plan.Scale, h/plan.Scale, plan.Rays, nil)
	if err != nil {
		return err
	}
	side, _ := wanted.Film().Rays()
	plateAsk, err := session.NewAsk(w*side, h*side, 1, nil)
	if err != nil {
		return err
	}
	var plate *frame.Frame
	var view session.Drawn
	plateIndex := -1
	for index := plan.From; index < plan.To; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		timing := SceneTiming{Index: index}
		var linear *frame.Frame
		if reference {
			start := time.Now()
			view, _, err = composite.Pass(ctx, index, session.Ask{}, gpu.CompletePicture)
			if err != nil {
				return err
			}
			linear = view.Light()
			timing.PlateSeconds = time.Since(start).Seconds()
			timing.TraceGPUSeconds = view.Profile().TraceGPU.Seconds()
			plateIndex = index
		} else {
			if plate == nil || (index-plan.From)%plan.PlateEvery == 0 {
				start := time.Now()
				view, _, err = composite.Pass(ctx, index, plateAsk, gpu.SurfacePlate)
				if err != nil {
					return err
				}
				plate, plateIndex = view.Light(), index
				timing.PlateSeconds = time.Since(start).Seconds()
				timing.TraceGPUSeconds = view.Profile().TraceGPU.Seconds()
			}
			start := time.Now()
			volume, transmission, err := composite.Pass(ctx, index, ask, gpu.VolumeTransport)
			if err != nil {
				return err
			}
			timing.TransportSeconds = time.Since(start).Seconds()
			timing.TraceGPUSeconds += volume.Profile().TraceGPU.Seconds()
			start = time.Now()
			linear, err = CompositeTransport(plate, volume.Light(), transmission)
			if err != nil {
				return err
			}
			linear, err = ResolveTransport(linear, side)
			if err != nil {
				return err
			}
			timing.CompositeSeconds = time.Since(start).Seconds()
			timing.SceneSeconds = volume.At().Seconds()
		}
		if reference {
			timing.SceneSeconds = view.At().Seconds()
		}
		timing.PlateFrame = plateIndex
		start := time.Now()
		picture, err := developer.Develop(packFrame(linear))
		if err != nil {
			return err
		}
		// Set HUD coordinates belong to the full-resolution view. A
		// supersampled plate has a different image size, so no overlay is
		// applied here; OpenComposite refuses sets that request a HUD.
		timing.DevelopSeconds = time.Since(start).Seconds()
		if err := consume(SceneResult{Timing: timing, Linear: linear, Image: picture}); err != nil {
			return err
		}
	}
	return nil
}
