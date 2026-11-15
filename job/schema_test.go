package job

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// tree is the schema decoded, for walking.
func tree(t *testing.T) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(Schema(), &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

// walk calls visit with every object in the schema and its place. An
// if is a condition on a shape, not a shape, and is not walked.
func walk(node any, where string, visit func(map[string]any, string)) {
	switch value := node.(type) {
	case map[string]any:
		visit(value, where)
		for key, child := range value {
			if key != "if" {
				walk(child, where+"/"+key, visit)
			}
		}
	case []any:
		for i, child := range value {
			walk(child, fmt.Sprintf("%s/%d", where, i), visit)
		}
	}
}

// TestEveryPropertyHasADescription holds that the editor has words to
// show for every key, and that they say something.
func TestEveryPropertyHasADescription(t *testing.T) {
	counted := nothing
	walk(tree(t), "", func(node map[string]any, where string) {
		properties, has := node["properties"].(map[string]any)
		if !has {
			return
		}
		for key, value := range properties {
			counted++
			words, _ := value.(map[string]any)["description"].(string)
			if len(strings.Fields(words)) < len(strings.Fields("a key")) {
				t.Errorf("%s/properties/%s has no description", where,
					key)
			}
		}
	})
	if counted == nothing {
		t.Fatal("the schema has no properties")
	}
}

// TestAPropertyWithoutADescriptionIsFound is the anti-test: the walk
// finds a property with no description where one is hidden.
// anti-test of TestEveryPropertyHasADescription
func TestAPropertyWithoutADescriptionIsFound(t *testing.T) {
	hidden := map[string]any{"$defs": map[string]any{"deep": []any{
		map[string]any{"properties": map[string]any{"bare": map[string]any{
			"type": "string"}}}}}}
	found := ""
	walk(hidden, "", func(node map[string]any, where string) {
		if properties, has := node["properties"].(map[string]any); has {
			for key := range properties {
				found = where + "/" + key
			}
		}
	})
	if found != "/$defs/deep/0/bare" {
		t.Errorf("the walk found %q", found)
	}
}

// open is the one object a job's schema leaves open: a named scene's
// parameters, which the scene checks.
const open = "/properties/scene/properties/with"

// TestEveryObjectIsClosed holds that an object with properties refuses a
// key it does not list, and that only with is left open.
func TestEveryObjectIsClosed(t *testing.T) {
	walk(tree(t), "", func(node map[string]any, where string) {
		_, listed := node["properties"]
		closed, said := node["additionalProperties"].(bool)
		if listed && (!said || closed) {
			t.Errorf("%s takes keys it does not list", where)
		}
		if node["type"] == "object" && !listed && where != open &&
			node["additionalProperties"] == nil {
			t.Errorf("%s is an object with no properties", where)
		}
	})
}

// enum is the schema's enum at a path of properties, each written as
// fmt writes it, without repeats, in order.
func enum(t *testing.T, path ...string) []string {
	t.Helper()
	node := tree(t)
	for _, key := range path {
		node = node["properties"].(map[string]any)[key].(map[string]any)
	}
	var names []string
	for _, value := range node["enum"].([]any) {
		name := fmt.Sprint(value)
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

// presetRow is a row of the README's presets table: a name, then a size.
var presetRow = regexp.MustCompile("(?m)^\\| `([a-z0-9-]+)` \\| [0-9]+ by ")

// tabledPresets is the preset names a README's text tables, in order.
func tabledPresets(text string) []string {
	var tabled []string
	for _, match := range presetRow.FindAllStringSubmatch(text, -one) {
		tabled = append(tabled, match[one])
	}
	return tabled
}

// TestThePresetsAreOneTable holds the schema's preset enum, presets.yaml
// and the README's table to the same names in the same order.
func TestThePresetsAreOneTable(t *testing.T) {
	var shelved []string
	for _, each := range Presets() {
		shelved = append(shelved, each.Name())
	}
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	tabled := tabledPresets(string(readme))
	schemaNames := enum(t, "film", "preset")
	if !slices.Equal(shelved, schemaNames) || !slices.Equal(shelved, tabled) {
		t.Errorf("presets.yaml %v, schema %v, README %v", shelved,
			schemaNames, tabled)
	}
}

// TestEachPresetSaysItsSizeAndWhoPickedIt holds every preset to a size,
// a source, and a rate only where the table gives one.
func TestEachPresetSaysItsSizeAndWhoPickedIt(t *testing.T) {
	previews := []string{"portrait-preview", "video-preview",
		"video-preview-720"}
	for _, each := range Presets() {
		width, height := each.Size()
		_, rated := each.Rate()
		if width == nothing || height == nothing || each.By() == "" {
			t.Errorf("preset %s is missing its size or its source",
				each.Name())
		}
		if rated != slices.Contains(previews, each.Name()) {
			t.Errorf("preset %s has a rate %v", each.Name(), rated)
		}
	}
}

// TestAPresetsTableWithAStrangeKeyIsRefused is the anti-test of the
// table's reading.
// anti-test of TestEachPresetSaysItsSizeAndWhoPickedIt
func TestAPresetsTableWithAStrangeKeyIsRefused(t *testing.T) {
	_, err := readPresets("presets.yaml",
		[]byte("presets:\n  - name: a\n    widht: 3\n"))
	if err == nil || !strings.Contains(err.Error(), "widht") {
		t.Errorf("a misspelt preset key read as %v", err)
	}
}

// TestTheStocksAreTheShelf holds the schema's stocks to the README's.
func TestTheStocksAreTheShelf(t *testing.T) {
	shelf := []string{"soot", "ash", "smoke", "dusk", "pollen", "lantern"}
	if got := enum(t, "stock", "name"); !slices.Equal(got, shelf) {
		t.Errorf("the schema's stocks are %v, want %v", got, shelf)
	}
}

// TestSchemaIsACopy holds that a caller cannot change the schema read
// through.
func TestSchemaIsACopy(t *testing.T) {
	first := Schema()
	first[nothing] = ' '
	if Schema()[nothing] == ' ' {
		t.Error("changing Schema's bytes changed the schema")
	}
}
