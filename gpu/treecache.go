package gpu

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	bolt "go.etcd.io/bbolt"
	berrors "go.etcd.io/bbolt/errors"

	"github.com/janearc/slayer/maths"
)

// The trees the kernel walks are grown on the cpu, one for each mesh,
// every time a world is encoded: for moonlake's fire frames about six
// seconds a glimpse, the same trees every glimpse and every frame, in
// every process. CacheTrees keeps them in a bbolt file between runs.
//
// A tree is kept under the hash of the program that grew it and the
// faces it was grown over, in order. A rebuilt program or a changed mesh
// misses and grows afresh, so a kept tree is the one grow makes. A kept
// tree begins with the sha256 of the rest of it, checked when it is
// read, since bbolt does not check what it holds.
//
// bbolt lets one process write at a time, so the file is opened for one
// look or one store and closed; processes drawing side by side take
// turns. A process that waits lockWait for its turn stops using the file
// for the rest of its run, so a lock held elsewhere costs one wait.
//
// The cache never decides a picture: a tree it cannot read, cannot
// trust or cannot store is grown as if it were not there, and counted,
// the first such failure kept, for TreesCached to say. The file is never
// emptied: it grows by a scene's trees for every program that draws it,
// about 588 MB for moonlake's, and whoever owns the disk clears it.

const (
	// treeBucket is the bucket the trees are kept in
	treeBucket = "trees"
	// fewestCached is the fewest faces worth a tree in the file; a
	// smaller mesh grows in less time than a look at the file takes
	fewestCached = 4096
	// lockWait is the longest a process waits for another's turn at the
	// file. picked: six minutes (2026-11-11), since one file now
	// serves every render on the machine, and on a busy night waiting
	// is cheaper than growing a scene's trees again. a store holds the
	// file under a second, so a wait that long is a process that is
	// stuck, and this one goes on without the file
	lockWait = 6 * time.Minute
	// fileMode is the file's permissions: its owner's alone
	fileMode = 0o600
	// numberBytes is a float64's bytes, and a count's
	numberBytes = 8
	// pointNumbers is a point's numbers
	pointNumbers = 3
	// facePoints is a face's corners and normals
	facePoints = 6
	// faceMarks is the numbers a face's mark takes: smooth or flat
	faceMarks = 1
	// faceBytes is a face packed: its points, then its mark
	faceBytes = (facePoints*pointNumbers + faceMarks) * numberBytes
	// boxCorners is a node's box's corners, low and high
	boxCorners = 2
	// nodeCounts is a node's first and its count
	nodeCounts = 2
	// nodeBytes is a node packed: its box's corners, first and count
	nodeBytes = (boxCorners*pointNumbers + nodeCounts) * numberBytes
	// headCounts is a packed tree's counts, of faces and of nodes
	headCounts = 2
	// headBytes is those counts packed
	headBytes = headCounts * numberBytes
	// sumBytes is the sha256 a kept tree begins with
	sumBytes = sha256.Size
	// keyShown is how much of a key an error shows: enough to find it
	keyShown = 8
)

// flatMark and smoothMark are a face's mark: flat, or smooth
const (
	flatMark   uint64 = iota
	smoothMark uint64 = iota
)

// The marks a node's indices are checked against: a leaf holds a face
// at least, a branch holds none itself, and its second child follows its
// first. firstPlace is the first index there is.
const (
	leafLeast   = 1
	branchCount = 0
	nextChild   = 1
	firstPlace  = 0
)

// programKey is a sha256: of the program, or a tree's key
type programKey = [sha256.Size]byte

// shelf is the tree cache when it is on, or nil
var shelf atomic.Pointer[treeShelf]

// treeShelf is the file trees are kept in, the program that grows them,
// how long it waits for its turn, what it has done, why it stopped
// using the file, when it has, and where it says so.
type treeShelf struct {
	path    string
	program programKey
	wait    time.Duration
	said    io.Writer
	form    string
	guard   sync.Mutex
	tally   TreeTally
	stop    error
}

// TreeTally is what the tree cache did: trees taken from its file,
// trees grown and stored there, and trees grown because the file
// failed, with the first failure.
type TreeTally struct {
	kept, grown, failed int
	first               error
}

// String is the tally in a line.
func (tally TreeTally) String() string {
	said := fmt.Sprintf("trees: %d from the cache, %d grown and kept, "+
		"%d grown past it", tally.kept, tally.grown, tally.failed)
	if tally.first != nil {
		said += fmt.Sprintf(", the first because: %v", tally.first)
	}
	return said
}

// CacheTrees keeps the trees grown over meshes' faces in the bbolt file
// at path, made when it is missing, and looks for each there before
// growing it; an empty path turns it off. It fails only when the running
// program cannot be read, since its hash is part of every key.
func CacheTrees(path string) error {
	if path == "" {
		shelf.Store(nil)
		return nil
	}
	form, err := formOf(os.Getenv(storeVariable))
	if err != nil {
		return err
	}
	kept := &treeShelf{path: path, wait: lockWait, said: os.Stderr,
		form: form}
	if err := kept.hashProgram(); err != nil {
		return fmt.Errorf("tree cache: %w", err)
	}
	shelf.Store(kept)
	return nil
}

// TreesCached is what the tree cache has done since CacheTrees turned it
// on; nothing when it is off.
func TreesCached() TreeTally {
	kept := shelf.Load()
	if kept == nil {
		return TreeTally{}
	}
	kept.guard.Lock()
	defer kept.guard.Unlock()
	return kept.tally
}

// hashProgram puts the running program's sha256 on the shelf.
func (kept *treeShelf) hashProgram() error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	hash.Sum(kept.program[:firstPlace])
	return nil
}

// grownOver is the tree grow makes over the faces: from the cache when
// it is on and holds it, and kept there when it does not.
func grownOver(faces []triangle) tree {
	kept := shelf.Load()
	if kept == nil || len(faces) < fewestCached {
		return grow(faces)
	}
	return kept.grown(faces)
}

// grown is the tree over the faces from the file, or grown and stored
// there. A tree the file cannot give is grown, and stored over whatever
// was under its key when the file itself answered.
func (kept *treeShelf) grown(faces []triangle) tree {
	if stop := kept.stopped(); stop != nil {
		kept.note(false, stop)
		return grow(faces)
	}
	key := kept.key(faces)
	value, err := kept.lookAs(key)
	if err != nil {
		kept.note(false, err)
		return grow(faces)
	}
	if value != nil {
		made, bad := unpackTree(value, len(faces))
		if bad == nil {
			kept.note(true, nil)
			return *made
		}
		err = fmt.Errorf("tree %x: %w", key[:keyShown], bad)
	}
	made := grow(faces)
	err = errors.Join(err, kept.storeAs(key, packTree(made)))
	kept.note(false, err)
	return made
}

// note counts a tree: taken from the file, grown and kept, or grown
// past a failure.
func (kept *treeShelf) note(found bool, err error) {
	kept.guard.Lock()
	defer kept.guard.Unlock()
	switch {
	case found:
		kept.tally.kept++
	case err != nil:
		kept.tally.failed++
		if kept.tally.first == nil {
			kept.tally.first = err
		}
	default:
		kept.tally.grown++
	}
}

// halt stops the shelf using its file for the rest of the run, and
// says so on a line of its own the moment it does: every tree after is
// grown, which can cost minutes, and a tally at the end is too late.
func (kept *treeShelf) halt(why error) {
	kept.guard.Lock()
	defer kept.guard.Unlock()
	if kept.stop != nil {
		return
	}
	kept.stop = why
	if kept.said != nil {
		fmt.Fprintf(kept.said, "tree cache: gave up on %s; every tree "+
			"from here is grown, not read: %v\n", kept.path, why)
	}
}

// stopped is why the shelf stopped using its file, or nil.
func (kept *treeShelf) stopped() error {
	kept.guard.Lock()
	defer kept.guard.Unlock()
	return kept.stop
}

// key is the hash of the program and of the faces, in order, as grow
// reads them.
func (kept *treeShelf) key(faces []triangle) programKey {
	hash := sha256.New()
	hash.Write(kept.program[:])
	packed := make([]byte, firstPlace, faceBytes)
	for _, face := range faces {
		hash.Write(packFace(packed[:firstPlace], face))
	}
	var key programKey
	hash.Sum(key[:firstPlace])
	return key
}

// open opens the file for a look or a store, waiting for another
// process's turn at it; a wait that runs out stops the shelf using the
// file for the rest of the run.
func (kept *treeShelf) open(reading bool) (*bolt.DB, error) {
	file, err := bolt.Open(kept.path, fileMode,
		&bolt.Options{Timeout: kept.wait, ReadOnly: reading})
	if errors.Is(err, berrors.ErrTimeout) {
		err = fmt.Errorf("%s: held elsewhere past %v, so not used again "+
			"this run: %w", kept.path, kept.wait, err)
		kept.halt(err)
	}
	return file, err
}

// look is the value under a key, or nil when there is none: a copy,
// since bbolt's own lasts only while its file is open. A file not made
// yet holds nothing.
func (kept *treeShelf) look(key programKey) ([]byte, error) {
	if _, err := os.Stat(kept.path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	file, err := kept.open(true)
	if err != nil {
		return nil, err
	}
	reading, err := file.Begin(false)
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	var value []byte
	if bucket := reading.Bucket([]byte(treeBucket)); bucket != nil {
		value = bytes.Clone(bucket.Get(key[:]))
	}
	return value, errors.Join(reading.Rollback(), file.Close())
}

// store keeps a value under a key, making the file and its bucket when
// they are not there.
func (kept *treeShelf) store(key programKey, value []byte) error {
	file, err := kept.open(false)
	if err != nil {
		return err
	}
	writing, err := file.Begin(true)
	if err != nil {
		return errors.Join(err, file.Close())
	}
	bucket, err := writing.CreateBucketIfNotExists([]byte(treeBucket))
	if err == nil {
		err = bucket.Put(key[:], value)
	}
	if err != nil {
		return errors.Join(err, writing.Rollback(), file.Close())
	}
	return errors.Join(writing.Commit(), file.Close())
}

// packTree is a tree as kept: the sha256 of the rest, then its counts of
// faces and nodes, its faces in the order its leaves hold them, and its
// nodes.
func packTree(grown tree) []byte {
	faces, nodes := uint64(len(grown.triangles)), uint64(len(grown.nodes))
	body := make([]byte, firstPlace, headBytes+faces*faceBytes+nodes*nodeBytes)
	body = binary.LittleEndian.AppendUint64(body, faces)
	body = binary.LittleEndian.AppendUint64(body, nodes)
	for _, face := range grown.triangles {
		body = packFace(body, face)
	}
	for _, each := range grown.nodes {
		body = packNode(body, each)
	}
	sum := sha256.Sum256(body)
	return append(sum[:], body...)
}

// packFace appends a face: its corners, its normals, and its mark.
func packFace(packed []byte, face triangle) []byte {
	for _, corner := range face.corners {
		packed = packPoint(packed, corner)
	}
	for _, normal := range face.normals {
		packed = packPoint(packed, normal)
	}
	mark := flatMark
	if face.smooth {
		mark = smoothMark
	}
	return binary.LittleEndian.AppendUint64(packed, mark)
}

// packNode appends a node: its box's low and high corners, its first and
// its count.
func packNode(packed []byte, each node) []byte {
	packed = packPoint(packed, each.bounds.low)
	packed = packPoint(packed, each.bounds.high)
	first, count := uint64(each.first), uint64(each.count)
	packed = binary.LittleEndian.AppendUint64(packed, first)
	return binary.LittleEndian.AppendUint64(packed, count)
}

// packPoint appends a point's three numbers, every bit of each.
func packPoint(packed []byte, at maths.Vec) []byte {
	for _, part := range [pointNumbers]float64{at.X, at.Y, at.Z} {
		bits := math.Float64bits(part)
		packed = binary.LittleEndian.AppendUint64(packed, bits)
	}
	return packed
}

// unpackTree is a kept tree over the given number of faces, refused when
// its sha256 is not the rest's, or its size, its counts or any node's
// indices do not fit.
func unpackTree(packed []byte, faces int) (*tree, error) {
	if len(packed) < sumBytes+headBytes {
		return nil, fmt.Errorf("%d bytes is too few for a tree", len(packed))
	}
	sum, body := packed[:sumBytes], packed[sumBytes:]
	if want := sha256.Sum256(body); !bytes.Equal(sum, want[:]) {
		return nil, errors.New("the tree is not the tree its sha256 was " +
			"taken of: the file is damaged")
	}
	read := unpacker{rest: body}
	count, nodes := read.count(), read.count()
	if count != uint64(faces) {
		return nil, fmt.Errorf("a tree of %d faces kept for %d", count, faces)
	}
	if nodes > uint64(len(body)/nodeBytes) ||
		uint64(len(body)) != headBytes+count*faceBytes+nodes*nodeBytes {
		return nil, fmt.Errorf("%d bytes is not %d faces and %d nodes",
			len(body), count, nodes)
	}
	made := &tree{triangles: make([]triangle, count), nodes: make([]node, nodes)}
	for i := range made.triangles {
		made.triangles[i] = read.face()
	}
	for i := range made.nodes {
		made.nodes[i] = read.node()
		if err := made.fits(i); err != nil {
			return nil, err
		}
	}
	return made, nil
}

// fits is whether the node at index points inside the tree: a leaf at
// its faces, a branch at two nodes after it.
func (grown tree) fits(index int) error {
	each := grown.nodes[index]
	leaf := each.count >= leafLeast && each.first >= firstPlace &&
		each.first+each.count <= len(grown.triangles)
	branch := each.count == branchCount && each.first > index &&
		each.first+nextChild < len(grown.nodes)
	if !leaf && !branch {
		return fmt.Errorf("node %d points outside its tree", index)
	}
	return nil
}

// unpacker takes numbers off the front of packed bytes, which unpackTree
// has measured first.
type unpacker struct{ rest []byte }

// count is the next count.
func (read *unpacker) count() uint64 {
	value := binary.LittleEndian.Uint64(read.rest)
	read.rest = read.rest[numberBytes:]
	return value
}

// number is the next float64, every bit of it.
func (read *unpacker) number() float64 {
	bits := read.count()
	return math.Float64frombits(bits)
}

// point is the next point.
func (read *unpacker) point() maths.Vec {
	x, y, z := read.number(), read.number(), read.number()
	return maths.V(x, y, z)
}

// face is the next face.
func (read *unpacker) face() triangle {
	var made triangle
	for i := range made.corners {
		made.corners[i] = read.point()
	}
	for i := range made.normals {
		made.normals[i] = read.point()
	}
	made.smooth = read.count() == smoothMark
	return made
}

// node is the next node.
func (read *unpacker) node() node {
	low, high := read.point(), read.point()
	first, count := read.count(), read.count()
	return node{bounds: box{low: low, high: high}, first: int(first),
		count: int(count)}
}
