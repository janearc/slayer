// Package searching is the first shoot for Something answers: a checked
// song edit, deliberately separate from the observatory scene recipes.
package searching

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/beats"
)

// Plan is an authored edit, not a measured account of the music or sites.
type Plan struct {
	Name           string    `yaml:"name"`
	Music          string    `yaml:"music"`
	Score          string    `yaml:"score"`
	Width          int       `yaml:"width"`
	Height         int       `yaml:"height"`
	FPS            int       `yaml:"fps"`
	Rays           int       `yaml:"rays"`
	Lock           string    `yaml:"lock"`
	RadioOne       string    `yaml:"radio-one"`
	RadioTwo       string    `yaml:"radio-two"`
	Reveal         string    `yaml:"reveal"`
	CodaBars       int       `yaml:"coda-bars"`
	SearchCutBars  int       `yaml:"search-cut-bars"`
	BeatMotion     []float64 `yaml:"beat-motion"`
	BeatTwoAccents []string  `yaml:"beat-two-accents"`
	SearchDays     float64   `yaml:"search-days"`
	Azimuth        float64   `yaml:"shared-azimuth-deg"`
	Altitude       float64   `yaml:"shared-altitude-deg"`
	Diameter       float64   `yaml:"optical-diameter-metres"`
	Aperture       float64   `yaml:"optical-aperture-metres"`
	ArrayCount     int       `yaml:"array-count"`
	Exposure       float64   `yaml:"exposure"`
	Bloom          float64   `yaml:"bloom"`
}

// Edit is a plan bound to its checked score, with all cuts in song seconds.
type Edit struct {
	Plan                                   Plan
	Grid                                   *beats.Grid
	Lock, RadioOne, RadioTwo, Reveal, Coda float64
	Accents                                []float64
}

// Read checks the whole plan before any scene, decoder or GPU is opened.
func Read(path string) (*Edit, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var plan Plan
	if err := yamlfile.Read(path, raw, &plan); err != nil {
		return nil, err
	}
	for _, value := range []*string{&plan.Music, &plan.Score} {
		if !filepath.IsAbs(*value) {
			*value = filepath.Join(filepath.Dir(path), *value)
		}
	}
	if err := plan.check(); err != nil {
		return nil, err
	}
	grid, err := beats.Read(plan.Score)
	if err != nil {
		return nil, err
	}
	if grid.Score.Meter != 4 {
		return nil, fmt.Errorf("this shoot uses four-beat bars")
	}
	if err := grid.CheckAudio(plan.Music); err != nil {
		return nil, err
	}
	edit := &Edit{Plan: plan, Grid: grid}
	previous := 0.0
	for index, cue := range []string{plan.Lock, plan.RadioOne, plan.RadioTwo, plan.Reveal} {
		at, err := edit.Cue(cue)
		if err != nil || at <= previous || at >= grid.Score.Duration {
			return nil, fmt.Errorf("shoot cue %q is invalid or out of order: %v", cue, err)
		}
		*[]*float64{&edit.Lock, &edit.RadioOne, &edit.RadioTwo, &edit.Reveal}[index] = at
		previous = at
	}
	lastBeat := grid.BeatAt(grid.Score.Duration)
	edit.Coda = grid.SecondAt(math.Floor(lastBeat/4-float64(plan.CodaBars)) * 4)
	if edit.Coda <= edit.Reveal {
		return nil, fmt.Errorf("coda overlaps reveal")
	}
	for _, cue := range plan.BeatTwoAccents {
		at, err := edit.Cue(cue)
		if err != nil {
			return nil, err
		}
		edit.Accents = append(edit.Accents, at)
	}
	return edit, nil
}

// check keeps this first proof small and refuses non-finite authoring values.
func (plan Plan) check() error {
	if plan.Name == "" || plan.Music == "" || plan.Score == "" ||
		plan.Width < 160 || plan.Width > 480 || plan.Height < 90 ||
		plan.Height > 480 || plan.FPS < 1 || plan.FPS > 24 ||
		plan.Rays < 1 || plan.Rays > 2 || plan.SearchCutBars < 1 ||
		plan.CodaBars < 1 || plan.ArrayCount < 1 || plan.ArrayCount > 40 ||
		len(plan.BeatMotion) != 4 {
		return fmt.Errorf("invalid or excessive preview plan")
	}
	for _, value := range append([]float64{plan.SearchDays, plan.Azimuth,
		plan.Altitude, plan.Diameter, plan.Aperture, plan.Exposure, plan.Bloom}, plan.BeatMotion...) {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("non-finite or negative shoot value")
		}
	}
	if plan.Exposure == 0 || plan.Diameter <= 0 || plan.Aperture <= 0 ||
		plan.Aperture >= plan.Diameter/2 || plan.Altitude > 85 ||
		plan.BeatMotion[0] <= 0 {
		return fmt.Errorf("invalid scene dimensions or motion")
	}
	return nil
}

// Cue maps a one-based bar.beat position onto the retained song grid.
func (edit *Edit) Cue(written string) (float64, error) {
	parts := strings.Split(written, ".")
	if len(parts) != 2 {
		return 0, fmt.Errorf("cue %q needs bar.beat", written)
	}
	bar, a := strconv.Atoi(parts[0])
	beat, b := strconv.Atoi(parts[1])
	if a != nil || b != nil || bar < 1 || beat < 1 || beat > 4 {
		return 0, fmt.Errorf("invalid cue %q", written)
	}
	return edit.Grid.SecondAt(float64((bar-1)*4 + beat - 1)), nil
}
