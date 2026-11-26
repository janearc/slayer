package manufacture

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestExternalFishProfilesUseTheirOwnSpeciesAndOptions(t *testing.T) {
	book, err := ReadFishProfiles("fish.yaml", fishRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		make func(FishOptions) (Kind, error)
	}{
		{"pacific-bluefin-tuna", book.NewPacificBluefinTuna}, {"california-yellowtail", book.NewCaliforniaYellowtail},
		{"california-barracuda", book.NewCaliforniaBarracuda}, {"pacific-sardine", book.NewPacificSardine},
		{"convict-tang", book.NewConvictTang}, {"dusky-grouper", book.NewDuskyGrouper},
		{"mediterranean-chromis", book.NewMediterraneanChromis}, {"european-perch", book.NewEuropeanPerch},
		{"flathead-grey-mullet", book.NewFlatheadGreyMullet},
	} {
		row, err := tc.make(FishOptions{})
		if err != nil || row.name != tc.name {
			t.Errorf("%s: %q, %v", tc.name, row.name, err)
		}
	}
	for _, tc := range []struct {
		options FishOptions
		name    string
	}{
		{FishOptions{}, "pacific-sardine"}, {FishOptions{Glide: true}, "pacific-sardine-glide"},
		{FishOptions{Bones: true}, "pacific-sardine-bones"}, {FishOptions{School: 500}, "pacific-sardine-school-500"},
	} {
		row, err := book.NewPacificSardine(tc.options)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := row.how.made(seed{}); !errors.Is(err, errGood) {
			t.Errorf("%s accepted an unsown seed: %v", tc.name, err)
		}
		if rungs := RungsOf(row); Deforms(row) || len(rungs) != 1 || rungs[0] != "quick" {
			t.Errorf("%s acquired an unsupported capability", tc.name)
		}
		if _, err := MakeWithComplexity(row, Sow(5), "hull"); err == nil {
			t.Errorf("%s accepted hull detail", tc.name)
		}
		made, err := Make(row, Sow(5))
		if err != nil {
			t.Fatal(err)
		}
		if made.PartCount() == 0 {
			t.Errorf("%s made no parts", tc.name)
		}
	}
	// A same-named profile comes from this book, never the embedded registry.
	changed := bytes.Replace(fishRaw, []byte("length: {least: 1.5, most: 2.4"), []byte("length: {least: 1.8, most: 2.4"), 1)
	if bytes.Equal(changed, fishRaw) {
		t.Fatal("missing tuna length fixture")
	}
	other, err := ReadFishProfiles("other.yaml", changed)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := book.NewPacificBluefinTuna(FishOptions{})
	b, _ := other.NewPacificBluefinTuna(FishOptions{})
	first, err := Make(a, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Make(b, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	if referenceDigest(t, first) == referenceDigest(t, second) {
		t.Fatal("different books reused the embedded fish")
	}
	for i := range changed {
		changed[i] = 0
	}
	if _, err := Make(b, Sow(5)); err != nil {
		t.Fatalf("fish book did not retain its checked data: %v", err)
	}
}

func TestFishProfilesRejectUnsafeOrIncompleteInput(t *testing.T) {
	for _, tc := range [][2]string{
		{"least: 1.5, most: 2.4", "least: null, most: 2.4"},
		{"least: 1.5, most: 2.4", "most: 2.4"},
		{"colour: [.010, .016, .050]", "colour: [null, .016, .050]"},
		{"school-variants: {least: 8, most: 8", "school-variants: {least: 0, most: 8"},
		{"school-long: {least: 3, most: 3", "school-long: {least: 3, most: 100000"},
		{"schools: [500, 5000]", "schools: [500, 5001]"},
		{"vertebrae: {least: 39, most: 39", "vertebrae: {least: 39, most: 1000000"},
		{"spot-pitch: {least: .045, most: .06", "spot-pitch: {least: 0, most: .06"},
		{"gape: {least: .34, most: .38", "gape: {least: 0, most: 0"},
		{"pelvic-at: {least: .31, most: .32", "pelvic-at: {least: 0, most: 0"},
		{"pectoral-at: {least: .28, most: .29", "pectoral-at: {least: 0, most: 0"},
		{"eye-at: {least: .36, most: .38", "eye-at: {least: 10, most: 10"},
		{"eye-height: {least: .28, most: .32", "eye-height: {least: 2, most: 2"},
		{"stripe-width: {least: 0, most: 0", "stripe-width: {least: -1, most: 0"},
	} {
		changed := bytes.Replace(fishRaw, []byte(tc[0]), []byte(tc[1]), 1)
		if bytes.Equal(changed, fishRaw) {
			t.Errorf("missing fixture %q", tc[0])
			continue
		}
		if _, err := ReadFishProfiles("bad.yaml", changed); err == nil {
			t.Errorf("accepted %q", tc[1])
		}
	}
	for _, extra := range []string{"\n---\nspecies: {}\n", "\n---\nspecies: [\n"} {
		if _, err := ReadFishProfiles("fish.yaml", append(bytes.Clone(fishRaw), extra...)); err == nil {
			t.Errorf("accepted trailing document %q", extra)
		}
	}
	merged := bytes.Replace(fishRaw, []byte("    figures:\n"), []byte("    figures:\n      <<: {length: {least: 1.5, most: 2.4, basis: picked}}\n"), 1)
	if _, err := ReadFishProfiles("fish.yaml", merged); err == nil || !strings.Contains(err.Error(), "merge keys") {
		t.Fatalf("fish merge key: %v", err)
	}
	noFinletRoom := bytes.Replace(fishRaw, []byte("second-dorsal-end: {least: .57, most: .57"), []byte("second-dorsal-end: {least: .85, most: .85"), 1)
	if _, err := ReadFishProfiles("fish.yaml", noFinletRoom); err == nil || !strings.Contains(err.Error(), "no room for dorsal-finlets") {
		t.Fatalf("finlets need room behind the dorsal fin: %v", err)
	}
	book, err := ReadFishProfiles("fish.yaml", fishRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, options := range []FishOptions{{Bones: true, Glide: true}, {School: 1, Bones: true}, {School: -1}, {School: 5001}} {
		if _, err := book.NewPacificSardine(options); err == nil {
			t.Errorf("accepted options %+v", options)
		}
	}
	if _, err := book.NewPacificBluefinTuna(FishOptions{School: 1}); err == nil {
		t.Error("accepted school without school figures")
	}
	if _, err := book.NewFish("missing", FishOptions{}); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("missing fish: %v", err)
	}
	zero := bytes.Replace(fishRaw, []byte("gloss: .45"), []byte("gloss: 0"), 1)
	if _, err := ReadFishProfiles("fish.yaml", zero); err != nil {
		t.Fatalf("zero gloss should be valid: %v", err)
	}
}

func TestFishProfilesAcceptMappingAliases(t *testing.T) {
	text := strings.Replace(string(fishRaw), "    figures:\n", "    figures: &tuna-figures\n", 1)
	text = strings.Replace(text, "    looks:\n", "    looks: &tuna-looks\n", 1)
	yellowtail := strings.Index(text, "  california-yellowtail:\n")
	if yellowtail < 0 {
		t.Fatal("missing yellowtail fixture")
	}
	figures := yellowtail + strings.Index(text[yellowtail:], "    figures:\n")
	next := figures + strings.Index(text[figures:], "  california-barracuda:\n")
	if figures < yellowtail || next < figures {
		t.Fatal("missing figure or species fixture")
	}
	text = text[:figures] + "    figures: *tuna-figures\n    looks: *tuna-looks\n" + text[next:]
	book, err := ReadFishProfiles("aliases.yaml", []byte(text))
	if err != nil {
		t.Fatalf("mapping aliases: %v", err)
	}
	if _, err := book.NewCaliforniaYellowtail(FishOptions{}); err != nil {
		t.Fatal(err)
	}
}
