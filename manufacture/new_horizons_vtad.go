package manufacture

import (
	"embed"
	"encoding/binary"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"strings"
)

// NASA VTAD model, with transforms baked, eight non-manifold faces removed,
// texture materials approximated as flat colours. See SOURCES.md.
//
//go:embed new-horizons-vtad/*.stl
var newHorizonsVTADData embed.FS

const (
	vtadDark   slot = spaceSecondary + 1
	vtadSilver      = vtadDark + 1
	vtadWhite       = vtadSilver + 1
	vtadWarm        = vtadWhite + 1
)

type newHorizonsVTAD struct{}
type newHorizonsVTADPart struct{ data []byte }

// skin is the part's triangles read from its binary STL and scaled to the
// craft's size. The cut is not read.
func (p newHorizonsVTADPart) skin(_ cut) surface {
	n := int(binary.LittleEndian.Uint32(p.data[80:84]))
	out := make(surface, n)
	const scale = 2.1 / (.9442630410194397 + .9435498714447021)
	for i := 0; i < n; i++ {
		for v := 0; v < 3; v++ {
			k := 84 + i*50 + 12 + v*12
			x := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k : k+4])))
			y := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k+4 : k+8])))
			z := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k+8 : k+12])))
			out[i][v] = maths.V(x*scale, y*scale, z*scale)
		}
	}
	return out
}

// madeAt is made at any detail: a model has the one representation.
func (row newHorizonsVTAD) madeAt(from seed, _ string) (good, error) { return row.made(from) }

// made assembles the craft from its model parts and materials. It refuses
// a seed that was not made.
func (newHorizonsVTAD) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	a := &instrumentAssembly{object: good{dressed: looks{
		body:           thing.Polished("gold blanket", maths.RGB(.5, .3, .07), .2, 1.7),
		spaceStructure: thing.Diffuse("dish", maths.Grey(.68)),
		spaceFoil:      thing.Diffuse("dark apertures", maths.Grey(.025)),
		vtadDark:       thing.Diffuse("graphite instruments", maths.Grey(.09)),
		vtadSilver:     thing.Polished("instrument metal", maths.Grey(.25), .12, 1.6),
		vtadWhite:      thing.Diffuse("light instrument panels", maths.Grey(.58)),
		vtadWarm:       thing.Polished("warm instrument insulation", maths.RGB(.31, .23, .11), .16, 1.6),
	}}}
	entries, err := newHorizonsVTADData.ReadDir("new-horizons-vtad")
	if err != nil {
		return good{}, err
	}
	for _, entry := range entries {
		data, e := newHorizonsVTADData.ReadFile("new-horizons-vtad/" + entry.Name())
		if e != nil {
			return good{}, e
		}
		m := spaceFoil
		switch {
		case strings.Contains(entry.Name(), "-dark.stl"):
			m = vtadDark
		case strings.Contains(entry.Name(), "-silver.stl"):
			m = vtadSilver
		case strings.Contains(entry.Name(), "-white.stl"):
			m = vtadWhite
		case strings.Contains(entry.Name(), "-warm.stl"):
			m = vtadWarm
		}
		if entry.Name() == "foil_gold-0.stl" {
			m = body
		}
		if entry.Name() == "AO_dish-0.stl" {
			m = spaceStructure
		}
		a.add(newHorizonsVTADPart{data}, maths.Local(), m)
	}
	bounds := maths.SegmentBox(maths.V(-2, -1, -2), maths.V(2, 2, 3), 0)
	out, err := a.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *out, nil
}
