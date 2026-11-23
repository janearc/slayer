package manufacture

import (
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// retailTowerSalt is mixed into the seed of the shops a tower carries, so
// that they are not the seeds of its signs or its gargoyles. A choice.
const retailTowerSalt = 0x53484f50544f5752

// retailSetback is how far behind the plane of a tower's wall the fronts of
// its shops stand, in metres: a hand's width. A choice.
const retailSetback = 0.1

// retailTie is how far the plain wall at a tower's foot reaches into the
// shops' own walls, in metres, so that no two faces lie in one plane. A
// choice.
const retailTie = 0.2

// retailed is the good with shops in the foot of a tower, if its recipe
// pins retail on and the tower is one that can have them: a single plain
// rectangle, which cities.Tower.Footed has lifted on an open storey of the
// height of a unit.
//
// The shops stand in that storey, each wall of the tower
// given a row of the retail units of retail_ground.go, drawn once and
// shared, their fronts a hand's width behind the plane of the wall and
// their rooms running back into the building.
//
// A corner of the tower and its
// middle are plain wall. A tower whose recipe does not pin retail on, or
// that cannot have shops, is returned as it is, with no open storey.
func retailed(made good, spec TowerSpec, tower *cities.Tower, from seed,
	detail string) (good, error) {
	if spec.Retail == nil || !*spec.Retail || tower.Foot() <= 0 {
		return made, nil
	}
	if retailBookError != nil {
		return good{}, retailBookError
	}
	faces, err := tower.FootFaces()
	if err != nil {
		return good{}, err
	}
	depth := retailBookRead.Grounds.Podium
	level := max(retailLevel(detail), 0)
	all := []good{made}
	var plain []footPiece
	for number, face := range faces {
		width := face.Length - 2*depth
		strip, found, err := retailStrip(sow(from.number^retailTowerSalt^
			uint64(number)), level, width)
		if err != nil {
			return good{}, err
		}
		if !found {
			plain = append(plain, footPiece{face: number, low: maths.V(
				-face.Length*spaceHalf, 0, -depth), high: maths.V(
				face.Length*spaceHalf, tower.Foot(), 0)})
			continue
		}
		placed, err := strip.placed(maths.NewSubspace(face.Mid.Add(
			face.Along.Scale(-width*spaceHalf)).Add(face.Out.Scale(
			-depth-retailSetback)), faceTurn(face), spaceOne))
		if err != nil {
			return good{}, err
		}
		all = append(all, placed)
	}
	wall, err := footWall(faces, plain, depth, tower.Foot())
	if err != nil {
		return good{}, err
	}
	return gather(append(all, wall)...)
}

// footPiece is a plain box in the frame of a face of a tower, which has its
// x along the face, its y up and its z out of it, with its middle at the
// middle of the face's foot.
type footPiece struct {
	face      int
	low, high maths.Vec
}

// faceTurn is the way a face of a tower turns a thing stood against it: x
// along the face, y up, z out of it.
func faceTurn(face cities.FootFace) maths.Orientation {
	return maths.Orientation{X: face.Along, Y: maths.Up, Z: face.Out}
}

// footWall is the plain wall in the open storey at a tower's foot, in the
// material the data fills a podium with: the pieces given, which are bands
// of wall where a face had no room for a unit, and, in all of the
// rectangle but the bands the shops fill, the four corners and the middle.
//
// The tower is a rectangle, so the first face and the one after it say how
// wide and how deep it is. A tower with no room for a room in it is plain
// wall all through.
func footWall(faces []cities.FootFace, pieces []footPiece, depth,
	high float64) (good, error) {
	across, deep := faces[0].Length, faces[1].Length
	half := across * spaceHalf
	if across <= 2*depth || deep <= 2*depth {
		pieces = []footPiece{{low: maths.V(-half, 0, -deep),
			high: maths.V(half, high, 0)}}
	} else {
		pieces = append(pieces, footPiece{low: maths.V(depth-half-retailTie,
			0, depth-deep-retailTie), high: maths.V(half-depth+retailTie,
			high, -depth+retailTie)})
		for _, left := range []float64{-half, half - depth - retailTie} {
			for _, back := range []float64{-deep, -depth - retailTie} {
				pieces = append(pieces, footPiece{low: maths.V(left, 0, back),
					high: maths.V(left+depth+retailTie, high,
						back+depth+retailTie)})
			}
		}
	}
	var members []shape.MeshMember
	for _, piece := range pieces {
		face := faces[piece.face]
		polygon, err := cityBox(piece.low.X, piece.high.X, piece.low.Y,
			piece.high.Y, piece.low.Z, piece.high.Z).placed(
			maths.NewSubspace(face.Mid, faceTurn(face), spaceOne)).meshed()
		if err != nil {
			return good{}, err
		}
		members = append(members, shape.MeshMember{Mesh: polygon,
			Space: maths.Local()})
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		return good{}, err
	}
	filler := retailBookRead.Grounds.Filler
	place := retailBookRead.slots()[filler]
	return good{dressed: looks{place: retailBookRead.look(filler)},
		groups:     map[slot]*shape.MeshGroup{place: group},
		groupParts: map[slot]int{place: len(members)}}, nil
}

// retailStrip is a podium of units a seed draws, as wide as a stretch of
// wall, standing on the ground with its back at the origin and its front
// out along z, its left end at the origin: the same units, stood the same
// way, as a block's. It says whether there was room for a unit.
func retailStrip(from seed, level int, width float64) (good, bool, error) {
	grounds := retailBookRead.Grounds
	placed, used, err := retailBlockRow{}.placed(from, level,
		width-2*grounds.Margin)
	if err != nil || len(placed) == 0 {
		return good{}, false, err
	}
	box := maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(width, 0, 0)}
	shops, err := retailBlockRow{}.podium(placed, box, used)
	return shops, err == nil, err
}
