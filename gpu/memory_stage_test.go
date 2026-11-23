package gpu

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMemoryStageIsOptIn(t *testing.T) {
	var out bytes.Buffer
	reportMemoryStage(&out, false, "before-encode")
	if out.Len() != 0 {
		t.Fatal("disabled instrumentation wrote a line")
	}
	reportMemoryStage(&out, true, "before-encode")
	if !strings.Contains(out.String(), `"stage":"before-encode"`) {
		t.Fatalf("enabled instrumentation omitted the stage: %s", out.String())
	}
}

func TestMemoryStageReadsTheEnvironment(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	previous := os.Stderr
	os.Stderr = writer
	defer func() { os.Stderr = previous }()
	t.Setenv("SLAYER_MEMORY_STAGES", "")
	memoryStage("disabled")
	t.Setenv("SLAYER_MEMORY_STAGES", "1")
	memoryStage("enabled")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte(`"stage":"disabled"`)) ||
		!bytes.Contains(got, []byte(`"stage":"enabled"`)) {
		t.Fatalf("environment control failed: %s", got)
	}
}

func TestMemoryStagesObserveScenePacking(t *testing.T) {
	if os.Getenv("SLAYER_TEST_PACK_TRACE") == "1" {
		if err := newScene().packPool(); err != nil {
			t.Fatal(err)
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMemoryStagesObserveScenePacking$")
	cmd.Env = append(os.Environ(), "SLAYER_TEST_PACK_TRACE=1", "SLAYER_MEMORY_STAGES=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("packing diagnostic failed: %v\n%s", err, out)
	}
	for _, stage := range []string{"pool-before", "pool-packed", "pool-retained"} {
		if !bytes.Contains(out, []byte(`"stage":"`+stage+`"`)) {
			t.Errorf("packing did not report %s: %s", stage, out)
		}
	}
}
