package cheeptune

import (
	"strings"
	"testing"
)

// elsewhere are the studio errors SCORING.md explains that are raised
// when a pair is bound or rendered, each held by a test of its own in
// studio_rules_test.go.
var elsewhere = []string{
	"the studio has 16 channels and the voicing has 17",
	"a note on a drum channel; it plays hits",
	"is outside the studio's range c0 to c8",
	"and clips; lower the voicing's master",
	// and these by acid_test.go.
	"filter accent and snap need a filter envelope to act on",
	"filter poles 3; a filter has 2 or 4",
	"glide is not for a pluck channel; fm and analog slide",
	"a struck drum needs a tone and takes no sweep",
	"hardness is for a struck drum",
}

func TestEveryStudioErrorThePageExplainsIsOneTheStudioReturns(t *testing.T) {
	text := page(t)
	_, table, found := strings.Cut(text, "### studio errors")
	if !found {
		t.Fatal("SCORING.md has no studio errors table")
	}
	table, _, _ = strings.Cut(table, "\n## ")
	var raised []string
	for _, c := range studioNearMisses() {
		_, err := ParseVoicing(studioVoicing(c.doc, ""))
		if err != nil {
			raised = append(raised, err.Error())
		}
	}
	raised = append(raised, elsewhere...)
	all := strings.Join(raised, "\n")
	rows := 0
	for line := range strings.SplitSeq(table, "\n") {
		cell, _, ok := strings.Cut(strings.TrimPrefix(line, "| `"), "` |")
		if !ok || !strings.HasPrefix(line, "| `") {
			continue
		}
		rows++
		if !strings.Contains(all, cell) {
			t.Errorf("SCORING.md explains %q, which nothing returns", cell)
		}
	}
	if rows < 20 {
		t.Errorf("the studio errors table has %d rows", rows)
	}
	// and the marks and the tempo list are taught where a score is.
	for _, want := range []string{"`pp`, `p`, `mp`, `mf`, `f`, `ff`",
		`"tempos": [{"bar": 9, "tempo": 132}`, "`SOUNDS.md`"} {
		if !strings.Contains(text, want) {
			t.Errorf("SCORING.md does not say %s", want)
		}
	}
}
