//go:build !darwin

package flow

import "fmt"

// newMetalStepper reports that no native backend is available.
func newMetalStepper(config Config, initial Velocity) (metalStepper, error) {
	return nil, fmt.Errorf("flow: Metal requires Darwin")
}
