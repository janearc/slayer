package diskcache

import (
	"testing"
	"time"
)

// The numbers the tests choose; testdata/numbers.yaml says each.
const (
	testCap     = 1600 // bytes a test kind keeps
	entryBytes  = 300  // a test payload's size
	minuteLater = time.Minute
)

// clock is a test's clock: it says the time it is set to.
type clock struct{ at time.Time }

// now is the clock's time.
func (made *clock) now() time.Time { return made.at }

// started is the time every test's clock starts at.
var started = time.Date(2026, 11, 12, 14, 2, 0, 0, time.UTC)

// testKind is a kind of a name with the test cap, declared once.
func testKind(t *testing.T, name string) Kind {
	t.Helper()
	made, err := newKind(name, testCap, "a test's")
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// testStore is a store of a kind in a test's folder, on a test clock.
func testStore(t *testing.T, name string) (*Store, *clock) {
	t.Helper()
	ticking := &clock{at: started}
	store, err := opened(t.TempDir(), testKind(t, name), ticking.now)
	if err != nil {
		t.Fatal(err)
	}
	return store, ticking
}

// payload is bytes of a size, each its place's low byte, so two
// payloads of one size are alike and a changed one differs.
func payload(size int) []byte {
	made := make([]byte, size)
	for i := range made {
		made[i] = byte(i)
	}
	return made
}

// lookupOf is a Lookup of a test's own variables.
func lookupOf(said map[string]string) Lookup {
	return func(name string) (string, bool) {
		value, set := said[name]
		return value, set
	}
}
