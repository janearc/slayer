package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// publishRoot is a renders root with one folder holding one picture.
func publishRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "renders")
	folder := filepath.Join(root, "frames", "2026-11-15-studio-one")
	for _, dir := range []string{folder, filepath.Join(root, "animations")} {
		if err := os.MkdirAll(dir, openFolder); err != nil {
			t.Fatal(err)
		}
	}
	picture := filepath.Join(folder, "one.png")
	if err := os.WriteFile(picture, []byte("picture"), openFile); err != nil {
		t.Fatal(err)
	}
	return root
}

// The modes a test's renders tree is made with.
const (
	openFolder = 0o755 // defined: a folder anyone may read
	openFile   = 0o644 // defined: a file anyone may read
)

// publishing runs the publish verb and returns its exit, output and
// complaint.
func publishing(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, wrong bytes.Buffer
	code := queueTool(t.Context()).Run(append([]string{"publish"}, args...),
		&out, &wrong)
	return code, out.String(), wrong.String()
}

// TestPublishWritesOneFolderOnce writes the folder's page, the gallery
// and, with its pictures, barkeep's line.
func TestPublishWritesOneFolderOnce(t *testing.T) {
	root := publishRoot(t)
	code, out, wrong := publishing(t, "--renders", root, "--folder",
		"frames/2026-11-15-studio-one", "--pictures", "1")
	if code != 0 {
		t.Fatal(code, out, wrong)
	}
	page, err := os.ReadFile(filepath.Join(root,
		"frames/2026-11-15-studio-one/index.html"))
	line, wrongLine := os.ReadFile(filepath.Join(root, "render-progress.txt"))
	if err != nil || wrongLine != nil || !bytes.Contains(page,
		[]byte(`content="azazel"`)) || !strings.HasPrefix(string(line),
		"1 1 0 0 ") {
		t.Fatal(string(page), string(line), err, wrongLine)
	}
	if _, err = os.Stat(filepath.Join(root, "latest.html")); err != nil {
		t.Fatal(err)
	}
}

// TestPublishDryRunWritesNothing is the anti-test: a preview names what
// it would do and writes no page.
func TestPublishDryRunWritesNothing(t *testing.T) {
	root := publishRoot(t)
	code, out, wrong := publishing(t, "--renders", root, "--folder",
		"frames/2026-11-15-studio-one", "--dry-run")
	if code != 0 || !strings.Contains(out, `"dry_run":true`) {
		t.Fatal(code, out, wrong)
	}
	for _, name := range []string{"latest.html",
		"frames/2026-11-15-studio-one/index.html"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			t.Fatal("a dry run wrote", name)
		}
	}
}

// TestPublishRefusesWhatItCannotPlace refuses a relative root, a folder
// off the shelves and a count of none.
func TestPublishRefusesWhatItCannotPlace(t *testing.T) {
	root := publishRoot(t)
	for _, args := range [][]string{
		{"--renders", "renders", "--folder", "frames/2026-11-15-studio-one"},
		{"--renders", root, "--folder", "studio/notes"},
		{"--renders", root, "--folder", "frames/2026-11-15-studio-one",
			"--pictures", "0"},
	} {
		if code, _, _ := publishing(t, args...); code == 0 {
			t.Error("publish took", args)
		}
	}
}

// TestRequestSchemaTakesPositivePictures reads pictures through the
// shared contract, and refuses none.
func TestRequestSchemaTakesPositivePictures(t *testing.T) {
	base := "agent: studio\nkey: one\ncommand: /usr/bin/true\nargs: []\n" +
		"directory: /private/tmp\nlane: gpu\ntimeout_seconds: 60\n"
	request, err := decodeRequest([]byte(base + "pictures: 2005\n"))
	if err != nil || request.Pictures == nil || *request.Pictures != 2005 {
		t.Fatal(request, err)
	}
	if _, err = decodeRequest([]byte(base + "pictures: 0\n")); err == nil {
		t.Fatal("pictures 0 was taken")
	}
}

// TestAgentHelpSaysWhoKeepsThePages names the pictures field and the
// publish verb for agents.
func TestAgentHelpSaysWhoKeepsThePages(t *testing.T) {
	told, code := run(t, azazel, "--agent-help")
	for _, want := range []string{"pictures:", "publish", "progress-while.sh"} {
		if code != 0 || !strings.Contains(told, want) {
			t.Errorf("agent help omits %q", want)
		}
	}
}
