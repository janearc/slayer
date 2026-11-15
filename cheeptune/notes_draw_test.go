//go:build draw

package cheeptune

import "testing"

// A piece's notes are its roles' as the score writes them, once through:
// sunrun's drums strike named hits, each note lets go after it starts,
// and a role the piece has not has none.
func TestAPiecesNotesAreItsRoles(t *testing.T) {
	piece, err := Open("examples/sunrun.json", "@leap-dmg")
	if err != nil {
		t.Fatal(err)
	}
	drums := piece.Notes("perc")
	if len(drums) == 0 || drums[0].Hit == "" {
		t.Fatalf("sunrun's drums are %v", drums)
	}
	for _, one := range piece.Notes("lead") {
		if !(one.Off > one.On) || one.On < 0 ||
			one.On > piece.Score.Seconds() {
			t.Errorf("a lead note runs %v to %v", one.On, one.Off)
		}
	}
	if len(piece.Notes("kazoo")) != 0 {
		t.Error("sunrun has a kazoo")
	}
}
