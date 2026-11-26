package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/hatzegopteryx"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// HatzDensityStudy makes the authored pterosaur density sculpture.
func HatzDensityStudy(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	return hatzDensityStudy(phase, true, at)
}

func hatzDensityStudy(phase float64, animated bool, at maths.Subspace) ([]thing.Thing, error) {
	profile, _ := defaultReptileProfiles.book.Get("hatzegopteryx")
	return hatzDensity(profile, phase, animated, at)
}

func hatzDensity(profile reptiles.Profile, phase float64, animated bool, at maths.Subspace) ([]thing.Thing, error) {
	return hatzegopteryx.NewDensity(profile, phase, animated, at)
}
