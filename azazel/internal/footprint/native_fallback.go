//go:build !darwin || !cgo

package footprint

import "errors"

type unavailable struct{}

// Native returns an unavailable source on unsupported systems.
func Native() Source { return unavailable{} }

// BootID reports that native boot identity is unavailable.
func BootID() (string, error) { return "", errors.New("boot identity unavailable") }

// Group reports that process-group listing is unavailable.
func (unavailable) Group(int) ([]int, error) {
	return nil, errors.New("process-group footprint unavailable")
}

// Identity reports that process identity is unavailable.
func (unavailable) Identity(int) (Identity, error) {
	return Identity{}, errors.New("process identity unavailable")
}

// Footprint reports that footprint reads are unavailable.
func (unavailable) Footprint(int) (Reading, error) {
	return Reading{}, errors.New("process footprint unavailable")
}
