//go:build darwin

package main

import (
	"bytes"
	"context"
	stdflag "flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunReleasesActualFuelSessionOwnersOnSuccessAndFailure(t *testing.T) {
	if path := os.Getenv("SLAYER_BUFFER_LIFETIME_LOG"); path != "" {
		t.Cleanup(func() {
			for _, suffix := range []string{"", ".buffers", ".callers"} {
				if raw, err := os.ReadFile(path + suffix); err == nil {
					t.Logf("actual native observation%s:\n%s", suffix, raw)
				}
			}
		})
		given := fuelOnFixture(t)
		raw, err := os.ReadFile(given.set)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Replace(string(raw), "width: 24", "width: 8", 1)
		text = strings.Replace(text, "height: 16", "height: 6", 1)
		mode := os.Getenv("SLAYER_BUFFER_LIFETIME_MODE")
		if mode == "staging-failure" {
			text = strings.Replace(text, "object: the stone", "object: the tree", 1)
			text = strings.Replace(text, "standoff_z: 2", "standoff_z: 0", 1)
		}
		if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(t.TempDir(), "picture.png")
		if mode == "save-failure" {
			parent := filepath.Join(t.TempDir(), "regular-file")
			if err := os.WriteFile(parent, []byte("owned output obstacle"), 0600); err != nil {
				t.Fatal(err)
			}
			out = filepath.Join(parent, "picture.png")
		}
		var said, wrong bytes.Buffer
		code := run([]string{"-set", given.set, "-out", out}, &said, &wrong)
		if mode == "success" {
			if code != 0 || wrong.Len() != 0 {
				t.Fatalf("real fuel render failed: code%d, %s", code, wrong.String())
			}
			if info, err := os.Stat(out); err != nil || info.Size() == 0 {
				t.Fatalf("real render produced no PNG: %v", err)
			}
		} else if mode == "save-failure" {
			if code != 1 || !strings.Contains(wrong.String(), filepath.Dir(out)) {
				t.Fatalf("real output obstacle lost its refusal: code%d, %q", code, wrong.String())
			}
		} else if mode == "staging-failure" {
			if code != 1 || !strings.Contains(wrong.String(), "no heading") || said.Len() != 0 {
				t.Fatalf("post-allocation camera refusal was lost: code%d, %q, progress %q", code, wrong.String(), said.String())
			}
		} else {
			t.Fatalf("unknown observer mode %q", mode)
		}
		// Observe the real retained session owner, without collecting or closing
		// the model on its behalf. Buffers may also have driver-owned references.
		allocated, destroyed := readFuelBufferLifetimes(t, path+".buffers")
		if len(allocated) < 2 {
			t.Fatal("fixture did not allocate the two actual fuel state buffers")
		}
		t.Logf("actual buffers: %d allocated, %d destroyed; checking session ownership separately", len(allocated), len(destroyed))
		deadline := time.Now().Add(2 * time.Second)
		for {
			created, freed := readFuelBufferLifetimes(t, path)
			if len(created) >= 1 && len(created) == len(freed) {
				for id := range created {
					if !freed[id] {
						t.Fatalf("native session owner %d was not released", id)
					}
				}
				t.Logf("actual %s path released all %d observed fuel session owners", mode, len(created))
				return
			}
			if !time.Now().Before(deadline) {
				t.Fatalf("real run retained fuel session owners: created %v, freed %v", created, freed)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	dir := t.TempDir()
	library := filepath.Join(dir, "observer.dylib")
	compile := exec.Command("cc", "-dynamiclib", "-fobjc-arc",
		"-framework", "Foundation", "-framework", "Metal",
		"testdata/buffer_lifetime_observer.m", "-o", library)
	if out, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile native lifetime observer: %v\n%s", err, out)
	}
	for _, mode := range []string{"success", "save-failure", "staging-failure"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestRunReleasesActualFuelSessionOwnersOnSuccessAndFailure$", "-test.v"}
			if flag := stdflag.Lookup("test.gocoverdir"); testing.CoverMode() != "" &&
				flag != nil && flag.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+flag.Value.String())
			}
			child := exec.CommandContext(ctx, os.Args[0], args...)
			child.Env = append(os.Environ(), "DYLD_INSERT_LIBRARIES="+library,
				"SLAYER_BUFFER_LIFETIME_LOG="+filepath.Join(dir, mode+".log"),
				"SLAYER_BUFFER_LIFETIME_MODE="+mode, "GOMAXPROCS=1", "GOMEMLIMIT=512MiB")
			out, err := child.CombinedOutput()
			t.Logf("isolated real buffer lifetimes:\n%s", out)
			if err != nil {
				t.Fatalf("native buffer lifetime subprocess: %v (deadline %v)", err, ctx.Err())
			}
		})
	}
}

func readFuelBufferLifetimes(t *testing.T, path string) (map[int]bool, map[int]bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing actual native allocation log: %v", err)
	}
	created, freed := map[int]bool{}, map[int]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("invalid native lifetime record %q", line)
		}
		id, err := strconv.Atoi(fields[1])
		if err != nil || id < 1 {
			t.Fatalf("invalid native lifetime identifier %q", line)
		}
		switch fields[0] {
		case "new":
			if created[id] {
				t.Fatal("duplicate native allocation identifier", id)
			}
			created[id] = true
		case "free":
			if !created[id] || freed[id] {
				t.Fatal("unmatched native release", id)
			}
			freed[id] = true
		default:
			t.Fatalf("unsupported observer event %q", line)
		}
	}
	return created, freed
}
