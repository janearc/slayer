package publication

import "testing"

// TestReplacementAcceptsTheCleanRootParent checks the permitted boundary.
func TestReplacementAcceptsTheCleanRootParent(t *testing.T) {
	fixture := publicationTest(t)
	fixture.request.parent = "/"
	fixture.temporary = "/.state-private"
	fixture.publication()
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationVisible || got.err != nil {
		t.Fatalf("clean root parent: %+v", got)
	}
}

// TestReplacementRefusesUncleanParents is the anti-test: no filesystem call
// may treat a raw parent differently from the joined destination.
func TestReplacementRefusesUncleanParents(t *testing.T) {
	for _, parent := range []string{
		"/queue/../other", "/queue/.", "/queue//nested", "/queue/",
	} {
		witness := uncleanParentWitness{parent: parent}
		t.Run(parent, witness.check)
	}
}

// uncleanParentWitness retains the raw parent spelling for one refusal.
type uncleanParentWitness struct {
	parent string
}

// check forbids every filesystem call before an unclean parent is refused.
func (witness uncleanParentWitness) check(t *testing.T) {
	fixture := publicationTest(t)
	fixture.request.parent = witness.parent
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationRefused || got.err == nil {
		t.Fatalf("unclean parent %q: %+v", witness.parent, got)
	}
}
