package job

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/set"
)

// importSnapshot holds the checked geometry consumed under the job's identity.
type importSnapshot struct{ set set.Set }

// ImportedSet is the immutable source snapshot for an import-bearing job.
// Jobs without imported geometry retain the existing lazy set handling.
func (job Job) ImportedSet() (set.Set, bool) {
	if job.imported == nil {
		return set.Set{}, false
	}
	return job.imported.set, true
}

// freezeImported pins import-bearing sets before any picture or resume lookup.
// Other sets keep the historical deferred validation by the rendering boundary.
func freezeImported(path string, inputs *digested) (*importSnapshot, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	read, has, err := set.ReadImported(path, raw)
	if !has {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("imported set: %w", err)
	}
	sum := sha256.Sum256(raw)
	inputs.fixed = map[string]string{inputSet: hex.EncodeToString(sum[:])}
	for source, digest := range read.ImportSources() {
		name := "asset:" + filepath.ToSlash(source)
		inputs.paths[name] = source
		inputs.fixed[name] = hex.EncodeToString(digest[:])
	}
	return &importSnapshot{set: read}, nil
}
