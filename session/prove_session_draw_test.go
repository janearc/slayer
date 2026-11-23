//go:build draw

package session

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/raw"
)

// The pixel the proof tests name, inside the session tests' 48 by 27
// picture, and how many more pixels they have the raw's bytes pick.
const (
	proveNamedAcross, proveNamedDown = 20, 10
	provePicks                       = 6
	// proveSmallWide and proveSmallTall are a picture smaller than the
	// session tests' own, 24 by 14.
	proveSmallWide, proveSmallTall = 24, 14
)

// drawnWidow is frame 0 of the session drawn and kept as a raw in a
// folder of the test's own, and where it is kept.
func drawnWidow(t *testing.T, session *Session) string {
	t.Helper()
	drawn, err := session.Frame(context.Background(), 0, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "widow.pfm")
	if err := raw.Write(path, drawn.Light()); err != nil {
		t.Fatal(err)
	}
	return path
}

// a session proves the raw it drew: the pixel named first, and then the
// picks: seven pixels, or six where one pick is the pixel named, since
// a pixel is proved once; every one made again to the bit.
// anti-test of TestASampleChangedFailsTheProof
func TestASessionProvesItsOwnRaw(t *testing.T) {
	session := widowSession(t)
	path := drawnWidow(t, session)
	proof, err := session.Prove(0, Proving{Raw: path,
		Pixels: [][2]int{{proveNamedAcross, proveNamedDown}},
		Many:   provePicks, Asked: "the test"})
	if err != nil {
		t.Fatal(err)
	}
	if !proof.Holds() {
		t.Fatalf("the proof of the session's own raw fails: %v", proof)
	}
	first := proof.Pixels[0]
	if first.X != proveNamedAcross || first.Y != proveNamedDown {
		t.Errorf("the first pixel proved is %d, %d, not the one named",
			first.X, first.Y)
	}
	if got := len(proof.Pixels); got != provePicks &&
		got != provePicks+1 {
		t.Errorf("%d pixels proved, not %d or %d", got, provePicks,
			provePicks+1)
	}
	if proof.Asked != "the test" {
		t.Errorf("the proof records %q as what was asked", proof.Asked)
	}
	if proof.Width != sessionWidth || proof.Height != sessionHeight {
		t.Errorf("the proof is of %d by %d, not %d by %d", proof.Width,
			proof.Height, sessionWidth, sessionHeight)
	}
}

// a sample changed by the least a float can change, at the pixel named,
// fails the proof, and fails it at that pixel.
// anti-test of TestASessionProvesItsOwnRaw
func TestASampleChangedFailsTheProof(t *testing.T) {
	session := widowSession(t)
	path := drawnWidow(t, session)
	kept, err := raw.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := kept.At(proveNamedAcross, proveNamedDown)
	changed.R = math.Nextafter32(changed.R, math.MaxFloat32)
	kept.Set(proveNamedAcross, proveNamedDown, changed)
	if err := raw.Write(path, kept); err != nil {
		t.Fatal(err)
	}
	proof, err := session.Prove(0, Proving{Raw: path,
		Pixels: [][2]int{{proveNamedAcross, proveNamedDown}},
		Many:   provePicks})
	if err != nil {
		t.Fatal(err)
	}
	first := proof.Pixels[0]
	if proof.Holds() || first.Kept.R == first.Made.R {
		t.Errorf("a changed sample held: %v; kept %v, made %v", proof,
			first.Kept.R, first.Made.R)
	}
}

// a raw of another size than the job's frame is refused, by both sizes.
// anti-test of TestASessionProvesItsOwnRaw, whose raw is the frame's size
func TestAProofOfARawOfAnotherSizeIsRefused(t *testing.T) {
	path := drawnWidow(t, widowSession(t))
	smaller := widowSessionAt(t, proveSmallWide, proveSmallTall)
	_, err := smaller.Prove(0, Proving{Raw: path, Many: provePicks})
	for _, want := range []string{"48 by 27", "24 by 14"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("a raw of another size: %v, not saying %q", err, want)
		}
	}
}
