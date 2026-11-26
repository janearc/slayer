package manufacture

import (
	"bytes"
	"testing"
)

func TestConiferShootUsesCheckedSpeciesAndKeepsBooksIndependent(t *testing.T) {
	base, err := ReadConiferProfiles("base.yaml", defaultConiferProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	original, err := base.NewCoastRedwoodShoot()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := func(row Kind) string {
		t.Helper()
		made, err := MakeWithComplexity(row, Sow(6), "quick")
		if err != nil {
			t.Fatal(err)
		}
		return referenceDigest(t, made)
	}
	want := fingerprint(original)
	changed := bytes.Clone(defaultConiferProfilesRaw)
	for _, pair := range [][2]string{
		{`- kind: "coast-redwood"`, `- kind: "coastal-sample"`},
		{"    width: 0.002\n", "    width: 0.003\n"},
	} {
		if bytes.Count(changed, []byte(pair[0])) != 1 {
			t.Fatalf("ambiguous fixture %q", pair[0])
		}
		changed = bytes.Replace(changed, []byte(pair[0]), []byte(pair[1]), 1)
	}
	custom, err := ReadConiferProfiles("custom.yaml", changed)
	if err != nil {
		t.Fatal(err)
	}
	external, err := custom.NewConiferShoot("coastal-sample")
	if err != nil {
		t.Fatal(err)
	}
	customDigest := fingerprint(external)
	if customDigest == want {
		t.Fatal("changed external foliage did not change the shoot")
	}
	for i := range changed {
		changed[i] = 0
	}
	if got := fingerprint(external); got != customDigest {
		t.Fatal("changing the source bytes altered a checked shoot")
	}
	if got := fingerprint(original); got != want {
		t.Fatal("another book changed the default shoot")
	}

	coloured := bytes.Replace(defaultConiferProfilesRaw, []byte("  leaf-colour: [0.07, 0.24, 0.12]"), []byte("  leaf-colour: [0.17, 0.24, 0.12]"), 1)
	if bytes.Equal(coloured, defaultConiferProfilesRaw) {
		t.Fatal("missing colour fixture")
	}
	book, err := ReadConiferProfiles("coloured.yaml", coloured)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewCoastRedwoodShoot()
	if err != nil {
		t.Fatal(err)
	}
	if got := fingerprint(row); got == want {
		t.Fatal("changed external foliage colour did not change the shoot")
	}
	if got := fingerprint(original); got != want {
		t.Fatal("another book changed the original shoot materials")
	}
}

func TestConiferShootRefusesMissingOrUnsupportedSpecies(t *testing.T) {
	var missing *ConiferProfiles
	if _, err := missing.NewConiferShoot("coast-redwood"); err == nil {
		t.Fatal("accepted a nil book")
	}
	for _, species := range []string{"absent", "spruce"} {
		if _, err := defaultConiferProfiles.NewConiferShoot(species); err == nil {
			t.Fatalf("accepted unsupported shoot %q", species)
		}
	}
	if _, err := defaultConiferProfiles.NewGiantSequoiaShoot(); err != nil {
		t.Fatal(err)
	}
}
