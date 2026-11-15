package cheeptune

import (
	"testing"
)

func TestTheBuiltInsAreThereAndSayWhatTheyAre(t *testing.T) {
	scores, voicings := Builtin()
	want := map[string]bool{"@crypt": true, "@dusk": true, "@leap": true,
		"@overdrive": true, "@tape": true}
	for _, s := range scores {
		delete(want, s)
	}
	if len(want) > 0 {
		t.Errorf("scores %v are missing from %v", want, scores)
	}
	if len(voicings) != 15 {
		t.Errorf("voicings %v, want fifteen", voicings)
	}
	for _, v := range voicings {
		if _, err := ReadVoicing(v); err != nil {
			t.Errorf("built-in %s does not read: %v", v, err)
		}
	}
}

func TestAnUnknownBuiltInNamesWhatThereIs(t *testing.T) {
	_, err := Open("@nocturne", "@dmg-echo")
	wantErr(t, err, `no built-in "@nocturne"`, "@crypt", "@dmg-echo")
}

func TestASwappedPairIsRefusedByName(t *testing.T) {
	_, err := Open("@dmg-echo", "@dusk")
	wantErr(t, err, "@dmg-echo is a voicing, and a score belongs here")
}

func TestReadDocReadsBothKinds(t *testing.T) {
	b, err := ReadDoc("@crypt")
	if err != nil {
		t.Fatal(err)
	}
	if kind, err := Kind(b); kind != "score" || err != nil {
		t.Errorf("@crypt reads as %q (%v)", kind, err)
	}
	// the other kind, which is the half the name promises and this test
	// used not to read at all.
	b, err = ReadDoc("@crypt-dmg")
	if err != nil {
		t.Fatal(err)
	}
	if kind, err := Kind(b); kind != "voicing" || err != nil {
		t.Errorf("@crypt-dmg reads as %q (%v)", kind, err)
	}
	if _, err := ReadDoc("@nothing-like-this"); err == nil {
		t.Error("an unknown name read something")
	}
}
