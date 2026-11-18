package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runAtelier(args ...string) (int, string) {
	var out, errs bytes.Buffer
	code := command().RunWith(args, strings.NewReader(""), &out, &errs)
	return code, out.String() + errs.String()
}

func TestOpenDryRunAndArgumentRefusal(t *testing.T) {
	code, output := runAtelier("open", "--dry-run")
	if code != 0 || !strings.Contains(output, "dry-run opens no window and starts no preview") {
		t.Fatalf("open dry-run returned %d: %s", code, output)
	}
	code, output = runAtelier("open", "one.job.yaml", "two.job.yaml", "--dry-run")
	if code == 0 || !strings.Contains(output, "at most one YAML file") {
		t.Fatalf("open with extra arguments returned %d: %s", code, output)
	}
}

func TestProofDryRunRequiresOutputAndCreatesNoFiles(t *testing.T) {
	code, output := runAtelier("proof", "--dry-run")
	if code == 0 || !strings.Contains(output, "needs --out") {
		t.Fatalf("proof without output returned %d: %s", code, output)
	}
	path := filepath.Join(t.TempDir(), "proof.png")
	code, output = runAtelier("proof", "--out", path, "--dry-run")
	if code != 0 || !strings.Contains(output, "export its own widget tree") {
		t.Fatalf("proof dry-run returned %d: %s", code, output)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("proof dry-run output path stat error = %v; wanted no file", err)
	}
}

func TestGestureProofRequiresRenderedSourceAndKnownGesture(t *testing.T) {
	for _, args := range [][]string{
		{"proof", "--out", "unused.png", "--gesture", "orbit", "--dry-run"},
		{"proof", "--out", "unused.png", "--gesture", "orbit", "--render", "--dry-run"},
		{"proof", "--out", "unused.png", "--gesture", "spin", "--render", "--file", "source.yaml", "--dry-run"},
	} {
		code, output := runAtelier(args...)
		if code == 0 || !strings.Contains(strings.ToLower(output), "gesture") && !strings.Contains(strings.ToLower(output), "render proof needs --file") {
			t.Fatalf("invalid gesture proof %v returned %d: %s", args, code, output)
		}
	}
	path := filepath.Join(t.TempDir(), "gesture.png")
	code, output := runAtelier("proof", "--out", path, "--gesture", "orbit", "--render", "--file", "source.yaml", "--dry-run")
	if code != 0 || !strings.Contains(output, "export its own widget tree") {
		t.Fatalf("valid gesture proof dry-run returned %d: %s", code, output)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("gesture dry-run created output: %v", err)
	}
	code, output = runAtelier("proof", "--out", path, "--gesture", "undo", "--render", "--file", "source.yaml", "--composition", "compose", "--dry-run")
	if code == 0 || !strings.Contains(output, "requires --composition observe") {
		t.Fatalf("undo proof with hidden editor returned %d: %s", code, output)
	}
}

func TestCompositionFlagAcceptsOnlyDeclaredLayouts(t *testing.T) {
	for _, args := range [][]string{
		{"open", "--composition", "split", "--dry-run"},
		{"proof", "--out", "unused.png", "--composition", "split", "--dry-run"},
	} {
		code, output := runAtelier(args...)
		if code == 0 || !strings.Contains(output, "choose play, observe or compose") {
			t.Fatalf("invalid composition %v returned %d: %s", args, code, output)
		}
	}
	for _, composition := range []string{"observe", "compose"} {
		code, output := runAtelier("open", "--composition", composition, "--dry-run")
		if code != 0 || !strings.Contains(output, "dry-run opens no window and starts no preview") {
			t.Fatalf("composition %s dry-run returned %d: %s", composition, code, output)
		}
	}
}

func TestThemeFlagValidatesBeforeOpeningOrWriting(t *testing.T) {
	for _, args := range [][]string{
		{"open", "--theme", "missing-theme", "--dry-run"},
		{"proof", "--out", filepath.Join(t.TempDir(), "theme.png"), "--theme", "missing-theme", "--dry-run"},
	} {
		code, output := runAtelier(args...)
		if code == 0 || !strings.Contains(output, "unknown Chroma colourway") {
			t.Fatalf("unknown theme %v returned %d: %s", args, code, output)
		}
	}
	code, output := runAtelier("open", "--theme", "vaporwave", "--dry-run")
	if code != 0 || !strings.Contains(output, "dry-run opens no window and starts no preview") {
		t.Fatalf("known theme dry-run returned %d: %s", code, output)
	}
}

func TestDeclaredExamples(t *testing.T) {
	if err := command().Examples(os.DirFS(".")); err != nil {
		t.Fatal(err)
	}
}

func Example_open() {
	code, output := runAtelier("open", "--dry-run")
	fmt.Println(code, strings.TrimSpace(output))
	// Output: 0 open Ferruccio on GoGPU Metal; dry-run opens no window and starts no preview
}

func Example_proof() {
	directory, err := os.MkdirTemp("", "atelier-proof-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "proof.png")
	code, output := runAtelier("proof", "--out", path, "--dry-run")
	_, statErr := os.Stat(path)
	fmt.Println(code, strings.Contains(output, "export its own widget tree"), os.IsNotExist(statErr))
	// Output: 0 true true
}

// TestRenderProofRequiresSource refuses an implicit render of a reference image.
func TestRenderProofRequiresSource(t *testing.T) {
	code, output := runAtelier("proof", "--out", "unused.png", "--render", "--dry-run")
	if code == 0 || !strings.Contains(output, "needs --file") {
		t.Fatalf("missing source accepted: %d %s", code, output)
	}
	code, output = runAtelier("proof", "--out", "unused.png", "--file", "existing.yaml", "--render", "--dry-run")
	if code != 0 {
		t.Fatalf("declared source refused: %d %s", code, output)
	}
}

// TestFerruccioIdentity names the active native tool consistently.
func TestFerruccioIdentity(t *testing.T) {
	code, output := runAtelier("--help")
	if code != 0 || !strings.Contains(strings.ToLower(output), "ferruccio") || strings.Contains(strings.ToLower(output), "atelier") {
		t.Fatalf("native tool identity: %d %s", code, output)
	}
	code, output = runAtelier("open", "--dry-run")
	if code != 0 || !strings.Contains(output, "Ferruccio") || strings.Contains(output, "Atelier") {
		t.Fatalf("native open identity: %d %s", code, output)
	}
}

// TestPlayCompositionAndEvidenceDryRunAreDeclaredWithoutFilesystemChanges.
func TestPlayCompositionAndEvidenceDryRunAreDeclaredWithoutFilesystemChanges(t *testing.T) {
	evidence := filepath.Join(t.TempDir(), "evidence-not-created")
	code, output := runAtelier("open", "--composition", "play", "--evidence", evidence, "--dry-run")
	if code != 0 {
		t.Fatalf("play evidence dry-run: %d %s", code, output)
	}
	if _, err := os.Stat(evidence); !os.IsNotExist(err) {
		t.Fatalf("dry-run created evidence parent: %v", err)
	}
	for _, gesture := range []string{"move", "turn"} {
		code, output = runAtelier("proof", "--out", "unused.png", "--composition", "play", "--gesture", gesture, "--render", "--file", "source.yaml", "--evidence", evidence, "--dry-run")
		if code != 0 {
			t.Fatalf("%s proof dry-run: %d %s", gesture, code, output)
		}
		code, output = runAtelier("proof", "--out", "unused.png", "--gesture", gesture, "--dry-run")
		if code == 0 || !strings.Contains(output, "gesture proof needs --render") {
			t.Fatalf("unrendered %s accepted: %d %s", gesture, code, output)
		}
	}
}

// TestExternalQueueFlagsDryRunNeverStartsAClient permits inspecting queue configuration safely.
func TestExternalQueueFlagsDryRunNeverStartsAClient(t *testing.T) {
	root := t.TempDir()
	missingQueue := filepath.Join(root, "queue-not-created")
	for _, args := range [][]string{
		{"open", "--azazel", "/missing/operator/azazel", "--azazel-queue", missingQueue, "--agent", "preview-owner", "--dry-run"},
		{"proof", "--out", filepath.Join(root, "proof.png"), "--render", "--file", "scene.yaml", "--azazel", "/missing/operator/azazel", "--azazel-queue", missingQueue, "--agent", "preview-owner", "--dry-run"},
	} {
		code, output := runAtelier(args...)
		if code != 0 {
			t.Fatalf("external queue dry-run: %d %s", code, output)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("queue dry-run wrote artifacts: %v %v", entries, err)
	}
}

// TestQueueAgentDefaultsOnlyWhenAbsent preserves explicit submission identity.
func TestQueueAgentDefaultsOnlyWhenAbsent(t *testing.T) {
	for _, fixture := range []struct{ said, want string }{{"", "ferruccio"}, {"preview-owner", "preview-owner"}, {" operator ", " operator "}} {
		if got := queueAgent(fixture.said); got != fixture.want {
			t.Fatalf("agent %q became %q, want %q", fixture.said, got, fixture.want)
		}
	}
}

// TestQueueProofDescriptionIsUnwrapped keeps worker admission out of frontend-held lanes.
func TestQueueProofDescriptionIsUnwrapped(t *testing.T) {
	code, output := runAtelier("proof", "--out", "unused.png", "--dry-run")
	if code != 0 || !strings.Contains(output, "unwrapped") || !strings.Contains(output, "preview transport") {
		t.Fatalf("queue proof description: %d %s", code, output)
	}
	code, help := runAtelier("proof", "--help")
	if code != 0 || !strings.Contains(help, "--azazel") {
		t.Fatalf("proof queue help: %d %s", code, help)
	}
	for _, words := range []string{"lane gpu", "lane tests", "shared rendering lanes"} {
		if strings.Contains(output+help, words) {
			t.Fatalf("queue proof suggests frontend lane ownership: %s", words)
		}
	}
}

// TestEvidenceDestinationHasSharedDefaultAndExplicitPrecedence never falls back to temporary output.
func TestEvidenceDestinationHasSharedDefaultAndExplicitPrecedence(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(home, "mesh", "the-vorrh", "renders", "frames")
	for _, fixture := range []struct{ environment, explicit, want string }{
		{"", "", shared},
		{"shared parent with 'quotes' $value\nline", "", "shared parent with 'quotes' $value\nline"},
		{"environment-parent", "explicit-parent", "explicit-parent"},
	} {
		t.Setenv("FERRUCCIO_EVIDENCE", fixture.environment)
		if got := evidenceDirectory(fixture.explicit); got != fixture.want {
			t.Fatalf("evidence %q / %q became %q; want %q", fixture.environment, fixture.explicit, got, fixture.want)
		}
	}
}
