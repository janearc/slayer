//go:build draw

package cheeptune

import (
	"strings"
	"testing"
)

func TestAPairThatClipsIsRefusedAndOneThatFitsIsNot(t *testing.T) {
	// what Pairs reports, without rendering all thirty-five pairs: the
	// two ends of its table.
	if _, err := Open("@crypt", "@crypt-dmg"); err != nil {
		t.Errorf("the pair crypt was voiced for does not render: %v",
			err)
	}
	_, err := Open("@crypt", "@dmg-echo")
	wantErr(t, err, "clips")
}

func TestPairsCoversEveryBuiltIn(t *testing.T) {
	if testing.Short() {
		t.Skip("Pairs renders every pair; about ten seconds")
	}
	var out strings.Builder
	if err := Pairs(&out); err != nil {
		t.Fatal(err)
	}
	scores, voicings := Builtin()
	for _, s := range scores {
		if !strings.Contains(out.String(), s+"\n") {
			t.Errorf("no row for score %s", s)
		}
	}
	for _, v := range voicings {
		if n := strings.Count(out.String(), v+":"); n != len(scores) {
			t.Errorf("%s appears against %d scores, want %d", v, n,
				len(scores))
		}
	}
}
