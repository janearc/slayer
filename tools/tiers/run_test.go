package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGoScript stands in for the go command. It logs its arguments and
// answers from files in the directory named by FAKEGO_DIR.
const fakeGoScript = "#" + `!/bin/sh
echo "$*" >> "$FAKEGO_DIR/args.log"
code() { if [ -f "$FAKEGO_DIR/$1.code" ]; then cat "$FAKEGO_DIR/$1.code"; else echo 0; fi; }
case "$*" in
*"-list"*)
  case "$*" in *"-tags draw"*) f=list-draw;; *) f=list-plain;; esac
  [ -f "$FAKEGO_DIR/$f.out" ] && cat "$FAKEGO_DIR/$f.out"
  [ -f "$FAKEGO_DIR/$f.err" ] && cat "$FAKEGO_DIR/$f.err" >&2
  [ "$f" = list-plain ] && [ -f "$FAKEGO_DIR/selfdelete" ] && rm -f "$0"
  exit "$(code $f)";;
"list -e"*)
  cat "$FAKEGO_DIR/packages.out"; exit "$(code packages)";;
*"-run"*)
  for last; do :; done
  echo "draw ran $last"
  if grep -qx "$last" "$FAKEGO_DIR/failing" 2>/dev/null; then exit 1; fi
  exit 0;;
*)
  echo "hot ran"; exit "$(code hot)";;
esac
`

// fakeGo makes the stand-in the only go on PATH and returns its directory. Tests
// write the files it answers from into that directory.
func fakeGo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(fakeGoScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKEGO_DIR", dir)
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	return dir
}

// answer writes a file the fake go reads.
func answer(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// calls is the argument lines the fake go was run with.
func calls(t *testing.T, dir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "args.log"))
	if err != nil {
		t.Fatal(err)
	}
	trimmed := strings.TrimSpace(string(raw))
	return strings.Split(trimmed, "\n")
}

// drawListing is a go test -list answer for two packages, p/one with TestA and
// TestD and p/two with TestC, as the draw build would print it.
const drawListing = "TestA\nTestD\nok  \tp/one\t0.01s\nTestC\nok  \tp/two\t0.01s\n"

// plainListing is the plain build's answer: the draw-only TestD is missing.
const plainListing = "TestA\nok  \tp/one\t0.01s\nTestC\nok  \tp/two\t0.01s\n"

// savedNames is the saved file that matches the draw listing.
const savedNames = "p/one\tTestA\np/one\tTestD\np/two\tTestC\n"

// execute runs the command and returns its code and the two outputs.
func execute(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// TestRunRefusesMissingAndUnknownCommands checks the usage failures.
func TestRunRefusesMissingAndUnknownCommands(t *testing.T) {
	if code, _, errOut := execute(); code != exitUsage || !strings.Contains(errOut, "want snapshot, hot, draw or check") {
		t.Errorf("no arguments: %d %q", code, errOut)
	}
	if code, _, errOut := execute("bogus"); code != exitUsage || !strings.Contains(errOut, `unknown command "bogus"`) {
		t.Errorf("unknown command: %d %q", code, errOut)
	}
	if code, _, errOut := execute("check", "-nope"); code != exitUsage || !strings.Contains(errOut, "nope") {
		t.Errorf("unknown flag: %d %q", code, errOut)
	}
}

// TestSnapshotWritesEveryNameFromTheDrawBuild checks the saved file and
// the count it reports.
func TestSnapshotWritesEveryNameFromTheDrawBuild(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", drawListing)
	file := filepath.Join(t.TempDir(), "saved.txt")
	code, out, errOut := execute("snapshot", "-o", file)
	if code != 0 || errOut != "" || !strings.Contains(out, "3 names in "+file) {
		t.Fatalf("snapshot: %d %q %q", code, out, errOut)
	}
	raw, err := os.ReadFile(file)
	if err != nil || string(raw) != savedNames {
		t.Errorf("saved %q, %v", raw, err)
	}
	if got := calls(t, dir); len(got) != 1 || got[0] != "test -vet=off -list . -count=1 -tags draw ./..." {
		t.Errorf("go was run as %q", got)
	}
}

// TestSnapshotReportsItsFailures checks a tree that cannot be listed and a
// file that cannot be written.
func TestSnapshotReportsItsFailures(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", drawListing)
	missing := filepath.Join(t.TempDir(), "no", "such", "saved.txt")
	if code, _, errOut := execute("snapshot", "-o", missing); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("unwritable file: %d %q", code, errOut)
	}
	answer(t, dir, "list-draw.code", "1")
	answer(t, dir, "list-draw.err", "boom\n")
	target := filepath.Join(t.TempDir(), "x")
	if code, _, errOut := execute("snapshot", "-o", target); code != exitUsage || !strings.Contains(errOut, "boom") {
		t.Errorf("unlistable tree: %d %q", code, errOut)
	}
}

// TestCheckPassesWhenTheTiersAddUp checks a quiet pass, with and without
// the per package draw count.
func TestCheckPassesWhenTheTiersAddUp(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", drawListing)
	answer(t, dir, "list-plain.out", plainListing)
	saved := filepath.Join(t.TempDir(), "saved.txt")
	answer(t, filepath.Dir(saved), "saved.txt", savedNames)
	code, out, _ := execute("check", "-before", saved)
	if code != 0 || !strings.Contains(out, "3 saved, 3 with the draw tag, 2 without, 1 draw only, 0 problems") {
		t.Fatalf("check: %d %q", code, out)
	}
	if strings.Contains(out, "1 draw\t") {
		t.Errorf("the draw count was printed without -v: %q", out)
	}
	code, out, _ = execute("check", "-v", "-before", saved)
	if code != 0 || !strings.Contains(out, "p/one\t1 draw\t1 plain\n") || strings.Contains(out, "p/two\t") {
		t.Errorf("verbose check: %d %q", code, out)
	}
}

// TestCheckNamesWhatDiffers checks that a saved name the tree lacks and a
// tree name the file lacks each make the exit code one and are named.
func TestCheckNamesWhatDiffers(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", drawListing)
	answer(t, dir, "list-plain.out", plainListing)
	saved := filepath.Join(t.TempDir(), "saved.txt")
	answer(t, filepath.Dir(saved), "saved.txt", "p/one\tTestA\np/one\tTestGone\np/two\tTestC\n")
	code, out, _ := execute("check", "-before", saved)
	if code != exitDiffer || !strings.Contains(out, "p/one: lost: TestGone") ||
		!strings.Contains(out, "p/one: added: TestD") || !strings.Contains(out, "2 problems") {
		t.Errorf("check: %d %q", code, out)
	}
}

// TestCheckWithPatternsLooksAtThosePackagesOnly checks that the saved names
// of other packages are set aside, and that the pattern is passed on.
func TestCheckWithPatternsLooksAtThosePackagesOnly(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", "TestA\nTestD\nok  \tp/one\t0.01s\n")
	answer(t, dir, "list-plain.out", "TestA\nok  \tp/one\t0.01s\n")
	answer(t, dir, "packages.out", "p/one\n")
	saved := filepath.Join(t.TempDir(), "saved.txt")
	answer(t, filepath.Dir(saved), "saved.txt", savedNames)
	code, out, _ := execute("check", "-before", saved, "./p/one")
	if code != 0 || !strings.Contains(out, "2 saved, 2 with the draw tag, 1 without") {
		t.Fatalf("check: %d %q", code, out)
	}
	found := false
	for _, line := range calls(t, dir) {
		found = found || line == "list -e ./p/one"
	}
	if !found {
		t.Errorf("go list was not run for the pattern: %q", calls(t, dir))
	}
	answer(t, dir, "packages.code", "1")
	if code, _, errOut := execute("check", "-before", saved, "./p/one"); code != exitUsage || !strings.Contains(errOut, "go list -e ./p/one") {
		t.Errorf("a failed go list: %d %q", code, errOut)
	}
}

// TestCheckReportsItsFailures checks a missing saved file and a tree that
// cannot be listed either way.
func TestCheckReportsItsFailures(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", drawListing)
	answer(t, dir, "list-plain.out", plainListing)
	saved := filepath.Join(t.TempDir(), "saved.txt")
	if code, _, errOut := execute("check", "-before", saved); code != exitUsage || !strings.Contains(errOut, "saved.txt") {
		t.Errorf("missing saved file: %d %q", code, errOut)
	}
	answer(t, filepath.Dir(saved), "saved.txt", savedNames)
	answer(t, dir, "list-plain.code", "1")
	if code, _, errOut := execute("check", "-before", saved); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("plain listing failed: %d %q", code, errOut)
	}
	answer(t, dir, "list-draw.code", "1")
	if code, _, errOut := execute("check", "-before", saved); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("draw listing failed: %d %q", code, errOut)
	}
}

// TestListTreeTreatsAnExcludedPackageAsEmpty checks that go's complaint
// about build constraints is not an error, and that any other failure is.
func TestListTreeTreatsAnExcludedPackageAsEmpty(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-plain.code", "1")
	answer(t, dir, "list-plain.err", "package p/draw: build constraints exclude all Go files in /x\n")
	got, err := listTree("", []string{"./p/draw"})
	if err != nil || len(got) != 0 {
		t.Errorf("excluded package: %v, %v", got, err)
	}
	answer(t, dir, "list-plain.err", "something else\n")
	if _, err := listTree("", []string{"./p/draw"}); err == nil || !strings.Contains(err.Error(), "something else") {
		t.Errorf("other failure: %v", err)
	}
	answer(t, dir, "list-plain.code", "0")
	answer(t, dir, "list-plain.err", "")
	answer(t, dir, "list-plain.out", "FAIL\tp/one [build failed]\n")
	if _, err := listTree("", nil); err == nil || !strings.Contains(err.Error(), "a package failed") {
		t.Errorf("a failed package: %v", err)
	}
}

// TestHotRunsThePackagesOfTheList checks the arguments given to go and the
// exit code passed back.
func TestHotRunsThePackagesOfTheList(t *testing.T) {
	dir := fakeGo(t)
	list := filepath.Join(t.TempDir(), "hot.txt")
	answer(t, filepath.Dir(list), "hot.txt", "p/one # first\n\np/two\n")
	code, out, _ := execute("hot", "-list", list)
	if code != 0 || out != "hot ran\n" {
		t.Fatalf("hot: %d %q", code, out)
	}
	if got := calls(t, dir); len(got) != 1 || got[0] != "test -count=1 p/one p/two" {
		t.Errorf("go was run as %q", got)
	}
	answer(t, dir, "hot.code", "3")
	if code, _, _ := execute("hot", "-list", list); code != 3 {
		t.Errorf("a failing hot run returned %d, want 3", code)
	}
}

// TestHotReportsItsFailures checks a missing list and a go that cannot run.
func TestHotReportsItsFailures(t *testing.T) {
	fakeGo(t)
	absent := filepath.Join(t.TempDir(), "none")
	if code, _, errOut := execute("hot", "-list", absent); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("missing list: %d %q", code, errOut)
	}
	list := filepath.Join(t.TempDir(), "hot.txt")
	answer(t, filepath.Dir(list), "hot.txt", "p/one\n")
	t.Setenv("PATH", t.TempDir())
	if code, _, errOut := execute("hot", "-list", list); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("no go on the path: %d %q", code, errOut)
	}
}

// TestDrawRunsOnlyTheDrawOnlyTests checks which package is run, with which
// tests, and that a failing package is named while the others still run.
func TestDrawRunsOnlyTheDrawOnlyTests(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", "TestA\nTestD\nok  \tp/one\t0.01s\nTestC\nTestE\nok  \tp/two\t0.01s\n")
	answer(t, dir, "list-plain.out", "TestA\nok  \tp/one\t0.01s\nTestC\nTestE\nok  \tp/two\t0.01s\n")
	code, out, _ := execute("draw")
	if code != 0 || out != "draw ran p/one\n" {
		t.Fatalf("draw: %d %q", code, out)
	}
	got := calls(t, dir)
	if len(got) != 3 || got[2] != "test -count=1 -tags draw -p 1 -timeout 60m -run ^(TestD)$ p/one" {
		t.Errorf("go was run as %q", got)
	}
	answer(t, dir, "failing", "p/one\n")
	code, out, _ = execute("draw", "./p/...")
	if code != exitFailed || !strings.Contains(out, "tiers: draw failed in p/one") {
		t.Errorf("a failing package: %d %q", code, out)
	}
}

// TestDrawReportsItsFailures checks a tree that cannot be listed either way
// and a go command that stops existing before the run.
func TestDrawReportsItsFailures(t *testing.T) {
	dir := fakeGo(t)
	answer(t, dir, "list-draw.out", drawListing)
	answer(t, dir, "list-plain.out", plainListing)
	answer(t, dir, "list-plain.code", "1")
	if code, _, errOut := execute("draw"); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("plain listing failed: %d %q", code, errOut)
	}
	answer(t, dir, "list-draw.code", "1")
	if code, _, errOut := execute("draw"); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("draw listing failed: %d %q", code, errOut)
	}
	answer(t, dir, "list-draw.code", "0")
	answer(t, dir, "list-plain.code", "0")
	answer(t, dir, "selfdelete", "")
	if code, _, errOut := execute("draw"); code != exitUsage || !strings.Contains(errOut, "tiers:") {
		t.Errorf("go vanished before the run: %d %q", code, errOut)
	}
}

// TestReadersRefuseMissingFiles checks the file errors of readSaved and
// readHot, and a list output line parseList cannot read.
func TestReadersRefuseMissingFiles(t *testing.T) {
	none := filepath.Join(t.TempDir(), "none")
	if _, err := readSaved(none); err == nil {
		t.Error("a missing saved file was read")
	}
	if _, err := readHot(none); err == nil {
		t.Error("a missing hot list was read")
	}
	if _, err := parseList([]byte("three words here\n")); err == nil || !strings.Contains(err.Error(), "cannot read") {
		t.Errorf("an unreadable line: %v", err)
	}
	if got, err := parseList([]byte("?   \tp/none\t[no test files]\nTestX\nok  \tp/x\t0s\n")); err != nil || len(got["p/x"]) != 1 || len(got["p/none"]) != 0 {
		t.Errorf("a package with no test files: %v, %v", got, err)
	}
}
