package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The splitmix constants the session uses to mix a scene's seed and a thing's
// seed into the seed the thing is made from.
const (
	mixStep  = 0x9e3779b97f4a7c15
	mixFirst = 0xbf58476d1ce4e5b9
	mixLast  = 0x94d049bb133111eb
)

// orbitSeed is the seed of sets/scavenged-orbit.yaml's one thing, as
// the session's placement.Seed mixes it for a scene seed of 1, which
// the set does not change, and a thing seed of 55. It is written out here
// because that package imports this one.
func orbitSeed(object uint64) uint64 {
	scene := uint64(1)
	mixed := scene ^ ((object + 1) * mixStep)
	mixed ^= mixed >> 30
	mixed *= mixFirst
	mixed ^= mixed >> 27
	mixed *= mixLast
	return mixed ^ mixed>>31
}

// orbitThing is the seed of the orbit set's one thing, and orbitAsk what
// the set asks of it, as sets/scavenged-orbit.yaml says.
const orbitThing = 12

var orbitAsk = ClusterAsk{Parts: map[string]float64{"sls-core-tank": 6,
	"alfa-hero": 2, "winnebago": 2}}

// The cluster the orbit set draws is logged: its parts, and the people it
// holds and how crowded that is, for the set's comment.
func TestTheClusterOfTheOrbitSetSaysWhoLivesInIt(t *testing.T) {
	seed := orbitSeed(orbitThing)
	laid, err := laidOut(sow(seed), "hull", orbitAsk)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(laid.report())
	for _, join := range laid.cluster.Joins {
		for _, band := range join.Bands {
			var middle maths.Vec
			for _, corner := range band.Hull {
				middle = middle.Add(corner)
			}
			share := 1 / float64(len(band.Hull))
			t.Logf("part %d (%s): a band round %d corners about %v",
				join.Piece, laid.parts[join.Piece].kind, len(band.Hull),
				middle.Scale(share))
		}
	}
	have := laid.plan.Interior.Occupancy(laid.cluster.Volume)
	if len(laid.parts) != 12 || !(have.Occupants > 0) {
		t.Errorf("%d parts holding %v", len(laid.parts), have)
	}
}
