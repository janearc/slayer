package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/pliosaurs"
	"github.com/janearc/slayer/scene/thing"
)

// PliosaurContinuousSkin constructs the named study from its embedded profile.
func PliosaurContinuousSkin(phase float64) ([]thing.Thing, error) {
	profile, _ := defaultReptileProfiles.book.Get("pliosaur")
	return pliosaurContinuousSkin(profile, phase)
}

func pliosaurContinuousSkin(profile reptiles.Profile, phase float64) ([]thing.Thing, error) {
	return pliosaurs.NewContinuous(profile, phase)
}
