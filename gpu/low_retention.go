package gpu

import "os"

// lowRetentionVariable names the environment variable that turns on
// low-retention scene encoding. It is the only place that reads it.
const lowRetentionVariable = "SLAYER_LOW_RETENTION"

// lowRetentionEnabled reports whether the low-retention experiment is
// on: each mesh is pooled as it is grown and no trees are kept between
// draws. It is off unless the variable is exactly "1".
func lowRetentionEnabled() bool {
	return os.Getenv(lowRetentionVariable) == "1"
}
