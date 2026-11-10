package smoke

import (
	"fmt"
	"os"
	"strings"
)

// RequireInputs is Reads with strict startup checks for a requested demo.
// Unlike optional scene fallbacks, an unset input is an actionable error.
func (scene Smoke) RequireInputs() ([]string, error) {
	for _, input := range scene.Inputs {
		if strings.HasPrefix(input, "$") {
			variable := strings.TrimPrefix(input, "$")
			name, _, _ := strings.Cut(variable, "/")
			if os.Getenv(name) == "" {
				return nil, fmt.Errorf("smoke %d needs %s; "+
					"set it to the data directory",
					scene.Number, name)
			}
		}
	}
	paths, err := scene.Reads()
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("smoke %d input: %w",
				scene.Number, err)
		}
	}
	return paths, nil
}
