package set

import (
	"fmt"
	yaml "gopkg.in/yaml.v3"
	"path/filepath"
)

// ReadImported reads a set when it declares imports or a composition. The
// flag identifies the declaration even when validation or opening a source
// fails, so callers cannot reuse a stale cached result.
func ReadImported(file string, source []byte) (Set, bool, error) {
	top, err := parse(file, source)
	if err != nil {
		return Set{}, false, err
	}
	world := valueNode(valueNode(top, "scene"), "world")
	if !says(world, "imports") && !hasStructureComposition(top) && !hasModelProfiles(top) {
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

// hasStructureComposition identifies recipe sources that must be frozen with
// the set before job keys and resume decisions are made.
func hasStructureComposition(top *yaml.Node) bool {
	recipes := valueNode(valueNode(top, "scene"), "recipes")
	if recipes == nil {
		return false
	}
	for i := 1; i < len(recipes.Content); i += 2 {
		if says(recipes.Content[i], "composition") {
			return true
		}
	}
	return false
}

// CompositionSources returns external files exactly as read, rejecting a
// file changed between two recipes in the same snapshot.
func (made Set) CompositionSources() (map[string][32]byte, error) {
	sources := make(map[string][32]byte)
	for _, recipe := range made.recipes {
		if recipe.compositionPath == "" {
			continue
		}
		if prior, ok := sources[recipe.compositionPath]; ok && prior != recipe.compositionDigest {
			return nil, fmt.Errorf("composition source changed while reading: %s", recipe.compositionPath)
		}
		sources[recipe.compositionPath] = recipe.compositionDigest
	}
	return sources, nil
}

// hasPlantProfiles identifies external plant books before render-key lookup.
func hasModelProfiles(top *yaml.Node) bool {
	recipes := valueNode(valueNode(top, "scene"), "recipes")
	if recipes == nil {
		return false
	}
	for i := 1; i < len(recipes.Content); i += 2 {
		if says(recipes.Content[i], "plant") || says(recipes.Content[i], "animal") || says(recipes.Content[i], "alga") || says(recipes.Content[i], "tree-profiles") {
			return true
		}
	}
	return false
}

// ProfileSources returns exactly the external model books used by the set.
func (made Set) ProfileSources() (map[string][32]byte, error) {
	sources := make(map[string][32]byte)
	for _, recipe := range made.recipes {
		if recipe.profilePath == "" {
			continue
		}
		if prior, ok := sources[recipe.profilePath]; ok && prior != recipe.profileDigest {
			return nil, fmt.Errorf("profile source changed while reading: %s", recipe.profilePath)
		}
		sources[recipe.profilePath] = recipe.profileDigest
	}
	return sources, nil
}

// BuiltinProfileSources names embedded books and their actual byte digests.
func (made Set) BuiltinProfileSources() map[string][32]byte {
	sources := make(map[string][32]byte)
	for _, recipe := range made.recipes {
		if recipe.profileBuiltin != "" {
			sources[recipe.profileBuiltin] = recipe.profileDigest
		}
	}
	return sources
}
