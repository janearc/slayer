package workload

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// ErrIncomplete refuses evidence whose recursive input dependencies are unknown.
var ErrIncomplete = errors.New("workload: incomplete source dependencies")

// Dependencies is an authoritative format/scene resolver's result. Complete
// means it knows every external input this file/scene can read, including
// implicit assets. Opaque leaves explicitly return Complete with no Files.
// Unknown named scenes or dynamic input selection must remain incomplete.
type Dependencies struct {
	Files    []string
	Complete bool
}

// Resolver inspects the actual opened source without retaining or closing it.
// It must report indirect and implicit dependencies, not merely job.Inputs.
// Relative dependencies resolve beside the logical source path, preserving
// the renderer's path context even when a source is a symbolic link.
type Resolver func(context.Context, string, io.ReadSeeker) (Dependencies, error)

// SourceFile records content and resolved paths plus the dependency edges.
type SourceFile struct {
	Path         string   `json:"path"`
	Resolved     string   `json:"resolved"`
	SHA256       string   `json:"sha256"`
	Bytes        int64    `json:"bytes"`
	Dependencies []string `json:"dependencies"`
}

// Manifest is an observed input graph, not a claim that later reads cannot
// change. Runtime execution must verify it again under its attempt fence.
type Manifest struct {
	Roots []string     `json:"roots"`
	Files []SourceFile `json:"files"`
}

// SourceManifest walks and hashes the complete dependency graph. Hashing uses
// a streaming read, so a terrain file is not copied wholesale into Go memory.
// Missing files, unknown dependencies, cycles and files changed during
// inspection are errors, never a reusable partial source key.
func SourceManifest(ctx context.Context, roots []string, resolve Resolver) (Manifest, string, error) {
	if resolve == nil || len(roots) == 0 {
		return Manifest{}, "", ErrIncomplete
	}
	normal, err := absolutePaths("", roots)
	if err != nil {
		return Manifest{}, "", err
	}
	walk := sourceWalk{ctx: ctx, resolve: resolve, files: make(map[string]SourceFile), active: make(map[string]bool)}
	for _, path := range normal {
		if err := walk.visit(path); err != nil {
			return Manifest{}, "", err
		}
	}
	manifest := Manifest{Roots: normal}
	for _, file := range walk.files {
		manifest.Files = append(manifest.Files, file)
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	raw, err := json.Marshal(struct {
		Version  int
		Manifest Manifest
	}{1, manifest})
	if err != nil {
		return Manifest{}, "", err
	}
	return manifest, fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}

// sourceWalk retains only metadata, not source payloads or open descriptors.
type sourceWalk struct {
	ctx     context.Context
	resolve Resolver
	files   map[string]SourceFile
	active  map[string]bool
}

// visit refuses dependency cycles, including cycles through symbolic aliases.
func (walk *sourceWalk) visit(path string) error {
	if err := walk.ctx.Err(); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if walk.active[resolved] {
		return fmt.Errorf("%w: dependency cycle at %s", ErrIncomplete, path)
	}
	if _, seen := walk.files[path]; seen {
		return nil
	}
	walk.active[resolved] = true
	defer delete(walk.active, resolved)
	record, err := walk.inspect(path, resolved)
	if err != nil {
		return err
	}
	for _, next := range record.Dependencies {
		if err := walk.visit(next); err != nil {
			return err
		}
	}
	walk.files[path] = record
	return nil
}

// inspect hashes the same open file the resolver inspected, checking mutation.
func (walk *sourceWalk) inspect(path, resolved string) (SourceFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return SourceFile{}, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return SourceFile{}, fmt.Errorf("%w: nonregular/unreadable %s", ErrIncomplete, path)
	}
	dependencies, err := walk.resolve(walk.ctx, path, file)
	if err != nil {
		return SourceFile{}, err
	}
	if !dependencies.Complete {
		return SourceFile{}, fmt.Errorf("%w: %s", ErrIncomplete, path)
	}
	paths, err := absolutePaths(filepath.Dir(path), dependencies.Files)
	if err != nil {
		return SourceFile{}, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return SourceFile{}, err
	}
	hash := sha256.New()
	count, err := io.Copy(hash, contextReader{walk.ctx, file})
	if err != nil {
		return SourceFile{}, err
	}
	after, err := file.Stat()
	if err != nil {
		return SourceFile{}, err
	}
	current, err := os.Stat(path)
	if err != nil {
		return SourceFile{}, err
	}
	currentTarget, err := filepath.EvalSymlinks(path)
	if err != nil {
		return SourceFile{}, err
	}
	if currentTarget != resolved || !os.SameFile(before, current) || before.Size() != after.Size() || count != after.Size() || current.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || !current.ModTime().Equal(after.ModTime()) {
		return SourceFile{}, fmt.Errorf("%w: changed source %s", ErrIncomplete, path)
	}
	return SourceFile{path, resolved, fmt.Sprintf("%x", hash.Sum(nil)), count, paths}, nil
}

// absolutePaths sorts and deduplicates paths without guessing a root directory.
func absolutePaths(base string, paths []string) ([]string, error) {
	unique := make(map[string]bool)
	for _, path := range paths {
		if path == "" {
			return nil, ErrIncomplete
		}
		if !filepath.IsAbs(path) {
			if base == "" {
				return nil, ErrIncomplete
			}
			path = filepath.Join(base, path)
		}
		unique[filepath.Clean(path)] = true
	}
	result := make([]string, 0, len(unique))
	for path := range unique {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

// contextReader permits cancellation while hashing large external assets.
type contextReader struct {
	ctx    context.Context
	source io.Reader
}

// Read returns the context error when the context is done, and otherwise
// reads from the source.
func (reader contextReader) Read(into []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.source.Read(into)
}
