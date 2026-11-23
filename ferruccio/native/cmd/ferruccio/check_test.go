package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/session/placement"
)

// checkSetSource is a space set the working display takes: one plain kind,
// a seed, a film size and a fixed camera.
const checkSetSource = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Check
  setting: space
  seed: 19
  light: none
  set:
    things:
      visitor: {kind: human-scale-2m, seed: 7, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -60, 20]}
      orientation: {look_at: [0, 0, 1]}
`

// checkFile writes text to a new file in directory and returns its path.
func checkFile(t *testing.T, directory, name, text string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckNeedsFilesOrASetsDirectory(t *testing.T) {
	code, output := runAtelier("check")
	if code == 0 || !strings.Contains(output, "needs FILE... or --sets DIR") {
		t.Fatalf("check with nothing to check returned %d: %s", code, output)
	}
}

func TestCheckRefusesABadTimeout(t *testing.T) {
	for _, given := range []string{"soon", "0s", "-5s"} {
		code, output := runAtelier("check", "--timeout", given, "any.yaml")
		if code == 0 || !strings.Contains(output, "timeout must be a positive duration") {
			t.Fatalf("timeout %q returned %d: %s", given, code, output)
		}
	}
}

func TestCheckRefusesAMissingSetsDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nowhere")
	if code, _ := runAtelier("check", "--sets", missing); code == 0 {
		t.Fatal("a sets directory that is not there was checked")
	}
}

func TestCheckTakesASetAndRefusesABrokenOne(t *testing.T) {
	directory := t.TempDir()
	good := checkFile(t, directory, "good.yaml", checkSetSource)
	broken := checkFile(t, directory, "broken.yaml", "scene: [unclosed\n")
	code, output := runAtelier("check", good, broken)
	if code != 0 {
		t.Fatalf("a refusal made check fail with %d: %s", code, output)
	}
	if !strings.Contains(output, good+"\ttakes\n") {
		t.Fatalf("the good set was not taken: %s", output)
	}
	if !strings.Contains(output, broken+"\trefused: ") {
		t.Fatalf("the broken set was not refused: %s", output)
	}
	if !strings.Contains(output, "2 sets: 1 takes, 1 refused, 0 timed out, 0 not checked") {
		t.Fatalf("the tally is wrong: %s", output)
	}
}

func TestCheckSetsTakesOnlyYamlFilesInOrderByName(t *testing.T) {
	directory := t.TempDir()
	checkFile(t, directory, "b.yaml", checkSetSource)
	checkFile(t, directory, "a.yaml", checkSetSource)
	checkFile(t, directory, "notes.txt", "not a set")
	if err := os.Mkdir(filepath.Join(directory, "c.yaml.d"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, output := runAtelier("check", "--sets", directory)
	first, second := strings.Index(output, "a.yaml\t"), strings.Index(output, "b.yaml\t")
	if first < 0 || second < first {
		t.Fatalf("the sets are not listed in order: %s", output)
	}
	if strings.Contains(output, "notes.txt") || strings.Contains(output, "c.yaml.d") {
		t.Fatalf("a file that is not a set was checked: %s", output)
	}
}

func TestCheckLeavesTheSetsDirectoryAsItFound(t *testing.T) {
	directory := t.TempDir()
	checkFile(t, directory, "good.yaml", checkSetSource)
	before, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	runAtelier("check", "--meshes", "--sets", directory)
	after, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("check left %d files where it found %d", len(after), len(before))
	}
}

func TestCheckWithReportsATimeoutNotARefusal(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	slow := func(string, bool) error { <-release; return nil }
	result := checkWith(checkPath{path: "slow.yaml", name: "slow.yaml"}, 10*time.Millisecond, true, slow)
	if result.Verdict != checkTimedOut {
		t.Fatalf("a set that ran out was %q", result.Verdict)
	}
	if len(checkTally([]checkResult{result})) != 0 {
		t.Fatal("a timeout was counted as a refusal reason")
	}
}

func TestCheckWithCallsRunWithTheMeshesChoice(t *testing.T) {
	var seen bool
	spy := func(_ string, meshes bool) error { seen = meshes; return nil }
	checkWith(checkPath{path: "a.yaml", name: "a.yaml"}, time.Second, true, spy)
	if !seen {
		t.Fatal("the meshes choice did not reach the check")
	}
	checkWith(checkPath{path: "a.yaml", name: "a.yaml"}, time.Second, false, spy)
	if seen {
		t.Fatal("meshes were built when they were not asked for")
	}
}

func TestCheckWithCountsAPanicAsARefusal(t *testing.T) {
	boom := func(string, bool) error { panic("no mesh") }
	result := checkWith(checkPath{path: "a.yaml", name: "a.yaml"}, time.Second, true, boom)
	if result.Verdict != checkRefused || !strings.Contains(result.Refusal, "panic: no mesh") {
		t.Fatalf("a panic was %q, %q", result.Verdict, result.Refusal)
	}
}

func TestCheckFinishedDropsThePlacementNameFromTheReason(t *testing.T) {
	err := &placement.DisplayThingError{Thing: "tower northeast", Err: errors.New("no valid placed hitter")}
	result := checkFinished(checkResult{Name: "x"}, checkPath{path: "x.yaml"}, err)
	if result.Refusal != "tower northeast: no valid placed hitter" {
		t.Fatalf("the refusal lost the display's words: %q", result.Refusal)
	}
	if result.Reason != "no valid placed hitter" {
		t.Fatalf("the reason kept the placement name: %q", result.Reason)
	}
}

func TestCheckFinishedFoldsLinesAndHidesThePath(t *testing.T) {
	err := fmt.Errorf("a/b/x.yaml: line 3:\n  bad key")
	result := checkFinished(checkResult{}, checkPath{path: "a/b/x.yaml"}, err)
	if strings.Contains(result.Reason, "\n") || strings.Contains(result.Reason, "a/b/x.yaml") {
		t.Fatalf("the reason is not one line without the path: %q", result.Reason)
	}
}

func TestCheckTallyPutsTheMostCommonReasonFirst(t *testing.T) {
	refused := func(reason string) checkResult { return checkResult{Verdict: checkRefused, Reason: reason} }
	tally := checkTally([]checkResult{refused("b"), refused("a"), refused("c"), refused("c"), {Verdict: checkTakes}})
	if len(tally) != 3 || tally[0].Reason != "c" || tally[0].Count != 2 || tally[1].Reason != "a" || tally[2].Reason != "b" {
		t.Fatalf("the tally is out of order: %v", tally)
	}
}

func TestCheckLineShowsTimeOnlyForMeshes(t *testing.T) {
	result := checkResult{Name: "x.yaml", Verdict: checkTakes, Elapsed: 1500 * time.Millisecond}
	if got := checkLine(result, false); got != "x.yaml\ttakes" {
		t.Fatalf("a cheap line was %q", got)
	}
	if got := checkLine(result, true); got != "x.yaml\ttakes (1.5s)" {
		t.Fatalf("a meshes line was %q", got)
	}
	skipped := checkResult{Name: "y.yaml", Verdict: checkNotChecked}
	if got := checkLine(skipped, true); got != "y.yaml\tnot checked" {
		t.Fatalf("an unchecked line was %q", got)
	}
}

func Example_check() {
	directory, err := os.MkdirTemp("", "ferruccio-check-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "broken.yaml")
	if err := os.WriteFile(path, []byte("scene: [unclosed\n"), 0o600); err != nil {
		panic(err)
	}
	code, output := runAtelier("check", path)
	fmt.Println(code, strings.Contains(output, "\trefused: "), strings.Contains(output, "1 sets: 0 takes, 1 refused"))
	// Output: 0 true true
}
