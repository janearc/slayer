package pages

import (
	"bytes"
	"testing"
)

// A render that exits well but left fewer pictures than its total has its
// line say the pictures that exist and the time left unknown.
func TestEndLineCountsThePicturesThatExist(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 8)
	var log bytes.Buffer
	coordinator := New(settingsFor(root, nil, hands), &log)
	coordinator.Start(jobFor(1, root, pair[0], 12))
	coordinator.tick(t.Context())
	hands.advance(TickEvery)
	coordinator.End(jobFor(1, root, pair[0], 12), succeeded)
	line := lineFieldsOf(t, root)
	if line[0] != "8" || line[1] != "12" || line[3] != "-1" {
		t.Fatal("the end line said more than the folder holds", line)
	}
}

// Anti-test: when every picture is there the line still says all done and
// nought left.
func TestEndLineSaysDoneWhenEveryPictureExists(t *testing.T) {
	root, hands := renders(t), &clock{at: fixtureNoon}
	stills(t, root, pair[0], 0, 12)
	var log bytes.Buffer
	coordinator := New(settingsFor(root, nil, hands), &log)
	coordinator.Start(jobFor(1, root, pair[0], 12))
	coordinator.tick(t.Context())
	hands.advance(TickEvery)
	coordinator.End(jobFor(1, root, pair[0], 12), succeeded)
	line := lineFieldsOf(t, root)
	if line[0] != "12" || line[1] != "12" || line[3] != "0" {
		t.Fatal(line)
	}
}
