package gpu

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// TestPrimaryBeamPassRejectsMixedSourceCache keeps the reconstruction
// contract from silently comparing direct sources with an approximate sum.
func TestPrimaryBeamPassRejectsMixedSourceCache(t *testing.T) {
	for _, pass := range []CompositePass{WithoutPrimaryBeams, PrimaryBeams} {
		drawer, err := NewCompositeFrames(CachedLighting, pass)
		if drawer != nil {
			drawer.Close()
		}
		if err == nil {
			t.Fatalf("pass %d accepted an inseparable mixed source cache", pass)
		}
	}
}

// TestPrimaryBeamPassRefusesNilPinhole must refuse a typed nil camera
// before dereferencing it or opening device resources.
func TestPrimaryBeamPassRefusesNilPinhole(t *testing.T) {
	drawer, err := NewCompositeFrames(Software, PrimaryBeams)
	if err != nil {
		t.Fatal(err)
	}
	defer drawer.Close()
	var lens *film.Pinhole
	picture, _, err := drawer.Draw(thing.NewWorld(), 1,
		maths.Subspace{}, lens, 0, 1, 1, 1)
	if err == nil || picture != nil {
		t.Fatal("typed nil lens was admitted")
	}
}
