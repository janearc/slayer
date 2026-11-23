package session

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/treebolt"
)

// treeFile is the bbolt file the GPU's trees are kept in between runs,
// so a frame after the first takes them rather than growing them again.
// It lives in the cache home, beside the trees kind's folder.
const treeFile = treebolt.File // defined: the cache's name for the file

// treePath is where the GPU's trees are kept: the tree file in the
// cache home, which diskcache.Home finds from the lookup. It refuses by
// name where SLAYER_CACHE is not set; there is no other place.
func treePath(lookup diskcache.Lookup) (string, error) {
	home, err := diskcache.Home(lookup)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, treeFile), nil
}

// cachingTrees turns the GPU's tree cache on, in the cache home the
// lookup names, or refuses by name where there is none.
func cachingTrees(lookup diskcache.Lookup) error {
	path, err := treePath(lookup)
	if err != nil {
		return err
	}
	return gpu.CacheTrees(path)
}

// reportTrees says what the tree cache did, when it did anything, for
// the per-process way merges this run's trees into the main file, and
// keeps the file to its cap: the run's uses are written, and a file over
// the cap is written fresh without its least recently used trees.
func reportTrees(said io.Writer) {
	if tally := gpu.TreesCached(); tally.Used() {
		fmt.Fprintln(said, tally)
	}
	merged, err := gpu.MergeTrees()
	if merged != "" || err != nil {
		fmt.Fprintf(said, "tree cache: %s %v\n", merged, err)
	}
	trimmed, err := gpu.TrimTrees()
	if trimmed != "" || err != nil {
		fmt.Fprintf(said, "tree cache: %s %v\n", trimmed, err)
	}
}
