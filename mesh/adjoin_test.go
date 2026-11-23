package mesh

import (
	"testing"
	"time"
)

// finishing is work run in a goroutine of its own, and the channel it
// closes when it is done.
type finishing struct {
	work func()
	done chan struct{}
}

// finish runs the work, and says it is done.
func (running finishing) finish() {
	running.work()
	close(running.done)
}

// within fails the test if work has not finished in so long, and leaves
// it running: a root that never ends must fail a test, not hold the rest.
func within(t *testing.T, wait time.Duration, work func()) {
	t.Helper()
	running := finishing{work: work, done: make(chan struct{})}
	go running.finish()
	select {
	case <-running.done:
	case <-time.After(wait):
		t.Fatalf("still going after %v", wait)
	}
}

// joining is the faces a test joins, the pairs it asks about after, and
// the join and the question it makes last, all read before the work runs
// in a goroutine of its own, which must not stop the test.
type joining struct {
	test               *testing.T
	faces              int
	joins, same, apart rows
	lateOne, lateOther int
	askedOne           int
	askedOther         int
}

// run joins the faces and asks the pairs.
func (check joining) run() {
	t := check.test
	joined := newJoins(check.faces)
	for _, row := range check.joins {
		joined.join(whole(t, row, "one"), whole(t, row, "other"))
	}
	for _, row := range check.same {
		one, other := whole(t, row, "one"), whole(t, row, "other")
		if joined.root(one) != joined.root(other) {
			t.Errorf("faces %d and %d are not joined", one, other)
		}
	}
	for _, row := range check.apart {
		one, other := whole(t, row, "one"), whole(t, row, "other")
		if joined.root(one) == joined.root(other) {
			t.Errorf("faces %d and %d are joined", one, other)
		}
	}
	joined.join(check.lateOne, check.lateOther)
	if joined.root(check.askedOne) != joined.root(check.askedOther) {
		t.Error("joining two pieces left them two")
	}
}

// faces joined are one piece with one root, whichever way round they
// were joined and through whom, and faces not joined each have their
// own. this is early so that a root which never ends is caught first.
func TestJoinedFacesShareARoot(t *testing.T) {
	check := joining{test: t, faces: pickedCount(t, "adjoin-faces"),
		joins:      read(t, "testdata/adjoin-joins.csv"),
		same:       read(t, "testdata/adjoin-same.csv"),
		apart:      read(t, "testdata/adjoin-apart.csv"),
		lateOne:    pickedCount(t, "adjoin-late-one"),
		lateOther:  pickedCount(t, "adjoin-late-other"),
		askedOne:   pickedCount(t, "adjoin-asked-one"),
		askedOther: pickedCount(t, "adjoin-asked-other")}
	wait := time.Duration(picked(t, "adjoin-wait")) * time.Second
	within(t, wait, check.run)
}
