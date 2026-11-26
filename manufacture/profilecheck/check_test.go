package profilecheck

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type numericRow struct {
	Values []struct{ Value float64 } `yaml:"values"`
	Triple [3]float64                `yaml:"triple"`
}

func TestProfileCheckRejectsWrongShapesAndDeepDocuments(t *testing.T) {
	typ := reflect.TypeOf(numericRow{})
	for name, raw := range map[string]string{
		"empty document":       "",
		"null document":        "---\n",
		"missing profiles":     "other: []\n",
		"mapping profiles":     "profiles: {}\n",
		"scalar values":        "profiles: [{values: 1, triple: [1,2,3]}]\n",
		"short array":          "profiles: [{values: [], triple: [1,2]}]\n",
		"scalar value entry":   "profiles: [{values: [1], triple: [1,2,3]}]\n",
		"missing nested value": "profiles: [{values: [{}], triple: [1,2,3]}]\n",
		"deeply nested":        "profiles:\n" + strings.Repeat("  - ", 102) + "1\n",
	} {
		t.Run(name, func(t *testing.T) {
			if err := NumbersPresent(name, []byte(raw), typ, nil); err == nil {
				t.Fatal("accepted malformed profile")
			}
		})
	}
	sections := []Section{{"figures", reflect.TypeOf(struct{ Value float64 }{})}}
	for _, raw := range []string{"other: {}\n", "species: []\n", "species: {fish: {figures: []}}\n", "species: {fish: {other: {}}}\n"} {
		if err := SpeciesNumbersPresent("species.yaml", []byte(raw), sections); err == nil {
			t.Fatalf("accepted malformed species: %q", raw)
		}
	}
	if Field(nil, "missing") != nil || Field(&yaml.Node{Kind: yaml.ScalarNode}, "missing") != nil {
		t.Fatal("found a field outside a mapping")
	}
}

func TestNumberCheckerGuardsRecursiveCalls(t *testing.T) {
	value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "1"}
	if err := checkProfileNumbers(value, reflect.TypeOf(float64(0)), value, "x", nil, 101); err == nil {
		t.Fatal("accepted excessively nested numeric input")
	}
	null := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	if err := checkProfileNumbers(null, reflect.TypeOf(float64(0)), null, "x", nil, 0); err == nil {
		t.Fatal("accepted null numeric input")
	}
}

func TestAuthoredProvenanceRequiresANameAuthorDateAndSourceWhenRead(t *testing.T) {
	for _, status := range []string{"picked", "read", "derived"} {
		if !Authored("fern", status, "observer", "2026", "source") {
			t.Fatalf("refused %s provenance", status)
		}
	}
	if !Authored("fern", "picked", "observer", "2026", "") {
		t.Fatal("picked data must not require a source")
	}
	for _, row := range [][5]string{
		{"", "picked", "observer", "2026", ""},
		{"fern", "picked", "", "2026", ""},
		{"fern", "picked", "observer", "", ""},
		{"fern", "unknown", "observer", "2026", "source"},
		{"fern", "read", "observer", "2026", ""},
		{"fern", "derived", "observer", "2026", ""},
	} {
		if Authored(row[0], row[1], row[2], row[3], row[4]) {
			t.Fatalf("accepted incomplete provenance %v", row)
		}
	}
}

func TestRowsKeepFamilyErrorsAndRejectIncompleteData(t *testing.T) {
	type row struct{ Size float64 }
	for _, tc := range []struct{ raw, want string }{
		{"", "book.yaml: empty or oversized whale book"},
		{"profiles: []\n", "book.yaml: invalid whale profile count"},
		{"profiles: [{}]\n", "missing numeric field size"},
		{"profiles: [{size: null}]\n", "null field"},
	} {
		_, err := Rows[row]("book.yaml", []byte(tc.raw), "whale book", "whale profile", nil)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("input %q: got %v, want %q", tc.raw, err, tc.want)
		}
	}
	rows, err := Rows[row]("book.yaml", []byte("profiles: [{size: 2}]\n"), "whale book", "whale profile", nil)
	if err != nil || len(rows) != 1 || rows[0].Size != 2 {
		t.Fatalf("valid profile: %+v, %v", rows, err)
	}
}
