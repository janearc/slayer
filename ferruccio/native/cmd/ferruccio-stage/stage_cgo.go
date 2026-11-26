//go:build cgo

package main

import (
	"context"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
	"github.com/janearc/slayer/ferruccio/native/internal/workspace"
	"github.com/janearc/slayer/job"
)

// stage runs Slayer staging inside this short-lived helper process.
func stage(ctx context.Context, path, source, kind string) (stagewire.Snapshot, error) {
	return workspace.StageSnapshot(ctx, path, source, kind)
}

// filmPresets exposes the producer's output-size table to the app.
func filmPresets() ([]stagewire.FilmPreset, error) {
	var result []stagewire.FilmPreset
	for _, preset := range job.Presets() {
		width, height := preset.Size()
		result = append(result, stagewire.FilmPreset{Name: preset.Name(), Width: width, Height: height})
	}
	return result, nil
}
