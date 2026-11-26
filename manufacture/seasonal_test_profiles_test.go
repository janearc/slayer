package manufacture

// Internal profile assertions keep their private view without a second
// production map; the real book remains the sole owner of these profiles.
var plantProfiles = func() map[string]plantProfile {
	profiles := make(map[string]plantProfile)
	for _, name := range defaultPlantProfiles.book.Species() {
		profiles[name], _ = defaultPlantProfiles.book.Get(name)
	}
	return profiles
}()
