package manufacture

import (
	"math"
	"sync"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/plants/conifers"
)

// defaultShootProfile reads the same species data as the default kinds.
func defaultShootProfile(style int) shootProfile {
	name := "coast-redwood"
	if style == 2 {
		name = "giant-sequoia"
	}
	if style == spruceShootStyle {
		name = "spruce"
	}
	profile, _, err := defaultConiferProfiles.book.Select(name)
	if err != nil {
		panic(err)
	}
	return profile.Needle.Geometry()
}

var shootPrototypes sync.Map

// prototype uses fixed physical size bins: scaling a twig must not turn
// a centimetre needle into a metre leaf. The variants are seeded microforms.
func prototype(style, bin, variant int, detail string) (*shootPrototype, error) {
	return prototypeWith(style, bin, variant, detail, defaultShootProfile(style), &shootPrototypes)
}

func planShoot(style int, length float64, from seed) shootPlan {
	return conifers.PlanShoot(style, length, from, defaultShootProfile(style), shootValue, shootNormal)
}

func needleSpacing(style int) float64 { return defaultShootProfile(style).Spacing }

// recursiveFoliage keeps the existing seeded woody scaffold and replaces
// terminal sleeves with repeated, recursively branched physical shoots.
func recursiveFoliage(made good, from seed, style int, detail string) (good, error) {
	return recursiveFoliageWith(made, from, style, detail, &shootConfig{profile: defaultShootProfile(style), store: &shootStore{}})
}

// defaultConiferRow retains the private scaffold entry points used by the
// independent whorled-reference checks, with species values from data.
func defaultConiferRow(name string) conifer {
	row, _, err := defaultConiferProfiles.selectTree(name)
	if err != nil {
		panic(err)
	}
	return row
}

// spruceRow and norwayRow select their whorled architecture from data.
func spruceRow() conifer { return defaultConiferRow("spruce") }
func norwayRow() conifer { return defaultConiferRow("norway") }

// Redwood and sequoia share the whorled construction; their dimensions,
// branch habit, foliage and appearance come from conifer-profiles.yaml.
func redwoodRow() conifer { return defaultConiferRow("coast-redwood") }
func sequoiaRow() conifer { return defaultConiferRow("giant-sequoia") }

func defaultSaguaro() columnarCactus {
	book, err := ReadCactusProfiles("cactus-profiles.yaml", cactusProfilesRaw)
	if err != nil {
		panic(err)
	}
	row, err := book.NewSaguaro()
	if err != nil {
		panic(err)
	}
	return row.how.(columnarCactus)
}

// spruceHabit is how a spruce sets its branches.
//
// the numbers are ninja's, from BranchOf, set by eye from ninja-gaiden's lesson
// of 2026-10-06; none is measured. they move to the kinds table, where
// published ones can replace them. the turn is the golden angle, which is
// worked out: the turn that never lines two whorls up.
func spruceHabit() habit {
	return habit{
		Turn:  measure.Radians(math.Pi * (3 - math.Sqrt(5))),
		Start: 0.10, Top: 0.97,
		Thinning: 0.55, Fewest: 3,
		Stray: 0.35,
		Low:   measure.Degrees(100), Lift: measure.Degrees(38),
		Sway:  measure.Degrees(6),
		Reach: 0.30, Shortening: 0.88, Uneven: 0.15,
		Girth: 0.35, Least: 0.012,
	}
}
