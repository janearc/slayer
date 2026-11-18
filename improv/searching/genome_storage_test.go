package searching

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"os"
	"testing"
)

// These witnesses pin the complete authored point/face arrays before changing
// storage ownership. No geometry is simplified, merged or reseeded.
func TestGenomeStorageKeepsAuthoredGeometry(t *testing.T) {
	var got []string
	for id := 0; id < 32; id++ {
		stage := &heardStage{world: thing.NewWorld()}
		being := crowdBeing{identity: identityOf(id)}
		if err := stage.buildIndividual(&being, false); err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		for _, part := range being.parts {
			var points []maths.Vec
			var faces [][3]int
			switch skin := part.part.Shape.(type) {
			case *shape.Mesh:
				points, faces = skin.Vertices, skin.Faces
			case *mesh.Mesh:
				points, faces = skin.Points(), skin.Faces()
			default:
				continue
			}
			raw, err := json.Marshal(struct {
				Points []maths.Vec
				Faces  [][3]int
			}{points, faces})
			if err != nil {
				t.Fatal(err)
			}
			hash.Write(raw)
		}
		got = append(got, hex.EncodeToString(hash.Sum(nil)))
	}
	if path := os.Getenv("GENOME_STORAGE_REFERENCE_OUT"); path != "" {
		raw, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile("testdata/genome-storage-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatal("missing geometry witnesses")
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("individual %d geometry changed: %s vs %s", i, got[i], want[i])
		}
	}
}
