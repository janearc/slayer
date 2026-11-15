//go:build darwin && arm64

package manufacture

import (
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"fmt"
	"hash"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// This is an authored CPU catalogue compatibility record for Darwin/ARM64.
// It holds geometry, normals, prototype sharing, poses, physical bounds and
// materials across the assembly refactor. It is not a physical validation
// of plants or an exact-pixel GPU comparison. The original 144 entries and
// their provenance are unchanged; additions have separate files and producer
// records, captured after their independent geometry checks.

func referenceWrite(h hash.Hash, values any) {
	if err := binary.Write(h, binary.LittleEndian, values); err != nil {
		panic(err)
	}
}
func referenceVector(h hash.Hash, point maths.Vec) {
	referenceWrite(h, [3]float64{point.X, point.Y, point.Z})
}
func referenceSpace(h hash.Hash, pose maths.Subspace) {
	referenceVector(h, pose.Origin)
	referenceWrite(h, pose.Scale)
	for _, axis := range []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis} {
		referenceVector(h, pose.Orientation.Out(axis))
	}
}
func referenceGeometry(h hash.Hash, solid shape.Shape, seen map[shape.Shape]int) {
	if index, exists := seen[solid]; exists {
		referenceWrite(h, int64(index))
		return
	}
	index := len(seen)
	seen[solid] = index
	referenceWrite(h, int64(-1-index))
	switch made := solid.(type) {
	case *mesh.Mesh:
		referenceWrite(h, byte(1))
		referenceWrite(h, int64(made.FaceCount()))
		for i := range made.FaceCount() {
			a, b, c := made.Corners(i)
			referenceVector(h, a)
			referenceVector(h, b)
			referenceVector(h, c)
			normals, smooth := made.CornerNormals(i)
			referenceWrite(h, smooth)
			if smooth {
				for _, normal := range normals {
					referenceVector(h, normal)
				}
			}
		}
	case *shape.Mesh:
		referenceWrite(h, byte(2))
		referenceWrite(h, int64(len(made.Faces)))
		for _, face := range made.Faces {
			for _, corner := range face {
				referenceVector(h, made.Vertices[corner])
			}
		}
		referenceWrite(h, made.Smooth)
		referenceWrite(h, int64(len(made.Normals)))
		for _, normal := range made.Normals {
			referenceVector(h, normal)
		}
	case *shape.MeshGroup:
		members := made.Members()
		referenceWrite(h, byte(3))
		referenceWrite(h, int64(len(members)))
		for _, member := range members {
			referenceSpace(h, member.Space)
			referenceGeometry(h, member.Mesh, seen)
		}
	default:
		panic(fmt.Sprintf("unrecorded manufactured shape %T", solid))
	}
}

// referenceEpoch is defined as the stationary specimen's zero time.
const referenceEpoch = 0

// referenceAbsent and referencePresent are defined material record tags.
const referenceAbsent, referencePresent = 0, 1

// referenceMaterialFields is defined by RGB, glow RGB, gloss and index.
const referenceMaterialFields = 8

// referenceDigest is the record's hash of a specimen, for checking a
// row and for writing a new addition's rows.
func referenceDigest(t *testing.T, made good) string {
	t.Helper()
	// Preserve the original record's valid zero-value placement exactly.
	var placement maths.Subspace
	things, err := made.Things(placement)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := sha256.New()
	bounds := made.Bounds()
	referenceVector(fingerprint, bounds.Min)
	referenceVector(fingerprint, bounds.Max)
	count := int64(len(things))
	referenceWrite(fingerprint, count)
	shared := map[shape.Shape]int{}
	for _, thing := range things {
		referenceSpace(fingerprint, thing.At(referenceEpoch))
		referenceGeometry(fingerprint, thing.Shape, shared)
		material := thing.Material
		if material == nil {
			referenceWrite(fingerprint, byte(referenceAbsent))
			continue
		}
		referenceWrite(fingerprint, byte(referencePresent))
		nameLength := int64(len(material.Name))
		referenceWrite(fingerprint, nameLength)
		if _, err := fingerprint.Write([]byte(material.Name)); err != nil {
			t.Fatal(err)
		}
		values := [referenceMaterialFields]float64{
			material.Colour.R, material.Colour.G, material.Colour.B,
			material.Glow.R, material.Glow.G, material.Glow.B,
			material.Gloss, material.Index}
		referenceWrite(fingerprint, values)
	}
	return fmt.Sprintf("%x", fingerprint.Sum(nil))
}
