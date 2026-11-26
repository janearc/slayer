package specimens

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

//go:embed testdata/kinds-9fabd006.txt
var beforeRetirement string

// retainedPopulation requires every captured name exactly once, while
// allowing later additions to the current registry and profile books.
func retainedPopulation(want, current []string) error {
	seen := make(map[string]bool, len(current))
	for _, name := range current {
		if seen[name] {
			return fmt.Errorf("duplicate candidate %q", name)
		}
		seen[name] = true
	}
	for _, name := range want {
		if !seen[name] {
			return fmt.Errorf("missing historical candidate %q", name)
		}
	}
	return nil
}

// The captured population predates profile retirement. New species may be
// added without hiding a missing or duplicated historical specimen.
func TestProfilesRetainThePreRetirementPopulation(t *testing.T) {
	profiles, err := Profiles()
	if err != nil {
		t.Fatal(err)
	}
	names := manufacture.KindNames()
	names = slices.DeleteFunc(names, func(name string) bool { return name == manufacture.ClusterKind })
	seen := make(map[string]bool)
	for _, specimen := range profiles {
		if specimen.Name == "" || seen[specimen.Name] {
			t.Fatalf("duplicate or empty specimen name %q", specimen.Name)
		}
		seen[specimen.Name] = true
		if strings.HasPrefix(specimen.Name, "animal/reptile/") && specimen.Legacy != "" {
			t.Errorf("reptile study %s gained a historical factory label %q", specimen.Name, specimen.Legacy)
		}
		if specimen.Legacy != "" {
			names = append(names, specimen.Legacy)
		}
	}
	witness := strings.TrimSuffix(beforeRetirement, "\n")
	want := strings.Split(witness, "\n")
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(witness)))
	if len(want) != 282 || digest != "993d9acaaa98393646779329fd70e15dcf1bef5d6a14c69ba6b7585fb78c53cd" {
		t.Fatalf("historical witness changed: %d, %s", len(want), digest)
	}
	if err := retainedPopulation(want, names); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"plant/fern/fern/whole", "plant/conifer/spruce/whole",
		"plant/conifer/coast-redwood/shoot", "animal/fish/pacific-sardine/school-5000",
		"animal/bird/grey-heron/outline", "animal/pinniped/walrus/volumes",
		"animal/pinniped/harbor-seal/study-bay", "animal/whale/orca/swim",
		"animal/whale/orca/spy-hop", "animal/whale/orca/bones",
		"animal/theropod/allosaurus/default", "animal/squid/humboldt-squid/default",
		"alga/kelp/macrocystis-pyrifera/plant/default", "alga/kelp/macrocystis-pyrifera/forest/default",
		"alga/kelp/macrocystis-pyrifera/plant/slack", "alga/kelp/macrocystis-pyrifera/forest/slack",
		"alga/kelp/macrocystis-pyrifera/plant/moderate", "alga/kelp/macrocystis-pyrifera/forest/moderate",
		"alga/kelp/macrocystis-pyrifera/plant/surge", "alga/kelp/macrocystis-pyrifera/forest/surge",
		"animal/reptile/pliosaur/baseline", "animal/reptile/pliosaur/continuous",
		"animal/reptile/purussaurus/study", "animal/reptile/hatzegopteryx/composite",
		"animal/reptile/hatzegopteryx/cage", "animal/reptile/hatzegopteryx/density",
	} {
		if !seen[name] {
			t.Errorf("missing %s", name)
		}
	}
}

// A newly introduced kind or profile must not conceal a missing old one or
// turn a duplicated historical name into a successful inventory.
func TestHistoricalPopulationAllowsAdditionsButRefusesLossOrDuplicates(t *testing.T) {
	want := []string{"fern", "boulder"}
	for _, tc := range []struct {
		name    string
		current []string
		valid   bool
	}{
		{"new species", []string{"fern", "boulder", "new-species"}, true},
		{"missing despite addition", []string{"boulder", "new-species"}, false},
		{"duplicated name", []string{"fern", "boulder", "fern"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := retainedPopulation(want, tc.current); (err == nil) != tc.valid {
				t.Fatalf("population %v: %v", tc.current, err)
			}
		})
	}
}

// A new checked water condition must enter both specimen forms without
// acquiring a made-up historical name.
func TestKelpInventoryIncludesNewWaterConditions(t *testing.T) {
	input := embeddedSources()
	read := input.alga
	input.alga = func(family string) (string, []byte, error) {
		file, raw, err := read(family)
		if err != nil {
			return file, nil, err
		}
		return file, append(raw, []byte("\n  tide: {current: 0.1, swell-height: 0.2, swell-period: 10, basis: 'picked: test tide'}\n")...), nil
	}
	got, err := profiles(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"alga/kelp/macrocystis-pyrifera/plant/tide", "alga/kelp/macrocystis-pyrifera/forest/tide"} {
		found := false
		for _, specimen := range got {
			if specimen.Name == name {
				found = true
				if specimen.Legacy != "" {
					t.Fatalf("new condition %s gained historical label %q", name, specimen.Legacy)
				}
			}
		}
		if !found {
			t.Errorf("new condition missing %s", name)
		}
	}
}

func TestDifferentKelpSpeciesGetsNoHistoricalLabels(t *testing.T) {
	input := embeddedSources()
	read := input.alga
	input.alga = func(family string) (string, []byte, error) {
		file, raw, err := read(family)
		if err != nil {
			return file, nil, err
		}
		changed := strings.Replace(string(raw), "species: macrocystis-pyrifera", "species: another-kelp", 1)
		return file, []byte(changed), nil
	}
	got, err := profiles(input)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, specimen := range got {
		if strings.HasPrefix(specimen.Name, "alga/kelp/another-kelp/") {
			count++
			if specimen.Legacy != "" {
				t.Fatalf("new species gained historical label %q", specimen.Legacy)
			}
		}
	}
	if count != 8 {
		t.Fatalf("new species has %d forms, want 8", count)
	}
}

// An unrecognised anatomical form cannot silently disappear from the checks.
func TestUnknownPinnipedFormIsRefused(t *testing.T) {
	if _, err := pinnipedVariants("new-seal", "unrecognised", nil); err == nil {
		t.Fatal("an unknown form was accepted")
	}
}
