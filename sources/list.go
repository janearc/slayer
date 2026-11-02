// Package sources fetches and verifies the published works cited by Slayer. A
// repository manifest records origins, licences, and SHA-256 hashes; downloaded
// bytes live on a shelf outside the repository.
//
// A hash mismatch rejects the download. A source without a pinned hash can be
// fetched and its hash reported for manual review and pinning. Entries that
// must not be fetched again remain as tombstones with a reason. The original
// plan is vanguard’s 2026-10-15-citations-fetch.md.
package sources

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"

	"go.yaml.in/yaml/v3"
)

// row is one source as the list writes it.
type row struct {
	Key     string `yaml:"key"`
	Title   string `yaml:"title"`
	By      string `yaml:"by"`
	Year    int    `yaml:"year"`
	From    string `yaml:"from"`
	DOI     string `yaml:"doi"`
	Licence string `yaml:"licence"`
	Sha256  string `yaml:"sha256"`
	Bytes   int64  `yaml:"bytes"`
	Fetched string `yaml:"fetched"`
	Never   string `yaml:"never"`
}

// A Source is one published work in the list.
type Source struct{ row row }

// Key is the name the source is cited by, as fermat and the code say it.
func (source Source) Key() string { return source.row.Key }

// From is where its bytes are fetched from.
func (source Source) From() string { return source.row.From }

// Licence is what the source may be used under, as the list says.
func (source Source) Licence() string { return source.row.Licence }

// Pinned is whether the list holds the source's sha256.
func (source Source) Pinned() bool { return source.row.Sha256 != "" }

// Never is why the source must not be fetched, or nothing.
func (source Source) Never() string { return source.row.Never }

// A List is the sources a repository cites, in the order it lists them.
type List struct{ sources []Source }

// Sources is every source in the list, tombstones too.
func (list List) Sources() []Source { return list.sources }

// Find is the source of a key.
func (list List) Find(key string) (Source, bool) {
	for _, source := range list.sources {
		if source.Key() == key {
			return source, true
		}
	}
	return Source{}, false
}

var errList = errors.New("sources: a list that cannot be")

// shapes the list's values must take: a key of lowercase letters,
// digits and hyphens, and a sha256 of 64 lowercase hex digits.
var (
	keyShape  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	hashShape = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Read is the list in a file. it refuses a key it does not know, a
// source with no key or a key given twice, one with no from or licence
// unless it is a tombstone, and a sha256 that is not one.
func Read(path string) (List, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return List{}, err
	}
	var file struct {
		Sources []row `yaml:"sources"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(text))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return List{}, fmt.Errorf("%s: %w", path, err)
	}
	list, seen := List{}, map[string]bool{}
	for _, each := range file.Sources {
		if err := sound(each, seen); err != nil {
			return List{}, fmt.Errorf("%s: %w", path, err)
		}
		seen[each.Key] = true
		list.sources = append(list.sources, Source{row: each})
	}
	return list, nil
}

// sound is what is wrong with one row, given the keys before it.
func sound(each row, seen map[string]bool) error {
	switch {
	case !keyShape.MatchString(each.Key):
		return fmt.Errorf("%w: the key %q", errList, each.Key)
	case seen[each.Key]:
		return fmt.Errorf("%w: %s is listed twice", errList, each.Key)
	case each.Never != "":
		return nil
	case each.From == "" || each.Licence == "":
		return fmt.Errorf("%w: %s has no from or no licence", errList,
			each.Key)
	case each.Sha256 != "" && !hashShape.MatchString(each.Sha256):
		return fmt.Errorf("%w: %s's sha256 %q", errList, each.Key,
			each.Sha256)
	}
	return nil
}
