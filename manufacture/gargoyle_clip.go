package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// clipDepth is how far behind the face of a gargoyle's panel, in metres, a
// face of the creature may stand and still be kept. A face with any corner
// in front of this plane is kept whole, so that none is cut across and
// the creature meets the stone with no gap; a face wholly behind it is in
// the wall and is left out, as is anything of the creature that is not a
// mesh and has its middle behind it.
//
// picked: enough for the faces of a
// fine mesh that straddle the plane, and no more, so that nothing of
// the creature comes through a thin wall at the back.
const clipDepth = 0.1

// the numbers a gargoyle's small features are carved coarse by
const (
	// smallFeature is how long, in metres of the creature's own frame, the
	// diagonal of the box of a piece that is not a mesh may be for it to
	// be a feature: an eye, a tooth, a sucker. picked: under the width of
	// a head and over the largest eye.
	smallFeature = 0.3
	// coarseGrowth is how many times larger a feature is carved than the
	// creature has it, about its own middle, so that it is seen from the
	// street: a mason cuts an eye deeper than an animal has it. picked.
	coarseGrowth = 2.5
)

// soup is a surface as bare triangles: corners, the faces that join them,
// and a normal at each corner if the surface has them.
type soup struct {
	corners []maths.Vec
	faces   [][3]int
	normals []maths.Vec
	smooth  bool
}

// soupOfMesh is a mesh of package mesh as bare triangles. The normals are
// the corner normals of its faces, which for a smooth mesh are one to a
// corner.
func soupOfMesh(from *mesh.Mesh) soup {
	bare := soup{corners: from.Points(), faces: from.Faces(),
		smooth: from.Smooth()}
	for index := range bare.faces {
		around, has := from.CornerNormals(index)
		if !has {
			bare.smooth = false
			continue
		}
		if bare.normals == nil {
			bare.normals = make([]maths.Vec, len(bare.corners))
		}
		for corner, vertex := range bare.faces[index] {
			bare.normals[vertex] = around[corner]
		}
	}
	return bare
}

// soupOfShape is a mesh of package shape as bare triangles.
func soupOfShape(from *shape.Mesh) soup {
	return soup{corners: from.Vertices, faces: from.Faces,
		normals: from.Normals, smooth: from.Smooth}
}

// within is the soup taken through a place into another frame, keeping
// only the faces that have a corner in front of the plane z is -clipDepth
// there. It says whether any face is kept.
func (bare soup) within(place maths.Subspace) (soup, bool) {
	kept := soup{smooth: bare.smooth}
	index := map[int]int{}
	for _, face := range bare.faces {
		if !bare.reaches(face, place) {
			continue
		}
		var joined [3]int
		for corner, vertex := range face {
			at, known := index[vertex]
			if !known {
				at = len(kept.corners)
				index[vertex] = at
				kept.corners = append(kept.corners,
					place.Out(bare.corners[vertex]))
				if bare.normals != nil {
					kept.normals = append(kept.normals,
						place.Orientation.Out(bare.normals[vertex]).Unit())
				}
			}
			joined[corner] = at
		}
		kept.faces = append(kept.faces, joined)
	}
	return kept, len(kept.faces) > 0
}

// reaches is whether a face, taken through a place, has a corner in front
// of the plane z is -clipDepth.
func (bare soup) reaches(face [3]int, place maths.Subspace) bool {
	for _, vertex := range face {
		if place.Out(bare.corners[vertex]).Z > -clipDepth {
			return true
		}
	}
	return false
}

// carvedPiece is a piece of a creature taken through a place into the
// block's frame with the part behind the wall cut away, in the material
// given. It says whether any of the piece is left. A mesh is rebuilt, in
// the block's frame, as the kind of mesh it was; a piece that is not a
// mesh is kept where it is if its middle is in front of the plane.
func carvedPiece(piece thing.Thing, place maths.Subspace,
	stone *thing.Material) (thing.Thing, bool, error) {
	piece.Material = stone
	var err error
	switch shaped := piece.Shape.(type) {
	case *mesh.Mesh:
		kept, left := soupOfMesh(shaped).within(place)
		if !left {
			return piece, false, nil
		}
		piece.Motion = nil
		piece.Shape, err = mesh.New(kept.corners, kept.faces,
			kept.normals, kept.smooth)
		return piece, err == nil, err
	case *shape.Mesh:
		kept, left := soupOfShape(shaped).within(place)
		if !left {
			return piece, false, nil
		}
		piece.Motion = nil
		piece.Shape, err = shape.NewMesh(kept.corners, kept.faces,
			kept.normals, kept.smooth)
		return piece, err == nil, err
	}
	middle := place.Out(maths.Vec{}).Z > -clipDepth
	piece.Motion = thing.Still(coarsened(piece.Shape, place))
	return piece, middle, nil
}

// coarsened is the place of a piece that is not a mesh, with a small
// feature made larger about its own middle by coarseGrowth, and any other
// piece where it was.
func coarsened(shaped shape.Shape, place maths.Subspace) maths.Subspace {
	box := shaped.Bounds()
	if box.Max.Sub(box.Min).Len() > smallFeature {
		return place
	}
	if place.Scale == 0 {
		place.Scale = 1
	}
	place.Scale *= coarseGrowth
	return place
}
