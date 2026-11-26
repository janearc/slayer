package stagewire

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
)

// FilmPreset is one producer-defined output size, not a physical film gauge.
type FilmPreset struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// ReadFilmPresets asks the producer helper for its current size table. It
// waits for the child and refuses malformed or ambiguous output.
func ReadFilmPresets(ctx context.Context, helper string) ([]FilmPreset, error) {
	if ctx == nil || !filepath.IsAbs(helper) {
		return nil, fmt.Errorf("film preset helper must be an absolute path")
	}
	cmd := exec.CommandContext(ctx, helper, "presets")
	raw, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("film preset helper failed: %w", err)
	}
	return decodeFilmPresets(raw)
}

// decodeFilmPresets checks the entire producer response before presenting it.
func decodeFilmPresets(raw []byte) ([]FilmPreset, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var presets []FilmPreset
	if err := decoder.Decode(&presets); err != nil {
		return nil, fmt.Errorf("film presets: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("film presets contain trailing data")
	}
	seen := make(map[string]bool)
	for _, preset := range presets {
		if preset.Name == "" || preset.Width < 1 || preset.Height < 1 || seen[preset.Name] {
			return nil, fmt.Errorf("film presets contain an invalid or duplicate entry")
		}
		seen[preset.Name] = true
	}
	if len(presets) == 0 {
		return nil, fmt.Errorf("film presets are empty")
	}
	return presets, nil
}
