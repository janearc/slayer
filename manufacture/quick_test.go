package manufacture

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

// todaysTrees is each spruce's woody skin as the named factory makes it
// at 85c1c37, when both spruces took physical needle shoots and their
// foliage moved to shared mesh groups: a digest of every corner of
// every face of the parts, matter by matter, at each seed from one to
// twelve. doc/families.md asks that quick stay today's trees, face for
// face; these are today's trees.
var todaysTrees = map[string][12]string{
	"spruce": {"b0cce761bece8c6a", "b3a7af1111ee150a", "e6f01496bfdbc704",
		"f339177fc6db00b0", "8b60dff2a02e8d26", "0d6de3ab703c1906",
		"78c4179272328e73", "d39e780562d87f4e", "4baf61910bb9c555",
		"d8e8d1504078f415", "0c0a12bd2870b293", "94f9afb2c74641c4"},
	"norway": {"7cf07c6df25bb71d", "53f0f2581096fc2c", "1e0abaf314015034",
		"d25029b944ac98b8", "229a404e8a1a2a61", "c81f7eece21e6a63",
		"2c5bafa8c6a36fd9", "a5218615c190eb38", "8a96b513b4be219e",
		"d199e65a0c348efc", "0f065d092220210e", "0f170c85c71db069"},
}

// spruceGrowth is a spruce grown by its row's architecture at quick, of
// so many whorls and so many branches to the lowest, set by a habit:
// the door ninja's truths for the spruce are held at.
func spruceGrowth(count, lowest int, set habit) (growth, error) {
	return spruceRow().grows.grown(count, lowest, 0, set, atQuick)
}

// fingerprint is a digest of a thing's skin: each matter in order, how many
// faces it has, and every corner's three numbers bit for bit.
func fingerprint(made good) string {
	digest := sha256.New()
	skins := made.skins()
	word := make([]byte, 8)
	for _, matter := range made.slots() {
		faces := skins[matter]
		binary.LittleEndian.PutUint64(word, uint64(matter))
		digest.Write(word)
		binary.LittleEndian.PutUint64(word, uint64(len(faces)))
		digest.Write(word)
		for _, one := range faces {
			for _, corner := range one {
				for _, number := range [3]float64{corner.X, corner.Y,
					corner.Z} {
					binary.LittleEndian.PutUint64(word,
						math.Float64bits(number))
					digest.Write(word)
				}
			}
		}
	}
	return fmt.Sprintf("%x", digest.Sum(nil))[:16]
}

// both spruces at quick are the trees they were, face for face, at
// every seed from one to twelve.
func TestQuickIsTodaysTrees(t *testing.T) {
	t.Parallel()
	for name, want := range todaysTrees {
		for seed := range uint64(12) {
			got := fingerprint(mustMake(t, name, seed+1))
			if got != want[seed] {
				t.Errorf("a %s of seed %d is %s, want today's %s", name,
					seed+1, got, want[seed])
			}
		}
	}
}
