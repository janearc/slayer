package preview

import (
	"fmt"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
)

// ResolveSpecimen uses an explicit selection or the adjacent render receipt.
func ResolveSpecimen(request Request) (string, error) {
	name := request.Specimen
	if name == "" {
		path := filepath.Join(filepath.Dir(request.Path), "render.json")
		var err error
		name, err = receiptSpecimen(path)
		if err != nil {
			return "", fmt.Errorf("select a specimen before rendering: %w", err)
		}
	}
	if err := validSpecimen(name); err != nil {
		return "", err
	}
	return name, nil
}

// receiptSpecimen bounds the companion read and ignores unrelated receipt fields.
func receiptSpecimen(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	const limit = 4 * 1024 * 1024
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return "", err
	}
	if len(raw) > limit {
		return "", fmt.Errorf("render receipt exceeds 4 MiB")
	}
	// Only specimen identity is consumed. Other receipt fields belong to its producer.
	schema, err := yamlfile.NewSchema("atelier:catalogue-receipt", []byte(`{"type":"object","required":["specimen"],"properties":{"specimen":{"type":"object","required":["entry"],"properties":{"entry":{"type":"object","required":["kind"],"properties":{"kind":{"type":"string","minLength":1}}}}}}}`))
	if err != nil {
		return "", err
	}
	tree, err := schema.Read(path, raw)
	if err != nil {
		return "", err
	}
	return field(field(field(tree.Content[0], "specimen"), "entry"), "kind").Value, nil
}

// catalogueSnapshot keeps authored lighting and camera values in a single small view.
func catalogueSnapshot(tree *yaml.Node, profile Profile) string {
	root := tree.Content[0]
	field(root, "width").Value = strconv.Itoa(profile.Width)
	field(root, "height").Value = strconv.Itoa(profile.Height)
	field(root, "rays").Value = "1"
	details := field(root, "details")
	detail := details.Content[0].Value
	details.Content = details.Content[:1]
	return detail
}

// validSpecimen accepts exactly one kind in the producer's checked menu.
func validSpecimen(name string) error {
	names, err := Specimens()
	if err != nil {
		return err
	}
	if !slices.Contains(names, name) {
		return fmt.Errorf("select a specimen from the catalogue before rendering (got %q)", name)
	}
	return nil
}

// Specimens returns the named kinds supported by the pinned catalogue producer.
func Specimens() ([]string, error) {
	entries, err := catalogueEntries()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries.Content {
		names = append(names, field(entry, "kind").Value)
	}
	return names, nil
}

// CatalogueProfile describes a checked catalogue entry backed by an external
// animal or plant book. Spec is a fresh checked YAML mapping for that entry.
type CatalogueProfile struct {
	Name, Kind, Group, About string
	Spec                     *yaml.Node
}

var cachedCatalogueProfiles = sync.OnceValues(readCatalogueProfiles)

// CatalogueProfiles lists independent copies of the checked profile choices.
func CatalogueProfiles() ([]CatalogueProfile, error) {
	profiles, err := cachedCatalogueProfiles()
	if err != nil {
		return nil, err
	}
	result := make([]CatalogueProfile, len(profiles))
	for index, profile := range profiles {
		result[index] = profile
		result[index].Spec = cloneCatalogueNode(profile.Spec)
	}
	return result, nil
}

// cloneCatalogueNode keeps one caller's edits out of the cached menu.
func cloneCatalogueNode(node *yaml.Node) *yaml.Node {
	copy := *node
	copy.Content = make([]*yaml.Node, len(node.Content))
	for index, child := range node.Content {
		copy.Content[index] = cloneCatalogueNode(child)
	}
	return &copy
}

// readCatalogueProfiles checks the embedded menu once for all workspaces.
func readCatalogueProfiles() ([]CatalogueProfile, error) {
	entries, err := catalogueEntries()
	if err != nil {
		return nil, err
	}
	var profiles []CatalogueProfile
	for _, entry := range entries.Content {
		for _, kind := range []string{"animal", "plant"} {
			if spec := field(entry, kind); spec != nil {
				profiles = append(profiles, CatalogueProfile{Name: field(entry, "kind").Value, Kind: kind, Group: field(entry, "group").Value, About: field(entry, "about").Value, Spec: spec})
			}
		}
	}
	return profiles, nil
}

// catalogueEntries reads the pinned producer menu through its checked schema.
func catalogueEntries() (*yaml.Node, error) {
	raw, err := assets.Read("catalogue-menu.yaml")
	if err != nil {
		return nil, err
	}
	contract, err := assets.Read("catalogue-menu.schema.json")
	if err != nil {
		return nil, err
	}
	schema, err := yamlfile.NewSchema("atelier:catalogue-menu", contract)
	if err != nil {
		return nil, err
	}
	tree, err := schema.Read("catalogue-menu.yaml", raw)
	if err != nil {
		return nil, err
	}
	entries := field(tree.Content[0], "entries")
	if entries == nil {
		return nil, fmt.Errorf("catalogue has no entries")
	}
	return entries, nil
}

// field reads a mapping already checked at the Puzzlemouth boundary.
func field(root *yaml.Node, name string) *yaml.Node {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == name {
			return root.Content[index+1]
		}
	}
	return nil
}
