package manufacture

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

func birdAllosaurusWitness(parts []thing.Thing) string {
	hash := sha256.New()
	put := func(v uint64) { var b [8]byte; binary.LittleEndian.PutUint64(b[:], v); _, _ = hash.Write(b[:]) }
	put(uint64(len(parts)))
	for _, part := range parts {
		fmt.Fprint(hash, part.Material.Name, "\x00")
		if m, ok := part.Shape.(*mesh.Mesh); ok {
			put(uint64(len(m.Points())))
			for _, p := range m.Points() {
				put(math.Float64bits(p.X))
				put(math.Float64bits(p.Y))
				put(math.Float64bits(p.Z))
			}
			put(uint64(len(m.Faces())))
			for _, f := range m.Faces() {
				for _, i := range f {
					put(uint64(i))
				}
			}
		} else {
			b := part.Shape.Bounds()
			for _, v := range []float64{b.Min.X, b.Min.Y, b.Min.Z, b.Max.X, b.Max.Y, b.Max.Z} {
				put(math.Float64bits(v))
			}
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

var birdBaselineWitness = map[string]string{
	"black-faced-spoonbill/standing":   "7b118b47ce4f2d9a03053a161998fe72ec3072857fc68e1daa13643342461910",
	"black-faced-spoonbill/gliding":    "0974858b39cc8639267ec0686067952fad625e193e5aae1edafd7bce2d8dfe65",
	"black-faced-spoonbill/bones":      "cf4236060b4c696772a379c2effe364e0dc97c18954fa30728c58599501ae564",
	"black-faced-spoonbill/outline":    "2a39fdb3d83d3b3d40c3093adddecd26d9176eb86c55cd3b7bcd8ce506450da2",
	"great-frigatebird/standing":       "1db32aab72f4e1ba3337c8a3517512ad0a64b99bae9165b2668165d6bbcb488d",
	"great-frigatebird/gliding":        "0234299c1f74e653a3110df48914b4b3aa2fca0889a99430dc84430ff70e922d",
	"great-frigatebird/bones":          "36f642c4be1639f2a8182f1e18d94a26e67541f0bc444bf49f4328c1666c0dd4",
	"great-frigatebird/outline":        "96ca8d065d0622b78c2f70ca2b03958e453874f5cb39d642df970600ad61cb25",
	"grey-heron/standing":              "82897ac847d5223500f4320027e397de47fe0c4e2b9f4d8202348acf3dbf80c1",
	"grey-heron/gliding":               "b62954e1816a161c08b7ca3b1d5807039503b6201354759eaf51db1f29f3cbb4",
	"grey-heron/bones":                 "c7138f04a48f599b8b4014d08654f418acb453a85678c316936f700ee27c57ce",
	"grey-heron/outline":               "e7c7f92c9b7d35d524f7a4b93976b4d4a9ae64e0b3a92453a0bed770c1304741",
	"little-egret/standing":            "4ffb1e68a067feb276439c2d047af425f0bf1d2591c7f04d5bde9f80e35fad87",
	"little-egret/gliding":             "4ceba5c37ec17794e38b612fcc93f566cb4a7f65a8f4c22ebed284a7b9593991",
	"little-egret/bones":               "5c449a0859df746f4c555febe2abd3b2cdb9cd777af4cd4aa49f7bc950c2bac4",
	"little-egret/outline":             "78f13b21d4aa79f1ec86256c294fd8954fcdc7460a4cc6b34587f6aa5ef4c029",
	"pacific-reef-heron/standing":      "639b858ed52c91b4564cf3f84889eb19dbad6d4f11bdfcbffff7af2ff411b710",
	"pacific-reef-heron/gliding":       "4a9cc8c41dc2a8bafbc99c352b96cede1f8c9cd6bde0e4b7bfce2824121118aa",
	"pacific-reef-heron/bones":         "d71caddf5e12bf2c9ed5968803e9bb6b93f743f96641ff54873a7095a8b6b766",
	"pacific-reef-heron/outline":       "360a0cd4a893565e2fee81da511771951d1d17e48b5538fd57a0762d8a47e76f",
	"white-tailed-tropicbird/standing": "5c2003a892ed5560f64a002056321aabf4fd9ab1deba056a3b2d9ff0bf78b6eb",
	"white-tailed-tropicbird/gliding":  "6ba1e03afd0eaac123df2705a1de6097ea5996ed3c413388e9a8ba4d64ee67a3",
	"white-tailed-tropicbird/bones":    "33aa7fac99dc7b67a4740ff3148a6c4fe405f55591efb240a27ec96ad0d299ad",
	"white-tailed-tropicbird/outline":  "0095cf8a3607c6e1662834373f4318bfa3f68a0f986ca0cd5063b040e73732ef",
	"white-tern/standing":              "4258665f4ef247e61f5ab04cc37d2baa9d6574c1ac25188cef5db550a1a889e3",
	"white-tern/gliding":               "73be2105ee751b996a268d47b69180d98177a999608545f91509dce9cfa67180",
	"white-tern/bones":                 "98b25e88362f655dca62bff412e3643bd3b7cde82f52d519cf9d25380e3e8833",
	"white-tern/outline":               "34afd4eed053dd339f3e3854f81f5a9a07bc26ee2aae49e3d4fe773d85cf8893",
	"yellow-legged-gull/standing":      "dcd0a79e360ed119cc8d5ba395bfa12c8029f235598d24a62383ef016d500607",
	"yellow-legged-gull/gliding":       "fe7e33370715c6112363dd6b10488858d98be3326158a31ece073659f86b8ede",
	"yellow-legged-gull/bones":         "8b4427f9dab4f36e4f48fd396ee917321578fb57df6c2753b399ce87fd61dea4",
	"yellow-legged-gull/outline":       "437331a124952deadb040d9b9cc015d5a2407302be89befc41c758544528784d",
}

var allosaurusBaselineWitness = map[float64]string{
	0:   "5d42d45b81fcd07e04a9676a63db248b8e69875e253693c98172db4b64ec2b7e",
	0.5: "7071fd04b9889b3324e64c4eab407696018c47a6700805fae30e8289d721b19e",
	1.5: "0128ed57851b79a9f808a3ae852a560101d9ec10e417d8e81684d31790fbcc5a",
}

func TestBirdAllosaurusBaselineWitness(t *testing.T) {
	for _, species := range birdRead.Species() {
		for _, stance := range birdStances {
			parts, _, err := birdThings(species, stance, sow(7), maths.Local())
			if err != nil {
				t.Fatal(err)
			}
			key := species + "/" + stance
			if got, want := birdAllosaurusWitness(parts), birdBaselineWitness[key]; got != want {
				t.Errorf("bird %s witness = %s; want %s", key, got, want)
			}
		}
	}
	for _, seconds := range []float64{0, .5, 1.5} {
		parts, err := AllosaurusStudy(seconds, maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		if got, want := birdAllosaurusWitness(parts), allosaurusBaselineWitness[seconds]; got != want {
			t.Errorf("allosaurus %g witness = %s; want %s", seconds, got, want)
		}
	}
}
