// Package stagewire carries a detached staged display snapshot.
package stagewire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"

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
	Frame      *FilmFrame               `json:"frame,omitempty"`
	Meshes     []*placement.DisplayMesh `json:"meshes,omitempty"`
	Places     []MeshPlace              `json:"places,omitempty"`
	MeshCount  int                      `json:"mesh_count,omitempty"`
	BatchCount int                      `json:"batch_count,omitempty"`
}

// FilmFrame identifies one inspection frame on the producer's film clock.
// Number is one-based. Still snapshots have no FilmFrame.
type FilmFrame struct {
	Number, Count int
	Time          time.Duration
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
	return readSized(in, -1)
}

// readSized uses a known whole-stream size only as an upper bound for
// allocations; -1 preserves streaming behavior when the size is unknown.
func readSized(in io.Reader, size int64) (Snapshot, error) {
	var snapshot Snapshot
	buffered := bufio.NewReader(in)
	if magic, err := buffered.Peek(len(binaryMagic)); err == nil && (string(magic) == binaryMagic || string(magic) == binaryMagicWithOrigins) {
		return readBinaryBatches(buffered, size)
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
	if frame := snapshot.Frame; frame != nil && (snapshot.Version < 5 || frame.Number < 1 || frame.Number > frame.Count) {
		return fmt.Errorf("invalid staged film frame")
	}
	if snapshot.Version < 5 && len(scene.Origins) != 0 {
		return fmt.Errorf("staged authored origins require version 5")
	}
	if err := validateLights(snapshot.Version, scene.Lights); err != nil {
		return err
	}
	if len(scene.VolumeBounds) > placement.VolumeBoundLimit || scene.OmittedVolumeBounds < 0 ||
		(snapshot.Version < 5 && (len(scene.VolumeBounds) != 0 || scene.OmittedVolumeBounds != 0)) {
		return fmt.Errorf("unexpected staged volume bounds")
	}
	for _, bound := range scene.VolumeBounds {
		if !bound.Valid() {
			return fmt.Errorf("invalid staged volume bound")
		}
	}
	if len(scene.GiantBoundaries) > placement.GiantBoundaryLimit || (snapshot.Version < 5 && len(scene.GiantBoundaries) > 0) {
		return fmt.Errorf("unexpected staged gas giant boundaries")
	}
	for _, boundary := range scene.GiantBoundaries {
		if !boundary.Valid() {
			return fmt.Errorf("invalid staged gas giant boundary")
		}
	}
	if len(scene.GlobeOutlines) > placement.GlobeOutlineLimit || (snapshot.Version < 5 && len(scene.GlobeOutlines) > 0) {
		return fmt.Errorf("unexpected staged globe outlines")
	}
	for _, outline := range scene.GlobeOutlines {
		if !outline.Valid() {
			return fmt.Errorf("invalid staged globe outline")
		}
	}
	if scene.SkyGradient != nil && (snapshot.Version < 5 || !scene.SkyGradient.Valid()) {
		return fmt.Errorf("invalid staged sky gradient")
	}
	for id, origin := range scene.Origins {
		if id == "" || !vector(origin) {
			return fmt.Errorf("invalid staged authored origin")
		}
	}
	if (snapshot.Version != 1 && snapshot.Version != 2 && snapshot.Version != 3 && snapshot.Version != 4 && snapshot.Version != 5) || snapshot.Notice == "" || scene.Width <= 0 || scene.Height <= 0 ||
		!shape.ValidPlacement(scene.Eye) || !vector(scene.Focus) || !finite(scene.Lens.Wide) || !finite(scene.Lens.Pixel) ||
		scene.Lens.Wide < 0 || scene.Lens.Wide >= math.Pi || scene.Lens.Pixel < 0 || (snapshot.Version < 5 && len(scene.Objects) == 0 && len(scene.Batches) == 0) {
		return fmt.Errorf("invalid staged snapshot camera or metadata")
	}
	if snapshot.Version == 1 && (len(scene.Objects) != 1 || scene.Objects[0].ID != "scene.world") {
		return fmt.Errorf("version 1 snapshot requires one aggregate world")
	}
	if snapshot.Version != 3 && snapshot.Version != 4 && snapshot.Version != 5 && len(scene.Batches) != 0 {
		return fmt.Errorf("older staged snapshot has unexpected batches")
	}
	if snapshot.Version >= 3 && len(scene.Objects) != 0 {
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

func validateLights(version int, lights []placement.DisplayLight) error {
	if len(lights) > placement.LightLimit || (version < 5 && len(lights) != 0) {
		return fmt.Errorf("unexpected staged lights")
	}
	seen := make(map[int]bool, len(lights))
	for _, light := range lights {
		if !light.Valid() || seen[light.Ordinal] {
			return fmt.Errorf("invalid staged light")
		}
		seen[light.Ordinal] = true
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
