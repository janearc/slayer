package set_test

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// v2Name is the v2 schema's identity, as its $id says.
const v2Name = "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"

// v2Drafts are the scenes and the example edit written against v2, ahead
// of the reader (2026-11-09).
func v2Drafts() []string {
	return []string{
		"keyhole-to-ground.yaml",
		"webb-looking-out.yaml",
		"corona.yaml",
		"orbital-and-lantern.yaml",
		"music-timeline.yaml",
	}
}

// v2Schema is the compiled v2 schema, or the test stops.
func v2Schema(t *testing.T) *yamlfile.Schema {
	t.Helper()
	raw, err := os.ReadFile("ecosystem-v2.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := yamlfile.NewSchema(v2Name, raw)
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

// draft is one of the drafts, read from doc, or the test stops.
func draft(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "doc", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Every draft written against v2 is one v2 takes, key for key.
func TestV2TakesEveryDraft(t *testing.T) {
	schema := v2Schema(t)
	for _, name := range v2Drafts() {
		if _, err := schema.Read(name, draft(t, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Anti-test: v2 refuses a key no schema has, a v2 key in the wrong
// place, and a value outside its kind, so taking the drafts is a check.
func TestV2RefusesWhatItDoesNotSay(t *testing.T) {
	schema := v2Schema(t)
	keyhole := string(draft(t, "keyhole-to-ground.yaml"))
	for why, from := range map[string][2]string{
		"an unknown key": {"        orbit:\n", "        orbits:\n"},
		"a date that is not a day": {`date: "2026-06-21"`,
			`date: "the solstice"`},
		"an eye that is no instrument": {
			"eye: {instrument: the keyhole}", "eye: {glass: the keyhole}"},
		"a move with no look": {
			"          look: {object: the keyhole}\n          duration: {value: 6",
			"          duration: {value: 6"},
		"a placement for no thing": {"        place: {at: anchor}",
			"        place: {at: elsewhere}"},
	} {
		changed := strings.Replace(keyhole, from[0], from[1], 1)
		if changed == keyhole {
			t.Fatalf("%s: the draft no longer holds %q", why, from[0])
		}
		if _, err := schema.Read("changed.yaml", []byte(changed)); err == nil {
			t.Errorf("%s was taken", why)
		}
	}
}

// propertyPaths is every place a schema names a key, as a path through
// its properties, definitions and branches, so two schemas can be
// compared key for key.
func propertyPaths(node any, path string, into map[string]bool) {
	switch value := node.(type) {
	case map[string]any:
		for key, child := range value {
			switch key {
			case "properties":
				names, _ := child.(map[string]any)
				for name, property := range names {
					into[path+"/"+name] = true
					propertyPaths(property, path+"/"+name, into)
				}
			case "$defs":
				definitions, _ := child.(map[string]any)
				for name, definition := range definitions {
					propertyPaths(definition, "#"+name, into)
				}
			default:
				propertyPaths(child, path, into)
			}
		}
	case []any:
		for _, child := range value {
			propertyPaths(child, path, into)
		}
	}
}

// schemaPaths is a schema file's property paths, or the test stops.
func schemaPaths(t *testing.T, file string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var schema any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	propertyPaths(schema, "", paths)
	return paths
}

// v2 keeps every key v1 has, where v1 has it, so a v1 scene says the same
// thing in v2 and the reader grows into v2 one key at a time.
func TestV2KeepsEveryV1Key(t *testing.T) {
	v1 := schemaPaths(t, "ecosystem.schema.json")
	v2 := schemaPaths(t, "ecosystem-v2.schema.json")
	var lost []string
	for path := range v1 {
		if !v2[path] {
			lost = append(lost, path)
		}
	}
	slices.Sort(lost)
	if len(lost) > 0 {
		t.Errorf("v2 lost %d of v1's keys: %v", len(lost), lost)
	}
	if len(v2) <= len(v1) {
		t.Errorf("v2 names %d keys and v1 %d: nothing was added", len(v2),
			len(v1))
	}
}
