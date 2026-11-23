package manufacture

import "github.com/janearc/slayer/manufacture/cities"

// bivouacRow is the kind bivouac: a one-person bivy sack with its guy
// lines, a pack and boots, drawn by package cities from its design and
// made a good here, which is where a material becomes a slot. The seed
// decides whether it has a stove out.
type bivouacRow struct{}

// made is the bivouac at the quick detail level.
func (row bivouacRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the bivouac at the named detail level. It returns errGood when
// the seed is not made.
func (bivouacRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.BivouacAt(cities.Detail(detail),
		cities.Seed(from.number))
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}
