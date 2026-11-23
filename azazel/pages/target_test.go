package pages

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// targetRoot is the renders root the target tests read paths against.
const targetRoot = "/renders"

// request is a job's request with these arguments, run from directory.
func request(directory string, args ...string) queue.Request {
	return queue.Request{Directory: directory, Args: args}
}

// TestFolderOfReadsTheJobsOut finds the render folder in each form of
// --out and --output, absolute and relative.
func TestFolderOfReadsTheJobsOut(t *testing.T) {
	want := Folder{Shelf: framesShelf, Name: "2026-11-15-studio-sky"}
	for _, each := range []queue.Request{
		request("/slayer", "draw", "set.yaml", "--out",
			"/renders/frames/2026-11-15-studio-sky/sky.png"),
		request("/renders/frames", "--out=2026-11-15-studio-sky/a/b.png"),
		request("/slayer", "--output", "/renders/frames/2026-11-15-studio-sky"),
	} {
		if got, reason := folderOf(targetRoot, each); reason != "" ||
			got != want {
			t.Errorf("%v: %v %s", each.Args, got, reason)
		}
	}
}

// TestFolderOfRefusesOtherPaths is the anti-test: no out, a path outside
// the root, another folder under it and a picture on a shelf are in no
// render folder, and each says why.
func TestFolderOfRefusesOtherPaths(t *testing.T) {
	for _, each := range []queue.Request{
		request("/slayer", "draw", "set.yaml"),
		request("/slayer", "--out", "/elsewhere/frames/x/a.png"),
		request("/slayer", "--out", "/renders/studio/x/a.png"),
		request("/slayer", "--out", "/renders/frames/a.png"),
		request("/slayer", "--out"),
	} {
		if got, reason := folderOf(targetRoot, each); reason == "" {
			t.Errorf("%v gave %v", each.Args, got)
		}
	}
}

// jobFile writes a job file in the job's directory and returns the
// directory.
func jobFile(t *testing.T, name, text string) string {
	t.Helper()
	directory := t.TempDir()
	err := os.WriteFile(filepath.Join(directory, name), []byte(text), listing)
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

// chunkFrames is where one of heard's chunks writes its frames.
const chunkFrames = "/renders/animations/2026-11-15-team-heard/chunks/" +
	"000153/frames"

// TestFolderOfReadsAJobFilesOutputs finds the render folder in a draw
// job's outputs, in json and in yaml, as heard's chunks name it.
func TestFolderOfReadsAJobFilesOutputs(t *testing.T) {
	want := Folder{Shelf: animationsShelf, Name: "2026-11-15-team-heard"}
	for name, text := range map[string]string{
		"job.yaml": `{"scene": {"named": "heard"}, "outputs": {"frames": "` +
			chunkFrames + `"}}`,
		"set.yml": "outputs:\n  frames: " + chunkFrames + "\n  movie: " +
			"/renders/animations/2026-11-15-team-heard/film.mp4\n",
		"job.json": `{"outputs": {"frames": "` + chunkFrames + `"}}`,
	} {
		directory := jobFile(t, name, text)
		got, reason := folderOf(targetRoot, request(directory, "draw", name))
		if reason != "" || got != want {
			t.Errorf("%s: %v %s", name, got, reason)
		}
	}
}

// TestFolderOfRefusesJobFilesWithoutOneFolder is the anti-test: a job
// file with no outputs, or outputs in two render folders, gives none.
func TestFolderOfRefusesJobFilesWithoutOneFolder(t *testing.T) {
	for _, text := range []string{
		`{"scene": {"named": "heard"}}`,
		`{"outputs": {"frames": "/elsewhere/frames"}}`,
		`{"outputs": {"frames": "` + chunkFrames + `", "movie": ` +
			`"/renders/animations/another/film.mp4"}}`,
	} {
		directory := jobFile(t, "job.yaml", text)
		if got, reason := folderOf(targetRoot, request(directory, "draw",
			"job.yaml")); reason == "" {
			t.Errorf("%s gave %v", text, got)
		}
	}
}

// TestAJobWithNoFolderIsLoggedOnce: the coordinator says once why it
// publishes nothing for a job, and keeps it out of its live jobs.
func TestAJobWithNoFolderIsLoggedOnce(t *testing.T) {
	var log bytes.Buffer
	settings := settingsFor(renders(t), nil, &clock{at: fixtureNoon})
	coordinator := New(settings,
		&log)
	job := queue.Job{Sequence: 7, State: queue.Running,
		Request: request(t.TempDir(), "draw", "job.yaml")}
	ledger := ledgerOf(t, []queue.Job{job}, []queue.Job{job})
	coordinator.look(t.Context(), ledger)
	coordinator.look(t.Context(), ledger)
	if strings.Count(log.String(), "nothing is published") != 1 ||
		len(coordinator.liveJobs()) != 0 {
		t.Fatal(log.String())
	}
}

// TestParseFolderTakesAShelfAndAName reads the publish verb's --folder.
func TestParseFolderTakesAShelfAndAName(t *testing.T) {
	want := Folder{Shelf: animationsShelf, Name: "heard"}
	for _, given := range []string{"animations/heard",
		"/renders/animations/heard"} {
		if got, err := ParseFolder(targetRoot, given); err != nil || got != want {
			t.Errorf("%s: %v %v", given, got, err)
		}
	}
}

// TestParseFolderRefusesAnythingElse is the anti-test.
func TestParseFolderRefusesAnythingElse(t *testing.T) {
	for _, given := range []string{"animations", "animations/heard/frames",
		"studio/heard", "../frames/heard", "frames/.."} {
		if _, err := ParseFolder(targetRoot, given); err == nil {
			t.Errorf("%s was taken", given)
		}
	}
}
