//go:build darwin && arm64

package manufacture

import "testing"

// Captured from registry specimens at 8d95e0e405fb, before their loose-shoot
// profile constructor. The digest covers surfaces, placement, bounds and dress.
func TestConiferShootsKeepPreProfileOutput(t *testing.T) {
	for _, tc := range []struct {
		species   string
		odd, even [4]string
	}{
		{"coast-redwood", [4]string{
			"32bff80c685e94161fd70acd0c27bc3f9f466ce910192f55c5b1e5ad73e0fe11",
			"eb28131e19e50127b7d0ddf7cd5e432cf53808f3afcbc5ea95e3d25f55a27a82",
			"f3244a96bd8a60ddda2da7774d74e5ff5cc43c700bbcb5cbcff7669a493fa116",
			"17a544f86ea2f67851c00a434d3cac3ba31c8b83c97a6bd1c0352968da298115",
		}, [4]string{
			"aeebaed3c5de09a9cd216bc49fe234895ce41142958025450955c211fdfcedb1",
			"cec658b2d2900731ed50bab05e6d2f1c2cb03465503440e0f6fc7471774fb9f5",
			"5b24e2eec38ba78978d3159e19dcfd034ec32fa0727cdd93c75d9db0090fc8b7",
			"336026f368213f12c2075d1220c988d6584decad575347b751806a0c2d06ab91",
		}},
		{"giant-sequoia", [4]string{
			"2b78629836c2c75f3b666be2ca03d8f634a7ea2e8b61dceddc7b50e66ecb2db7",
			"e9d38562f168f416ea651fe683623dab9d395476a6fad28b18d2fc10d756c074",
			"20564597e012e6b0797276267f9132a096fbff2ce2875713cfa6771a16124fd6",
			"a76b87fe708779fa2c1a2541b3847b5720cc076af6f02420ef88f838da5407c4",
		}, [4]string{
			"618afb8268886776b66b676b67975fd282a8e8cce889a84da7b979ffaaa6c5ce",
			"d08c19308071f6aa007d05526f5a9c73425a074a7735dcbe6d3cf6a53396bfca",
			"9d58f79db42ecdecb33f57f0cb0c5069c00895ab7c4b4c30bbc07cd66d4416db",
			"8a5b274c0d1dba80a513ddeb5b0a4b109b6f7274baad2056b2688b25bdfa3f65",
		}},
	} {
		row, err := defaultConiferProfiles.NewConiferShoot(tc.species)
		if err != nil {
			t.Fatal(err)
		}
		for _, variant := range []struct {
			seed    uint64
			digests [4]string
		}{{5, tc.odd}, {6, tc.even}, {11, tc.odd}} {
			for i, detail := range []string{"hull", "clumps", "quick", "demo"} {
				made, err := MakeWithComplexity(row, Sow(variant.seed), detail)
				if err != nil {
					t.Fatal(err)
				}
				if got := referenceDigest(t, made); got != variant.digests[i] {
					t.Fatalf("%s seed %d %s: got %s, want %s", tc.species, variant.seed, detail, got, variant.digests[i])
				}
			}
		}
	}
}
