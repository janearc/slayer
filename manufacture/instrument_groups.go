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
	if assembly.trouble != nil {
		return assembly.trouble
	}
	prototypes := map[[sha256.Size]byte]*mesh.Mesh{}
	members := map[slot][]shape.MeshMember{}
	for _, piece := range assembly.object.parts {
		skin := piece.shape.skin(piece.fine)
		if len(skin) > mostFaces {
			return errTooMany
		}
		key := componentFingerprint(skin)
		polygon := prototypes[key]
		if polygon == nil {
			var err error
			polygon, err = skin.meshed()
			if err != nil {
				return err
			}
			prototypes[key] = polygon
		}
		members[piece.of] = append(members[piece.of],
			shape.MeshMember{Mesh: polygon, Space: piece.space})
	}
	return assembly.keepComponentGroups(members)
}

// keepComponentGroups publishes only after every material's hierarchy is valid.
func (assembly *instrumentAssembly) keepComponentGroups(members map[slot][]shape.MeshMember) error {
	groups := map[slot]*shape.MeshGroup{}
	for role, parts := range members {
		group, err := shape.NewMeshGroup(parts)
		if err != nil {
			return err
		}
		groups[role] = group
	}
	assembly.object.parts = nil
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
