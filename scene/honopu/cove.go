// Package honopu stages surveyed relief along the Nā Pali coast at Honopū.
package honopu

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
)

//go:embed honopu-window.bin.gz
var elevations []byte

type relief struct {
	window   terrain.Window
	low      float64
	detail   bool
	vertical bool
}

var read = sync.OnceValues(func() (relief, error) {
	reader, err := gzip.NewReader(bytes.NewReader(elevations))
	if err != nil {
		return relief{}, err
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return relief{}, err
	}
	window, err := terrain.WindowFromBytes(raw)
	return relief{window: window}, err
})

// NewWithDetail builds the coastline with a still sea. See NewWithWind.
func NewWithDetail(wet bool, detail string) (*arid.Cove, error) {
	return NewWithWind(wet, detail, 0, 0)
}

// NewWithWind optionally adds authored surface colour and shading detail
// to the same surveyed terrain, and raises the sea by a wind: its speed ten
// metres over the water in metres a second, and the direction it blows
// toward in radians from east, counterclockwise. The generated detail is not
// survey data.
func NewWithWind(
	wet bool, detail string, wind, toward float64,
) (*arid.Cove, error) {
	return NewWithSea(wet, detail, wind, toward, 32, 65)
}

// NewWithSea is NewWithWind with the square of sea that carries waves said
// outright: its side in metres and the samples along a side. Past the square
// the sea is the flat mean plane.
func NewWithSea(
	wet bool, detail string, wind, toward, extent float64, samples int,
) (*arid.Cove, error) {
	return NewNoted(io.Discard, wet, detail, wind, toward, extent, samples)
}

// NewNoted is NewWithSea saying to said what any made ground laid over the
// survey is: its surveyed and made corners, and how far it departs.
func NewNoted(
	said io.Writer, wet bool, detail string,
	wind, toward, extent float64, samples int,
) (*arid.Cove, error) {
	if detail != "survey" && detail != "procedural" && detail != "bump" &&
		detail != "vertical" && !madeKind(detail) {
		return nil, fmt.Errorf("honopu: unknown surface detail %q", detail)
	}
	data, err := read()
	if err != nil {
		return nil, err
	}
	solid, low, err := ground(data.window, detail, said)
	if err != nil {
		return nil, err
	}
	data.low = low
	data.detail = detail != "survey" && !madeKind(detail)
	data.vertical = detail == "vertical"
	// WGS84 UTM 4N origin 431000 E, 2451500 N, with local MSL at Y=0.
	origin := maths.V(-1632, low, -1440)
	material := &thing.Material{Name: "Honopū coastal relief", Texture: data}
	if detail == "bump" {
		material.Crinkle = &thing.Crinkle{Size: 18, Slope: .34, Seed: 967}
	}
	shore := []thing.Thing{{Shape: solid,
		Motion:   thing.Still(maths.NewSubspace(origin, maths.Identity, 1)),
		Material: material,
	}}
	if !wet {
		return arid.New(shore, nil)
	}
	return arid.New(shore, &arid.Water{
		Name: "Honopū coastal water", Extinction: maths.RGB(.18, .042, .034),
		Sea:    oceantex.Sea{Wind: wind, Toward: toward, Fetch: 2900, Depth: 24},
		Extent: extent, Collar: 8, Samples: samples, Seed: 16,
	})
}

// At shades the basalt, vegetation, and shallow seabed using height and slope.
func (r relief) At(p maths.Vec) maths.Light {
	return r.Within(p, 0)
}

// Within filters fine authored texture at distant pixel footprints.
func (r relief) Within(p maths.Vec, footprint float64) maths.Light {
	h := p.Y + r.low
	if h < 0 {
		deep := maths.Smooth(3, 130, -h)
		return maths.RGB(.23, .50, .41).Scale(1 - deep).
			Add(maths.RGB(.033, .13, .19).Scale(deep))
	}
	slope := triangleSlope(r.window, p.X, p.Z)
	cliff := maths.Smooth(.72, 2.4, slope)
	variation := noise.Gradient(967, p.X/37, h/31, p.Z/37)
	green := maths.RGB(.17, .37, .095).Scale(.89 + .24*variation)
	rock := maths.RGB(.30, .22, .16).Scale(.9 + .29*variation)
	if r.detail {
		// Three-dimensional fields avoid stretching an overhead image down
		// a cliff. They are synthetic variation, not mapped rock features.
		coarse := noise.Fractal(1301, p.X/48, h/48, p.Z/48, 3)
		medium := noise.Gradient(1302, p.X/12, h/12, p.Z/12)
		fine := noise.Gradient(1303, p.X/3, h/3, p.Z/3)
		medium *= 1 - maths.Smooth(3, 12, footprint)
		fine *= 1 - maths.Smooth(.75, 3, footprint)
		rock = rock.Scale(math.Max(.35, 1+.65*coarse+.40*medium+.22*fine))
		green = green.Scale(math.Max(.4, 1+.4*coarse+.28*medium+.13*fine))
		if r.vertical {
			// Elongated 3D noise suggests weathering down steep exposed faces.
			// The pattern is authored appearance, not surveyed geological detail.
			ribs := verticalWeathering(p, footprint)
			rock = rock.Scale(math.Max(.25, 1+1.25*ribs))
		}
	}
	land := green.Scale(1 - cliff).Add(rock.Scale(cliff))
	beach := (1 - maths.Smooth(3, 18, h)) * (1 - maths.Smooth(.20, .6, slope))
	return land.Scale(1 - beach).Add(maths.RGB(.72, .55, .29).Scale(beach))
}

// verticalWeathering varies more slowly along gravity than across a cliff.
// The fine octave is filtered as the pixel footprint grows.
func verticalWeathering(point maths.Vec, footprint float64) float64 {
	long := noise.Gradient(2318, point.X/17, point.Y/180, point.Z/17)
	fine := noise.Gradient(2319, point.X/5, point.Y/100, point.Z/5)
	fine *= 1 - maths.Smooth(1.5, 6, footprint)
	return .75*long + .25*fine
}

// triangleSlope follows the diagonal used by terrain.Window.Ground. This
// classifies the surface that was hit, rather than a neighbouring vertex.
func triangleSlope(window terrain.Window, east, south float64) float64 {
	across, down, step := window.Size()
	x := max(0, min(float64(across-1), east/step))
	z := max(0, min(float64(down-1), south/step))
	col := min(across-2, int(math.Floor(x)))
	row := min(down-2, int(math.Floor(z)))
	near := window.At(col, row)
	eastHeight := window.At(col+1, row)
	southHeight := window.At(col, row+1)
	far := window.At(col+1, row+1)
	along, away := eastHeight-near, far-eastHeight
	if x-float64(col) < z-float64(row) {
		along, away = far-southHeight, southHeight-near
	}
	return maths.V(along, 0, away).Len() / step
}
