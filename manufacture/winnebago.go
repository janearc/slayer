package manufacture

import "github.com/janearc/slayer/manufacture/cities"

// winnebagoRow is the kind winnebago: a motorhome, drawn by package
// cities from its design and made a good here, which is where a material
// becomes a slot. It is made as the yurt is, and differs only in the
// design.
type winnebagoRow struct{}

// made is the winnebago at the quick detail level.
func (row winnebagoRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the winnebago at the named detail level. It returns errGood
// when the seed is not made.
func (winnebagoRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.WinnebagoAt(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}
