package diskcache

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lookup is how the home reads the environment: os.LookupEnv, or a
// test's own.
type Lookup func(name string) (string, bool)

// Home is the folder every kind lives under: SLAYER_CACHE where it is
// set, cache/ under SLAYER_RENDERS where that is, and slayer/ under the
// user's cache folder otherwise. A home said that is not absolute is
// refused by name, since the folder would move with the working one.
func Home(lookup Lookup) (string, error) {
	if said, set := lookup(homeVariable); set && said != "" {
		return absolute(homeVariable, said)
	}
	if renders, set := lookup(rendersVariable); set && renders != "" {
		home, err := absolute(rendersVariable, renders)
		return filepath.Join(home, rendersFolder), err
	}
	user, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("diskcache: no %s, no %s and no user "+
			"cache folder (%w); set %s to a folder", homeVariable,
			rendersVariable, err, homeVariable)
	}
	return filepath.Join(user, userFolder), nil
}

// absolute is a folder a variable said, refused where it is relative.
func absolute(variable, said string) (string, error) {
	if !filepath.IsAbs(said) {
		return "", fmt.Errorf("diskcache: %s is %q, which is not an "+
			"absolute folder; give it one", variable, said)
	}
	return filepath.Clean(said), nil
}
