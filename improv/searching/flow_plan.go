package searching

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

// FlowGeometryMaxBytes bounds authored geometry assets at read and copy time.
const FlowGeometryMaxBytes = 2 << 20

// FlowPlan is an authored motion study. It deliberately cannot label its
// points as a measured galaxy catalogue or a reconstructed velocity field.
// Points run from the outside toward the opening, in arbitrary scene units.
type FlowPlan struct {
	Travel     *flowTravel            `yaml:"travel,omitempty"`
	Geometry   *FlowGeometryReference `yaml:"geometry,omitempty"`
	Provenance string                 `yaml:"provenance"`
	Start      string                 `yaml:"start"`
	Fold       string                 `yaml:"fold"`
	Answer     string                 `yaml:"answer"`
	Eye        [3]float64             `yaml:"eye"`
	Wide       float64                `yaml:"wide-degrees"`
	Paths      []flowPath             `yaml:"paths,omitempty"`
	Landmarks  []flowPoint            `yaml:"landmarks,omitempty"`
	Clouds     []flowCloud            `yaml:"clouds,omitempty"`
	ReturnPath int                    `yaml:"return-path"`
}

// FlowGeometryReference pins a relative YAML file containing only authored
// paths, landmarks and clouds. A plan's camera and timing remain local.
type FlowGeometryReference struct {
	File string `yaml:"file"`
	SHA  string `yaml:"sha256"`
}

// flowGeometry is the reusable authored content shared by camera/timing plans.
type flowGeometry struct {
	Paths     []flowPath  `yaml:"paths"`
	Landmarks []flowPoint `yaml:"landmarks"`
	Clouds    []flowCloud `yaml:"clouds,omitempty"`
}

// resolveFlowGeometry loads a pinned relative geometry asset without changing
// any camera, timing or provenance fields. Inline geometry remains supported.
func resolveFlowGeometry(planPath string, plan *FlowPlan) error {
	if plan == nil || plan.Geometry == nil {
		return nil
	}
	if plan.Paths != nil || plan.Landmarks != nil || plan.Clouds != nil {
		return fmt.Errorf("flow geometry cannot be both inline and referenced")
	}
	if err := checkFlowGeometryReference(plan.Geometry); err != nil {
		return err
	}
	path := filepath.Join(filepath.Dir(planPath), plan.Geometry.File)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("flow geometry %q: %w", plan.Geometry.File, err)
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, FlowGeometryMaxBytes+1))
	if err != nil {
		return fmt.Errorf("read flow geometry %q: %w", plan.Geometry.File, err)
	}
	if len(raw) > FlowGeometryMaxBytes {
		return fmt.Errorf("flow geometry %q exceeds %d bytes", plan.Geometry.File, FlowGeometryMaxBytes)
	}
	return resolveFlowGeometryBytes(plan.Geometry.File, raw, plan)
}

// resolveFlowGeometryBytes validates and parses the already-read referenced
// asset bytes, then assigns only the geometry fields to the plan.
func resolveFlowGeometryBytes(file string, raw []byte, plan *FlowPlan) error {
	if plan == nil || plan.Geometry == nil {
		return nil
	}
	if err := checkFlowGeometryReference(plan.Geometry); err != nil {
		return err
	}
	if file != plan.Geometry.File {
		return fmt.Errorf("flow geometry source %q differs from reference %q", file, plan.Geometry.File)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != plan.Geometry.SHA {
		return fmt.Errorf("flow geometry %q SHA256 differs from the plan", file)
	}
	var geometry flowGeometry
	if err := yamlfile.Read(file, raw, &geometry); err != nil {
		return fmt.Errorf("parse flow geometry %q: %w", file, err)
	}
	if len(geometry.Paths) == 0 || len(geometry.Paths) > 128 ||
		len(geometry.Landmarks) > 4096 || len(geometry.Clouds) > 48 {
		return fmt.Errorf("flow geometry %q exceeds authored preview bounds", file)
	}
	plan.Paths, plan.Landmarks, plan.Clouds = geometry.Paths, geometry.Landmarks, geometry.Clouds
	return nil
}

// checkFlowGeometryReference rejects a reference that is not a clean relative
// YAML path.
func checkFlowGeometryReference(reference *FlowGeometryReference) error {
	if reference == nil || reference.File == "" || !filepath.IsLocal(reference.File) ||
		filepath.Clean(reference.File) != reference.File ||
		filepath.ToSlash(filepath.Clean(reference.File)) != reference.File ||
		!strings.HasSuffix(reference.File, ".yaml") {
		return fmt.Errorf("flow geometry file must be a clean relative YAML path")
	}
	if filepath.Dir(reference.File) == "." {
		switch reference.File {
		case "shoot.yaml", "shoot.yaml.txt", "score.yaml", "score.yaml.txt",
			"catalogue.json", "catalogue.json.txt", "render.json", "animation.mp4":
			return fmt.Errorf("flow geometry file %q is reserved by render output", reference.File)
		}
	}
	if len(reference.SHA) != sha256.Size*2 {
		return fmt.Errorf("flow geometry SHA256 must be 64 lowercase hexadecimal characters")
	}
	decoded, err := hex.DecodeString(reference.SHA)
	if err != nil || hex.EncodeToString(decoded) != reference.SHA {
		return fmt.Errorf("flow geometry SHA256 must be 64 lowercase hexadecimal characters")
	}
	return nil
}

// flowPath retains an explicit polyline, its visible width and beat delay.
// Arc length, not point index, determines the travelling light's position.
type flowPath struct {
	Points [][3]float64 `yaml:"points"`
	Width  float64      `yaml:"width"`
	Delay  float64      `yaml:"delay-beats"`
}

// flowPoint is a fixed luminous landmark. Its display size is authored.
type flowPoint struct {
	At     [3]float64 `yaml:"at"`
	Radius float64    `yaml:"radius"`
	RGB    [3]float64 `yaml:"rgb"`
}

// flowCloud supplies faint authored volume emission behind the filaments.
// Density and radius use scene units, without an astrophysical gas claim.
type flowCloud struct {
	At      [3]float64 `yaml:"at"`
	RGB     [3]float64 `yaml:"rgb"`
	Radius  float64    `yaml:"radius"`
	Density float64    `yaml:"density"`
	Seed    uint64     `yaml:"seed"`
}

// checkFlow refuses malformed geometry and unordered cues before opening
// the GPU. These bounds keep the first study within its preview budget.
func (edit *Edit) checkFlow() error {
	plan := edit.Plan.FlowStudy
	if plan == nil {
		return nil
	}
	if plan.Provenance != "authored-previs" {
		return fmt.Errorf("flow-study provenance must be authored-previs")
	}
	if len(plan.Paths) < 1 || len(plan.Paths) > 128 ||
		len(plan.Landmarks) > 4096 || len(plan.Clouds) > 48 || plan.ReturnPath < 0 ||
		plan.ReturnPath >= len(plan.Paths) || !bounded(plan.Wide, 10, 120) ||
		!pointOK(plan.Eye) || vector(plan.Eye).Len() < 1 {
		return fmt.Errorf("invalid flow-study size, camera or return path")
	}
	previous := -1.0
	for index, cue := range []string{plan.Start, plan.Fold, plan.Answer} {
		at, err := edit.cue(cue)
		if err != nil || at <= previous || at >= edit.Grid.Score.Duration {
			return fmt.Errorf("invalid or unordered flow-study cue %q", cue)
		}
		*[]*float64{&edit.FlowStart, &edit.FlowFold, &edit.FlowAnswer}[index] = at
		previous = at
	}
	if err := edit.checkFlowTravel(); err != nil {
		return err
	}
	for index, path := range plan.Paths {
		if len(path.Points) < 2 || len(path.Points) > 128 ||
			!bounded(path.Width, 0.001, 0.5) || !bounded(path.Delay, 0, 4) {
			return fmt.Errorf("flow path %d exceeds preview bounds", index)
		}
		for point, at := range path.Points {
			if !pointOK(at) || point > 0 &&
				vector(at).Sub(vector(path.Points[point-1])).Len() < 1e-5 {
				return fmt.Errorf("flow path %d has invalid segment %d", index, point)
			}
		}
	}
	for index, point := range plan.Landmarks {
		if !pointOK(point.At) || !bounded(point.Radius, 0.001, 1) {
			return fmt.Errorf("invalid flow landmark %d", index)
		}
		for _, channel := range point.RGB {
			if !bounded(channel, 0, 20) {
				return fmt.Errorf("invalid flow landmark colour %d", index)
			}
		}
	}
	for index, cloud := range plan.Clouds {
		if !pointOK(cloud.At) || !bounded(cloud.Radius, 0.1, 10) ||
			!bounded(cloud.Density, 0, 0.5) {
			return fmt.Errorf("invalid flow cloud %d", index)
		}
		for _, channel := range cloud.RGB {
			if !bounded(channel, 0, 20) {
				return fmt.Errorf("invalid flow cloud colour %d", index)
			}
		}
	}
	return nil
}

// bounded includes finite endpoints and rejects NaN through comparisons.
func bounded(value, low, high float64) bool {
	return value >= low && value <= high
}

// pointOK keeps coordinates finite and small enough for this authored set.
func pointOK(point [3]float64) bool {
	return bounded(point[0], -100, 100) && bounded(point[1], -100, 100) &&
		bounded(point[2], -100, 100)
}

// vector converts the YAML triple without changing axes or units.
func vector(point [3]float64) maths.Vec {
	return maths.V(point[0], point[1], point[2])
}

// fractions measures cumulative distance along a retained polyline.
func fractions(points [][3]float64) []float64 {
	distance := make([]float64, len(points))
	for index := 1; index < len(points); index++ {
		distance[index] = distance[index-1] +
			vector(points[index]).Sub(vector(points[index-1])).Len()
	}
	total := distance[len(distance)-1]
	for index := range distance {
		distance[index] /= total
	}
	return distance
}

// packet is a compact smooth pulse; it does not wrap to the path's start.
func packet(position, centre, width float64) float64 {
	inside := math.Max(0, 1-math.Abs(position-centre)/width)
	return inside * inside * (3 - 2*inside)
}
