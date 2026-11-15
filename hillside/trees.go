package hillside

import (
	"fmt"
	"io"
	"os"

	"github.com/janearc/slayer/gpu"
)

// treeCacheVariable names the bbolt file the GPU's trees are kept in
// between runs, so a frame after the first takes them rather than
// growing them again; unset, every run grows its own
const treeCacheVariable = "SLAYER_TREE_CACHE"

// cachingTrees turns the GPU's tree cache on when the environment names
// its file.
func cachingTrees() error {
	return gpu.CacheTrees(os.Getenv(treeCacheVariable))
}

// reportTrees says what the tree cache did, when it was on, and for
// the per-process way merges this run's trees into the main file.
func reportTrees(said io.Writer) {
	if os.Getenv(treeCacheVariable) == "" {
		return
	}
	fmt.Fprintln(said, gpu.TreesCached())
	merged, err := gpu.MergeTrees()
	if merged != "" || err != nil {
		fmt.Fprintf(said, "tree cache: %s %v\n", merged, err)
	}
}
