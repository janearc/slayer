package stagewire

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
)

type cancelAfterRead struct {
	data   *bytes.Reader
	cancel context.CancelFunc
}

func (reader cancelAfterRead) Read(data []byte) (int, error) {
	n, err := reader.data.Read(data)
	reader.cancel()
	return n, err
}

func sample() Snapshot {
	return Snapshot{Version: 1, Notice: "aggregate; no source picking", Scene: placement.DisplayScene{
		Width: 640, Height: 400, Focus: maths.V(0, 0, -1),
		Objects: []placement.DisplayObject{{ID: "scene.world", Name: "staged world", Mesh: &placement.DisplayMesh{
			Vertices: []placement.DisplayVertex{{Position: maths.V(0, 0, 0)}, {Position: maths.V(1, 0, 0)}, {Position: maths.V(0, 1, 0)}},
			Indices:  []uint32{0, 1, 2}, Edges: []uint32{0, 1},
		}}},
	}}
}

func TestSnapshotRoundTripKeepsGeometryCameraAndNotice(t *testing.T) {
	var encoded bytes.Buffer
	if err := Write(&encoded, sample()); err != nil {
		t.Fatal(err)
	}
	got, err := Read(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got.Scene.Width != 640 || got.Scene.Focus.Z != -1 || len(got.Scene.Objects[0].Mesh.Vertices) != 3 || got.Notice != "aggregate; no source picking" {
		t.Fatalf("round trip = %+v", got)
	}
	if got.Scene.Pick(.5, .5) != "" {
		t.Fatal("snapshot claimed source picking")
	}
}

func TestSnapshotRejectsMalformedOrExtraData(t *testing.T) {
	bad := sample()
	bad.Scene.Objects[0].Mesh.Indices[2] = 3
	var sink bytes.Buffer
	if err := Write(&sink, bad); err == nil || sink.Len() != 0 {
		t.Fatal("wrote an invalid index")
	}
	for _, input := range []string{"{}", `{"version":1} {}`, `{"version":1,"unknown":true}`} {
		if _, err := Read(strings.NewReader(input)); err == nil {
			t.Fatal("accepted malformed snapshot")
		}
	}
	raw, err := json.Marshal(bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Read(bytes.NewReader(raw)); err == nil {
		t.Fatal("accepted out-of-range index")
	}
}

func TestSnapshotAcceptsGeometryAboveFormerLimits(t *testing.T) {
	large := sample()
	mesh := large.Scene.Objects[0].Mesh
	mesh.Vertices = make([]placement.DisplayVertex, 1_000_001)
	mesh.Indices = make([]uint32, 6_000_003)
	if err := validate(large); err != nil {
		t.Fatalf("valid large geometry was refused: %v", err)
	}
}

func TestSharedSnapshotRoundTripKeepsOneMeshAndDistinctPlacements(t *testing.T) {
	original := sample()
	original.Version = 2
	second := original.Scene.Objects[0]
	second.ID = "second"
	second.At.Origin = maths.V(4, 0, 0)
	original.Scene.Objects = append(original.Scene.Objects, second)
	var encoded bytes.Buffer
	if err := Write(&encoded, original); err != nil {
		t.Fatal(err)
	}
	bad := bytes.Replace(encoded.Bytes(), []byte(`"Mesh":0`), []byte(`"Mesh":7`), 1)
	if _, err := Read(bytes.NewReader(bad)); err == nil {
		t.Fatal("accepted an invalid shared mesh reference")
	}
	got, err := Read(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Scene.Objects) != 2 || got.Scene.Objects[0].Mesh != got.Scene.Objects[1].Mesh || got.Scene.Objects[1].At.Origin.X != 4 {
		t.Fatalf("shared scene = %+v", got.Scene.Objects)
	}
}

func TestStreamedBatchesKeepSharingAndAuthors(t *testing.T) {
	original := sample()
	original.Version = 3
	mesh := original.Scene.Objects[0].Mesh
	mesh.Vertices[0].Normal = maths.V(.1, .2, .3)
	mesh.Vertices[0].Colour = maths.Light{R: .4, G: .5, B: .6}
	mesh.Notice = "test notice"
	original.Scene.Objects = nil
	original.Scene.Batches = []placement.StagedBatch{
		{Mesh: mesh, Places: []maths.Subspace{{}, {Origin: maths.V(4, 0, 0)}}, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "one"}, Authored: true},
		{Mesh: mesh, Places: []maths.Subspace{{Origin: maths.V(0, 2, 0)}}},
	}
	var data bytes.Buffer
	if err := Write(&data, original); err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(data.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Scene.Batches) != 2 || got.Scene.Batches[0].Mesh != got.Scene.Batches[1].Mesh || len(got.Scene.Batches[0].Places) != 2 || got.Scene.Batches[0].Author.Name != "one" {
		t.Fatalf("streamed batches = %+v", got.Scene.Batches)
	}
	if _, err := Read(bytes.NewReader(append(bytes.Clone(data.Bytes()), []byte("{}")...))); err == nil {
		t.Fatal("accepted trailing data")
	}
}

func TestBinaryBatchesKeepTransformsAcrossChunks(t *testing.T) {
	original := sample()
	original.Version = 5
	mesh := original.Scene.Objects[0].Mesh
	mesh.Vertices[0].Normal = maths.V(.1, .2, .3)
	mesh.Vertices[0].Colour = maths.Light{R: .4, G: .5, B: .6}
	mesh.Notice = "test notice"
	original.Scene.Objects = nil
	original.Scene.Origins = map[string]maths.Vec{"scene.set.things/city": maths.V(3, 4, 5)}
	places := make([]maths.Subspace, placeChunk+1)
	places[placeChunk] = maths.NewSubspace(maths.V(3, 4, 5), maths.About(maths.YAxis, .4), 2)
	original.Scene.Batches = []placement.StagedBatch{{Mesh: mesh, Places: places, Bounds: maths.Box{Min: maths.V(-20, -20, -20), Max: maths.V(20, 20, 20)}, HasBounds: true, Author: thing.AuthorRef{Collection: "scene.set.things", Name: "city"}, Authored: true}}
	var data bytes.Buffer
	if err := Write(&data, original); err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(data.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	batch := got.Scene.Batches[0]
	if len(batch.Places) != len(places) || batch.Places[placeChunk] != places[placeChunk] || !batch.Authored || batch.Author.Name != "city" || !batch.HasBounds || batch.Bounds != original.Scene.Batches[0].Bounds || batch.Mesh.Vertices[0] != mesh.Vertices[0] || batch.Mesh.Notice != mesh.Notice || len(batch.Mesh.Indices) != len(mesh.Indices) || got.Scene.Origins["scene.set.things/city"] != maths.V(3, 4, 5) {
		t.Fatal("binary batch lost its placement or author")
	}
	if _, err := Read(bytes.NewReader(data.Bytes()[:data.Len()-1])); err == nil {
		t.Fatal("accepted a truncated binary snapshot")
	}
	if _, err := Read(bytes.NewReader(append(bytes.Clone(data.Bytes()), 0))); err == nil {
		t.Fatal("accepted trailing binary data")
	}
	wrongMagic := bytes.Clone(data.Bytes())
	copy(wrongMagic, binaryMagic)
	if _, err := Read(bytes.NewReader(wrongMagic)); err == nil {
		t.Fatal("accepted version 5 metadata under the version 4 signature")
	}
	original.Version = 4
	if err := Write(io.Discard, original); err == nil {
		t.Fatal("wrote authored origins under the version 4 contract")
	}
	original.Scene.Origins = nil
	var legacy bytes.Buffer
	if err := Write(&legacy, original); err != nil {
		t.Fatal(err)
	}
	if old, err := Read(&legacy); err != nil || old.Version != 4 || len(old.Scene.Batches) != 1 {
		t.Fatalf("version 4 snapshot no longer reads: version %d, %v", old.Version, err)
	}
	original.Version = 5
	original.Scene.Batches[0].Bounds.Max = maths.V(0, 0, 0)
	if err := Write(io.Discard, original); err == nil {
		t.Fatal("accepted bounds that hide placements")
	}
}

func TestSnapshotDecodeRespondsToCancellation(t *testing.T) {
	original := sample()
	original.Version = 4
	mesh := original.Scene.Objects[0].Mesh
	original.Scene.Objects = nil
	original.Scene.Batches = []placement.StagedBatch{{Mesh: mesh, Places: make([]maths.Subspace, placeChunk+1)}}
	var data bytes.Buffer
	if err := Write(&data, original); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := Read(contextReader{ctx: ctx, source: cancelAfterRead{data: bytes.NewReader(data.Bytes()), cancel: cancel}})
	if err == nil || ctx.Err() != context.Canceled {
		t.Fatalf("decode after cancellation: %v", err)
	}
}
