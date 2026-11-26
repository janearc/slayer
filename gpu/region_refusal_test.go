//go:build darwin

package gpu

import (
	"errors"
	"image"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

type regionOtherLens struct{ film.Sensor }

func TestSoftwareRegionRefusesFieldsAndOtherLenses(t *testing.T) {
	for _, check := range []struct {
		world *thing.World
		lens  film.Sensor
	}{
		{&thing.World{Fields: []*thing.Field{{Mass: 1}}}, film.Pinhole{}},
		{&thing.World{}, regionOtherLens{film.Pinhole{}}},
	} {
		picture, _, err := SampledRegion(Software, check.world, maths.Subspace{}, check.lens,
			0, 10, 8, 2, image.Rect(2, 3, 5, 7))
		if picture != nil || !errors.Is(err, ErrRegionUnavailable) {
			t.Errorf("unsupported scene returned picture %v and error %v", picture, err)
		}
	}
}
