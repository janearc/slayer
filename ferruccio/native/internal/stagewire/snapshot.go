// Package stagewire carries a detached staged display snapshot.
package stagewire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/session/placement"
)

// Snapshot carries display meshes and authored identities; private native hit
// targets do not cross the process boundary. Version 2 and later transmit
// shared meshes separately from placements.
type Snapshot struct {
	Version    int                      `json:"version"`
	Scene      placement.DisplayScene   `json:"scene"`
	Notice     string                   `json:"notice"`
	Meshes     []*placement.DisplayMesh `json:"meshes,omitempty"`
	Places     []MeshPlace              `json:"places,omitempty"`
	MeshCount  int                      `json:"mesh_count,omitempty"`
	BatchCount int                      `json:"batch_count,omitempty"`
}

// MeshPlace references a single shared mesh and its staged transform.
type MeshPlace struct {
	ID, Name string
	Mesh     int
	At       maths.Subspace
}

// Write validates a snapshot before writing one JSON document.
func Write(out io.Writer, snapshot Snapshot) error {
	if len(snapshot.Meshes) != 0 || len(snapshot.Places) != 0 || snapshot.MeshCount != 0 || snapshot.BatchCount != 0 {
		return fmt.Errorf("staged snapshot has unexpected encoded fields")
	}
	if err := validate(snapshot); err != nil {
		return err
	}
	if snapshot.Version == 4 || snapshot.Version == 5 {
		return writeBinaryBatches(out, snapshot)
	}
	if snapshot.Version == 3 {
		return writeBatches(out, snapshot)
	}
	if snapshot.Version == 2 {
		known := make(map[*placement.DisplayMesh]int)
		for _, object := range snapshot.Scene.Objects {
			index, found := known[object.Mesh]
			if !found {
				index = len(snapshot.Meshes)
				known[object.Mesh] = index
				snapshot.Meshes = append(snapshot.Meshes, object.Mesh)
			}
			snapshot.Places = append(snapshot.Places, MeshPlace{ID: object.ID, Name: object.Name, Mesh: index, At: object.At})
		}
		snapshot.Scene.Objects = nil
	}
	return json.NewEncoder(out).Encode(snapshot)
}

// Read rejects incomplete, extra, and structurally invalid output.
func Read(in io.Reader) (Snapshot, error) {
	var snapshot Snapshot
	buffered := bufio.NewReader(in)
	if magic, err := buffered.Peek(len(binaryMagic)); err == nil && (string(magic) == binaryMagic || string(magic) == binaryMagicWithOrigins) {
		return readBinaryBatches(buffered)
	}
	decoder := json.NewDecoder(buffered)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Version == 3 {
		return readBatches(decoder, snapshot)
	}
	if snapshot.MeshCount != 0 || snapshot.BatchCount != 0 {
		return Snapshot{}, fmt.Errorf("unexpected batch counts in older staged snapshot")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Snapshot{}, fmt.Errorf("staged snapshot has trailing data")
	}
	if snapshot.Version == 2 {
		if len(snapshot.Scene.Objects) != 0 || len(snapshot.Places) == 0 || len(snapshot.Meshes) == 0 {
			return Snapshot{}, fmt.Errorf("invalid shared staged snapshot")
		}
		used := make([]bool, len(snapshot.Meshes))
		for _, place := range snapshot.Places {
			if place.Mesh < 0 || place.Mesh >= len(snapshot.Meshes) {
				return Snapshot{}, fmt.Errorf("staged mesh reference is out of range")
			}
			used[place.Mesh] = true
			snapshot.Scene.Objects = append(snapshot.Scene.Objects, placement.DisplayObject{ID: place.ID, Name: place.Name, Mesh: snapshot.Meshes[place.Mesh], At: place.At})
		}
		for _, present := range used {
			if !present {
				return Snapshot{}, fmt.Errorf("staged snapshot contains an unreferenced mesh")
			}
		}
		snapshot.Meshes, snapshot.Places = nil, nil
	} else if len(snapshot.Meshes) != 0 || len(snapshot.Places) != 0 {
		return Snapshot{}, fmt.Errorf("unexpected shared fields in version 1 snapshot")
	}
	if err := validate(snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// finite identifies a number the display can safely consume.
func finite(number float64) bool { return !math.IsNaN(number) && !math.IsInf(number, 0) }

// vector checks all three coordinates of a display vector.
func vector(value maths.Vec) bool { return finite(value.X) && finite(value.Y) && finite(value.Z) }

// validate refuses geometry that could make the display draw invalid indices.
func validate(snapshot Snapshot) error {
	scene := snapshot.Scene
	if snapshot.Version < 5 && len(scene.Origins) != 0 {
		return fmt.Errorf("staged authored origins require version 5")
	}
	for id, origin := range scene.Origins {
		if id == "" || !vector(origin) {
			return fmt.Errorf("invalid staged authored origin")
		}
	}
	if (snapshot.Version != 1 && snapshot.Version != 2 && snapshot.Version != 3 && snapshot.Version != 4 && snapshot.Version != 5) || snapshot.Notice == "" || scene.Width <= 0 || scene.Height <= 0 ||
		!shape.ValidPlacement(scene.Eye) || !vector(scene.Focus) || !finite(scene.Lens.Wide) || !finite(scene.Lens.Pixel) ||
		scene.Lens.Wide < 0 || scene.Lens.Wide >= math.Pi || scene.Lens.Pixel < 0 || (len(scene.Objects) == 0 && len(scene.Batches) == 0) {
		return fmt.Errorf("invalid staged snapshot camera or metadata")
	}
	if snapshot.Version == 1 && (len(scene.Objects) != 1 || scene.Objects[0].ID != "scene.world") {
		return fmt.Errorf("version 1 snapshot requires one aggregate world")
	}
	if snapshot.Version != 3 && snapshot.Version != 4 && snapshot.Version != 5 && len(scene.Batches) != 0 {
		return fmt.Errorf("older staged snapshot has unexpected batches")
	}
	if (snapshot.Version == 3 || snapshot.Version == 4 || snapshot.Version == 5) && (len(scene.Objects) != 0 || len(scene.Batches) == 0) {
		return fmt.Errorf("staged snapshot requires batches")
	}
	seen := make(map[*placement.DisplayMesh]bool)
	ids := make(map[string]bool)
	for _, object := range scene.Objects {
		if object.ID == "" || ids[object.ID] || object.Mesh == nil || !shape.ValidPlacement(object.At) {
			return fmt.Errorf("invalid staged object")
		}
		ids[object.ID] = true
		if seen[object.Mesh] {
			continue
		}
		seen[object.Mesh] = true
		if err := validateMesh(object.Mesh); err != nil {
			return err
		}
	}
	for _, batch := range scene.Batches {
		if batch.Mesh == nil || len(batch.Places) == 0 {
			return fmt.Errorf("invalid staged batch")
		}
		low, high, boundsOK := batch.Mesh.Bounds()
		if batch.HasBounds && (!boundsOK || !vector(batch.Bounds.Min) || !vector(batch.Bounds.Max) || batch.Bounds.Min.X > batch.Bounds.Max.X || batch.Bounds.Min.Y > batch.Bounds.Max.Y || batch.Bounds.Min.Z > batch.Bounds.Max.Z) {
			return fmt.Errorf("invalid staged batch bounds")
		}
		centre, radius := low.Add(high).Scale(.5), high.Sub(low).Len()*.5
		for _, place := range batch.Places {
			if !shape.ValidPlacement(place) {
				return fmt.Errorf("invalid staged batch placement")
			}
			if batch.HasBounds {
				point, scale := place.Out(centre), place.Scale
				if scale == 0 {
					scale = 1
				}
				reach := radius * scale
				if point.X-reach < batch.Bounds.Min.X || point.Y-reach < batch.Bounds.Min.Y || point.Z-reach < batch.Bounds.Min.Z || point.X+reach > batch.Bounds.Max.X || point.Y+reach > batch.Bounds.Max.Y || point.Z+reach > batch.Bounds.Max.Z {
					return fmt.Errorf("staged batch bounds omit a placement")
				}
			}
		}
		if !seen[batch.Mesh] {
			seen[batch.Mesh] = true
			if err := validateMesh(batch.Mesh); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateMesh checks finite vertices and indices before GPU upload.
func validateMesh(mesh *placement.DisplayMesh) error {
	if len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 || len(mesh.Indices)%3 != 0 || len(mesh.Edges)%2 != 0 {
		return fmt.Errorf("invalid staged snapshot mesh size")
	}
	for _, vertex := range mesh.Vertices {
		if !vector(vertex.Position) || !vector(vertex.Normal) || !finite(vertex.Colour.R) || !finite(vertex.Colour.G) || !finite(vertex.Colour.B) {
			return fmt.Errorf("invalid staged snapshot vertex")
		}
	}
	for _, indices := range [][]uint32{mesh.Indices, mesh.Edges} {
		for _, index := range indices {
			if uint64(index) >= uint64(len(mesh.Vertices)) {
				return fmt.Errorf("staged snapshot index is outside its mesh")
			}
		}
	}
	return nil
}
