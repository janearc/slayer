package job

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/set"
)

// importSnapshot holds checked geometry consumed under the job's identity.
type importSnapshot struct{ set set.Set }

// ImportedSet is the immutable snapshot for a job with imports, a
// composition or an explicitly supplied set buffer. Other sets retain the
// existing lazy handling.
func (job Job) ImportedSet() (set.Set, bool) {
	if job.imported == nil {
		return set.Set{}, false
	}
	return job.imported.set, true
}

// freezeImported pins sets with imports or compositions before any picture
// or resume lookup. Other sets keep deferred validation by the renderer.
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
	return freezeSet(read, raw, inputs)
}

// freezeSet records the source closure of a set already read from raw.
func freezeSet(read set.Set, raw []byte, inputs *digested) (*importSnapshot, error) {
	sum := sha256.Sum256(raw)
	inputs.fixed = map[string]string{inputSet: hex.EncodeToString(sum[:])}
	compositionSources, err := read.CompositionSources()
	if err != nil {
		return nil, fmt.Errorf("composition set: %w", err)
	}
	for source, digest := range compositionSources {
		name := "composition:" + filepath.ToSlash(source)
		inputs.paths[name] = source
		inputs.fixed[name] = hex.EncodeToString(digest[:])
	}
	profileSources, err := read.ProfileSources()
	if err != nil {
		return nil, fmt.Errorf("profile set: %w", err)
	}
	for source, digest := range profileSources {
		name := "profile:" + filepath.ToSlash(source)
		inputs.paths[name] = source
		inputs.fixed[name] = hex.EncodeToString(digest[:])
	}
	for source, digest := range read.BuiltinProfileSources() {
		name := "profile:@" + source
		inputs.fixed[name] = hex.EncodeToString(digest[:])
	}
	for source, digest := range read.ImportSources() {
		name := "asset:" + filepath.ToSlash(source)
		inputs.paths[name] = source
		inputs.fixed[name] = hex.EncodeToString(digest[:])
	}
	return &importSnapshot{set: read}, nil
}
