// Package rules holds the harness to the two of its rules a test can
// see: it reaches slayer only through slayer's exported doors, and no
// commit changes the harness and what the harness judges together. the
// third, that it works out every answer itself, is in INSIDE.md.
package rules

import (
	"bytes"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// harness is the harness's own tree, from here.
const harness = ".."

// folded is the commit that brought the harness into slayer. the rule
// on commits holds from there; before it the two were apart by being
// two repositories. slayer is taken forward by fast-forward or merge,
// never by rebase, so this commit stays in every history.
const folded = "03d5d4421a855727670c6d48dcfa09bf6cff6ade"

// mechanical are the commits that changed the harness and what it judges
// together and judged nothing differently, each read whole: tree-wide
// moves of import paths, and comments, prose and helpers renamed. the
// rule holds every commit in neither this list nor readWhole, and every
// one to come.
var mechanical = map[string]string{
	"8a86563b71dc959d81ee71f7d19f589fb513d0b2": "lamb's imports became " +
		"slayer's audited copies: import paths alone in the harness",
	"3cdacecc8c2861256b04ee8900b7564bd5df4f00": "two comments take a " +
		"paragraph break: comments alone in the harness",
	"0a9bae933d1da07da18fc167034ac9fc9d7cbce3": "comments say where " +
		"lamb's parts went: comments and its notes alone in the harness",
	"b7cdea4486cda7572e56cba708d7c64b213dfee1": "lamb and goat leave: " +
		"import paths, its notes, a test and two helpers renamed and a " +
		"message reworded in the harness",
	"d8936f670fa99f8b33aa5a45745c5fe8abc99dac": "maths and ninjafilm " +
		"become slayer's own: import paths alone in the harness, the " +
		"pins they moved",
}

// readWhole are the commits that changed the harness and what it judges
// together and are not mechanical: each changes what the harness checks,
// and each was read whole and found to leave it at least as strict. each
// says what it changed in the harness, the loosening first, and when it
// was read. they are not mechanical and are not listed as if they were.
var readWhole = map[string]string{
	"03d6b0aa10a1f3a3a48a0d7e290962390bb1cc4a": "recursive foliage: one " +
		"view of three moves off edge-on, (1, 0.1, 0) to (1, 0.1, 0.2), " +
		"since a flat spray edge-on has no pixels to compare; the harness " +
		"learns mesh groups, each placement's winding and finite corners " +
		"and each prototype's faces checked, where groups went unchecked " +
		"before. read 2026-11-06",
	"96fb7c74700c0c8c3f9ed96608133535df6a9e8d": "a merge resolved by " +
		"hand: mesh_test takes 6ce998d's comparison, which expands each " +
		"group into plain reference meshes so no group is judged against " +
		"itself, as 03d6b0a's was; the rest is gofumpt's reflow. read " +
		"2026-11-06",
}

// shared is what a commit to the harness may also touch: the module's
// own files, which the harness and slayer both need, but only where no
// changed line names one of janearc's libraries. those lines pin
// maths, the theme and holemath, which is the physics the tracer runs
// on, so a change to them is a change to what the harness judges.
var shared = []string{"go.mod", "go.sum"}

// pinned is what names one of janearc's libraries in go.mod or go.sum.
const pinned = "github.com/janearc/"

// importsOf is every import path of the harness's go files.
func importsOf(t *testing.T) map[string][]string {
	t.Helper()
	found := map[string][]string{}
	err := filepath.WalkDir(harness, func(path string, entry fs.DirEntry,
		err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil,
			parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range parsed.Imports {
			imported, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			found[path] = append(found[path], imported)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestTheHarnessUsesOnlySlayersExportedDoors(t *testing.T) {
	const slayer = "github.com/janearc/slayer/"
	files := importsOf(t)
	if len(files) < 20 {
		t.Fatalf("found %d go files under %s; is this the harness?",
			len(files), harness)
	}
	for file, imported := range files {
		for _, path := range imported {
			inner := strings.Contains("/"+path+"/", "/internal/")
			if strings.HasPrefix(path, slayer) && inner {
				t.Errorf("%s imports %s, which is inside slayer", file,
					path)
			}
		}
	}
}

// git is what git says, run in the harness, or the test stops.
func git(t *testing.T, arguments ...string) []byte {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = harness
	out, err := command.Output()
	if err != nil {
		t.Fatalf("git %s: %v. this test reads slayer's history since "+
			"the harness was folded in, %s: run it in a clone of slayer "+
			"that has that commit. a history taken forward by rebase "+
			"does not", strings.Join(arguments, " "), err, folded[:7])
	}
	return out
}

// touched is, for every commit since the harness was folded in, the top-level
// names it changes.
//
// a merge counts by what its own resolution changed beyond git's automatic
// merge, so a plain merge changes nothing and a merge resolved into both sides
// is caught. renames count as the delete and the add they are, so a file moved
// out of the harness counts on both sides.
func touched(t *testing.T) map[string][]string {
	t.Helper()
	out := git(t, "log", "--diff-merges=remerge", "--no-renames",
		"--format=@%H", "--name-only", folded+"..HEAD")
	commits := map[string][]string{}
	var commit string
	for _, line := range bytes.Split(out, []byte("\n")) {
		text := string(line)
		switch {
		case strings.HasPrefix(text, "@"):
			commit = text[1:]
			commits[commit] = nil
		case text != "":
			top := strings.SplitN(text, "/", 2)[0]
			commits[commit] = append(commits[commit], top)
		}
	}
	return commits
}

// judged is whether a top-level name is something the harness judges:
// anything of slayer's that is not the harness and not shared.
func judged(top string) bool {
	if top == "burning-trousers" {
		return false
	}
	for _, name := range shared {
		if top == name {
			return false
		}
	}
	return true
}

// movesPins is whether a commit changes a line of slayer's own go.mod or
// go.sum, at the top of the repository, that names one of janearc's
// libraries.
func movesPins(t *testing.T, commit string) bool {
	t.Helper()
	out := git(t, "show", "--diff-merges=remerge", "--format=", commit,
		"--", ":/go.mod", ":/go.sum")
	for _, line := range bytes.Split(out, []byte("\n")) {
		text := string(line)
		changed := strings.HasPrefix(text, "+") ||
			strings.HasPrefix(text, "-")
		header := strings.HasPrefix(text, "+++") ||
			strings.HasPrefix(text, "---")
		if changed && !header && strings.Contains(text, pinned) {
			return true
		}
	}
	return false
}

// alsoJudged is what else a commit to the harness changes that the
// harness judges, or nothing.
func alsoJudged(t *testing.T, commit string, tops []string) []string {
	t.Helper()
	var others []string
	var shares bool
	for _, top := range tops {
		shares = shares || !judged(top) && top != "burning-trousers"
		if judged(top) {
			others = append(others, top)
		}
	}
	if shares && movesPins(t, commit) {
		others = append(others, "go.mod's pins")
	}
	return others
}

func TestNoCommitChangesTheHarnessAndWhatItJudges(t *testing.T) {
	broke := map[string]bool{}
	for commit, tops := range touched(t) {
		var harnessToo bool
		for _, top := range tops {
			harnessToo = harnessToo || top == "burning-trousers"
		}
		if !harnessToo {
			continue
		}
		others := alsoJudged(t, commit, tops)
		if len(others) == 0 {
			continue
		}
		_, isMechanical := mechanical[commit]
		_, isRead := readWhole[commit]
		if isMechanical || isRead {
			broke[commit] = true
			continue
		}
		judges := slices.Compact(slices.Sorted(slices.Values(others)))
		t.Errorf("%s changes the harness, burning-trousers, and what it "+
			"judges, %s, in one commit.\n%s", commit[:7],
			strings.Join(judges, ", "), apart)
	}
	// a commit named in either list that the rule would not hold, or that
	// is not in the history, is a name with nothing behind it; one named
	// in both is mechanical and not at once
	for _, named := range namedLists() {
		for commit, why := range named.commits {
			if !broke[commit] {
				t.Errorf("%s is named %s (%s), and the rule has "+
					"nothing against it.\n%s", commit[:7], named.list,
					why, unbacked)
			}
			if _, both := mechanical[commit]; both && named.list != "mechanical" {
				t.Errorf("%s is named both mechanical and %s.\n%s",
					commit[:7], named.list, oneList)
			}
		}
	}
}

// What the rule's refusals say, for a reader who has not read the rule.
const (
	apart = "The harness is slayer's judge. A commit that changes the " +
		"judge and the judged together can change what passes without " +
		"anyone seeing it, so the rule asks for them apart.\n" +
		"Not yet pushed: split it into a commit to the harness and a " +
		"commit to the rest.\n" +
		"Already in the history: read the harness's part whole. If it " +
		"leaves the harness at least as strict, name the commit in " +
		"readWhole in burning-trousers/rules/rules_test.go, saying what " +
		"it changed there, any loosening first, and when it was read. " +
		"mechanical is for import paths, comments and renames alone."
	unbacked = "It is not in this history, or it does not change the " +
		"harness and what it judges together. A name the rule does not " +
		"need would excuse a commit no one read: take it out."
	oneList = "A commit is mechanical or it changes what the harness " +
		"checks; it cannot be both. Take it out of one list."
)

// namedList is one of the lists of commits the rule lets stand.
type namedList struct {
	list    string
	commits map[string]string
}

// namedLists are both lists, by the names their errors give them.
func namedLists() []namedList {
	return []namedList{{"mechanical", mechanical}, {"read whole", readWhole}}
}
