package preview

import (
	"fmt"
	"os"
)

// CacheVariable is the environment variable Slayer reads for the one
// absolute folder it keeps what it can work out again in.
const CacheVariable = "SLAYER_CACHE"

// ShelfVariable is the environment variable Slayer reads for its source shelf.
const ShelfVariable = "SLAYER_SHELF"

// ErrNoCache is the one plain line said when there is no cache folder to give
// Slayer, which exits at once without one. The window shows it and opens
// Settings; a render refuses with it.
var ErrNoCache = fmt.Errorf("slayer needs a cache folder: choose one in Settings, or give --cache DIR or set %s to one absolute folder", CacheVariable)

// RequireCache refuses a direct render that has no cache folder to give
// Slayer. The folder is tools.Cache, which holds the settings' value, or the
// variable in Ferruccio's own environment, which the child inherits.
func RequireCache(tools Tools) error {
	if tools.Cache != "" || os.Getenv(CacheVariable) != "" {
		return nil
	}
	return ErrNoCache
}
