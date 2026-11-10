package manufacture

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
)

// FireLayout arranges source centres in a disk or along a line. Radius is
// the disk radius; Length is the full line length. Variation is the authored
// fractional range of source heights about the nominal flame height.
type FireLayout struct {
	Pattern        string
	Count          int
	Radius, Length measure.Length
	Variation      float64
}

// FuelRadius encloses the source patches before their turbulent displacement.
func (spec BurnSpec) FuelRadius() measure.Length {
	radius := spec.Flame.Radius
	switch spec.Layout.Pattern {
	case "cluster":
		radius += spec.Layout.Radius.InMetres()
	case "line":
		radius += spec.Layout.Length.InMetres() / 2
	}
	return measure.Metres(radius)
}

type burningField interface {
	medium.Medium
	Shine() texture.Timed
	Tick(time.Duration) medium.Medium
	Sources(int) []flame.Source
}

// makeBurningField keeps the single-body path when no layout is requested.
func makeBurningField(spec BurnSpec, exposure float64) (burningField, error) {
	patches, err := sourcePatches(spec)
	if err != nil {
		return nil, err
	}
	if len(patches) == 0 {
		return flame.NewBody(spec.Flame, exposure)
	}
	return flame.NewField(patches, exposure)
}

// Placement and height variation are authored, not a combustion distribution.
func sourcePatches(spec BurnSpec) ([]flame.Ignition, error) {
	layout := spec.Layout
	if layout.Pattern == "" || layout.Pattern == "single" {
		if (layout.Count != 0 && layout.Count != 1) ||
			layout.Radius.InMetres() != 0 ||
			layout.Length.InMetres() != 0 ||
			layout.Variation != 0 {
			return nil, errors.New(
				"manufacture: single fire takes no layout parameters")
		}
		return nil, nil
	}
	if layout.Count < 2 || layout.Count > 32 || !(layout.Variation >= 0 &&
		layout.Variation <= 0.5) {
		return nil, errors.New(
			"manufacture: invalid fire patch count or variation")
	}
	radius, length := layout.Radius.InMetres(), layout.Length.InMetres()
	switch layout.Pattern {
	case "cluster":
		if !(radius > 0) || !burnFinite(radius) || length != 0 {
			return nil, errors.New(
				"manufacture: cluster needs only a radius")
		}
	case "line":
		if !(length > 0) || !burnFinite(length) || radius != 0 {
			return nil, errors.New(
				"manufacture: line needs only a length")
		}
	default:
		return nil, errors.New("manufacture: unknown fire layout")
	}
	return laidFirePatches(spec), nil
}

// laidFirePatches gives each checked source its own repeatable phase.
func laidFirePatches(spec BurnSpec) []flame.Ignition {
	layout := spec.Layout
	from := sow(spec.Flame.Seed)
	angle := math.Pi * (3 - math.Sqrt(5))
	phase := 2 * math.Pi * from.draw(flaring, 0, 0)
	patches := make([]flame.Ignition, layout.Count)
	for i := range patches {
		at := maths.V((float64(i)/float64(
			layout.Count-1)-0.5)*layout.Length.InMetres(), 0, 0)
		if layout.Pattern == "cluster" {
			fraction := (float64(i) + 0.5) / float64(layout.Count)
			radius := layout.Radius.InMetres() * math.Sqrt(fraction)
			turn := phase + float64(i)*angle
			at = maths.V(radius*math.Cos(turn), 0,
				radius*math.Sin(turn))
		}
		fire := spec.Flame
		fire.Seed = from.sprout(flaring, i).number
		fire.Height *= 1 + layout.Variation*from.swing(flaring, i, 1)
		patches[i] = flame.Ignition{At: at, Fire: fire}
	}
	return patches
}
