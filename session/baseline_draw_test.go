//go:build draw

package session

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// stagedBaselineFile is the table of digests of what the session stages from
// each version 1 set the repository ships, written once from the stage as
// it stood before the reader took version 2. It is never regenerated: a
// digest that moves means what a version 1 set stages moved.
const stagedBaselineFile = "testdata/baseline-v1-staged.digests"

// readBaselineFile is the table of the reader's baseline, in the set
// package's folder; its paths say which files are version 1 sets.
const readBaselineFile = "../set/testdata/baseline-v1.digests"

// stagedArguments are the flags every baseline staging is asked with: the
// smallest picture, at the first instant.
var stagedArguments = []string{"-w", "32", "-n", "1", "-at", "0"}

// shelfFolder is the environment variable naming where the baseline's sets
// find their sources, as the other heavy tests have it.
const shelfFolder = "SLAYER_SHELF"

// drawnOnCPU are the sets the baseline also draws, on the CPU, at the
// stagedArguments' width, and records the digest of the raw: the ones that
// say no renderer and draw in a blink. Picked by time, the others take
// minutes.
var drawnOnCPU = []string{"set/testdata/built.yaml", "sets/shoebox.yaml",
	"sets/widow.yaml", "sets/widow-smoke.yaml", "sets/widow-tail.yaml"}

// digestText is the sha256 of a text, in hexadecimal.
func digestText(text []byte) string {
	sum := sha256.Sum256(text)
	return hex.EncodeToString(sum[:])
}

// stagedBaselinePaths is the version 1 sets the reader's baseline lists,
// each by its path from the repository's root, that the reader takes.
func stagedBaselinePaths(t *testing.T) []string {
	t.Helper()
	file, err := os.Open(readBaselineFile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var paths []string
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		path, _, _ := strings.Cut(lines.Text(), " ")
		source, err := os.ReadFile(filepath.Join("..", path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := set.Read(path, source); err == nil {
			paths = append(paths, path)
		}
	}
	if err := lines.Err(); err != nil {
		t.Fatal(err)
	}
	return paths
}

// stagedDump is what the session stages from a set of the repository, as text:
// its refusal, or its tally and the digest of the table of what it stood.
func stagedDump(t *testing.T, path string) string {
	t.Helper()
	placed := filepath.Join(t.TempDir(), "places.csv")
	arguments := append(append([]string{}, stagedArguments...),
		"-set", filepath.Join("..", path), "-places", placed)
	given, err := parse(arguments, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	_, made, ready, err := staged(given)
	if ready.live != nil {
		defer ready.live.closeBeds()
	}
	if err != nil {
		return "refused: " + strings.ReplaceAll(err.Error(),
			filepath.Join("..", path), path)
	}
	table, err := os.ReadFile(placed)
	if err != nil {
		table = []byte("no table")
	}
	said := fmt.Sprintf("tally %+v\nplaces %s\n", made, digestText(table))
	if slices.Contains(drawnOnCPU, path) {
		said += "raw " + rawDigest(t, path) + "\n"
	}
	return said
}

// rawDigest is the digest of the raw floats the command draws for a set
// of the repository, on the CPU, or the command's complaint.
func rawDigest(t *testing.T, path string) string {
	t.Helper()
	raw := filepath.Join(t.TempDir(), "drawn.raw")
	arguments := append(append([]string{}, stagedArguments...),
		"-set", filepath.Join("..", path), "-raw", raw)
	var said, wrong bytes.Buffer
	if code := run(arguments, &said, &wrong); code != nothing {
		return fmt.Sprintf("exit %d: %s", code, wrong.String())
	}
	drawn, err := os.ReadFile(raw)
	if err != nil {
		return "no raw: " + err.Error()
	}
	return digestText(drawn)
}

// stagedDigests is the recorded table, a path and its digest to a line.
func stagedDigests(t *testing.T) map[string]string {
	t.Helper()
	file, err := os.Open(stagedBaselineFile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	table := map[string]string{}
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		path, digest, _ := strings.Cut(lines.Text(), " ")
		table[path] = digest
	}
	if err := lines.Err(); err != nil {
		t.Fatal(err)
	}
	return table
}

// Every version 1 set the repository ships stages as it always did: the
// same tally, the same table of what stood where, and for the sets that
// draw in a blink the same raw floats on the CPU, or the same refusal.
// It needs the shelf, so it skips with the other heavy set tests.
func TestVersionOneStagesAsItAlwaysDid(t *testing.T) {
	if os.Getenv(shelfFolder) == "" || testing.Short() {
		t.Skip("no shelf named by SLAYER_SHELF, or short")
	}
	recorded := stagedDigests(t)
	if len(recorded) == nothing {
		t.Fatal("the staged baseline holds no set")
	}
	for _, path := range stagedBaselinePaths(t) {
		want, found := recorded[path]
		if !found {
			continue
		}
		t.Run(path, func(t *testing.T) {
			dump := stagedDump(t, path)
			if got := digestText([]byte(dump)); got != want {
				t.Errorf("what it stages moved: digest %s, baseline %s; "+
					"it stages:\n%s", got, want, dump)
			}
		})
	}
	for path := range recorded {
		if !slices.Contains(stagedBaselinePaths(t), path) {
			t.Errorf("%s is in the staged baseline and the reader no "+
				"longer takes it", path)
		}
	}
}
