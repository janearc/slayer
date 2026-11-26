package manufacture

import (
	"fmt"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// gargoyleStudies are the studies of the creatures a gargoyle may be
// carved as, by the name gargoyle-creatures.yaml gives each: a function
// that makes the creature at rest, in its own frame and its own
// materials, which the gargoyle sets aside. This is the one place a
// creature's name meets its function, since data cannot name a function.
//
// To add a creature, such as the hatzegopteryx when it is here: write its
// entry in gargoyle-creatures.yaml, and add one line here naming its
// study. The cornice, the tower and the schema's kind list follow the
// data.
func gargoyleStudies() map[string]func() ([]thing.Thing, error) {
	rest := maths.Local()
	return map[string]func() ([]thing.Thing, error){
		"allosaurus": func() ([]thing.Thing, error) {
			return AllosaurusStudy(0, rest)
		},
		"hatzegopteryx": func() ([]thing.Thing, error) {
			return hatzGargoyleStudy()
		},
		"orca": func() ([]thing.Thing, error) { return orcaSwimming(0, rest) },
		"pliosaur": func() ([]thing.Thing, error) {
			return PliosaurStudy(0, rest)
		},
		"squid": func() ([]thing.Thing, error) { return NewHumboldtSquid(0) },
	}
}

// gargoyleStudy is the study of a name, or a refusal naming the studies
// there are.
func gargoyleStudy(name string) (func() ([]thing.Thing, error), error) {
	studies := gargoyleStudies()
	study, known := studies[name]
	if known {
		return study, nil
	}
	names := make([]string, 0, len(studies))
	for each := range studies {
		names = append(names, each)
	}
	slices.Sort(names)
	return nil, fmt.Errorf("manufacture: no study named %q for a gargoyle, "+
		"only %v", name, names)
}
