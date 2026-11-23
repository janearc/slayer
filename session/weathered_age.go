package session

import (
	"fmt"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// weatheredAge is how old the weathering says the place is, in years.
//
// It is the number the set says, or, where the set says the age follows a
// superblock's accretion, the age at planned and the age at accreted
// with the accretion that superblock is made at between them: the more a
// block is grown over, the more the agents have laid on it. The accretion
// is the one the thing is made from, drawn from its seed where the set
// leaves it to the seed, so the age and the block never disagree.
func (making *staging) weatheredAge(read set.Set,
	laid set.Weathering) (float64, error) {
	name, planned, accreted, follows := laid.AgeFollows()
	if !follows {
		return laid.Age(), nil
	}
	for _, each := range read.Things() {
		if each.Name() != name {
			continue
		}
		if each.Kind() != manufacture.SuperblockKind {
			return 0, fmt.Errorf("%s: the weathering's age follows %s, "+
				"which is a %s and not a superblock", making.given.set, name,
				each.Kind())
		}
		value, drawn := each.Accretion()
		grown, err := manufacture.SuperblockAccretion(
			manufacture.SuperblockAsk{Accretion: value, Drawn: drawn},
			placement.Seed(making.given.seed, int(each.Seed())))
		if err != nil {
			return 0, err
		}
		return planned + (accreted-planned)*grown, nil
	}
	return 0, fmt.Errorf("%s: the weathering's age follows %s, and the set "+
		"has no thing of that name", making.given.set, name)
}
