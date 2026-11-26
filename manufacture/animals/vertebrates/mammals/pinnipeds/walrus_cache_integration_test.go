//go:build walrusintegration

package pinnipeds

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"

	"github.com/janearc/slayer/mesh"
)

// Two independent profile snapshots must never share a surface. Concurrent
// builds of one recipe also exercise first-use synchronization of its mesh.
func TestWalrusCacheConcurrentRecipes(t *testing.T) {
	original := defaultPinniped("walrus")
	changed := defaultPinniped("walrus")
	changed.WalrusOutline[10][1] += .02
	a, err := NewWalrus(original, "lying", "skin")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewWalrus(changed, "lying", "skin")
	if err != nil {
		t.Fatal(err)
	}
	changed.WalrusOutline[10][1] += .3
	type outcome struct {
		digest [32]byte
		err    error
	}
	results := make(chan outcome, 3)
	start := make(chan struct{})
	for _, row := range []*Walrus{a, a, b} {
		go func(row *Walrus) {
			<-start
			parts, err := row.Build()
			if err != nil {
				results <- outcome{err: err}
				return
			}
			hide := parts[0].Shape.(*mesh.Mesh)
			hash := sha256.New()
			var bytes [8]byte
			for _, point := range hide.Points() {
				for _, coordinate := range []float64{point.X, point.Y, point.Z} {
					binary.LittleEndian.PutUint64(bytes[:], math.Float64bits(coordinate))
					_, _ = hash.Write(bytes[:])
				}
			}
			var digest [32]byte
			copy(digest[:], hash.Sum(nil))
			results <- outcome{digest: digest}
		}(row)
	}
	close(start)
	got := make(map[[32]byte]int)
	for i := 0; i < 3; i++ {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		got[result.digest]++
	}
	if len(got) != 2 {
		t.Fatalf("independent profiles produced %d geometries", len(got))
	}
	for _, count := range got {
		if count != 1 && count != 2 {
			t.Fatalf("unexpected repeat count %d", count)
		}
	}
}
