package manufacture

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

// todaysTrees is each spruce's skin as it was made at fermat e0859a6,
// before the trees were rows of an architecture: a digest of every
// corner of every face, matter by matter, at each seed from one to
// twelve. doc/families.md asks that quick stay today's trees, face for
// face; these are today's trees.
var todaysTrees = map[string][12]string{
	"spruce": {"13716071c6c1fb24", "80932c95d786a200", "0da8ca98d389cbc0",
		"03325430ae8896b3", "ffb407df3454b128", "525133f22e64ee6a",
		"de46bad9ca76ccc5", "25bfa92a7b339fb6", "136da8e58045bb34",
		"afd7ef4a5ee3ba5d", "f190e71411ba5ced", "0e62b2ae064d762d"},
	"norway": {"c285fe9f085ec653", "75a6017000bacf14", "2eb31aab3af61c33",
		"0f9bfb84d3e634eb", "c7e98b20c206475e", "5174ab6577552dd7",
		"e013f56e90b99eb1", "3e4a661b136bc0d4", "18b32bb688b0a877",
		"4c90ea5c1e47d044", "6ddda672dcfbca07", "95bb240304a203fd"},
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
