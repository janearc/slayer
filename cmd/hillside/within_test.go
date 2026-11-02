package main

import (
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// wedged is the set package's kīlauea iki, its trees on the rim grown
// within a wedge seen from its first tree, written as it is given.
func wedged(t *testing.T, wedge string) set.Stand {
	t.Helper()
	source, err := os.ReadFile("../../set/testdata/kilauea-iki.yaml")
	if err != nil {
		t.Fatal(err)
	}
	written := strings.Replace(string(source), "        alight: 4\n",
		"        alight: 4\n        within: "+wedge+"\n", 1)
	read, err := set.Read("kilauea-iki.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	return read.Stands()[1]
}

// every place offered a stand with a wedge is inside it: its bearing
// from the thing between the wedge's, round past north where it wraps,
// and its distance between its near and far. and they fill it: some
// fall in each tenth of its turn.
func TestAWedgesStandIsOfferedOnlyItsWedge(t *testing.T) {
	for _, x := range []struct {
		wedge                 string
		from, turn, near, far float64
	}{
		{"{of: the first tree, from: 80, to: 140, near: 300, far: 900}",
			80, 60, 300, 900},
		{"{of: the first tree, from: 350, to: 20, near: 0, far: 50}",
			350, 30, 0, 50},
	} {
		stage := &staging{west: 264000, north: 2149000,
			places: map[string]maths.Vec{"the first tree": maths.V(400, 7,
				300)}}
		offer, err := stage.offering(wedged(t, x.wedge),
			rand.New(rand.NewPCG(5, 1)))
		if err != nil {
			t.Fatal(err)
		}
		east, north := 264000.0+400, 2149000.0-300
		var tenths [10]int
		for range 4000 {
			at, up := offer()
			bearing := math.Mod(math.Atan2(at-east, up-north)*180/math.Pi+
				360, 360)
			along := math.Mod(bearing-x.from+360, 360)
			out := math.Hypot(at-east, up-north)
			if along > x.turn+1e-9 || out < x.near-1e-9 ||
				out > x.far+1e-9 {
				t.Fatalf("%s: offered %.1f m out at %.2f degrees", x.wedge,
					out, bearing)
			}
			tenths[min(9, int(along/x.turn*10))]++
		}
		for k, many := range tenths {
			if many == 0 {
				t.Errorf("%s: no offers in tenth %d", x.wedge, k)
			}
		}
	}
}

// a wedge seen from something that stands nowhere on the ground is
// refused where the stand is staged.
func TestAWedgeFromNowhereIsRefused(t *testing.T) {
	stage := &staging{places: map[string]maths.Vec{}}
	_, err := stage.offering(wedged(t,
		"{of: the first tree, from: 0, to: 90, near: 0, far: 10}"),
		rand.New(rand.NewPCG(5, 1)))
	if err == nil || !strings.Contains(err.Error(), "stands nowhere") {
		t.Errorf("refused with %v", err)
	}
}
