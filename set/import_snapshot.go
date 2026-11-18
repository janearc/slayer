package set

import (
	"fmt"
	"path/filepath"
)

// ReadImported reads a set only when its world declares imports. The flag
// identifies that declaration even when validation or opening an asset fails.
func ReadImported(file string, source []byte) (Set, bool, error) {
	top, err := parse(file, source)
	if err != nil {
		return Set{}, false, err
	}
	world := valueNode(valueNode(top, "scene"), "world")
	if !says(world, "imports") {
		return Set{}, false, nil
	}
	read, err := Read(file, source)
	return read, true, err
}

// ImportSources is the exact decoded closure, by absolute path, in a fresh map.
func (made Set) ImportSources() map[string][32]byte {
	sources := map[string][32]byte{}
	if made.world == nil {
		return sources
	}
	for _, part := range made.world.imports {
		for _, source := range part.asset.SourceClosure() {
			sources[filepath.Join(part.root, filepath.FromSlash(source.Path()))] = source.SHA256()
		}
	}
	return sources
}

// consistentImports refuses a source changed between two policy-specific reads.
func (reading reader) consistentImports() error {
	known := map[string][32]byte{}
	for _, part := range reading.set.world.imports {
		for _, source := range part.asset.SourceClosure() {
			path := filepath.Join(part.root, filepath.FromSlash(source.Path()))
			if prior, has := known[path]; has && prior != source.SHA256() {
				return fmt.Errorf("%s: imported source changed while reading: %s", reading.file, path)
			}
			known[path] = source.SHA256()
		}
	}
	return nil
}
