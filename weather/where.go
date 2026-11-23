package weather

import (
	"path/filepath"

	"github.com/janearc/slayer/diskcache"
)

// refusal is a refusal of this package, held as a constant so that no one
// can assign it.
type refusal string

// Error is the refusal's words.
func (r refusal) Error() string { return string(r) }

// ErrNoGPU is a machine the weather's kernels cannot run on: not a mac,
// or a mac with no metal device. Run on the cpu there.
const ErrNoGPU refusal = "weather: no metal device to run on; " +
	"run on the cpu"

// cacheFolder is the folder of the cache home the weather is kept in.
const cacheFolder = "weather" // defined: the cache's name for weather

// CacheDir is where weather is kept between shots, so a job, a set
// and the weather command share checkpoints: the weather folder
// under the cache home, which diskcache.Home finds from the lookup
// given. Where the home is not set, it refuses by name and nothing is
// kept anywhere else.
func CacheDir(lookup diskcache.Lookup) (string, error) {
	home, err := diskcache.Home(lookup)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, cacheFolder), nil
}
