package sauropods

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"
)

func checkedBook(t *testing.T) *Book {
	t.Helper()
	file, raw := Embedded()
	book, err := Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	return book
}

func TestSauropodBookOwnsFourDistinctReconstructions(t *testing.T) {
	file, raw := Embedded()
	book, err := Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alamosaurus-sanjuanensis", "apatosaurus-louisae", "brachiosaurus-altithorax", "diplodocus-carnegii"}
	if got := book.Species(); !reflect.DeepEqual(got, want) {
		t.Fatalf("species %v, want %v", got, want)
	}
	profiles := make(map[string]Profile)
	for _, name := range want {
		p, err := book.Profile(name)
		if err != nil {
			t.Fatal(err)
		}
		if p.Kind != name || p.Scientific == "" || p.Source == "" || p.Notes == "" || p.Basis != "picked" {
			t.Fatalf("missing identity or provenance: %+v", p)
		}
		profiles[name] = p
	}
	dip := profiles["diplodocus-carnegii"]
	apa := profiles["apatosaurus-louisae"]
	bra := profiles["brachiosaurus-altithorax"]
	// These comparisons concern the authored studies, not universal claims.
	if !(apa.Stations[8].Height > dip.Stations[8].Height && bra.Stations[bra.Fore.Station].Y > bra.Stations[bra.Hind.Station].Y) {
		t.Fatal("neck or shoulder profile distinction was lost")
	}
	for _, p := range profiles {
		for _, q := range profiles {
			if p.Kind != q.Kind && reflect.DeepEqual(p.Stations, q.Stations) {
				t.Fatalf("%s and %s share an identical outline", p.Kind, q.Kind)
			}
		}
	}
	before := profiles["alamosaurus-sanjuanensis"]
	changed, _ := book.Profile(before.Kind)
	changed.Stations[0].X = 10
	again, _ := book.Profile(before.Kind)
	if !reflect.DeepEqual(before, again) {
		t.Fatal("caller changed owned station data")
	}
	raw[0] ^= 1
	if again, _ := book.Profile(before.Kind); !reflect.DeepEqual(before, again) {
		t.Fatal("caller changed the checked book's source bytes")
	}
	_, fresh := Embedded()
	if bytes.Equal(raw, fresh) {
		t.Fatal("embedded book bytes shared with caller")
	}
	if _, err := book.Profile("alamosaurus"); err == nil {
		t.Fatal("accepted unlisted alias")
	}
	var missing *Book
	if _, err := missing.Profile(before.Kind); err == nil {
		t.Fatal("accepted nil book")
	}
}

func TestSauropodBookRejectsMalformedProfiles(t *testing.T) {
	_, raw := Embedded()
	cases := []struct{ name, old, replacement string }{
		{"missing position", "x: -12.000, ", ""},
		{"null radius", "width: 0.050", "width: null"},
		{"nonfinite radius", "width: 0.050", "width: .nan"},
		{"unknown field", "    basis: picked", "    unknown: true\n    basis: picked"},
		{"missing provenance", "    basis: picked", "    basis: ''"},
		{"inverted rings", "x: -9.000", "x: -13.000"},
		{"subresolution rings", "x: -9.000", "x: -11.999999"},
		{"below ground", "height: 0.050", "height: 1.0"},
		{"missing limb index", "station: 6, ", ""},
		{"overlapping legs", "footx: 1.50", "footx: -2.50"},
		{"invalid head index", "eyestation: 11", "eyestation: 99"},
		{"oversized eye", "eyeradius: 0.040", "eyeradius: 1.0"},
		{"invalid colour", "r: 0.290", "r: 1.1"},
		{"duplicate species", "kind: diplodocus-carnegii", "kind: alamosaurus-sanjuanensis"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Contains(raw, []byte(tc.old)) {
				t.Fatalf("missing fixture %q", tc.old)
			}
			bad := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			if _, err := Read("bad.yaml", bad); err == nil {
				t.Fatal("accepted malformed profile")
			}
		})
	}
	if _, err := Read("empty.yaml", nil); err == nil {
		t.Fatal("accepted empty book")
	}
	if _, err := Read("extra.yaml", append(append([]byte(nil), raw...), []byte("\n---\nprofiles: []\n")...)); err == nil {
		t.Fatal("accepted trailing document")
	}
}

func TestSauropodValidateRejectsMutatedProfiles(t *testing.T) {
	book := checkedBook(t)
	tests := []struct {
		field  string
		mutate func(*Profile)
	}{
		{"stations", func(p *Profile) { p.Stations[1].X = math.NaN() }},
		{"stations", func(p *Profile) { p.Stations[1].Width = math.Inf(1) }},
		{"stations", func(p *Profile) { p.Stations = p.Stations[:3] }},
		{"limb", func(p *Profile) { p.Hind.Station = p.Fore.Station }},
		{"fore", func(p *Profile) { p.Fore.FootZ = 0 }},
		{"head", func(p *Profile) { p.Head.EyeRadius = 5 }},
	}
	for _, tc := range tests {
		p, _ := book.Profile("alamosaurus-sanjuanensis")
		tc.mutate(&p)
		if err := p.Validate(); err == nil || !strings.Contains(err.Error(), tc.field) {
			t.Errorf("%s: got %v", tc.field, err)
		}
	}
}
