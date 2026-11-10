//go:build darwin

package main

import (
	"bytes"
	"context"
	stdflag "flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Lens exposures discard GPU profiles. Observe the real Metal compilation
// instead, so a software-rendered picture cannot mask lost native routing.
func TestRigCompilesTheRequestedActualTraversalBackend(t *testing.T) {
	if path := os.Getenv("SLAYER_TRAVERSAL_OBSERVER_LOG"); path != "" {
		mode := os.Getenv("SLAYER_TRAVERSAL_OBSERVER_MODE")
		scene := onFixtureRigged(t, "0.5", false)
		raw, err := os.ReadFile(scene)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Replace(string(raw), "scene:", "scene:\n  renderer: "+mode, 1)
		text = strings.Replace(text, "width: 24", "width: 4", 1)
		if err := os.WriteFile(scene, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		var said, wrong bytes.Buffer
		code := run([]string{"-set", scene, "-out", filepath.Join(t.TempDir(), "rig.png")}, &said, &wrong)
		if code != 0 || wrong.Len() != 0 {
			t.Fatalf("real rig render failed: code%d, %s", code, wrong.String())
		}
		observed, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("missing actual compiler observations: %v", err)
		}
		want, known := map[string]string{
			"gpu": "0 0", "gpu-native-shadows": "0 1", "gpu-native": "1 1",
		}[mode]
		if !known || len(bytes.TrimSpace(observed)) == 0 {
			t.Fatal("observer lost its real traversal mode or compilation")
		}
		lines := strings.Split(strings.TrimSpace(string(observed)), "\n")
		for _, line := range lines {
			if line != want {
				t.Fatalf("requested %s, actual Metal compiler received primary/shadows %q, want %q", mode, line, want)
			}
		}
		t.Logf("actual %s rig compiled %d trace libraries with primary/shadows=%s", mode, len(lines), want)
		return
	}

	dir := t.TempDir()
	library := filepath.Join(dir, "observer.dylib")
	compile := exec.Command("cc", "-dynamiclib", "-fobjc-arc",
		"-framework", "Foundation", "-framework", "Metal",
		"testdata/traversal_observer.m", "-o", library)
	if out, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile actual traversal observer: %v\n%s", err, out)
	}
	for _, mode := range []string{"gpu", "gpu-native-shadows", "gpu-native"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestRigCompilesTheRequestedActualTraversalBackend$", "-test.v"}
			if flag := stdflag.Lookup("test.gocoverdir"); testing.CoverMode() != "" &&
				flag != nil && flag.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+flag.Value.String())
			}
			child := exec.CommandContext(ctx, os.Args[0], args...)
			child.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+library,
				"SLAYER_TRAVERSAL_OBSERVER_LOG="+filepath.Join(dir, mode+".log"),
				"SLAYER_TRAVERSAL_OBSERVER_MODE="+mode, "GOMAXPROCS=1", "GOMEMLIMIT=512MiB")
			out, err := child.CombinedOutput()
			t.Logf("isolated actual compiler observations:\n%s", out)
			if err != nil {
				t.Fatalf("native traversal observer subprocess: %v (deadline %v)", err, ctx.Err())
			}
		})
	}
}
