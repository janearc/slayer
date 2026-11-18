package searching

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

// CataloguePlan keeps measured positions separate from authored visibility
// and camera travel. No fictional flow paths are added to this layer.
type CataloguePlan struct {
	File       string     `yaml:"file"`
	SHA        string     `yaml:"sha256"`
	Start      string     `yaml:"start"`
	End        string     `yaml:"end"`
	Near       float64    `yaml:"near-parsecs"`
	Far        float64    `yaml:"far-parsecs"`
	Eye        [3]float64 `yaml:"eye-direction"`
	Wide       float64    `yaml:"wide-degrees"`
	Pixels     float64    `yaml:"point-radius-pixels"`
	HomePixels float64    `yaml:"home-radius-pixels"`
}

// catalogue is the retained celestial-data display product, not an archive
// response. The original measurements and selection remain in its JSON.
type catalogue struct {
	Kind    string           `json:"kind"`
	Source  string           `json:"source"`
	SHA     string           `json:"sha256"`
	Adapter string           `json:"adapter"`
	Method  string           `json:"distance-method"`
	Points  []cataloguePoint `json:"points"`
}

// cataloguePoint uses Galactic directions about the Sun and parsec distance.
type cataloguePoint struct {
	ID        string  `json:"id"`
	Epoch     float64 `json:"epoch"`
	Longitude float64 `json:"longitude-deg"`
	Latitude  float64 `json:"latitude-deg"`
	Parsecs   float64 `json:"parsecs"`
	Magnitude float64 `json:"g-magnitude"`
}

// readCatalogue checks the pinned small product before any GPU is opened.
func (edit *Edit) readCatalogue(path string) error {
	plan := edit.Plan.CatalogueStudy
	if plan == nil {
		return nil
	}
	if (edit.Plan.FlowStudy != nil && !edit.Plan.FullCut) || !bounded(plan.Near, .01, 1e9) ||
		!bounded(plan.Far, plan.Near, 1e9) || plan.Far == plan.Near ||
		!bounded(plan.Wide, 10, 120) || !bounded(plan.Pixels, .1, 3) ||
		!bounded(plan.HomePixels, .1, 6) || !pointOK(plan.Eye) || vector(plan.Eye).Len() < .01 {
		return fmt.Errorf("invalid catalogue camera or conflicting study")
	}
	var err error
	edit.CatalogueStart, err = edit.Cue(plan.Start)
	if err != nil {
		return err
	}
	edit.CatalogueEnd, err = edit.Cue(plan.End)
	if err != nil {
		return err
	}
	if edit.CatalogueEnd <= edit.CatalogueStart || edit.CatalogueEnd > edit.Grid.Score.Duration {
		return fmt.Errorf("unordered catalogue cues")
	}
	if !filepath.IsAbs(plan.File) {
		plan.File = filepath.Join(filepath.Dir(path), plan.File)
	}
	file, err := os.Open(plan.File)
	if err != nil {
		return err
	}
	defer file.Close()
	const limit = 8 * 1024 * 1024 // Authored upper bound on a preview product.
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return err
	}
	if len(raw) > limit || fmt.Sprintf("%x", sha256.Sum256(raw)) != plan.SHA {
		return fmt.Errorf("catalogue size or SHA256 differs from the plan")
	}
	var data catalogue
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if err := data.check(); err != nil {
		return err
	}
	edit.Catalogue = &data
	return nil
}

// check refuses missing provenance, duplicate IDs and invalid coordinates.
func (data catalogue) check() error {
	if data.Kind != "gaia-dr3-display" || data.Source == "" || len(data.SHA) != 64 ||
		data.Adapter == "" || data.Method == "" || len(data.Points) == 0 || len(data.Points) > 8192 {
		return fmt.Errorf("invalid catalogue provenance or point count")
	}
	seen := map[string]bool{}
	for _, point := range data.Points {
		if point.ID == "" || seen[point.ID] || !bounded(point.Epoch, 1900, 2100) ||
			!bounded(point.Longitude, 0, 360) || !bounded(point.Latitude, -90, 90) ||
			!bounded(point.Parsecs, .001, 20000) || !bounded(point.Magnitude, -30, 40) {
			return fmt.Errorf("invalid catalogue point %q", point.ID)
		}
		seen[point.ID] = true
	}
	return nil
}

// distance is logarithmic camera travel with eased endpoints, not motion
// of the catalogue or a claim about the time needed to cross these scales.
func (plan CataloguePlan) distance(fraction float64) float64 {
	return math.Exp(math.Log(plan.Near) + smooth(fraction)*math.Log(plan.Far/plan.Near))
}
