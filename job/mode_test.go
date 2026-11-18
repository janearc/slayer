package job

import "testing"

// TestFilmModeWireIsReadAndMovesEveryKey holds that a film saying mode
// wire asks for the wire picture, and that the wire is a different
// picture of different light: all three keys move.
func TestFilmModeWireIsReadAndMovesEveryKey(t *testing.T) {
	wired, err := reading(t, still+"film: {width: 64, mode: wire}\n")
	if err != nil {
		t.Fatal(err)
	}
	if !wired.Film().Wire() {
		t.Fatal("a film of mode wire does not ask for the wire")
	}
	plain := keysOf(t, still+"film: {width: 64}\n")
	got := keysOf(t, still+"film: {width: 64, mode: wire}\n")
	if movedFrom(plain, got) != (moved{true, true, true}) {
		t.Errorf("mode wire moves %+v of the keys, not all three",
			movedFrom(plain, got))
	}
}

// anti-test of TestFilmModeWireIsReadAndMovesEveryKey
// TestFilmModeLightIsNoWireAndKeepsTheKeys holds that mode light, and a
// film that says no mode, ask for no wire, and that saying light keeps
// every key the job had without it.
func TestFilmModeLightIsNoWireAndKeepsTheKeys(t *testing.T) {
	for _, text := range []string{"film: {width: 64, mode: light}\n",
		"film: {width: 64}\n", ""} {
		read, err := reading(t, still+text)
		if err != nil {
			t.Fatal(err)
		}
		if read.Film().Wire() {
			t.Errorf("%q asks for the wire", text)
		}
	}
	plain := keysOf(t, still+"film: {width: 64}\n")
	said := keysOf(t, still+"film: {width: 64, mode: light}\n")
	if plain != said {
		t.Errorf("saying mode light moves the keys: %+v",
			movedFrom(plain, said))
	}
}

// anti-test of TestFilmModeWireIsReadAndMovesEveryKey
// TestFilmModeRefusesAModeThereIsNot holds that a mode the schema does
// not name is refused, and not read as light.
func TestFilmModeRefusesAModeThereIsNot(t *testing.T) {
	if _, err := reading(t, still+"film: {mode: chalk}\n"); err == nil {
		t.Fatal("a film of mode chalk was read")
	}
}
