package manufacture

import "github.com/janearc/slayer/manufacture/cities"

// pierKind is the name of the kind pier: a wharf pier, a deck on rows of
// piles with a shed on it, drawn by package cities and made a good here.
// Its origin stands at the sea, so a set stands it by an absolute position.
const pierKind = "pier"

// pierRow is the kind pier.
type pierRow struct{}

// made is the pier at the quick detail level.
func (row pierRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the pier at the named detail level. It returns errGood when
// the seed is not made.
func (pierRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.Pier(cities.Seed(from.number))
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}
