package rules

import (
	"strings"
	"testing"
)

// record is one commit as signedReads asks git to print it.
func record(verdict, key, message string) string {
	return "c0ffee\x00" + verdict + "\x00" + key + "\x00" + message + "\x1e"
}

const someone = "SHA256:somebodyElseEntirely"

var readHash = strings.Repeat("ab", 20)

// a good signature from the maintainer's key clears the commit its note
// names, and nothing else does: another key, a signature git cannot vouch
// for, no signature, a short hash, or a note that is not on a line of its
// own.
func TestOnlyTheMaintainersGoodSignatureClearsACommit(t *testing.T) {
	note := "harness-read: " + readHash + "\n\napart gains tests alone"
	for name, each := range map[string]struct {
		records string
		clears  bool
	}{
		"hers, good":        {record("G", signer, note), true},
		"hers, untrusted":   {record("U", signer, note), false},
		"hers, bad":         {record("B", signer, note), false},
		"unsigned":          {record("N", "", note), false},
		"another key":       {record("G", someone, note), false},
		"a short hash":      {record("G", signer, "harness-read: "+readHash[:12]), false},
		"inside a sentence": {record("G", signer, "see harness-read: "+readHash), false},
		"among other lines": {record("N", "", "x") + "\n" + record("G", signer, "merge\n"+note), true},
	} {
		_, cleared := notesSignedBy([]byte(each.records), signer)[readHash]
		if cleared != each.clears {
			t.Errorf("%s: cleared %v, want %v", name, cleared, each.clears)
		}
	}
}

// anti-test: a reader that ignored the verdict would let an unsigned
// note through; the table above must catch that reader.
func TestAnUnsignedNoteIsSeenByTheTable(t *testing.T) {
	ignoringVerdict := func(records []byte) bool {
		return strings.Contains(string(records), "harness-read: "+readHash)
	}
	note := []byte(record("N", "", "harness-read: "+readHash))
	if !ignoringVerdict(note) {
		t.Fatal("the anti-test's careless reader should clear the unsigned note")
	}
	if _, cleared := notesSignedBy(note, signer)[readHash]; cleared {
		t.Fatal("notesSignedBy clears an unsigned note, as the careless reader does")
	}
}
