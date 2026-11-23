package manufacture

import (
	_ "embed"
	"encoding/binary"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// NASA printable 3D parts, three separate physical components. These are
// illustration geometry, not flight-grade blueprints. Assembly transforms,
// surface colours and rotation are estimates; the scale uses the 2.1 m
// high gain antenna in NASA's spacecraft fact sheet.
// https://science.nasa.gov/3d-resources/new-horizons/
// https://science.nasa.gov/photojournal/new-horizons-at-pluto-artists-concept/
//
//go:embed new-horizons/body.stl
var newHorizonsBody []byte

//go:embed new-horizons/antenna.stl
var newHorizonsAntenna []byte

//go:embed new-horizons/rtg.stl
var newHorizonsRTG []byte

type newHorizons struct{}
type newHorizonsPart struct {
	data []byte
	role int
}

// skin is the part's triangles read from its binary STL and scaled to the
// craft's size. The cut is not read.
func (p newHorizonsPart) skin(_ cut) surface {
	n := int(binary.LittleEndian.Uint32(p.data[80:84]))
	out := make(surface, n)
	const scale = 2.1 / 4.1335
	for i := 0; i < n; i++ {
		for v := 0; v < 3; v++ {
			k := 84 + i*50 + 12 + v*12
			x := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k : k+4])))
			y := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k+4 : k+8])))
			z := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k+8 : k+12])))
			if p.role == 1 {
				y += 1.65
			}
			if p.role == 2 {
				x, y = y+1.95, -x+.85
			}
			out[i][v] = maths.V(x*scale, y*scale, z*scale)
		}
	}
	return out
}

// madeAt is made at any detail: a model has the one representation.
func (row newHorizons) madeAt(from seed, _ string) (good, error) { return row.made(from) }

// made assembles the craft from its model parts and materials. It refuses
// a seed that was not made.
func (newHorizons) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	a := &instrumentAssembly{object: good{dressed: looks{
		body:           thing.Polished("gold insulating blanket", maths.RGB(.45, .25, .04), .24, 1.8),
		spaceStructure: thing.Diffuse("high gain antenna", maths.Grey(.7)),
		spaceFoil:      thing.Diffuse("radioisotope generator", maths.Grey(.09)),
	}}}
	a.add(newHorizonsPart{newHorizonsBody, 0}, maths.Local(), body)
	a.add(newHorizonsPart{newHorizonsAntenna, 1}, maths.Local(), spaceStructure)
	a.add(newHorizonsPart{newHorizonsRTG, 2}, maths.Local(), spaceFoil)
	bounds := maths.SegmentBox(maths.V(-2, -1, -1.3), maths.V(2.5, 2.3, 1.3), 0)
	out, err := a.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *out, nil
}
