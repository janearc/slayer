package weather

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNoGPU is a machine the weather's kernels cannot run on: not a mac,
// or a mac with no metal device. Run on the cpu there.
var ErrNoGPU = errors.New("weather: no metal device to run on; " +
	"run on the cpu")

// cacheVariable is defined as the environment variable that names where
// weather is kept between shots, over the user's cache.
const cacheVariable = "SLAYER_WEATHER"

// CacheDir is where weather is kept between shots, so a job, a hillside
// set and the weather command share checkpoints: SLAYER_WEATHER, or
// slayer/weather under the system's cache for this user.
func CacheDir() (string, error) {
	if dir := os.Getenv(cacheVariable); dir != "" {
		return dir, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "slayer", "weather"), nil
}
