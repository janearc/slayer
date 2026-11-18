package workspace

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// locations resolves declared runtime paths before the renderer changes directory.
func locations(options Options) (Options, error) {
	for _, program := range []*string{&options.Tools.Azazel, &options.Tools.Game, &options.Tools.Slayer, &options.Tools.Catalogue} {
		if *program == "" {
			continue
		}
		if !strings.ContainsRune(*program, filepath.Separator) {
			path, err := exec.LookPath(*program)
			if err != nil {
				return options, fmt.Errorf("renderer tool %s: %w", *program, err)
			}
			*program = path
		}
		absolute, err := filepath.Abs(*program)
		if err != nil {
			return options, err
		}
		*program = absolute
	}
	for _, path := range []*string{&options.Evidence, &options.Starter, &options.Tools.Queue, &options.Tools.GameHome, &options.Tools.Shelf, &options.Tools.Cache, &options.Examples} {
		if *path == "" {
			continue
		}
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return options, err
		}
		*path = absolute
	}
	return options, nil
}
