// Package commandtest builds real commands for their integration tests.
// Covered children match their package test's instrumentation and inherit
// GOCOVERDIR. Go then collects them without a second profile or remapping.
package commandtest

import (
	"fmt"
	"os/exec"
)

// Build writes the requested command at binary. Mode is testing.CoverMode();
// an empty mode builds an ordinary executable for an ordinary Game check.
func Build(directory, name, binary, mode string) error {
	args := []string{"build", "-o", binary}
	if mode != "" {
		args = append(args, "-cover", "-covermode="+mode, "-coverpkg="+name)
	}
	args = append(args, name)
	build := exec.Command("go", args...)
	build.Dir = directory
	if output, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("build %s: %w\n%s", name, err, output)
	}
	return nil
}
