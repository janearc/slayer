package gpu

import (
	"os"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// This is a deliberately repeatable CPU-only encode experiment, not a GPU
// draw or a city. It is off during normal tests and keeps a single copy of
// the test world alive in both versions.
func TestLowRetentionMemoryExperiment(t *testing.T) {
	if os.Getenv("SLAYER_LOW_RETENTION_MEASURE") != "1" {
		t.Skip("measurement not requested")
	}
	world := &thing.World{}
	const meshes = 128
	const faces = 2048
	for i := range meshes {
		solid := shapeFan(t, float64(i)/100, faces, false)
		local := thing.Still(maths.Local())
		world.Add(thing.Thing{Shape: solid, Motion: local})
	}
	keeper := newKeptTrees()
	made, err := encodeKept(world, nil, maths.Local(), nil, time.Duration(0), 64, 64, keeper)
	if err != nil {
		t.Fatal(err)
	}
	if made.counts.triangles != meshes*faces {
		t.Fatalf("faces = %d", made.counts.triangles)
	}
	t.Logf("faces=%d packed=%d bytes", made.counts.triangles, len(made.triangles.bytes))
}
