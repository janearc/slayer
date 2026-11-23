package manufacture

import (
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// shareComponents keeps closed components local and instances their placements.
// The cache is per assembly and keys exact vertex/winding bits, so forms with
// slice-backed profiles need neither reflection nor comparable interface
// values.
func (assembly *instrumentAssembly) shareComponents() error {
	return assembly.shareWhere(func(part) bool { return true })
}

// shareRepeats instances only the parts that repeat in a flat matter: a
// matter of two parts or more, with no rounded part. A group holds flat
// meshes alone, so a rounded matter stays whole parts, shaded smooth, and
// so does a matter of one part, which has nothing to share.
func (assembly *instrumentAssembly) shareRepeats() error {
	round := assembly.object.roundSlots()
	counts := map[slot]int{}
	for _, piece := range assembly.object.parts {
		counts[piece.of]++
	}
	return assembly.shareWhere(func(piece part) bool {
		return !round[piece.of] && counts[piece.of] > 1
	})
}

// shareWhere instances the parts that share says to, each distinct
// component stored once, and leaves the others as parts.
func (assembly *instrumentAssembly) shareWhere(share func(part) bool) error {
	if assembly.trouble != nil {
		return assembly.trouble
	}
	prototypes := map[[sha256.Size]byte]*mesh.Mesh{}
	members := map[slot][]shape.MeshMember{}
	var kept []part
	for _, piece := range assembly.object.parts {
		if !share(piece) {
			kept = append(kept, piece)
			continue
		}
		polygon, err := componentMesh(prototypes, piece)
		if err != nil {
			return err
		}
		members[piece.of] = append(members[piece.of],
			shape.MeshMember{Mesh: polygon, Space: piece.space})
	}
	return assembly.keepComponentGroups(members, kept)
}

// componentMesh is the part's closed mesh in its own space, the one
// already made when an identical part has been met.
func componentMesh(prototypes map[[sha256.Size]byte]*mesh.Mesh,
	piece part,
) (*mesh.Mesh, error) {
	skin := piece.shape.skin(piece.fine)
	key := componentFingerprint(skin)
	polygon := prototypes[key]
	if polygon == nil {
		var err error
		polygon, err = skin.meshed()
		if err != nil {
			return nil, err
		}
		prototypes[key] = polygon
	}
	return polygon, nil
}

// keepComponentGroups publishes only after every material's hierarchy is
// valid, and keeps the parts that were not instanced.
func (assembly *instrumentAssembly) keepComponentGroups(members map[slot][]shape.MeshMember, kept []part) error {
	groups := map[slot]*shape.MeshGroup{}
	for role, parts := range members {
		group, err := shape.NewMeshGroup(parts)
		if err != nil {
			return err
		}
		groups[role] = group
	}
	assembly.object.groupParts = assembly.object.partCounts()
	assembly.object.parts = kept
	assembly.object.groups = groups
	return nil
}

// componentFingerprint includes every ordered triangle corner, without poses.
func componentFingerprint(skin surface) [sha256.Size]byte {
	hash := sha256.New()
	var word [8]byte
	for _, triangle := range skin {
		for _, point := range triangle {
			for _, value := range [3]float64{point.X, point.Y, point.Z} {
				binary.LittleEndian.PutUint64(word[:], math.Float64bits(value))
				hash.Write(word[:])
			}
		}
	}
	var fingerprint [sha256.Size]byte
	copy(fingerprint[:], hash.Sum(nil))
	return fingerprint
}
