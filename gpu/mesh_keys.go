package gpu

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"unsafe"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// meshKey is what a mesh's faces are, as a sha256 of the bytes they are
// made from. Two meshes with one key give the same faces in the same
// order, and so the same tree, whichever pointer holds them: the heard
// stages make meshes again as new pointers with the same faces in them.
//
// It is taken of the bytes and never of a pointer, so a shape.Mesh, whose
// fields can be changed in place after it is made, has a new key the
// frame its faces change. The key lives only in this process, so the
// bytes are hashed in the machine's own order.
type meshKey [sha256.Size]byte

// The kinds of mesh a key is taken of, hashed first, so two kinds whose
// bytes happen to agree never share a key: their faces are made apart.
const (
	shapeMeshKind byte = iota + 1
	slayersMeshKind
)

// vectorBytes is a maths.Vec's bytes: three float64s.
const vectorBytes = int(unsafe.Sizeof(maths.Vec{}))

// cornerBytes is a shape.Mesh face's bytes: three indices.
const cornerBytes = int(unsafe.Sizeof([3]int{}))

// shapeMeshKey is a shape.Mesh's key: whether it is smooth, and every
// vertex, normal and face it holds, each run after its length. These are
// everything meshFaces reads.
func shapeMeshKey(made *shape.Mesh) meshKey {
	sum := sha256.New()
	sum.Write([]byte{shapeMeshKind, byte(flag(made.Smooth))})
	hashRun(sum, len(made.Vertices), vectorsAsBytes(made.Vertices))
	hashRun(sum, len(made.Normals), vectorsAsBytes(made.Normals))
	faces := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(
		made.Faces))), len(made.Faces)*cornerBytes)
	hashRun(sum, len(made.Faces), faces)
	return keyOfSum(sum)
}

// slayersMeshKey is the key of one of slayer's meshes, taken of its faces
// as slayersFaces makes them: every corner, normal and mark, as the tree
// cache packs a face.
func slayersMeshKey(faces []triangle) meshKey {
	sum := sha256.New()
	sum.Write([]byte{slayersMeshKind})
	packed := make([]byte, firstPlace, faceBytes)
	hashRun(sum, len(faces), nil)
	for _, face := range faces {
		sum.Write(packFace(packed[:firstPlace], face))
	}
	return keyOfSum(sum)
}

// hashRun writes a run's length and then its bytes, so where one run
// ends and the next begins is part of the key.
func hashRun(sum hash.Hash, length int, run []byte) {
	var counted [numberBytes]byte
	binary.LittleEndian.PutUint64(counted[:], uint64(length))
	sum.Write(counted[:])
	sum.Write(run)
}

// vectorsAsBytes is the bytes under a run of points, read in place.
func vectorsAsBytes(points []maths.Vec) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(points))),
		len(points)*vectorBytes)
}

// keyOfSum is the key a finished hash gives.
func keyOfSum(sum hash.Hash) meshKey {
	var key meshKey
	sum.Sum(key[:firstPlace])
	return key
}
