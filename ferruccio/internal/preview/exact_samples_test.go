package preview

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFullExactSamplesReachTheProducer(t *testing.T) {
	plan, err := Prepare(Request{
		Path: filepath.Join(t.TempDir(), "scene.yaml"), Text: string(firstPicture(t)),
		Profile: Profile{Full: true, Samples: 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	args, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Join(args, " ")
	if !strings.Contains(line, "--spp 7") || !strings.Contains(line, "--renderer cpu") || strings.Contains(line, "--rays") || plan.lane() != "tests" {
		t.Fatalf("exact request = %q, lane %q", line, plan.lane())
	}
}

func TestFullExactSamplesReachGPU(t *testing.T) {
	plan, err := Prepare(Request{
		Path: filepath.Join(t.TempDir(), "scene.yaml"), Text: string(firstPicture(t)),
		Profile: Profile{Full: true, Samples: 7, GPU: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	args, err := plan.arguments(Tools{Slayer: "slayer"})
	if err != nil || !strings.Contains(strings.Join(args, " "), "--spp 7") || !strings.Contains(strings.Join(args, " "), "--renderer gpu") || plan.lane() != "gpu" {
		t.Fatalf("GPU exact request = %v, lane %q, %v", args, plan.lane(), err)
	}
}
