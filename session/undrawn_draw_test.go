//go:build draw

package session

import (
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// text is the set file a row says.
func (row undrawnRow) text() string {
	film, cameras := row.film, row.cameras
	if film == "" {
		film = undrawnFilm
	}
	if cameras == "" {
		cameras = undrawnCameras
	}
	return undrawnHead + film + cameras + row.scene
}

// place is the line and column of the row's key in its set file, counted
// by hand from the text: the first line that holds the key.
func (row undrawnRow) place(t *testing.T) (line, column int) {
	t.Helper()
	for at, each := range strings.Split(row.text(), "\n") {
		if found := strings.Index(each, row.key); found >= 0 {
			return at + 1, found + 1
		}
	}
	t.Fatalf("%s: the set holds no %q", row.feature, row.key)
	return 0, 0
}

// everythingBut is a set of features, each one drawn but the one named.
func everythingBut(rows []undrawnRow, feature string) map[string]bool {
	drawn := map[string]bool{}
	for _, each := range rows {
		drawn[each.feature] = each.feature != feature
	}
	return drawn
}

// Each feature of version 2 that the reader keeps and no stage draws
// refuses the set that holds it, by name: the file, the key and where it
// stands in the file, and that it is read and not yet drawn. Each row is
// a set whose every other feature is counted drawn, so it is the row's
// own that is refused.
func TestEachUndrawnFeatureRefusesItsSetByName(t *testing.T) {
	rows := undrawnRows()
	for _, row := range rows {
		t.Run(row.feature, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "t.yaml")
			if err := os.WriteFile(path, []byte(row.text()), 0o644); err != nil {
				t.Fatal(err)
			}
			read, err := set.Read(path, []byte(row.text()))
			if err != nil {
				t.Fatal(err)
			}
			line, column := row.place(t)
			want := path + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column) +
				": " + row.feature
			err = refuseUndrawn(read, path, everythingBut(rows, row.feature))
			if err == nil || !strings.HasPrefix(err.Error(), want) ||
				!strings.HasSuffix(err.Error(), "is read and not yet drawn") {
				t.Errorf("refusal is %v, want it to start %q and end \"is read "+
					"and not yet drawn\"", err, want)
			}
		})
	}
}

// A set whose every feature is drawn is not refused, and one with no
// feature of version 2 holds none.
func TestADrawnSetIsNotRefused(t *testing.T) {
	rows := undrawnRows()
	all := map[string]bool{}
	for _, row := range rows {
		all[row.feature] = true
		read, err := set.Read("t.yaml", []byte(row.text()))
		if err != nil {
			t.Fatal(err)
		}
		if err := refuseUndrawn(read, "t.yaml", everythingBut(rows,
			"none")); err != nil {
			t.Errorf("%s: a set of drawn features is refused: %v", row.feature, err)
		}
	}
	plain, err := set.Read("t.yaml", []byte(undrawnHead+undrawnFilm+undrawnCameras))
	if err != nil {
		t.Fatal(err)
	}
	if err := refuseUndrawn(plain, "t.yaml", map[string]bool{}); err != nil {
		t.Errorf("a set with no feature of version 2 is refused: %v", err)
	}
}

// The stage for a scene in space draws its setting, and a still has no
// use for a film's frames a second: a set holding either, and no other
// feature, is not refused. Every other feature of the table still is.
func TestTheSpaceStagesFeaturesAreDrawnAndNoOthers(t *testing.T) {
	drawn := map[string]bool{"scene.setting": true, "film.fps": true}
	if !maps.Equal(drawnFeatures, drawn) {
		t.Errorf("drawn are %v, want %v", drawnFeatures, drawn)
	}
	for _, row := range undrawnRows() {
		read, err := set.Read("t.yaml", []byte(row.text()))
		if err != nil {
			t.Fatal(err)
		}
		err = refuseUndrawn(read, "t.yaml", drawnFeatures)
		if drawn[row.feature] && err != nil {
			t.Errorf("%s is drawn, and its set is refused: %v", row.feature,
				err)
		}
		if !drawn[row.feature] && err == nil {
			t.Errorf("%s is not drawn, and its set is not refused",
				row.feature)
		}
	}
}

// The four scenes written against version 2 read, and the session refuses
// each by the first feature in it that nothing draws, as the command says
// it: past the setting, which the space stage draws, to the next.
func TestTheFourScenesAreRefusedByTheirFirstUndrawnFeature(t *testing.T) {
	for file, want := range map[string]string{
		"doc/corona.yaml":                     "../doc/corona.yaml:23:9: thing.kind: star is read and not yet drawn",
		"doc/orbital-and-lantern.yaml":        "../doc/orbital-and-lantern.yaml:24:9: thing.kind: lantern is read and not yet drawn",
		"set/testdata/webb-looking-out.yaml":  "../set/testdata/webb-looking-out.yaml:21:3: scene.frame is read and not yet drawn",
		"set/testdata/keyhole-to-ground.yaml": "../set/testdata/keyhole-to-ground.yaml:34:3: scene.anchor is read and not yet drawn",
	} {
		given := smallSetAsked(filepath.Join("..", file))
		_, _, _, err := staged(given)
		if err == nil || err.Error() != want {
			t.Errorf("%s: staged says %v, want %q", file, err, want)
		}
	}
}
