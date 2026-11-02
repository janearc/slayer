package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

// A Norway spruce's needles are on branchlets its branches carry, many
// to a branch: from every seed it has more than twice as many parts of
// needles as parts of branch, where a spruce has one for one.
func TestANorwaySpruceIsBranchedTwice(t *testing.T) {
	for number := range uint64(20) {
		made := mustMake(t, "norway", number)
		branches, needles := -1, 0 // the trunk is a body part
		for _, one := range made.parts {
			if one.of == leaf {
				needles++
			} else {
				branches++
			}
		}
		if branches < 1 || needles <= 2*branches {
			t.Errorf("seed %d: %d parts of needles on %d of branch",
				number, needles, branches)
		}
	}
}

// The largest Norway spruce its row allows, the tallest with its whorls
// closest, the most branches to a whorl and the most branchlets to a
// piece, has no more faces than a made thing may: so no seed is refused.
func TestTheLargestNorwaySpruceFits(t *testing.T) {
	row := norwayRow()
	whorls := int(math.Round(row.tall.most / row.between.least))
	grown, err := row.grows.grown(whorls, row.lowest.count(1),
		row.grows.shoots.per.count(1), row.hangs, atQuick)
	if err != nil {
		t.Fatal(err)
	}
	tall := measure.Metres(row.tall.most)
	made, err := grow(grown, size{length: tall,
		girth: tall.Times(row.slender.most)}, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	if faces := made.faces(); faces > mostFaces {
		t.Errorf("the largest norway spruce is %d faces, past %d", faces,
			mostFaces)
	}
}
