package commandtest_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/internal/commandtest"
)

func TestBuildProducesRunnableCommandsAndReportsCompilerErrors(t *testing.T) {
	root := t.TempDir()
	for name, text := range map[string]string{"go.mod": "module example.test/commandfixture\n\ngo 1.26.10\n", "main.go": "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"command fixture\")}\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"", "set"} {
		binary := filepath.Join(t.TempDir(), "fixture")
		if err := commandtest.Build(root, ".", binary, mode); err != nil {
			t.Fatal(err)
		}
		run := exec.Command(binary)
		// This synthetic external module is an oracle for the builder, not Slayer.
		run.Env = append(os.Environ(), "GOCOVERDIR="+t.TempDir())
		out, err := run.CombinedOutput()
		if err != nil || string(out) != "command fixture\n" {
			t.Fatalf("%q %v", out, err)
		}
	}
	if err := commandtest.Build(root, "./does-not-exist", filepath.Join(root, "unused"), ""); err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("lost compiler refusal: %v", err)
	}
}
