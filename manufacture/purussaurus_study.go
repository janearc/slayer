package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/crocodylians"
	"github.com/janearc/slayer/scene/thing"
)

// PurussaurusStudy constructs the named study from its embedded profile.
func PurussaurusStudy(phase, yaw, posture float64) ([]thing.Thing, error) {
	profile, _ := defaultReptileProfiles.book.Get("purussaurus")
	return purussaurusStudy(profile, phase, yaw, posture)
}

func purussaurusStudy(profile reptiles.Profile, phase, yaw, posture float64) ([]thing.Thing, error) {
	return crocodylians.NewCrocodylian(profile, phase, yaw, posture)
}
