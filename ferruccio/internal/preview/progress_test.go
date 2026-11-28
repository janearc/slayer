package preview

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttemptProgressTracksCompleteLinesOfTheCurrentAttempt(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "00000000000000000042.0001.log")
	second := filepath.Join(root, "00000000000000000042.0002.log")
	if err := os.WriteFile(first, []byte("progress rows=90/90 taken=0 elapsed=5m\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("staging at=things 50/394 elapsed=15s\nprogress rows=0/0 taken=0 elapsed=15s\nprogress rows=13/90 taken=0 elapsed=4m\nprogress rows=30/90"), 0600); err != nil {
		t.Fatal(err)
	}
	got := readAttemptProgress(root, 42, 2)
	if got != (Progress{Phase: "Rendering", Done: 13, Total: 90}) {
		t.Fatalf("current attempt progress = %+v", got)
	}
	if got := readAttemptProgress(root, 42, 3); got != (Progress{}) {
		t.Fatalf("missing attempt borrowed previous progress: %+v", got)
	}
	link := filepath.Join(root, "00000000000000000042.0003.log")
	if err := os.Symlink(first, link); err != nil {
		t.Fatal(err)
	}
	if got := readAttemptProgress(root, 42, 3); got != (Progress{}) {
		t.Fatalf("a linked log supplied progress: %+v", got)
	}
}

func TestProgressReportsPhaseWithoutInventingAResult(t *testing.T) {
	for _, sample := range []struct {
		line string
		want Progress
	}{
		{"staging at=things 50/394 elapsed=15s", Progress{Phase: "Staging · things", Done: 50, Total: 394}},
		{"progress rows=0/0 taken=0 elapsed=15s", Progress{Phase: "Rendering"}},
		{"progress rows=91/90 taken=0 elapsed=15s", Progress{Phase: "Rendering"}},
		{"staging at=atmosphere elapsed=3s", Progress{Phase: "Staging · atmosphere"}},
	} {
		got, ok := parseProgressLine(sample.line)
		if !ok || got != sample.want {
			t.Errorf("%q = %+v, %v; want %+v", sample.line, got, ok, sample.want)
		}
	}
}
