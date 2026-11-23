//go:build draw

package cheeptune

import (
	"testing"
)

func TestAPairThatClipsIsRefusedAndOneThatFitsIsNot(t *testing.T) {
	// the two ends of the table of pairs in SCORING.md.
	if _, err := Open("@crypt", "@crypt-dmg"); err != nil {
		t.Errorf("the pair crypt was voiced for does not render: %v",
			err)
	}
	_, err := Open("@crypt", "@dmg-echo")
	wantErr(t, err, "clips")
}
