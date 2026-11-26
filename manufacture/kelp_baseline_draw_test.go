//go:build draw && darwin && arm64

package manufacture

import (
	"strings"
	"testing"
	"time"
)

// The geometry, placements and material records were captured before the
// migration to profiles, at c48ec91a. They cover each water condition, a
// forest, two seeds and the animated state.
func TestKelpKeepsPreProfileGeometry(t *testing.T) {
	for _, sample := range []struct {
		name string
		want [4]string
	}{
		{"giant-kelp-slack", [4]string{"ca21e968143ac50964bcf2e34da953f1c9fcc4640267277c58d3de864ddcaff4", "ca21e968143ac50964bcf2e34da953f1c9fcc4640267277c58d3de864ddcaff4", "4856cf6379e9d9a93191e64c630ff160b31394bdedc9e83234bea536461c36e1", "4856cf6379e9d9a93191e64c630ff160b31394bdedc9e83234bea536461c36e1"}},
		{"giant-kelp-moderate", [4]string{"0a0f003ebbeeb562f5e3d03c289a903d0075bcdf3b9ee351a654dd7310bcd2e2", "1184418476576b239c6f712841e60a03ed8781edefef74212784ed332aee8736", "4932cc579d2b2be1a93189f4309737a843478d6177e2caf73be566dbc5297d29", "01a84a7a92ca4d12d6d93488ed2b8e91a0dca5fe3be69949fec350cb3fe565ca"}},
		{"giant-kelp-surge", [4]string{"b2eab22d6cc60131f9362ce570d0587d6e1839ac7f6684e5da94dc03889e4371", "11b82dcf7eaa95f98f64f72f8fd1b10399b14d20148580a5d21cc207c4b2db02", "ff34ab8890d52d3d2a48c8402d54851d00e6bc939d0a78f946b4ecd2d7000c87", "9985a0cc1379cbb88372dd63e11d1fa48c0782bc3a3ff2b3ca6f723eed6ccd31"}},
		{"kelp-forest-moderate", [4]string{"d547b987b839198c48761ad11af4e5811ab379de580ecb3bc2a7da153b2dee70", "2c3500768334d9d3461f607b5c8977e4aab66df780b8d4dc909e06bfb0cbcb15", "1990312e64db8186feea604957f8f06f0ae5f140ea9522459cf088f854696fba", "1b7cff05c3ab92efc5e8862210ee63c24cc23b4e7a92079b7081026ce5177e49"}},
	} {
		forest := strings.HasPrefix(sample.name, "kelp-forest")
		condition := strings.TrimPrefix(sample.name, "giant-kelp-")
		if forest {
			condition = strings.TrimPrefix(sample.name, "kelp-forest-")
		}
		row, err := defaultKelpProfiles.NewGiantKelp(condition, forest)
		if err != nil {
			t.Fatal(err)
		}
		for i, number := range []uint64{5, 11} {
			for j, seconds := range []time.Duration{0, 3 * time.Second} {
				made, err := MakeAtTime(row, Sow(number), "quick", seconds)
				if err != nil {
					t.Fatal(err)
				}
				if got := referenceDigest(t, made); got != sample.want[2*i+j] {
					t.Errorf("%s seed %d time %s: got %s, want %s", sample.name, number, seconds, got, sample.want[2*i+j])
				}
			}
		}
	}
}
