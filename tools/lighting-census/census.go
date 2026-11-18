// Package census inventories opt-in lighting comparisons; it does not render
// unless the separate draw-tagged harness is explicitly invoked.
package census

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/smoke"
)

// Mode identifies an actual public renderer configuration. GPU modes use
// production density baking, including the two approximate lighting modes.
type Mode struct {
	Name       string `json:"name"`
	Fidelity   string `json:"fidelity"`
	Definition string `json:"definition"`
}

var modes = []Mode{
	{"cpu", "reference", "CPU visible-light reference"},
	{"gpu-software", "production-density-direct-light", "GPU software traversal, direct lighting"},
	{"gpu-native-shadows", "production-density-direct-light", "native shadow queries, direct lighting"},
	{"gpu-native-meshes", "production-density-direct-light", "native mesh and shadow queries, direct lighting"},
	{"gpu-cached-lighting", "approximate", "native meshes and approximate cached smoke lighting"},
	{"gpu-fast-lighting", "approximate", "native meshes and lamp store"},
	{"gpu-faster-lighting", "approximate", "native meshes, lamp store and shared sky looks"},
}

// Modes is a copy of the supported configurations.
func Modes() []Mode { return append([]Mode(nil), modes...) }

// Case identifies an intended comparison, even before it runs.
type Case struct {
	Scene      int           `json:"scene"`
	Name       string        `json:"name"`
	Group      string        `json:"group"`
	Samples    int           `json:"samples_per_side"`
	Bands      int           `json:"additional_spectral_bands"`
	BandValues []maths.Band  `json:"bands,omitempty"`
	Palette    []maths.Light `json:"palette,omitempty"`
	Tone       colour.Tone   `json:"tone"`
	Gauge      film.Gauge    `json:"gauge"`
	Wide       float64       `json:"field_of_view_radians"`
	Mode       Mode          `json:"mode"`
	Status     string        `json:"status"`
	Inputs     []string      `json:"declared_inputs,omitempty"`
}

// Key is stable across mode ordering.
func (c Case) Key() string { return fmt.Sprintf("%03d/%s", c.Scene, c.Mode.Name) }

// Matrix contains every registered smoke and every public mode. A pending case
// is never treated as successful or as evidence of mode compatibility.
func Matrix() []Case {
	var result []Case
	for _, scene := range smoke.All() {
		for _, mode := range modes {
			result = append(result, Case{Scene: scene.Number, Name: scene.Name,
				Group: scene.Group(), Samples: max(1, scene.Samples),
				Bands: len(scene.Bands), BandValues: append([]maths.Band(nil), scene.Bands...),
				Palette: append([]maths.Light(nil), scene.Palette...), Tone: scene.Tone,
				Gauge: scene.Gauge, Wide: scene.Wide, Mode: mode, Status: "not-run",
				Inputs: append([]string(nil), scene.Inputs...)})
		}
	}
	return result
}

// CreateOutput claims one previously nonexistent absolute path. It never
// empties, replaces, or resumes a previous run's directory.
func CreateOutput(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("census: output must be absolute")
	}
	return os.Mkdir(path, 0o700)
}

// MinFreeBytes matches the shared render machine's current floor. Space
// checks are not reservations; concurrent writers can consume the headroom.
const MinFreeBytes uint64 = 30 * 1024 * 1024 * 1024

// ErrCapacity stops the entire run when the output filesystem cannot
// establish enough free space for the shared floor and one frame.
var ErrCapacity = errors.New("census: output capacity unavailable or below floor")

// RequiredFree adds a conservative full RGB/coverage/depth/PNG frame and
// fixed metadata allowance to the floor. An unknown or overflowed estimate
// is refused rather than treated as an empty result.
func RequiredFree(width, height int) (uint64, error) {
	if width < 1 || height < 1 || uint64(width) > (math.MaxUint64-(1<<20))/28/uint64(height) {
		return 0, fmt.Errorf("census: invalid dimensions for output estimate")
	}
	estimate := uint64(width)*uint64(height)*28 + 1<<20
	if estimate > math.MaxUint64-MinFreeBytes {
		return 0, fmt.Errorf("census: output estimate overflows")
	}
	return MinFreeBytes + estimate, nil
}
