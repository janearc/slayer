package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/pinnipeds"
	"github.com/janearc/slayer/scene/thing"
)

// WalrusStudy builds the default walrus in an animated lying pose.
func WalrusStudy(yaw, phase float64) ([]thing.Thing, error) { return pinnipeds.WalrusStudy(yaw, phase) }
