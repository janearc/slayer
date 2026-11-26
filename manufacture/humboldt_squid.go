package manufacture

import "github.com/janearc/slayer/scene/thing"

// NewHumboldtSquid selects the authored Humboldt profile.
func NewHumboldtSquid(phase float64) ([]thing.Thing, error) {
	return NewSquid("humboldt-squid", phase)
}

// NewSquid constructs a named species from the default checked profile book.
func NewSquid(species string, phase float64) ([]thing.Thing, error) {
	return defaultSquidProfiles.NewSquid(species, phase)
}
