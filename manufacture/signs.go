package manufacture

import "github.com/janearc/slayer/manufacture/cities"

// signRow is a kind that is one sign, on its own: a roof sign standing on
// its posts, a screen, or a blade sign hanging from a bracket at its
// origin, drawn by package cities and made a good here. Its words, colours
// and look are the seed's, and it is lit: it carries the lamps that throw
// its colour on what is near it.
type signRow struct {
	mount cities.MountKind
	// lettering is the script the sign's words are in, which trade it
	// carries if one is asked, and where the files of its fonts are. It
	// is nothing for Latin capitals of the trades the culture draws.
	lettering cities.Lettering
}

// signRows are the four signs, by the name of the kind: sign-roof,
// sign-screen, sign-blade and sign-awning.
func signRows() map[string]signRow {
	return map[string]signRow{
		"sign-roof":   {mount: cities.MountRoof},
		"sign-screen": {mount: cities.MountScreen},
		"sign-blade":  {mount: cities.MountBlade},
		"sign-awning": {mount: cities.MountAwning},
	}
}

// made is the sign at the quick detail level.
func (row signRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the sign at the named detail level, a glowing panel at the
// hull and the clumps and letters above them. It returns errGood when the
// seed is not made.
func (row signRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.FreeSignIn(row.mount, cities.Seed(from.number),
		cities.Detail(detail), row.lettering)
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}
