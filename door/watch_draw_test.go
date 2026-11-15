//go:build draw

package door

import (
	"context"
	"testing"

	"github.com/janearc/slayer/hillside"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/live"
)

// the stager opens a session that draws the frame at the size asked,
// and says the job's shape, times and exposure; a set says its cameras.
func TestSessionsStageAJob(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "a.yaml", stillJob(t, ""))
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := sessions{stamp: hillside.Stamp{}}.Stage(
		context.Background(), wanted)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Close()
	size, _ := live.NewSize(8, 6, 1)
	drawn, err := stage.Frame(context.Background(), nothing, size)
	if err != nil || drawn.Light().W != 8 || drawn.Light().H != 6 {
		t.Fatalf("a frame of 8 by 6 drew %v", err)
	}
	width, height := stage.Shape()
	if width != number(t, "width") || height != number(t, "height") ||
		len(stage.Times()) != one || stage.Exposure() <= nothing {
		t.Errorf("the stage says %d by %d, %d times, exposure %v", width,
			height, len(stage.Times()), stage.Exposure())
	}
	if stage.Cameras() == nil {
		t.Error("a set's stage says no cameras")
	}
	_ = stage.Tone()
}
