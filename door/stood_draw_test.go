//go:build draw

package door

// The places flag against a set that stands on terrain, with the
// set, the shelf and the server of terrain_draw_test.go.

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/session"
)

// placesHeader is the first line of the places table: its columns, in
// their order, as the old command's -places writes them.
const placesHeader = "of,kind,east,north,height"

// rowsOf is a places table read: its header and its rows, each a record
// of five fields.
func rowsOf(t *testing.T, path string) [][]string {
	t.Helper()
	opened, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	rows, err := csv.NewReader(opened).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// rowFor is the row of a thing's name in a places table, or nil.
func rowFor(rows [][]string, name string) []string {
	for _, each := range rows {
		if each[nothing] == name {
			return each
		}
	}
	return nil
}

// workedRows checks a places table of the set of tendScene against
// numbers worked by hand. The things stand where the set says, printed
// to a tenth of a metre: east 1049 and north 8991 for the stone, east
// 1049 and north 8961 for the far stone. The ground window starts at
// west 1001, north 8999 with a step of 2, so its sample column, row is
// at east 1001+2 column, north 8999-2 row, and is the raster cell that
// holds that point: the cell of width 2 from 1000.5 east holds column
// (1001+2 column-1000.5)/2 = column+0.25, and the cell down from
// 9000.25 north holds row (9000.25-8999+2 row)/2 = row+0.625; so the
// sample is the raster's own, as high as (column+1000 row)/8
// (terrain/raster_test.go). A thing stands on the nearest sample. The
// stone is at column (1049-1001)/2 = 24 and row (8999-8991)/2 = 4, so
// (24+4000)/8 = 503, which prints as 503.0. The far stone is at row
// (8999-8961)/2 = 19, so (24+19000)/8 = 2378, which prints as 2378.0.
func workedRows(t *testing.T, rows [][]string) {
	t.Helper()
	if len(rows) != datasets+one {
		t.Fatalf("the table has %d lines, want a header and two things",
			len(rows))
	}
	if header := strings.Join(rows[nothing], ","); header != placesHeader {
		t.Errorf("the header is %q, want %q", header, placesHeader)
	}
	for name, want := range map[string]string{
		"the stone":     `the stone,cobble,1049.0,8991.0,503.0`,
		"the far stone": `the far stone,cobble,1049.0,8961.0,2378.0`} {
		got := strings.Join(rowFor(rows, name), ",")
		if got != want {
			t.Errorf("the row of %s is %q, want %q", name, got, want)
		}
	}
}

// assess --places stages the set, writes its places table, worked by
// hand, in the folder it is made in, and draws nothing.
func TestAssessPlacesWritesWhereEachThingWasStood(t *testing.T) {
	path, shelf, server := tendSetup(t)
	shelved(t, shelf, server)
	table := filepath.Join(filepath.Dir(path), "out", "places.csv")
	got := tendRun(t, "assess", path, shelf, append(smallOnCPU(),
		"--places", "out/places.csv")...)
	if got.code != nothing {
		t.Fatalf("assess gave %d: %s%s", got.code, got.said, got.wrongSaid)
	}
	mustHold(t, got, "wrote the places "+table)
	workedRows(t, rowsOf(t, table))
	for _, found := range files(t, filepath.Dir(path)) {
		if strings.HasSuffix(found, ".png") {
			t.Errorf("assess drew %s", found)
		}
	}
}

// draw --places writes the same table as assess does and as the old
// command's -places does for the same set, byte for byte, and draws the
// picture as well.
func TestDrawPlacesWritesTheOldCommandsTable(t *testing.T) {
	path, shelf, server := tendSetup(t)
	shelved(t, shelf, server)
	folder := filepath.Dir(path)
	table := filepath.Join(folder, "places.csv")
	got := drawTending(t, path, shelf, "--places", table)
	if got.code != nothing {
		t.Fatalf("draw gave %d: %s%s", got.code, got.said, got.wrongSaid)
	}
	mustHold(t, got, "wrote the places "+table)
	workedRows(t, rowsOf(t, table))
	if _, err := os.Stat(pictured(path)); err != nil {
		t.Errorf("no picture was drawn: %v", err)
	}
	old := filepath.Join(folder, "old.csv")
	var said, wrong bytes.Buffer
	code := session.Run([]string{"-set", path, "-w", "8", "-h", "6", "-n",
		"1", "-out", filepath.Join(folder, "old.png"), "-places", old},
		&said, &wrong, session.Stamped("", "", "", ""))
	if code != nothing {
		t.Fatalf("the old command gave %d: %s", code, wrong.String())
	}
	wanted, err := os.ReadFile(old)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wanted, fileBytes(t, table)) {
		t.Errorf("the door's table is not the old command's:\n%s\n%s",
			fileBytes(t, table), wanted)
	}
}

// the anti-test: with no --places, draw and assess write no table.
func TestDrawAndAssessWithNoPlacesFlagWriteNoTable(t *testing.T) {
	path, shelf, server := tendSetup(t)
	shelved(t, shelf, server)
	if got := tendRun(t, "assess", path, shelf,
		smallOnCPU()...); got.code != nothing {
		t.Fatalf("assess gave %d: %s%s", got.code, got.said, got.wrongSaid)
	}
	if got := drawTending(t, path, shelf); got.code != nothing {
		t.Fatalf("draw gave %d: %s%s", got.code, got.said, got.wrongSaid)
	}
	for _, found := range files(t, filepath.Dir(filepath.Dir(path))) {
		if strings.HasSuffix(found, ".csv") ||
			strings.Contains(found, "places") {
			t.Errorf("a run with no flag wrote %s", found)
		}
	}
}
