// Package regionmanifest identifies a caller-declared region draw.
package regionmanifest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Input is one named snapshot file and the digest of its actual bytes.
type Input struct {
	Name   string
	Bytes  int64
	SHA256 [32]byte
}

// Rectangle is a region in full-frame pixel coordinates.
type Rectangle struct {
	Left, Top, Width, Height int
}

// Manifest is the caller's declared inputs and effective region settings.
// TimeNS is a pointer so zero time is explicit rather than missing.
type Manifest struct {
	ClosureAsserted bool
	Inputs          []Input
	Executable      [32]byte
	Width, Height   int
	Region          Rectangle
	Renderer        string
	Rays            int
	Camera          string
	TimeNS          *int64
	// Scrooge explicitly selects whether view-dependent detail is enabled.
	Scrooge     *bool
	Settings    [32]byte
	Development [32]byte
}

// Digest validates the manifest and returns its deterministic identity.
func Digest(manifest Manifest) ([32]byte, error) {
	if !manifest.ClosureAsserted || len(manifest.Inputs) == 0 {
		return [32]byte{}, fmt.Errorf("regionmanifest: an asserted input closure is required")
	}
	if manifest.Executable == ([32]byte{}) || manifest.Settings == ([32]byte{}) ||
		manifest.Development == ([32]byte{}) || manifest.TimeNS == nil || manifest.Scrooge == nil ||
		manifest.Renderer == "" || strings.TrimSpace(manifest.Renderer) != manifest.Renderer ||
		manifest.Camera == "" || strings.TrimSpace(manifest.Camera) != manifest.Camera ||
		manifest.Rays < 1 || manifest.Rays > 64 {
		return [32]byte{}, fmt.Errorf("regionmanifest: build, settings, renderer, camera, time and valid rays are required")
	}
	box := manifest.Region
	if manifest.Width <= 0 || manifest.Height <= 0 || box.Left < 0 || box.Top < 0 ||
		box.Width <= 0 || box.Height <= 0 || box.Left >= manifest.Width ||
		box.Top >= manifest.Height || box.Width > manifest.Width-box.Left ||
		box.Height > manifest.Height-box.Top {
		return [32]byte{}, fmt.Errorf("regionmanifest: rectangle is outside the frame")
	}
	inputs := append([]Input(nil), manifest.Inputs...)
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].Name < inputs[j].Name })
	for i, input := range inputs {
		if !filepath.IsLocal(input.Name) || filepath.Clean(input.Name) != input.Name ||
			input.Name == "." || input.Bytes < 0 || input.SHA256 == ([32]byte{}) {
			return [32]byte{}, fmt.Errorf("regionmanifest: invalid input %q", input.Name)
		}
		if i > 0 && inputs[i-1].Name == input.Name {
			return [32]byte{}, fmt.Errorf("regionmanifest: duplicate input %q", input.Name)
		}
	}
	manifest.Inputs = inputs
	canonical := struct {
		Version string
		Draw    Manifest
	}{Version: "regionmanifest-v1", Draw: manifest}
	bytes, err := json.Marshal(canonical)
	if err != nil {
		return [32]byte{}, fmt.Errorf("regionmanifest: encode: %w", err)
	}
	return sha256.Sum256(bytes), nil
}
