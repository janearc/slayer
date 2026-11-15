//go:build draw

package door

import (
	_ "embed"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
)

// seconds of a number the tests chose, as a duration.
func duration(t *testing.T, name string) time.Duration {
	t.Helper()
	chosen, err := numbers.Read("testdata/numbers.yaml", numbersRaw)
	if err != nil {
		t.Fatal(err)
	}
	value, err := chosen.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return time.Duration(value.Value * float64(time.Second))
}

// macroblock is the square H.264 codes a picture in, which a film's
// height is padded to a multiple of. Defined by ITU-T H.264, clause 6.3.
const macroblock = 16

// padded is the height the movie writer codes the tests' height as: the
// next multiple of a macroblock.
func padded(t *testing.T) int {
	t.Helper()
	blocks := (number(t, "height") + macroblock - one) / macroblock
	return blocks * macroblock
}
