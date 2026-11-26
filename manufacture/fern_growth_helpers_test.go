package manufacture

import "github.com/janearc/slayer/manufacture/plants/ferns"

func fernGrowth(fronds, leaflets int) (growth, error) {
	book, err := ferns.Read("fern-profiles.yaml", fernProfilesRaw)
	if err != nil {
		return growth{}, err
	}
	profile, _ := book.Get("fern")
	return ferns.Growth(fronds, leaflets, int(body), int(leaf), profile.Anatomy)
}
