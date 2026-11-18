package diskcache

import (
	"fmt"
	"path/filepath"
)

// Lookup is how the home reads the environment: os.LookupEnv, or a
// test's own.
type Lookup func(name string) (string, bool)

// Home is the folder every cache slayer keeps lives under, and the one
// rule that finds it: SLAYER_CACHE, an absolute folder. Nothing else is
// read; there is no fallback to the renders home, the user's cache folder
// or the temporary folder.
//
// Where SLAYER_CACHE is unset, empty or not absolute, Home refuses by name
// and says how to set it, and whoever asked keeps nothing. Every package
// that keeps a cache asks Home, and puts its files in a folder of its own
// under it.
func Home(lookup Lookup) (string, error) {
	said, set := lookup(homeVariable)
	if !set || said == "" {
		return "", fmt.Errorf("diskcache: %s is not set; set it to "+
			"the one absolute folder slayer keeps what it can work "+
			"out again, in your game config or your shell",
			homeVariable)
	}
	if !filepath.IsAbs(said) {
		return "", fmt.Errorf("diskcache: %s is %q, which is not an "+
			"absolute folder; give it one", homeVariable, said)
	}
	return filepath.Clean(said), nil
}
