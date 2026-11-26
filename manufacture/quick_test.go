package manufacture

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"testing"
)

// todaysTrees is each spruce's woody skin as the named factory makes it
// since both spruces took physical needle shoots and their foliage moved
// to shared mesh groups: a digest of every corner of every face of the
// parts, to a millionth of a metre, matter by matter, at each seed from
// one to twelve. quick is to stay today's trees, face for face; these are
// today's trees. They were taken from the last build at which the earlier
// digest, of the corners to the last bit, still held, so they are those
// trees and not new ones.
var todaysTrees = map[string][12]string{
	"spruce": {"87c96e0246c766ac", "01757828d623ebc0", "07f27a09e16aadd4",
		"e5cd2ea563a77a88", "72b53f3895cde744", "f249f267a0ad7db2",
		"6b8cf64d747eb694", "a803b31a3b2d4653", "0fa87501c15b2030",
		"6372496cdbbea612", "551590864fca6cc2", "78653fb9ed71fae2"},
	"norway": {"fef888e9e6bb29ee", "c3957d275a68e699", "de027f3bf3a2ae55",
		"c7428964447a8643", "cbe7993e6c6abb0c", "ece5edc2efd334b4",
		"698ed488c0e48509", "f43559b4545a1bcc", "8a0c939571960868",
		"95c1b7e8c0546095", "775627494456c7e1", "edd27156fe2c6cf5"},
}

// spruceGrowth is a spruce grown by its row's architecture at quick, of
// so many whorls and so many branches to the lowest, set by a habit:
// the door ninja's truths for the spruce are held at.
func spruceGrowth(count, lowest int, set habit) (growth, error) {
	return spruceRow().Grows.Grown(count, lowest, 0, set, int(body), int(leaf))
}

// cornersPerMetre is how finely a corner is held in a fingerprint: to a
// millionth of a metre. The last bits of a number move with how a compiler
// fuses a multiply and an add, which a change of inlining alters; a
// millionth of a metre is coarser than that, and finer than any detail a
// spruce is drawn with.
const cornersPerMetre = 1e6

// fingerprint is a digest of a thing's skin: each matter in order, how many
// faces it has, and every corner's three numbers to a millionth of a metre.
func fingerprint(made good) string {
	digest := sha256.New()
	skins := made.skins()
	word := make([]byte, 8)
	for _, matter := range made.slots() {
		faces := skins[matter]
		binary.LittleEndian.PutUint64(word, uint64(matter))
		digest.Write(word)
		faceCount := uint64(len(faces))
		binary.LittleEndian.PutUint64(word, faceCount)
		digest.Write(word)
		for _, one := range faces {
			for _, corner := range one {
				for _, number := range [3]float64{corner.X, corner.Y,
					corner.Z} {
					rounded := int64(math.Round(number * cornersPerMetre))
					binary.LittleEndian.PutUint64(word,
						uint64(rounded))
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

// Fused multiplies move the last bits of a corner, a thousand-millionth
// of a millionth of a metre here, and a fingerprint must not see that; a
// corner moved by ten millionths of a metre it must see.
func TestAFingerprintSeesAMicrometreAndNotTheLastBits(t *testing.T) {
	const noise, step = 1e-12, 1e-5
	made := mustMake(t, "can", 1)
	moved := func(by float64) good {
		shifted := made
		shifted.parts = slices.Clone(made.parts)
		shifted.parts[0].space.Origin.X += by
		return shifted
	}
	if fingerprint(moved(noise)) != fingerprint(made) {
		t.Error("a corner moved by the last bits changed the fingerprint")
	}
	if fingerprint(moved(step)) == fingerprint(made) {
		t.Error("a corner moved by ten millionths of a metre did not")
	}
}
