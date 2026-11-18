//go:build darwin && arm64

package manufacture

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// TestSpacecraftFacesCapture writes, for each row of the reviewed
// spacecraft record, a digest of its meshes' corners and nothing else:
// no normals, no shading. Run with SPACECRAFT_FACES_OUT naming a file on
// two trees, the two files say whether a change moved any face; without
// it the test does nothing. Written 2026-11-15 for the round tube and
// dish, whose record moved in normals alone.
func TestSpacecraftFacesCapture(t *testing.T) {
	out := os.Getenv("SPACECRAFT_FACES_OUT")
	if out == "" {
		t.Skip("SPACECRAFT_FACES_OUT names no file")
	}
	var rows strings.Builder
	for _, name := range []string{"hubble", "space-observatory",
		"spitzer", "webb"} {
		row, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, seed := range []uint64{5, 11} { // defined: the record's
			for _, detail := range Complexities() {
				made, err := MakeWithComplexity(row, Sow(seed), detail)
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(&rows, "%s %d %s %s\n", name, seed, detail,
					facesDigest(t, made))
			}
		}
	}
	if err := os.WriteFile(out, []byte(rows.String()),
		0o644); err != nil { // defined: owner rw, others r
		t.Fatal(err)
	}
}

// facesDigest is the sha256 of every corner of every face of the
// thing's meshes, in order, and of nothing else.
func facesDigest(t *testing.T, made good) string {
	t.Helper()
	// the record's own placement: the zero subspace
	var placement maths.Subspace
	things, err := made.Things(placement)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := sha256.New()
	for _, one := range things {
		skin, isMesh := one.Shape.(*mesh.Mesh)
		if !isMesh {
			continue
		}
		for i := range skin.FaceCount() {
			a, b, c := skin.Corners(i)
			referenceVector(fingerprint, a)
			referenceVector(fingerprint, b)
			referenceVector(fingerprint, c)
		}
	}
	return fmt.Sprintf("%x", fingerprint.Sum(nil))
}
