package omnipositor

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/job"
)

// ScenePlan selects how a GPU set is decomposed. PlateEvery explicitly
// freezes surface lighting and geometry between refreshes. Scale reduces
// foreground transport resolution, not the scenery plate's resolution.
type ScenePlan struct {
	Version    int    `yaml:"version"`
	JobFile    string `yaml:"job"`
	From       int    `yaml:"from"`
	To         int    `yaml:"to"`
	Scale      int    `yaml:"transport-scale"`
	Rays       int    `yaml:"transport-rays"`
	PlateEvery int    `yaml:"plate-every"`
}

//go:embed scene.schema.json
var sceneSchemaBytes []byte

// ReadScenePlan checks the compositing recipe and its ordinary Slayer job.
// Paths are relative to the recipe, not the command's working directory.
func ReadScenePlan(path string) (ScenePlan, job.Job, error) {
	var plan ScenePlan
	schema, err := yamlfile.NewSchema("https://github.com/janearc/slayer/schemas/composite-scene-v1.json", sceneSchemaBytes)
	if err != nil {
		return plan, job.Job{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return plan, job.Job{}, err
	}
	node, err := schema.Read(path, content)
	if err != nil {
		return plan, job.Job{}, err
	}
	if err := node.Decode(&plan); err != nil {
		return plan, job.Job{}, err
	}
	if !filepath.IsAbs(plan.JobFile) {
		plan.JobFile = filepath.Join(filepath.Dir(path), plan.JobFile)
	}
	wanted, err := job.Read(plan.JobFile)
	if err != nil {
		return plan, wanted, err
	}
	return plan, wanted, plan.Validate(wanted)
}

// Validate refuses combinations that would silently reuse the wrong view.
// PlateEvery > 1 is an authored approximation even with a fixed camera:
// moving geometry and its lighting remain frozen until the next refresh.
func (plan ScenePlan) Validate(wanted job.Job) error {
	if _, said := wanted.Film().Samples(); said {
		return fmt.Errorf("composite scene: exact samples are not supported")
	}
	if plan.Version != 1 || plan.From < 0 || plan.To <= plan.From || plan.To > wanted.Clock().Frames() || plan.PlateEvery < 1 || plan.Rays < 1 || plan.Rays > 4 || (plan.Scale != 1 && plan.Scale != 2 && plan.Scale != 4) {
		return fmt.Errorf("composite scene: invalid range, version, plate interval, scale or rays")
	}
	w, _ := wanted.Film().Width()
	h, _ := wanted.Film().Height()
	side, _ := wanted.Film().Rays()
	if w <= 0 || h <= 0 || side < 1 || side > 4 || w > (1<<24)/side/side/h || w%plan.Scale != 0 || h%plan.Scale != 0 {
		return fmt.Errorf("composite scene: explicit dimensions must divide by transport-scale, rays must be 1..4, and the sample grid must fit 16777216 samples")
	}
	if wanted.Scene().Form() != "set" {
		return fmt.Errorf("composite scene: use a set job")
	}
	if _, yes := wanted.Stock(); yes {
		return fmt.Errorf("composite scene: stocks are not supported")
	}
	if _, yes := wanted.Titles(); yes {
		return fmt.Errorf("composite scene: add titles in the final edit")
	}
	if _, yes := wanted.Sound(); yes {
		return fmt.Errorf("composite scene: this study writes silent footage; add sound in the final edit")
	}
	if _, yes := wanted.Shutter(); yes {
		return fmt.Errorf("composite scene: a closed shutter is required")
	}
	if _, yes := wanted.Film().Aperture(); yes {
		return fmt.Errorf("composite scene: a pinhole is required")
	}
	if plan.PlateEvery > 1 {
		camera, _ := wanted.Camera()
		_, moving := camera.Path()
		_, zooming := camera.Zoom()
		if moving || zooming {
			return fmt.Errorf("composite scene: a camera move or zoom needs plate-every: 1")
		}
	}
	return nil
}
