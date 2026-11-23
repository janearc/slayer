package manufacture

import (
	"embed"
	"encoding/binary"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Model geometry: NASA/Brian E. Kumanchik, Cassini-Huygens (A), decoded
// from the corresponding separate NASA GLBs with node poses baked in.
// https://science.nasa.gov/3d-resources/cassini-huygens-a/
// Huygens diameter is scaled to ESA's 2.7 m; Cassini's dish is already
// approximately 4 m. The added rim and fittings are image-led estimates.
// https://www.esa.int/Science_Exploration/Space_Science/Cassini-Huygens/Huygens_spacecraft
//
//go:embed cassini-huygens/cassini/*.stl cassini-huygens/huygens/*.stl cassini-huygens/attached/*.stl
var cassiniGeometry embed.FS

const (
	cassGold  slot = leaf + 1
	cassDark       = cassGold + 1
	cassWhite      = cassDark + 1
)

type cassiniHuygens struct{ probe, attached bool }
type cassiniPart struct {
	data  []byte
	scale float64
}

// skin is the part's triangles read from its binary STL and scaled. The cut
// is not read: a model has the detail it was made with.
func (p cassiniPart) skin(_ cut) surface {
	n := int(binary.LittleEndian.Uint32(p.data[80:84]))
	out := make(surface, n)
	for i := 0; i < n; i++ {
		for v := 0; v < 3; v++ {
			k := 84 + i*50 + 12 + v*12
			x := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k : k+4])))
			y := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k+4 : k+8])))
			z := float64(math.Float32frombits(binary.LittleEndian.Uint32(p.data[k+8 : k+12])))
			out[i][v] = maths.V(x*p.scale, y*p.scale, z*p.scale)
		}
	}
	return out
}

// madeAt is made at any detail: a model has the one representation.
func (row cassiniHuygens) madeAt(from seed, _ string) (good, error) { return row.made(from) }

// made assembles the craft from its model parts and materials: the orbiter,
// the probe, or both joined, as the row says.
func (row cassiniHuygens) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	a := &instrumentAssembly{fine: cut{around: 12}, object: good{dressed: looks{
		body:      thing.Polished("aluminum", maths.Grey(.55), .25, 1.8),
		cassGold:  thing.Polished("gold insulation", maths.RGB(.65, .39, .08), .25, 1.8),
		cassDark:  thing.Diffuse("dark spacecraft hardware", maths.Grey(.045)),
		cassWhite: thing.Diffuse("antenna white", maths.Grey(.72)),
	}}}
	folder := "cassini-huygens/cassini"
	scale := 1.0
	if row.probe {
		folder = "cassini-huygens/huygens"
		scale = 2.7 / 2.6153474
	}
	entries, err := cassiniGeometry.ReadDir(folder)
	if err != nil {
		return good{}, err
	}
	for _, entry := range entries {
		data, e := cassiniGeometry.ReadFile(folder + "/" + entry.Name())
		if e != nil {
			return good{}, e
		}
		mat := body
		switch entry.Name() {
		case "foil_gold-0.stl", "foil_gold_2-0.stl", "foil_gold_h-0.stl":
			mat = cassGold
		case "plastic_dark-0.stl", "black_krinkle-0.stl", "tex_01-0.stl", "tex_01_h-0.stl":
			mat = cassDark
		case "plastic_white-0.stl", "dish-0.stl":
			mat = cassWhite
		}
		a.add(cassiniPart{data, scale}, maths.Local(), mat)
	}
	if row.attached {
		for _, name := range []string{"foil_gold_h-0.stl", "tex_01_h-0.stl"} {
			data, e := cassiniGeometry.ReadFile("cassini-huygens/attached/" + name)
			if e != nil {
				return good{}, e
			}
			material := cassGold
			if name == "tex_01_h-0.stl" {
				material = cassDark
			}
			a.add(cassiniPart{data, 1}, maths.Local(), material)
		}
	}
	if row.probe {
		// Circumferential edge seam and small perimeter fixtures, placed by eye
		// from ESA's entry assembly imagery; not measured hardware dimensions.
		r := 1.33
		for i := 0; i < 48; i++ {
			t := 2 * math.Pi * float64(i) / 48
			u := 2 * math.Pi * (float64(i) + .96) / 48
			a.rod(maths.V(-.05, r*math.Cos(t), r*math.Sin(t)), maths.V(-.05, r*math.Cos(u), r*math.Sin(u)), .015, body)
		}
		for i := 0; i < 16; i++ {
			t := 2 * math.Pi * float64(i) / 16
			a.box(.08, .06, .06, maths.V(-.04, r*math.Cos(t)-.03, r*math.Sin(t)-.03), cassGold)
		}
	}
	bounds := maths.SegmentBox(maths.V(-10, -9, -12), maths.V(10, 4, 3), 0)
	if row.probe {
		bounds = maths.SegmentBox(maths.V(-.6, -1.5, -1.5), maths.V(.6, 1.5, 1.5), 0)
	}
	out, err := a.finish(bounds)
	if err != nil {
		return good{}, err
	}
	return *out, nil
}
