package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// A webbed foot: a thin web between each pair of neighbouring front
// toes, from the foot out to the row's share of the toes' length, its
// edge straight from one toe to the next. A row whose web is nought has
// none. The web rides on the middle toe's first bone.

// birdWebThick is a web's half thickness, of the bird's length; picked.
const birdWebThick = .0006

// birdWebPairs are the front toes, by place in birdToes, that a web
// joins: the inner to the middle, the middle to the outer.
var birdWebPairs = [][2]int{{1, 2}, {2, 3}}

// birdWeb is one web: from the foot to the points on two toes it
// reaches, thin through its face, the face looking toward up.
type birdWeb struct {
	foot, inner, outer maths.Vec
	thick              float64
}

// Middle is the web's line, from the foot to the middle of its edge.
func (web birdWeb) Middle() []maths.Vec {
	return []maths.Vec{web.foot, web.inner.Add(web.outer).Scale(.5)}
}

// Build adds the web to the mesh: a blade from the foot, widening to
// the distance between the toes at its edge.
func (web birdWeb) Build(points []maths.Vec, faces [][3]int,
	place func(maths.Vec) maths.Vec) ([]maths.Vec, [][3]int) {
	const rings, least = 6, .05 // picked
	edge := web.inner.Add(web.outer).Scale(.5)
	half := web.outer.Sub(web.inner).Len() / 2
	path := func(t float64) maths.Vec {
		return web.foot.Add(edge.Sub(web.foot).Scale(t))
	}
	size := func(t float64) (float64, float64) {
		return half * math.Max(least, t), web.thick
	}
	return skeleton.Sweep(points, faces, rings, path, size, birdUp, place)
}

// webPieces are the webs of both feet, as far as the row's web reaches.
func (body *birdBody) webPieces() []birdPiece {
	m := body.measures
	if m.web <= 0 {
		return nil
	}
	pieces := []birdPiece{}
	for _, side := range birdSides {
		at, _ := body.bones.Find("tarsometatarsus-" + side.name)
		foot := body.bones.Bone(at).Tail()
		reach := func(n int) maths.Vec {
			toe := birdToes[n]
			way := maths.V(toe.forward, 0, side.sign*toe.out).Unit()
			return foot.Add(way.Scale(toe.length * m.toe * m.web))
		}
		for _, pair := range birdWebPairs {
			pieces = append(pieces, birdPiece{
				bone:   fmt.Sprintf("toe3-1-%s", side.name),
				region: "feet", form: birdWeb{foot: foot,
					inner: reach(pair[0]), outer: reach(pair[1]),
					thick: birdWebThick * m.length}})
		}
	}
	return pieces
}
