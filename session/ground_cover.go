package session

import (
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/set"
)

// surfacesNamed is the name of a covered ground, "ground cover" and its
// surfaces from the ground up, and the source each surface's reflectance
// says, one after another, for the material to carry.
func surfacesNamed(layers []set.CoverLayer) (name, source string) {
	names := make([]string, 0, len(layers))
	sources := make([]string, 0, len(layers))
	for _, layer := range layers {
		names = append(names, layer.Name())
		sources = append(sources, layer.Name()+": "+layer.Source())
	}
	return "ground cover: " + strings.Join(names, ", "),
		strings.Join(sources, "; ")
}

// coveredGround is the ground made of the cover the set says: the first
// surface everywhere, and each later one laid over it where its height
// above the sea and its slope hold. The sea is at level on the terrain's
// datum, and the ground is laid so its lowest height stands at nought, so
// the sea stands at level less lowest in the ground's own space.
func coveredGround(level, lowest float64,
	layers []set.CoverLayer) texture.Cover {
	sea := level - lowest
	cover := texture.NewCover(surfaceLight(layers[0]), sea)
	for _, layer := range layers[1:] {
		cover = cover.Over(surfaceLight(layer), surfaceRange(layer.Height),
			surfaceRange(layer.Slope))
	}
	return cover
}

// surfaceLight is the colour a cover's surface gives back.
func surfaceLight(layer set.CoverLayer) maths.Light {
	shares := layer.Reflectance()
	return maths.RGB(shares[0], shares[1], shares[2])
}

// surfaceRange is a rule of a surface as a range, from the method of the
// surface that says it.
func surfaceRange(rule func() (float64, float64, float64, bool)) texture.Range {
	from, until, blend, _ := rule()
	return texture.NewRange(from, until, blend)
}
