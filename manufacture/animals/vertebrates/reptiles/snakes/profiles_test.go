package snakes

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestEmbeddedBookNamesAndOwnership(t *testing.T) {
	file, raw := Embedded()
	book, err := Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"bitis-gabonica", "boa-constrictor-occidentalis", "eunectes-murinus", "madtsoia-bai", "malayopython-reticulatus", "palaeophis-colossaeus", "python-bivittatus", "python-sebae", "titanoboa-cerrejonensis"}
	if got := book.Species(); !reflect.DeepEqual(got, want) {
		t.Fatalf("species %v, want %v", got, want)
	}
	for _, name := range want {
		p, err := book.Profile(name)
		if err != nil {
			t.Fatal(err)
		}
		if p.Kind != name || p.Scientific == "" || p.Source == "" || p.Notes == "" || p.Basis != "picked" {
			t.Fatalf("%s: missing identity or provenance: %+v", name, p)
		}
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := book.Profile("eunectes-murinus")
	after, _ := book.Profile("eunectes-murinus")
	after.Stations[0].Width = .2
	if again, _ := book.Profile("eunectes-murinus"); !reflect.DeepEqual(again, before) {
		t.Fatal("returned station slice changed the book")
	}
	raw[0] ^= 1
	if again, _ := book.Profile("eunectes-murinus"); !reflect.DeepEqual(again, before) {
		t.Fatal("mutated source bytes changed the book")
	}
	_, fresh := Embedded()
	if bytes.Equal(raw, fresh) {
		t.Fatal("embedded bytes shared with caller")
	}
	if _, err := book.Profile("eunectes"); err == nil {
		t.Fatal("accepted unlisted alias")
	}
	var none *Book
	if _, err := none.Profile("eunectes-murinus"); err == nil {
		t.Fatal("accepted nil book")
	}
}

func TestBookRejectsMalformedData(t *testing.T) {
	_, raw := Embedded()
	cases := []struct{ name, old, replacement string }{
		{"missing length", "    length: 12.3\n", ""},
		{"null length", "length: 12.3", "length: null"},
		{"nonfinite length", "length: 12.3", "length: .nan"},
		{"resource length", "length: 12.3", "length: 51"},
		{"unknown field", "    length: 12.3", "    mystery: 1\n    length: 12.3"},
		{"missing pit", ", pit: false", ""},
		{"null pit", "pit: false", "pit: null"},
		{"invalid source", "basis: picked", "basis: measured"},
		{"empty scientific", "scientific: Palaeophis colossaeus", "scientific: ''"},
		{"invalid color", "r: 0.160", "r: 1.1"},
		{"invalid pattern", "kind: bands", "kind: paisley"},
		{"unbounded repeat", "repeat: 21", "repeat: 257"},
		{"missing radius", "at: 0.000, width: 0.001365,", "at: 0.000,"},
		{"unordered stations", "at: 0.040", "at: 0.000"},
		{"wrong endpoint", "at: 1.000", "at: 0.999"},
		{"duplicate species", "kind: titanoboa-cerrejonensis", "kind: palaeophis-colossaeus"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Contains(raw, []byte(tc.old)) {
				t.Fatalf("fixture lacks %q", tc.old)
			}
			bad := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			if _, err := Read("bad.yaml", bad); err == nil {
				t.Fatal("accepted malformed profile")
			}
		})
	}
	if _, err := Read("empty.yaml", nil); err == nil {
		t.Fatal("accepted empty document")
	}
	if _, err := Read("extra.yaml", append(append([]byte(nil), raw...), []byte("\n---\nprofiles: []\n")...)); err == nil {
		t.Fatal("accepted second document")
	}
}

func TestValidateRejectsMutatedProfile(t *testing.T) {
	file, raw := Embedded()
	book, err := Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		field  string
		mutate func(*Profile)
	}{
		{"length", func(p *Profile) { p.Length = math.Inf(1) }},
		{"stations", func(p *Profile) { p.Stations[1].Width = math.NaN() }},
		{"stations", func(p *Profile) { p.Stations = p.Stations[:3] }},
		{"head", func(p *Profile) { p.Head.EyeAt = -.1 }},
		{"pattern", func(p *Profile) { p.Pattern.Contrast = 2 }},
	}
	for _, tc := range tests {
		p, _ := book.Profile("eunectes-murinus")
		tc.mutate(&p)
		if err := p.Validate(); err == nil || !strings.Contains(err.Error(), tc.field) {
			t.Errorf("%s: got %v", tc.field, err)
		}
	}
}
