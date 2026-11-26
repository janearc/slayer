package stagewire

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
)

const binaryMagic = "FSTAGE4\n"
const binaryMagicWithOrigins = "FSTAGE5\n"
const placeBytes = 13 * 8
const placeChunk = 1024

type binaryBatch struct {
	Mesh      int
	Count     uint64
	Bounds    maths.Box
	HasBounds bool
	Author    thing.AuthorRef
	Authored  bool
}

type binaryMesh struct {
	Vertices, Indices, Edges uint64
	Notice, Diagnostic       string
}

// writeBinaryBatches streams meshes and placements without flattening them.
func writeBinaryBatches(out io.Writer, snapshot Snapshot) error {
	writer := bufio.NewWriterSize(out, 1<<20)
	magic := binaryMagic
	if snapshot.Version == 5 {
		magic = binaryMagicWithOrigins
	}
	if _, err := writer.WriteString(magic); err != nil {
		return err
	}
	known := make(map[*placement.DisplayMesh]int)
	var meshes []*placement.DisplayMesh
	for _, batch := range snapshot.Scene.Batches {
		if _, found := known[batch.Mesh]; !found {
			known[batch.Mesh] = len(meshes)
			meshes = append(meshes, batch.Mesh)
		}
	}
	batches := snapshot.Scene.Batches
	snapshot.Scene.Batches = nil
	snapshot.MeshCount, snapshot.BatchCount = len(meshes), len(batches)
	if err := writeJSONRecord(writer, snapshot); err != nil {
		return err
	}
	for _, mesh := range meshes {
		if err := writeBinaryMesh(writer, mesh); err != nil {
			return err
		}
	}
	buffer := make([]byte, placeBytes*placeChunk)
	for _, batch := range batches {
		row := binaryBatch{Mesh: known[batch.Mesh], Count: uint64(len(batch.Places)), Bounds: batch.Bounds, HasBounds: batch.HasBounds, Author: batch.Author, Authored: batch.Authored}
		if err := writeJSONRecord(writer, row); err != nil {
			return err
		}
		for start := 0; start < len(batch.Places); start += placeChunk {
			count := min(placeChunk, len(batch.Places)-start)
			for i, place := range batch.Places[start : start+count] {
				encodePlace(buffer[i*placeBytes:], place)
			}
			if _, err := writer.Write(buffer[:count*placeBytes]); err != nil {
				return err
			}
		}
	}
	return writer.Flush()
}

// writeJSONRecord writes one length-prefixed metadata record.
func writeJSONRecord(out io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var size [8]byte
	binary.LittleEndian.PutUint64(size[:], uint64(len(data)))
	if _, err := out.Write(size[:]); err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

// readJSONRecord checks and decodes one length-prefixed metadata record.
func readJSONRecord(in io.Reader, value any) error {
	var size [8]byte
	if _, err := io.ReadFull(in, size[:]); err != nil {
		return err
	}
	length := binary.LittleEndian.Uint64(size[:])
	if length > math.MaxInt64 {
		return fmt.Errorf("staged record length is invalid")
	}
	limited := &io.LimitedReader{R: in, N: int64(length)}
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF || limited.N != 0 {
		return fmt.Errorf("staged record has trailing or missing data")
	}
	return nil
}

// readBinaryBatches restores shared meshes and checks the complete stream.
func readBinaryBatches(in *bufio.Reader) (Snapshot, error) {
	magic := make([]byte, len(binaryMagic))
	if _, err := io.ReadFull(in, magic); err != nil || (string(magic) != binaryMagic && string(magic) != binaryMagicWithOrigins) {
		return Snapshot{}, fmt.Errorf("invalid staged snapshot signature")
	}
	var snapshot Snapshot
	if err := readJSONRecord(in, &snapshot); err != nil {
		return Snapshot{}, err
	}
	if (string(magic) == binaryMagic && snapshot.Version != 4) || (string(magic) == binaryMagicWithOrigins && snapshot.Version != 5) || snapshot.MeshCount <= 0 || snapshot.BatchCount <= 0 || len(snapshot.Scene.Objects) != 0 || len(snapshot.Scene.Batches) != 0 || len(snapshot.Meshes) != 0 || len(snapshot.Places) != 0 {
		return Snapshot{}, fmt.Errorf("invalid binary staged header")
	}
	var meshes []*placement.DisplayMesh
	for i := 0; i < snapshot.MeshCount; i++ {
		mesh, err := readBinaryMesh(in)
		if err != nil {
			return Snapshot{}, fmt.Errorf("read staged mesh %d: %w", i, err)
		}
		meshes = append(meshes, mesh)
	}
	used := make([]bool, len(meshes))
	buffer := make([]byte, placeBytes*placeChunk)
	for i := 0; i < snapshot.BatchCount; i++ {
		var row binaryBatch
		if err := readJSONRecord(in, &row); err != nil {
			return Snapshot{}, fmt.Errorf("read staged batch %d: %w", i, err)
		}
		if row.Mesh < 0 || row.Mesh >= len(meshes) || row.Count == 0 {
			return Snapshot{}, fmt.Errorf("invalid staged batch reference or size")
		}
		batch := placement.StagedBatch{Mesh: meshes[row.Mesh], Bounds: row.Bounds, HasBounds: row.HasBounds, Author: row.Author, Authored: row.Authored}
		used[row.Mesh] = true
		for left := row.Count; left > 0; {
			count := int(min(uint64(placeChunk), left))
			if _, err := io.ReadFull(in, buffer[:count*placeBytes]); err != nil {
				return Snapshot{}, fmt.Errorf("read staged placements: %w", err)
			}
			for j := 0; j < count; j++ {
				batch.Places = append(batch.Places, decodePlace(buffer[j*placeBytes:]))
			}
			left -= uint64(count)
		}
		snapshot.Scene.Batches = append(snapshot.Scene.Batches, batch)
	}
	for _, present := range used {
		if !present {
			return Snapshot{}, fmt.Errorf("staged snapshot contains an unreferenced mesh")
		}
	}
	if _, err := in.ReadByte(); err != io.EOF {
		return Snapshot{}, fmt.Errorf("staged snapshot has trailing data")
	}
	snapshot.MeshCount, snapshot.BatchCount = 0, 0
	if err := validate(snapshot); err != nil {
		return Snapshot{}, err
	}
	snapshot.Scene.IndexPicking()
	return snapshot, nil
}

// writeBinaryMesh writes local vertices and indices in bounded chunks.
func writeBinaryMesh(out io.Writer, mesh *placement.DisplayMesh) error {
	header := binaryMesh{Vertices: uint64(len(mesh.Vertices)), Indices: uint64(len(mesh.Indices)), Edges: uint64(len(mesh.Edges)), Notice: mesh.Notice, Diagnostic: mesh.Diagnostic}
	if err := writeJSONRecord(out, header); err != nil {
		return err
	}
	const vertexBytes = 9 * 8
	buffer := make([]byte, vertexBytes*placeChunk)
	for start := 0; start < len(mesh.Vertices); start += placeChunk {
		count := min(placeChunk, len(mesh.Vertices)-start)
		for i, vertex := range mesh.Vertices[start : start+count] {
			values := [9]float64{vertex.Position.X, vertex.Position.Y, vertex.Position.Z, vertex.Normal.X, vertex.Normal.Y, vertex.Normal.Z, vertex.Colour.R, vertex.Colour.G, vertex.Colour.B}
			for j, value := range values {
				binary.LittleEndian.PutUint64(buffer[i*vertexBytes+j*8:], math.Float64bits(value))
			}
		}
		if _, err := out.Write(buffer[:count*vertexBytes]); err != nil {
			return err
		}
	}
	for _, indices := range [][]uint32{mesh.Indices, mesh.Edges} {
		for start := 0; start < len(indices); start += placeChunk {
			count := min(placeChunk, len(indices)-start)
			for i, index := range indices[start : start+count] {
				binary.LittleEndian.PutUint32(buffer[i*4:], index)
			}
			if _, err := out.Write(buffer[:count*4]); err != nil {
				return err
			}
		}
	}
	return nil
}

// readBinaryMesh reads local vertices and indices in bounded chunks.
func readBinaryMesh(in io.Reader) (*placement.DisplayMesh, error) {
	var header binaryMesh
	if err := readJSONRecord(in, &header); err != nil {
		return nil, err
	}
	mesh := &placement.DisplayMesh{Notice: header.Notice, Diagnostic: header.Diagnostic}
	const vertexBytes = 9 * 8
	buffer := make([]byte, vertexBytes*placeChunk)
	for left := header.Vertices; left > 0; {
		count := int(min(uint64(placeChunk), left))
		if _, err := io.ReadFull(in, buffer[:count*vertexBytes]); err != nil {
			return nil, err
		}
		for i := 0; i < count; i++ {
			var value [9]float64
			for j := range value {
				value[j] = math.Float64frombits(binary.LittleEndian.Uint64(buffer[i*vertexBytes+j*8:]))
			}
			mesh.Vertices = append(mesh.Vertices, placement.DisplayVertex{Position: maths.V(value[0], value[1], value[2]), Normal: maths.V(value[3], value[4], value[5]), Colour: maths.Light{R: value[6], G: value[7], B: value[8]}})
		}
		left -= uint64(count)
	}
	readIndices := func(count uint64) ([]uint32, error) {
		var result []uint32
		for count > 0 {
			chunk := int(min(uint64(placeChunk), count))
			if _, err := io.ReadFull(in, buffer[:chunk*4]); err != nil {
				return nil, err
			}
			for i := 0; i < chunk; i++ {
				result = append(result, binary.LittleEndian.Uint32(buffer[i*4:]))
			}
			count -= uint64(chunk)
		}
		return result, nil
	}
	var err error
	if mesh.Indices, err = readIndices(header.Indices); err != nil {
		return nil, err
	}
	if mesh.Edges, err = readIndices(header.Edges); err != nil {
		return nil, err
	}
	return mesh, nil
}

// encodePlace preserves the exact floating-point values of one placement.
func encodePlace(out []byte, place maths.Subspace) {
	values := [13]float64{place.Origin.X, place.Origin.Y, place.Origin.Z, place.Orientation.X.X, place.Orientation.X.Y, place.Orientation.X.Z, place.Orientation.Y.X, place.Orientation.Y.Y, place.Orientation.Y.Z, place.Orientation.Z.X, place.Orientation.Z.Y, place.Orientation.Z.Z, place.Scale}
	for i, value := range values {
		binary.LittleEndian.PutUint64(out[i*8:], math.Float64bits(value))
	}
}

// decodePlace restores one placement from its fixed-width representation.
func decodePlace(in []byte) maths.Subspace {
	var value [13]float64
	for i := range value {
		value[i] = math.Float64frombits(binary.LittleEndian.Uint64(in[i*8:]))
	}
	return maths.Subspace{Origin: maths.V(value[0], value[1], value[2]), Orientation: maths.Orientation{X: maths.V(value[3], value[4], value[5]), Y: maths.V(value[6], value[7], value[8]), Z: maths.V(value[9], value[10], value[11])}, Scale: value[12]}
}
