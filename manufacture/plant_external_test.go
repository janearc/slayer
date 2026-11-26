package manufacture

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/profilecheck"
)

func TestExternalPlantProfilesConstructSpeciesWithoutRegistry(t *testing.T) {
	source := bytes.Replace(plantProfilesRaw, []byte("kind: valley-oak"), []byte("kind: meadow-oak"), 1)
	book, err := ReadPlantProfiles("plants.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewPlant("meadow-oak")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := KindNamed("meadow-oak"); err == nil {
		t.Fatal("test species unexpectedly in registry")
	}
	made, err := MakeWithComplexity(row, Sow(5), "hull")
	if err != nil {
		t.Fatal(err)
	}
	original, err := MakeWithComplexity(mustPlantKind(t, "valley-oak"), Sow(5), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if referenceDigest(t, made) != referenceDigest(t, original) {
		t.Fatal("external profile changed the plant")
	}
	source = bytes.ReplaceAll(source, []byte("meadow-oak"), []byte("vanished"))
	if _, err := Make(row, Sow(5)); err != nil {
		t.Fatalf("profile not retained: %v", err)
	}
}

func TestPlantProfileYAMLAliasExpansionIsBounded(t *testing.T) {
	var source strings.Builder
	source.WriteString("profiles: []\na: &a [0,0,0,0,0,0,0,0,0,0]\n")
	for letter := byte('b'); letter <= 'f'; letter++ {
		fmt.Fprintf(&source, "%c: &%c [", letter, letter)
		for i := 0; i < 10; i++ {
			if i > 0 {
				source.WriteByte(',')
			}
			fmt.Fprintf(&source, "*%c", letter-1)
		}
		source.WriteString("]\n")
	}
	err := profilecheck.NumbersPresent("aliases.yaml", []byte(source.String()), reflect.TypeOf(plantProfile{}), nil)
	if err == nil || !strings.Contains(err.Error(), "node budget") {
		t.Fatalf("alias expansion: %v", err)
	}
}

func mustPlantKind(t *testing.T, name string) Kind {
	t.Helper()
	row, err := testKind(name)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestExternalPlantProfilesRefuseInvalidBooksAndSpecies(t *testing.T) {
	cases := [][]byte{nil, []byte("profiles: []"), []byte("profiles: [{kind: nope}]"), []byte("profiles: ["), bytes.Repeat([]byte("x"), (1<<20)+1)}
	for _, raw := range cases {
		if _, err := ReadPlantProfiles("plants.yaml", raw); err == nil {
			t.Fatalf("accepted invalid book %q", string(raw[:min(len(raw), 40)]))
		}
	}
	text := string(plantProfilesRaw)
	first := strings.Index(text, "\n- status:")
	next := strings.Index(text[first+1:], "\n- status:")
	if first < 0 || next < 0 {
		t.Fatal("fixture has fewer than two profiles")
	}
	duplicate := append(bytes.Clone(plantProfilesRaw), []byte(text[first:first+1+next])...)
	if _, err := ReadPlantProfiles("plants.yaml", duplicate); err == nil || !strings.Contains(err.Error(), "invalid authored plant profile \"valley-oak\"") {
		t.Fatalf("duplicate profile refusal: %v", err)
	}
	book, err := ReadPlantProfiles("plants.yaml", plantProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewPlant("absent"); err == nil || !strings.Contains(err.Error(), "absent") {
		t.Fatalf("unknown species: %v", err)
	}
}

func TestNamedPlantConstructorsUseTheirOwnProfiles(t *testing.T) {
	book, err := ReadPlantProfiles("plants.yaml", plantProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		make func() (Kind, error)
	}{
		{"valley-oak", book.NewValleyOak}, {"alder", book.NewAlder},
		{"poplar", book.NewPoplar}, {"manzanita", book.NewManzanita},
		{"chaparral-brush", book.NewChaparralBrush}, {"joshua-tree", book.NewJoshuaTree},
		{"kandelia-obovata", book.NewKandeliaObovata}, {"aegiceras-corniculatum", book.NewAegicerasCorniculatum},
		{"avicennia-marina", book.NewAvicenniaMarina}, {"bruguiera-gymnorhiza", book.NewBruguieraGymnorhiza},
		{"sonneratia-apetala", book.NewSonneratiaApetala}, {"pandanus-tectorius", book.NewPandanusTectorius},
		{"scaevola-taccada", book.NewScaevolaTaccada}, {"metrosideros-polymorpha", book.NewMetrosiderosPolymorpha},
		{"barringtonia-asiatica", book.NewBarringtoniaAsiatica}, {"quercus-ilex", book.NewQuercusIlex},
		{"erica-arborea", book.NewEricaArborea}, {"salix-alba", book.NewSalixAlba},
	}
	for _, tc := range cases {
		row, err := tc.make()
		if err != nil || row.name != tc.name {
			t.Errorf("%s: row %q, %v", tc.name, row.name, err)
		}
	}
	var missing *PlantProfiles
	if _, err := missing.NewValleyOak(); err == nil {
		t.Fatal("nil book accepted")
	}
}

func TestPlantProfilesKeepProvenanceForReadAndDerivedData(t *testing.T) {
	for _, status := range []string{"read", "derived"} {
		raw := bytes.Replace(plantProfilesRaw, []byte("status: picked"), []byte("status: "+status), 1)
		if _, err := ReadPlantProfiles("plants.yaml", raw); err == nil {
			t.Fatalf("%s without source accepted", status)
		}
		raw = bytes.Replace(raw, []byte("status: "+status), []byte("status: "+status+"\n  source: source-id"), 1)
		if _, err := ReadPlantProfiles("plants.yaml", raw); err != nil {
			t.Fatalf("%s with source: %v", status, err)
		}
	}
	invalid := bytes.Replace(plantProfilesRaw, []byte("status: picked"), []byte("status: imagined"), 1)
	if _, err := ReadPlantProfiles("plants.yaml", invalid); err == nil {
		t.Fatal("unknown provenance accepted")
	}
}

func TestPlantProfilesDistinguishNullMissingAndZero(t *testing.T) {
	for _, edit := range [][2]string{
		{"  fork: 48\n", "  fork: null\n"},
		{"  twist: 150\n", ""},
		{"  orders: 4\n", "  orders: null\n"},
		{"    r: 0.27\n", "    r: null\n"},
		{"  bark:\n    r: 0.27\n    g: 0.2\n    b: 0.13\n", ""},
		{"    tip: 0.5\n", "    tip: null\n"},
		{"  rosette-leaves: 0\n", "  rosette-leaves: null\n"},
		{"  leader: false\n", "  leader: null\n"},
		{"  deciduous: true\n", "  deciduous: null\n"},
	} {
		changed := bytes.Replace(plantProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(changed, plantProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		if _, err := ReadPlantProfiles("plants.yaml", changed); err == nil {
			t.Errorf("accepted %q", edit[1])
		}
	}
	for _, edit := range [][2]string{
		{"  fork: 48\n", "  fork: 0\n"},
		{"    r: 0.27\n", "    r: 0\n"},
		{"  rosette-leaves: 0\n", ""},
		{"  rosette-angle: 70\n", ""},
		{"  leader-gain: 1.1\n", ""},
	} {
		changed := bytes.Replace(plantProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(changed, plantProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		if _, err := ReadPlantProfiles("plants.yaml", changed); err != nil {
			t.Errorf("refused allowed edit %q: %v", edit[1], err)
		}
	}
	rootStart := bytes.Index(plantProfilesRaw, []byte("  roots:\n"))
	rootEnd := bytes.Index(plantProfilesRaw[rootStart:], []byte("  kind:")) + rootStart
	rootNull := append(bytes.Clone(plantProfilesRaw[:rootStart]), []byte("  roots: null\n")...)
	rootNull = append(rootNull, plantProfilesRaw[rootEnd:]...)
	if _, err := ReadPlantProfiles("plants.yaml", rootNull); err == nil || !strings.Contains(err.Error(), "null field profiles[") {
		t.Fatalf("null roots: %v", err)
	}
	rosetteStart := bytes.Index(plantProfilesRaw, []byte("  form: rosette\n"))
	rosetteOptional := append(bytes.Clone(plantProfilesRaw[:rosetteStart]), bytes.Replace(plantProfilesRaw[rosetteStart:], []byte("  twig-count: 0\n"), nil, 1)...)
	if _, err := ReadPlantProfiles("plants.yaml", rosetteOptional); err != nil {
		t.Fatalf("rosette without twig-count: %v", err)
	}
	aliases := bytes.Replace(plantProfilesRaw, []byte("  form: twig\n"), []byte("  form: &twig twig\n"), 1)
	aliases = bytes.Replace(aliases, []byte("  form: twig\n"), []byte("  form: *twig\n"), 1)
	aliases = bytes.Replace(aliases, []byte("  leader: false\n"), []byte("  leader: &off false\n"), 1)
	aliases = bytes.Replace(aliases, []byte("  leader: true\n"), []byte("  leader: *off\n"), 1)
	aliases = bytes.Replace(aliases, []byte("  leader-gain: 1.3\n"), nil, 1)
	second := bytes.Index(aliases, []byte("  form: *twig\n"))
	aliases = append(bytes.Clone(aliases[:second]), bytes.Replace(aliases[second:], []byte("  rosette-leaves: 0\n"), nil, 1)...)
	if _, err := ReadPlantProfiles("plants.yaml", aliases); err != nil {
		t.Fatalf("aliased form and leader with omitted irrelevant fields: %v", err)
	}
	upperFalse := bytes.Replace(plantProfilesRaw, []byte("  leader: false\n"), []byte("  leader: FALSE\n"), 1)
	upperFalse = bytes.Replace(upperFalse, []byte("  leader-gain: 1.1\n"), nil, 1)
	if _, err := ReadPlantProfiles("plants.yaml", upperFalse); err != nil {
		t.Fatalf("upper-case false with omitted leader gain: %v", err)
	}
	merged := bytes.Replace(plantProfilesRaw, []byte("  deciduous: true\n"), []byte("  <<: {deciduous: null}\n"), 1)
	if _, err := ReadPlantProfiles("plants.yaml", merged); err == nil || !strings.Contains(err.Error(), "merge keys") {
		t.Fatalf("merged null deciduous accepted: %v", err)
	}
	merged = bytes.Replace(plantProfilesRaw, []byte("  fork: 48\n"), []byte("  <<: {fork: 48}\n"), 1)
	if _, err := ReadPlantProfiles("plants.yaml", merged); err == nil || !strings.Contains(err.Error(), "merge keys") {
		t.Fatalf("merged numeric field accepted: %v", err)
	}
	for _, extra := range []string{"\n---\nprofiles: []\n", "\n---\nprofiles: [\n"} {
		if _, err := ReadPlantProfiles("plants.yaml", append(bytes.Clone(plantProfilesRaw), extra...)); err == nil {
			t.Fatalf("accepted trailing document %q", extra)
		}
	}
}
