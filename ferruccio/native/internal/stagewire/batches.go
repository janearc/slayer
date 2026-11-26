package stagewire

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
)

type wireBatch struct {
	Mesh      int
	Places    []maths.Subspace
	Bounds    maths.Box
	HasBounds bool
	Author    thing.AuthorRef
	Authored  bool
}

// writeBatches writes one mesh or placement group at a time instead of
// allocating one JSON document for the whole scene.
func writeBatches(out io.Writer, snapshot Snapshot) error {
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
	encoder := json.NewEncoder(out)
	if err := encoder.Encode(snapshot); err != nil {
		return err
	}
	for _, mesh := range meshes {
		if err := encoder.Encode(mesh); err != nil {
			return err
		}
	}
	for _, batch := range batches {
		row := wireBatch{Mesh: known[batch.Mesh], Places: batch.Places, Bounds: batch.Bounds, HasBounds: batch.HasBounds, Author: batch.Author, Authored: batch.Authored}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	return nil
}

// readBatches restores shared pointers and checks the complete JSON stream.
func readBatches(decoder *json.Decoder, snapshot Snapshot) (Snapshot, error) {
	if snapshot.MeshCount <= 0 || snapshot.BatchCount <= 0 || len(snapshot.Scene.Objects) != 0 || len(snapshot.Scene.Batches) != 0 || len(snapshot.Meshes) != 0 || len(snapshot.Places) != 0 {
		return Snapshot{}, fmt.Errorf("invalid staged batch header")
	}
	var meshes []*placement.DisplayMesh
	for i := 0; i < snapshot.MeshCount; i++ {
		var mesh *placement.DisplayMesh
		if err := decoder.Decode(&mesh); err != nil {
			return Snapshot{}, fmt.Errorf("read staged mesh %d: %w", i, err)
		}
		meshes = append(meshes, mesh)
	}
	used := make([]bool, len(meshes))
	for i := 0; i < snapshot.BatchCount; i++ {
		var batch wireBatch
		if err := decoder.Decode(&batch); err != nil {
			return Snapshot{}, fmt.Errorf("read staged batch %d: %w", i, err)
		}
		if batch.Mesh < 0 || batch.Mesh >= len(meshes) {
			return Snapshot{}, fmt.Errorf("staged batch mesh reference is out of range")
		}
		used[batch.Mesh] = true
		snapshot.Scene.Batches = append(snapshot.Scene.Batches, placement.StagedBatch{Mesh: meshes[batch.Mesh], Places: batch.Places, Bounds: batch.Bounds, HasBounds: batch.HasBounds, Author: batch.Author, Authored: batch.Authored})
	}
	for _, present := range used {
		if !present {
			return Snapshot{}, fmt.Errorf("staged snapshot contains an unreferenced mesh")
		}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Snapshot{}, fmt.Errorf("staged snapshot has trailing data")
	}
	snapshot.MeshCount, snapshot.BatchCount = 0, 0
	if err := validate(snapshot); err != nil {
		return Snapshot{}, err
	}
	snapshot.Scene.IndexPicking()
	return snapshot, nil
}
