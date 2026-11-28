package job

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// An Azazel request accepts up to 8192 characters in one argument.
// Camera recipe fields use ASCII, so a byte limit also fits that contract.
const maxStillCameraRecipe = 8 * 1024

type stillCameraRecipe struct {
	Camera *cameraWire `json:"camera"`
	At     *quantity   `json:"at,omitempty"`
}

// StillCameraRecipe captures a path and its timing without copying job outputs
// or machine-local paths. The scene being edited supplies its own film and sources.
func (job Job) StillCameraRecipe() (string, error) {
	wire := job.wire
	if !job.Clock().Still() || job.lens.path == nil || wire.Scene.Set == "" {
		return "", fmt.Errorf("camera path authoring requires a still job referencing a set")
	}
	if len(wire.Sources) > 0 {
		return "", fmt.Errorf("camera path authoring cannot borrow job-specific sources")
	}
	for _, override := range []struct {
		key    string
		author bool
	}{
		{"print", wire.Print != nil},
		{"stock", wire.Stock != nil},
		{"titles", wire.Titles != nil},
		{"vibes", wire.Vibes != ""},
		{"fastlight", wire.Fastlight != ""},
		{"shortcuts", len(wire.Shortcuts) != 0},
		{"renderer", wire.Renderer != ""},
		{"quality", wire.Quality != ""},
		{"limits", wire.Limits != nil},
	} {
		if override.author {
			return "", fmt.Errorf("camera path authoring cannot preserve the job's %q setting", override.key)
		}
	}
	if wire.Film != nil && *wire.Film != (filmWire{}) {
		return "", fmt.Errorf("camera path authoring requires the set's own film settings")
	}
	raw, err := json.Marshal(stillCameraRecipe{Camera: wire.Camera, At: wire.At})
	if err != nil {
		return "", err
	}
	if len(raw) > maxStillCameraRecipe {
		return "", fmt.Errorf("camera path is too large to borrow into the editor")
	}
	return string(raw), nil
}

// StillCamera applies a portable still camera path to a set-based still.
// With checks the merged job against the same schema as an authored job.
func StillCamera(recipe string) Override {
	return Override{key: "still-camera", empty: recipe == "", apply: func(wire *document) error {
		if wire.film() || wire.Scene.Set == "" || wire.Camera != nil || wire.At != nil {
			return refusal{fmt.Errorf("still-camera requires a set-based still with no other camera or time override")}
		}
		if len(recipe) > maxStillCameraRecipe {
			return refusal{fmt.Errorf("still-camera recipe is too large")}
		}
		decoder := json.NewDecoder(strings.NewReader(recipe))
		decoder.DisallowUnknownFields()
		var bundle stillCameraRecipe
		if err := decoder.Decode(&bundle); err != nil {
			return refusal{fmt.Errorf("still-camera: %w", err)}
		}
		if decoder.Decode(new(json.RawMessage)) != io.EOF || bundle.Camera == nil || len(bundle.Camera.Moves) == 0 {
			return refusal{fmt.Errorf("still-camera requires one recipe containing a camera path")}
		}
		wire.Camera, wire.At = bundle.Camera, bundle.At
		return nil
	}}
}
