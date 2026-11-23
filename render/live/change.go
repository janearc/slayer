package live

import (
	"encoding/json"
	"reflect"

	"github.com/janearc/slayer/job"
)

// Change is how much a change to the job or the view costs, cheapest
// first. Changes fold by taking the dearer.
type Change int

// The changes, in order of cost.
const (
	// Unchanged draws nothing again.
	Unchanged Change = iota
	// Redevelop develops the kept light again: print, stock, titles,
	// exposure by key.
	Redevelop
	// Redraw draws the frame again from the open session: another
	// frame, size or rays.
	Redraw
	// Restage closes the session and opens the job again.
	Restage
)

// words are each change's name, in the order of Change.
var words = [...]string{"unchanged", "redevelop", "redraw", "restage"}

// String is the change's name.
func (change Change) String() string {
	if change < Unchanged || change > Restage {
		return "unknown"
	}
	return words[change]
}

// With is the dearer of two changes: what both need done.
func (change Change) With(other Change) Change { return max(change, other) }

// costOfKey is what a change of each top-level key of the job's
// canonical form costs; a key not here restages. See README.md.
var costOfKey = map[string]Change{
	"print":  Redevelop,
	"stock":  Redevelop,
	"titles": Redevelop,
	"sound":  Unchanged,
}

// costOfFilm is what a change of each key of the film costs; a film key
// not here restages.
var costOfFilm = map[string]Change{
	"width":  Redraw,
	"height": Redraw,
	"rays":   Redraw,
}

// costOfInput is what a change of each file the job reads costs, by
// its name under the canonical form's inputs: a file it does not name
// restages. A set's file changed on disk restages, as the set changed.
var costOfInput = map[string]Change{
	"theme":   Redevelop,
	"tune":    Unchanged,
	"voicing": Unchanged,
	"sound":   Unchanged,
	"beats":   Unchanged,
}

// within is the table of each top-level key whose own keys cost
// differently.
var within = map[string]map[string]Change{
	"film":   costOfFilm,
	"inputs": costOfInput,
}

// Compare is the change from one job to another: the dearest change of
// any key that differs in their canonical forms, and of the files they
// read. Outputs, limits and the note are not in the canonical form, so
// they change nothing.
func Compare(was, is job.Job) Change {
	wasTree := tree(was.Canonical())
	isTree := tree(is.Canonical())
	return differences(wasTree, isTree,
		costOfKey, true)
}

// differences is the dearest cost of the keys that differ between two
// trees: a key with a table of its own in within, at the top, by its
// own keys; any other by the table given; a key in neither restages.
func differences(before, after map[string]any, table map[string]Change,
	top bool) Change {
	cost := Unchanged
	for _, key := range keysOf(before, after) {
		was, is := before[key], after[key]
		if reflect.DeepEqual(was, is) {
			continue
		}
		own, has := within[key]
		if top && has {
			wasBranch := asTree(was)
			isBranch := asTree(is)
			cost = cost.With(differences(wasBranch, isBranch, own,
				false))
			continue
		}
		known, said := table[key]
		if !said {
			known = Restage
		}
		cost = cost.With(known)
	}
	return cost
}

// keysOf is every key of either tree.
func keysOf(trees ...map[string]any) []string {
	var keys []string
	seen := map[string]bool{}
	for _, each := range trees {
		for key := range each {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// tree is a canonical form as a map. The job package wrote it as JSON
// of an object, so it reads.
func tree(canonical []byte) map[string]any {
	var read map[string]any
	_ = json.Unmarshal(canonical, &read)
	return read
}

// asTree is a value of a tree as a tree itself, or none when it is not
// one, as a film the job does not say.
func asTree(value any) map[string]any {
	read, _ := value.(map[string]any)
	return read
}
