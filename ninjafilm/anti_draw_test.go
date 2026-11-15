//go:build draw

package ninjafilm_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm"
)

// anti-tests: no stock prints white as it prints black. White light
// makes every stock lighter than no light does, for these stocks are
// reversal stocks.
func TestNoStockPrintsWhiteAsItPrintsBlack(t *testing.T) {
	names, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		black, white := still(t, name, maths.Light{}), still(t, name,
			maths.Grey(lit))
		if !(white.R > black.R) {
			t.Errorf("%s prints white as %v and black as %v", name, white,
				black)
		}
	}
}
