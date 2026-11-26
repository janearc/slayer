package gpu

import (
	"errors"
	"image"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestRegionDoesNotSilentlyDrawAnUnimplementedMode(t *testing.T) {
	for _, mode := range []Mode{NativeShadows, NativeMeshes,
		CachedLighting, FastLighting, FasterLighting} {
		picture, _, err := SampledRegion(mode, nil, maths.Subspace{}, nil,
			0, 10, 8, 2, image.Rect(2, 3, 5, 7))
		if picture != nil || !errors.Is(err, ErrRegionUnavailable) {
			t.Errorf("mode %v returned picture %v and error %v", mode, picture, err)
		}
	}
}

func TestSoftwareRegionRefusesInvalidRectangles(t *testing.T) {
	for _, rect := range []image.Rectangle{image.Rect(-1, 0, 2, 2), image.Rect(0, 0, 0, 1), image.Rect(9, 7, 11, 8)} {
		picture, _, err := SampledRegion(Software, &thing.World{}, maths.Subspace{}, nil,
			0, 10, 8, 2, rect)
		if picture != nil || err == nil {
			t.Errorf("region %v returned picture %v and error %v", rect, picture, err)
		}
	}
}

func TestSoftwareRegionRefusesMissingWorld(t *testing.T) {
	picture, _, err := SampledRegion(Software, nil, maths.Subspace{}, nil,
		0, 10, 8, 2, image.Rect(2, 3, 5, 7))
	if picture != nil || err == nil {
		t.Fatalf("missing world returned picture %v and error %v", picture, err)
	}
}

func TestRegionRefusesUnknownMode(t *testing.T) {
	picture, _, err := SampledRegion(Mode(255), nil, maths.Subspace{}, nil,
		0, 10, 8, 2, image.Rect(2, 3, 5, 7))
	if picture != nil || !errors.Is(err, errNoMode) {
		t.Fatalf("unknown mode returned picture %v and error %v", picture, err)
	}
}
