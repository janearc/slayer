package set

import (
	yaml "gopkg.in/yaml.v3"
)

// ageFollowing is an age that is not a number of years but follows how
// far a superblock is grown over: the thing it follows, and the age the
// place has when that thing is planned and when it is accreted. Between
// them the age is as far along as the accretion is.
type ageFollowing struct {
	thing             string
	planned, accreted float64
}

// AgeFollows is the thing whose accretion sets the weathering's age, the
// age at an accretion of nought and the age at one, and whether the age
// follows a thing at all. Where it does not, Age is the age in years.
func (made Weathering) AgeFollows() (thing string, planned, accreted float64,
	follows bool) {
	following := made.ageFollows
	return following.thing, following.planned, following.accreted,
		following.thing != ""
}

// age reads how old the place is: a number of years, or a block that says
// the age follows a superblock's accretion, so that the more a block is
// grown over the more the agents have laid on it.
func (reading reader) age(made *Weathering) step {
	return func(name, value *yaml.Node) error {
		if value.Kind == yaml.ScalarNode {
			return reading.number(&made.age, 0, weatheringMost)(name, value)
		}
		following := &made.ageFollows
		err := reading.keys(value, map[string]step{
			"from":     reading.text(&following.thing),
			"planned":  reading.number(&following.planned, 0, weatheringMost),
			"accreted": reading.number(&following.accreted, 0, weatheringMost),
		}, nil)
		if err == nil {
			err = reading.all(value, "an age that follows an accretion",
				"from", "planned", "accreted")
		}
		if err == nil && following.accreted < following.planned {
			err = reading.refuse(value, "an age that follows an accretion "+
				"never falls as it rises, and accreted is %v, under "+
				"planned, %v", following.accreted, following.planned)
		}
		return err
	}
}
