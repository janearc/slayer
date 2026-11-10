//ninja:vow dojo the private inverse has a contract beyond the shipped layout
package manufacture

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"pgregory.net/rapid"
)

//go:embed testdata/hex-rings.yaml
var hexRingContractRaw []byte

// firstRing is defined as the first ring outside the empty centre.
const firstRing = 1

type hexRingContract struct {
	Neighbors, Minimum, Maximum int
}

func TestHexRingInverseAgreesWithCountingSuccessiveRings(test *testing.T) {
	var data hexRingContract
	if err := yamlfile.Read("hex-rings.yaml", hexRingContractRaw, &data); err != nil {
		test.Fatal(err)
	}
	rapid.Check(test, data.countRings)
}

func (data hexRingContract) countRings(test *rapid.T) {
	rings := rapid.IntRange(data.Minimum, data.Maximum).Draw(test, "rings")
	population := spaceZero
	// Count the cells geometrically, without the inverse's quadratic.
	for level := range rings {
		population += data.Neighbors * (level + firstRing)
	}
	if got := hexRings(population); got != rings {
		test.Fatalf("%d cells: inverse gave %d rings, want %d",
			population, got, rings)
	}
}
