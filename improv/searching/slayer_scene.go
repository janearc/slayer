package searching

import (
	"encoding/json"
	"fmt"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/scene/thing"
	"io"
	"math"
	"time"
)

// stageJob is scene content only. Slayer owns tracing, tone, files, caching
// and the film clock. All scene inputs are inline in the job's with block.
type stageJob struct {
	Stage StagePlan `yaml:"stage"`
	From  float64   `yaml:"from"`
	Until float64   `yaml:"until"`
	BPM   float64   `yaml:"bpm"`
}

// readStageJob reads the stage parameters from a job, rejecting unknown ones.
func readStageJob(with map[string]any) (stageJob, error) {
	var p stageJob
	for k := range with {
		if k != "stage" && k != "from" && k != "until" && k != "bpm" {
			return p, fmt.Errorf("heard: unknown parameter %q", k)
		}
	}
	raw, err := json.Marshal(with)
	if err != nil {
		return p, err
	}
	// yamlfile applies the same strict unknown-field validation as film plans.
	if err = yamlfile.Read("heard scene", raw, &p); err != nil {
		return p, err
	}
	if err = p.Stage.check(); err != nil {
		return p, err
	}
	if p.Stage.Kind == "listening" || p.Stage.Kind == "continuous-field" {
		return p, fmt.Errorf("heard: %s needs an explicit data-backed scene adapter", p.Stage.Kind)
	}
	if p.From < 0 || !bounded(p.Until, p.From+.001, 1e6) || !bounded(p.BPM, 1, 1000) {
		return p, fmt.Errorf("heard: invalid absolute clock")
	}
	return p, nil
}

// RegisterSlayerScenes connects authored geometry to slayer draw. It does
// not invoke the searching renderer or its bespoke output pipeline.
func RegisterSlayerScenes() error {
	if err := named.RegisterAuthored(named.Authored{
		FastLighting: true, Name: "heard", About: "Procedural heard. characters and observatory stages", Unit: "m",
		Width: 1280, Height: 720, Rays: 2, Wide: 65 * maths.Degree,
		Tone:  colour.Tone{Exposure: 1.6, Knee: .75, Top: 1.8, Bloom: .22},
		Check: func(with map[string]any) error { _, err := readStageJob(with); return err },
		Build: func(with map[string]any, said io.Writer) (*thing.World, maths.Subspace, func(time.Duration) (maths.Subspace, error), error) {
			p, err := readStageJob(with)
			if err != nil {
				return nil, maths.Subspace{}, nil, err
			}
			s, err := newHeardStage(&p.Stage, "")
			if err != nil {
				return nil, maths.Subspace{}, nil, err
			}
			tick := func(at time.Duration) (maths.Subspace, error) {
				t := at.Seconds()
				u := math.Max(0, math.Min(1, (t-p.From)/(p.Until-p.From)))
				return s.update(&p.Stage, State{At: t, Beat: t * p.BPM / 60}, u)
			}
			eye, err := tick(time.Duration(p.From * float64(time.Second)))
			if err != nil {
				return nil, maths.Subspace{}, nil, err
			}
			fmt.Fprintf(said, "heard %s: %d surfaces, %d volumes\n", p.Stage.Kind, len(s.world.Things), len(s.world.Volumes))
			return s.world, eye, tick, nil
		}}); err != nil {
		return err
	}
	return registerSlayerFilm()
}
