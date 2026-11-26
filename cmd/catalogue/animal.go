package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/manufacture"
)

// animalSpec identifies a checked species in a model book. File is relative
// to an explicitly supplied model directory; without one it must name the
// corresponding embedded default book.
type animalSpec struct {
	Family         string   `json:"family" yaml:"family"`
	File           string   `json:"file" yaml:"file"`
	Species        string   `json:"species" yaml:"species"`
	Pose           string   `json:"pose,omitempty" yaml:"pose"`
	Count          *int     `json:"count,omitempty" yaml:"count"`
	Seconds        *float64 `json:"seconds,omitempty" yaml:"seconds"`
	Phase          *float64 `json:"phase,omitempty" yaml:"phase"`
	Representation *string  `json:"representation,omitempty" yaml:"representation"`
	Study          *string  `json:"study,omitempty" yaml:"study"`
}

type modelBooks struct {
	root    string
	bytes   map[string][]byte
	animals map[string]*manufacture.AnimalBook
	plants  map[string]*manufacture.PlantBook
	algae   map[string]*manufacture.KelpProfiles
}

// read snapshots a bounded source once per menu, shared by both families.
func (books *modelBooks) read(category, family, file string) ([]byte, error) {
	if books.bytes == nil {
		books.bytes = map[string][]byte{}
	}
	name := filepath.Clean(file)
	if file == "" || filepath.IsAbs(file) || name == "." || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid model file %q", file)
	}
	key := category + "\x00" + family + "\x00" + name
	if books.root != "" {
		key = name
	}
	if raw, ok := books.bytes[key]; ok {
		return raw, nil
	}
	var raw []byte
	if books.root == "" {
		var defaultName string
		var err error
		switch category {
		case "animal":
			defaultName, raw, err = manufacture.EmbeddedAnimalBook(family)
		case "plant":
			defaultName, raw, err = manufacture.EmbeddedPlantBook(family)
		case "alga":
			defaultName, raw, err = manufacture.EmbeddedAlgaBook(family)
		default:
			return nil, fmt.Errorf("unknown model category %q", category)
		}
		if err != nil {
			return nil, err
		}
		if name != defaultName {
			return nil, fmt.Errorf("model %q requires an explicit models directory", file)
		}
	} else {
		path := filepath.Join(books.root, name)
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("model %s: %w", path, err)
		}
		raw, err = io.ReadAll(io.LimitReader(f, (1<<20)+1))
		closeErr := f.Close()
		if err != nil {
			return nil, fmt.Errorf("model %s: %w", path, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("model %s: %w", path, closeErr)
		}
		if len(raw) > 1<<20 {
			return nil, fmt.Errorf("model %s exceeds 1 MiB", path)
		}
	}
	books.bytes[key] = raw
	return raw, nil
}

// cachedModelBook keeps one checked book per family and cleaned source path.
func cachedModelBook[T any](cache *map[string]*T, family, file string, raw []byte, read func(string, string, []byte) (*T, error)) (*T, error) {
	if *cache == nil {
		*cache = make(map[string]*T)
	}
	key := family + "\x00" + filepath.Clean(file)
	if book, ok := (*cache)[key]; ok {
		return book, nil
	}
	book, err := read(family, file, raw)
	if err != nil {
		return nil, err
	}
	(*cache)[key] = book
	return book, nil
}

func (spec animalSpec) construct(books *modelBooks, raw []byte) (manufacture.Kind, error) {
	book, err := cachedModelBook(&books.animals, spec.Family, spec.File, raw, manufacture.ReadAnimalBook)
	if err != nil {
		return manufacture.Kind{}, err
	}
	options := manufacture.AnimalOptions{Species: spec.Species, Count: spec.Count, Seconds: spec.Seconds, Phase: spec.Phase,
		Representation: spec.Representation, Study: spec.Study}
	if spec.Pose != "" {
		options.Pose = &spec.Pose
	}
	return book.New(options)
}

// bindAnimal freezes a checked constructor and the identity of everything
// about this card that can affect its render, including exact model bytes.
func (entry *entry) bindAnimal(books *modelBooks) error {
	if entry.Animal == nil {
		return nil
	}
	raw, err := books.read("animal", entry.Animal.Family, entry.Animal.File)
	if err != nil {
		return err
	}
	kind, err := entry.Animal.construct(books, raw)
	if err != nil {
		return err
	}
	return entry.bindSource("animal", raw, kind)
}

// bindSource freezes the checked constructor and exact source identity.
func (entry *entry) bindSource(category string, raw []byte, kind manufacture.Kind) error {
	metadata, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	model := sha256.Sum256(raw)
	hash := sha256.New()
	hash.Write([]byte("catalogue-" + category + "-v1\x00"))
	hash.Write(metadata)
	hash.Write(model[:])
	entry.source = hex.EncodeToString(hash.Sum(nil))
	entry.checked = &kind
	return nil
}

func (entry entry) kind() (manufacture.Kind, error) {
	if entry.Animal != nil || entry.Plant != nil || entry.Alga != nil {
		if entry.checked == nil {
			return manufacture.Kind{}, fmt.Errorf("catalogue model %q has not been checked", entry.Kind)
		}
		return *entry.checked, nil
	}
	return manufacture.KindNamed(entry.Kind)
}
