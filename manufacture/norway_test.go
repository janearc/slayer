package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// A Norway spruce's needles are on branchlets its branches carry, many
// to a branch: from every seed it has more than twice as many needled
// shoots as parts of branch, where a spruce has one for one. The shoots
// are the leaf group's members; the parts are the woody scaffold.
func TestANorwaySpruceIsBranchedTwice(t *testing.T) {
	for number := range uint64(20) {
		made := mustMake(t, "norway", number)
		branches := len(made.parts) - 1 // the trunk is a body part
		needles := 0
		if group := made.groups[leaf]; group != nil {
			needles = len(group.Members())
		}
		if branches < 1 || needles <= 2*branches {
			t.Errorf("seed %d: %d shoots of needles on %d of branch",
				number, needles, branches)
		}
	}
}

// The largest Norway spruce its row allows, the tallest with its whorls
// closest, the most branches to a whorl and the most branchlets to a
// piece, has no more faces than a made thing may: so no seed is refused.
func TestTheLargestNorwaySpruceFits(t *testing.T) {
	row := norwayRow()
	whorls := int(math.Round(row.Tall[1] / row.Between[0]))
	grown, err := row.Grows.Grown(whorls, maths.CountBetween(row.Lowest[0], row.Lowest[1], 1),
		maths.CountBetween(row.Grows.Shoots.Per[0], row.Grows.Shoots.Per[1], 1), row.Hangs, int(body), int(leaf))
	if err != nil {
		t.Fatal(err)
	}
	tall := measure.Metres(row.Tall[1])
	made, err := grow(grown, size{Length: tall,
		Girth: tall.Times(row.Slender[1])}, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if faces := made.faces(); faces > mostFaces {
		t.Errorf("the largest norway spruce is %d faces, past %d", faces,
			mostFaces)
	}
}
