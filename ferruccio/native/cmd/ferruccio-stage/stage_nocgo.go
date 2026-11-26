//go:build !cgo

package main

import (
	"context"
	"fmt"

	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

// stage explains why a helper built without Slayer's Cgo cannot stage.
func stage(_ context.Context, _, _, _ string) (stagewire.Snapshot, error) {
	return stagewire.Snapshot{}, fmt.Errorf("staging helper requires a Cgo-enabled build")
}

// filmPresets refuses when this helper cannot use the producer.
func filmPresets() ([]stagewire.FilmPreset, error) {
	return nil, fmt.Errorf("film presets require a Cgo-enabled staging helper")
}
