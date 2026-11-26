//go:build draw && darwin && arm64

package manufacture

import (
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// Captured from unmodified 9fabd006 on Darwin/arm64, using the actual
// rejection loop and a PRNG initialized with the cluster stream.
func TestScavengedRetirementKeepsSeededGeometry(t *testing.T) {
	plan, err := cities.Scavenged()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		seed                                        uint64
		detail, first, firstError, accepted, digest string
		next                                        uint64
	}{
		{5, "hull", "california-barracuda-glide", "manufacture: california-barracuda-glide has no hull representation", "mees-solar", "7f6452e9ff39692a63c28d33c5217be0d024156f50be746c417ad7038ef864fa", 17300719576760621809},
		{5, "clumps", "california-barracuda-glide", "manufacture: california-barracuda-glide has no clumps representation", "mees-solar", "d6a1c6a459c700f9f4531c79a8dea23eaed446d199b64f6a52c3920ec76ee862", 17300719576760621809},
		{6, "hull", "saguaro", "", "saguaro", "ad6c6665d0a0908110dea39fcacaa1b07d2b94e04b318d7e56ac8fe05589a487", 7547635388514683346},
		{6, "clumps", "saguaro", "", "saguaro", "121fb1a7d3e3846e0786e91d3988ce611b66b59feaa1342423252adfbbd3bbf0", 7547635388514683346},
		{12, "hull", "walrus-fresh-5", "manufacture: walrus-fresh-5 has no hull representation", "mariner-10", "12e2650859d0d731b4aa8eda6708d94e49b2bf586705d5ac9c908dfd110eecc3", 6737662187949293555},
		{12, "clumps", "walrus-fresh-5", "manufacture: walrus-fresh-5 has no clumps representation", "mariner-10", "12e2650859d0d731b4aa8eda6708d94e49b2bf586705d5ac9c908dfd110eecc3", 6737662187949293555},
		{15, "hull", "bird-grey-heron-standing", "manufacture: bird-grey-heron-standing has no hull representation", "new-horizons", "fec2fa7a89aa06ca97333963843e0d1184cc49dbb00c79d4acb95ca0bf0a42d6", 6364569876147732996},
		{15, "clumps", "bird-grey-heron-standing", "manufacture: bird-grey-heron-standing has no clumps representation", "new-horizons", "fec2fa7a89aa06ca97333963843e0d1184cc49dbb00c79d4acb95ca0bf0a42d6", 6364569876147732996},
	}
	for _, tc := range cases {
		rng := rand.New(rand.NewPCG(tc.seed, scavengedStream))
		name, err := drawKind(ClusterAsk{}, rng)
		if err != nil || name != tc.first {
			t.Fatalf("seed %d %s first=%s: %v", tc.seed, tc.detail, name, err)
		}
		seed := rng.Uint64()
		row, err := scavengedModel(name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = MakeWithComplexity(row, Sow(seed), tc.detail)
		if tc.firstError == "" && err != nil || tc.firstError != "" && (err == nil || err.Error() != tc.firstError) {
			t.Fatalf("seed %d %s first refusal %v, want %q", tc.seed, tc.detail, err, tc.firstError)
		}
		rng = rand.New(rand.NewPCG(tc.seed, scavengedStream))
		part, err := drawPart(plan, ClusterAsk{}, rng, tc.detail)
		if err != nil {
			t.Fatal(err)
		}
		things, err := part.turned.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		if digest, next := birdAllosaurusWitness(things), rng.Uint64(); part.kind != tc.accepted || digest != tc.digest || next != tc.next {
			t.Errorf("seed %d %s got %s %s next=%d", tc.seed, tc.detail, part.kind, digest, next)
		}
	}
}
