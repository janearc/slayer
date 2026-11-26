package set

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"

	yaml "gopkg.in/yaml.v3"
)

// Digest names a recipe by what the file said of it: the SHA-256, in
// hexadecimal, of the recipe's block as the reader read it. Two recipes
// written alike have one digest and any edit to a recipe's block changes
// it, so a holder of work made from a recipe can tell a stale copy. It is
// empty for a recipe that did not come from a file.
func (recipe Recipe) Digest() string { return recipe.digest }

// digested wraps the step that reads one recipe so that the recipe it
// stored is stamped with the digest of the block it came from.
func (reading reader) digested(read func(name, block *yaml.Node) error,
) func(name, block *yaml.Node) error {
	return func(name, block *yaml.Node) error {
		if err := read(name, block); err != nil {
			return err
		}
		stored := reading.set.recipes[name.Value]
		stored.digest = digestOf(block)
		if stored.profilePath != "" || stored.profileBuiltin != "" {
			sum := sha256.Sum256([]byte(stored.digest + hex.EncodeToString(stored.profileDigest[:])))
			stored.digest = hex.EncodeToString(sum[:])
		}
		if stored.composition != nil {
			sum := sha256.Sum256([]byte(stored.digest + hex.EncodeToString(stored.compositionDigest[:])))
			stored.digest = hex.EncodeToString(sum[:])
		}
		reading.set.recipes[name.Value] = stored
		return nil
	}
}

// digestOf is the hexadecimal SHA-256 of a YAML node's content: its kind,
// tag and value and, in order, everything under it. Where the node sits
// in the file, and any comment on it, are left out.
func digestOf(node *yaml.Node) string {
	hasher := sha256.New()
	writeNode(hasher, node)
	return hex.EncodeToString(hasher.Sum(nil))
}

// writeNode feeds a node and what is under it to a hash. Each piece is
// quoted and counted, so no two different nodes write the same bytes.
func writeNode(hasher hash.Hash, node *yaml.Node) {
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		writeNode(hasher, node.Alias)
		return
	}
	fmt.Fprintf(hasher, "%d %q %q %d\n", node.Kind, node.ShortTag(),
		node.Value, len(node.Content))
	for _, child := range node.Content {
		writeNode(hasher, child)
	}
}
