package manufacture

import (
	"reflect"
	"strings"
	"testing"
)

func TestFernProfilesConstructCheckedNamedFern(t *testing.T) {
	book, err := ReadFernProfiles("ferns.yaml", fernProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewGenericFern()
	if err != nil {
		t.Fatal(err)
	}
	if row.name != "fern" {
		t.Fatalf("kind %q", row.name)
	}
	if _, err := Make(row, Sow(5)); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewFern("absent"); err == nil {
		t.Fatal("accepted absent profile")
	}
	if _, err := (*FernProfiles)(nil).NewFern("fern"); err == nil {
		t.Fatal("accepted nil book")
	}
}

func TestFernProfilesRejectInvalidBooksAndDimensions(t *testing.T) {
	base := string(fernProfilesRaw)
	for name, raw := range map[string]string{
		"empty":              "",
		"unknown field":      base + "    impossible: true\n",
		"no provenance":      strings.Replace(base, "by: fern scenery study", "by: ''", 1),
		"missing source":     strings.Replace(base, "status: picked", "status: read", 1),
		"negative size":      strings.Replace(base, "minimum: 0.4", "minimum: -0.4", 1),
		"inverted size":      strings.Replace(base, "maximum: 0.9", "maximum: 0.1", 1),
		"nonfinite size":     strings.Replace(base, "minimum: 0.4", "minimum: .inf", 1),
		"zero fronds":        strings.Replace(base, "minimum: 6", "minimum: 0", 1),
		"excessive leaflets": strings.Replace(base, "maximum: 24", "maximum: 1000000", 1),
		"invalid colour":     strings.Replace(base, "r: 0.2", "r: 2", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadFernProfiles("ferns.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted invalid fern book")
			}
		})
	}
	if _, err := ReadFernProfiles("ferns.yaml", make([]byte, (1<<20)+1)); err == nil {
		t.Fatal("accepted oversized book")
	}
}

func TestFernProfileBytesBelongToTheBook(t *testing.T) {
	raw := append([]byte(nil), fernProfilesRaw...)
	book, err := ReadFernProfiles("ferns.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw {
		raw[i] = 'x'
	}
	if _, err := book.NewGenericFern(); err != nil {
		t.Fatal(err)
	}
}

func TestFernProfileRequiresEveryColourComponent(t *testing.T) {
	base := string(fernProfilesRaw)
	colour := "    colour: {r: 0.2, g: 0.5, b: 0.12}"
	if !strings.Contains(base, colour) {
		t.Fatal("fixture missing colour")
	}
	for name, replacement := range map[string]string{
		"missing colour": "", "null colour": "    colour: null",
		"missing r": "    colour: {g: 0.5, b: 0.12}",
		"missing g": "    colour: {r: 0.2, b: 0.12}",
		"missing b": "    colour: {r: 0.2, g: 0.5}",
		"null r":    "    colour: {r: null, g: 0.5, b: 0.12}",
		"null g":    "    colour: {r: 0.2, g: null, b: 0.12}",
		"null b":    "    colour: {r: 0.2, g: 0.5, b: null}",
	} {
		t.Run(name, func(t *testing.T) {
			raw := strings.Replace(base, colour, replacement, 1)
			if _, err := ReadFernProfiles("ferns.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted missing or null colour")
			}
		})
	}
	zero := strings.Replace(base, colour, "    colour: {r: 0, g: 0, b: 0}", 1)
	book, err := ReadFernProfiles("ferns.yaml", []byte(zero))
	if err != nil {
		t.Fatalf("valid zero colour: %v", err)
	}
	if _, err := book.NewGenericFern(); err != nil {
		t.Fatal(err)
	}
}

func TestFernProfilesRejectTrailingDocumentsAndMergeKeys(t *testing.T) {
	base := string(fernProfilesRaw)
	for name, raw := range map[string]string{
		"second document":           base + "\n---\nprofiles: []\n",
		"malformed second document": base + "\n---\nprofiles: [\n",
		"merge key":                 strings.Replace(base, "colour: {r: 0.2, g: 0.5, b: 0.12}", "colour: {<<: {r: 0.2}, g: 0.5, b: 0.12}", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if raw == base {
				t.Fatal("fixture was not changed")
			}
			if _, err := ReadFernProfiles("ferns.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted trailing document or merge")
			}
		})
	}
}

func TestFernProfilesRejectUnmeshableDimensions(t *testing.T) {
	base := string(fernProfilesRaw)
	for _, tc := range []struct{ name, from, to string }{
		{"tiny stalk", "stalk-radius: {minimum: 0.005, maximum: 0.008}", "stalk-radius: {minimum: 1e-300, maximum: 1e-300}"},
		{"tiny frond", "frond-length: {minimum: 0.4, maximum: 0.9}", "frond-length: {minimum: 1e-300, maximum: 1e-300}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.from) {
				t.Fatal("fixture absent")
			}
			if _, err := ReadFernProfiles("bad.yaml", []byte(strings.Replace(base, tc.from, tc.to, 1))); err == nil {
				t.Fatal("accepted unmeshable fern")
			}
		})
	}
}

func TestFernAnatomyComesFromTheCheckedProfile(t *testing.T) {
	base := string(fernProfilesRaw)
	originalBook, err := ReadFernProfiles("original.yaml", []byte(base))
	if err != nil {
		t.Fatal(err)
	}
	original, err := originalBook.NewGenericFern()
	if err != nil {
		t.Fatal(err)
	}
	before, err := original.how.(fern).made(Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, from, to string }{
		{"crown span", "crown-span: 0.03", "crown-span: 0.06"},
		{"crown radius", "crown-radius: 3", "crown-radius: 4"},
		{"stalk taper", "stalk-tip: 0.15", "stalk-tip: 0.3"},
		{"leaflet taper", "leaflet-tip: 0.2", "leaflet-tip: 0.4"},
		{"leaflet reach", "leaflet-reach: 0.22", "leaflet-reach: 0.3"},
		{"leaflet shortening", "leaflet-shortening: 0.85", "leaflet-shortening: 0.5"},
		{"leaflet girth", "leaflet-girth: 3", "leaflet-girth: 4"},
		{"leaflet minimum", "leaflet-minimum: 0.01", "leaflet-minimum: 0.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.from) {
				t.Fatal("fixture absent")
			}
			book, err := ReadFernProfiles("variant.yaml", []byte(strings.Replace(base, tc.from, tc.to, 1)))
			if err != nil {
				t.Fatal(err)
			}
			row, err := book.NewGenericFern()
			if err != nil {
				t.Fatal(err)
			}
			after, err := row.how.(fern).made(Sow(5))
			if err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(before.parts, after.parts) {
				t.Fatal("profile change did not affect geometry")
			}
		})
	}
}

func TestFernAnatomyRefusesMissingAndOutOfRangeValues(t *testing.T) {
	base := string(fernProfilesRaw)
	for _, tc := range []struct{ from, to string }{
		{"crown-span: 0.03", "crown-span: 0"},
		{"crown-radius: 3", "crown-radius: .inf"},
		{"stalk-tip: 0.15", "stalk-tip: -0.1"},
		{"leaflet-tip: 0.2", "leaflet-tip: 2"},
		{"leaflet-reach: 0.22", "leaflet-reach: 0"},
		{"leaflet-shortening: 0.85", "leaflet-shortening: 1"},
		{"leaflet-girth: 3", "leaflet-girth: null"},
		{"leaflet-minimum: 0.01", ""},
	} {
		if !strings.Contains(base, tc.from) {
			t.Fatalf("fixture absent: %s", tc.from)
		}
		if _, err := ReadFernProfiles("invalid.yaml", []byte(strings.Replace(base, tc.from, tc.to, 1))); err == nil {
			t.Fatalf("accepted %s", tc.to)
		}
	}
}
