package planet

import (
	"testing"
)

// pointProbe is a face and the probes a test asks it about, one by one.
type pointProbe struct {
	face   *Face
	probes []probe
	at     int
}

// ask is the face asked at the next probe.
func (probing *pointProbe) ask() {
	each := probing.probes[probing.at%len(probing.probes)]
	probing.at++
	probing.face.Within(each.at, each.footprint)
}

// a point of a face as full as the giant's makes nothing on the heap: a
// 2K frame asks some twenty million points, and a list made for each was
// 1.6 GB of garbage for the collector to hand back.
func TestAFacesPointMakesNoGarbage(t *testing.T) {
	face := NewFace(stormyBody(t, true), look(t, "alloc pole"),
		look(t, "alloc star"), look(t, "alloc eye"), even(),
		picked(t, "alloc-exposure"))
	probing := &pointProbe{face: face, probes: giantProbes(t, face)}
	made := testing.AllocsPerRun(len(probing.probes), probing.ask)
	if !(made == nothing) {
		t.Fatalf("a point of the face makes %g things on the heap", made)
	}
}
