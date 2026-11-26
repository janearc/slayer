package ferns

import (
	"strings"
	"testing"
)

const authoredFern = `profiles:
  - kind: fern
    status: picked
    by: study
    date: '2026-10-06'
    frond-length: {minimum: 0.4, maximum: 0.9}
    stalk-radius: {minimum: 0.005, maximum: 0.008}
    fronds: {minimum: 6, maximum: 11}
    leaflets: {minimum: 14, maximum: 24}
    anatomy:
      crown-span: 0.03
      crown-radius: 3
      stalk-tip: 0.15
      leaflet-tip: 0.2
      leaflet-reach: 0.22
      leaflet-shortening: 0.85
      leaflet-girth: 3
      leaflet-minimum: 0.01
    colour: {r: 0.2, g: 0.5, b: 0.12}
`

func TestReadOwnsBytesAndRejectsInvalidNumbers(t *testing.T) {
	raw := []byte(authoredFern)
	book, err := Read("fern.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw {
		raw[i] = 'x'
	}
	if row, ok := book.Get("fern"); !ok || row.FrondLength.Minimum != .4 {
		t.Fatalf("profile was not retained: %+v", row)
	}
	for _, replacement := range []string{"minimum: null", "minimum: .inf", "minimum: -0.4"} {
		bad := strings.Replace(authoredFern, "minimum: 0.4", replacement, 1)
		if _, err := Read("fern.yaml", []byte(bad)); err == nil {
			t.Fatalf("accepted %s", replacement)
		}
	}
}
