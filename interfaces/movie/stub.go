//go:build !(darwin && cgo)

package movie

import "image"

// Without macOS and cgo nothing reaches AVFoundation. The settings are
// still checked, so a caller learns of a bad request first, and then of
// the platform.

// openPipe is ErrNeedsMacOS.
func openPipe(Settings, string) (pipe, error) { return nil, ErrNeedsMacOS }

// probeFile is ErrNeedsMacOS.
func probeFile(string) (Facts, error) { return Facts{}, ErrNeedsMacOS }

// decodeSound is ErrNeedsMacOS.
func decodeSound(string, int) ([]float32, int, error) {
	return nil, 0, ErrNeedsMacOS
}

// openFilm is ErrNeedsMacOS.
func openFilm(string, float64, image.Point) (pictures, filmFacts, error) {
	return nil, filmFacts{}, ErrNeedsMacOS
}
