package manufacture

import (
	"bytes"
	"testing"
)

// Kelp books select a species, water and stand without a public kind alias.
func TestKelpProfilesSelectCheckedVariants(t *testing.T) {
	book, err := ReadKelpProfiles("kelp.yaml", kelpRaw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewGiantKelp("", false); err != nil {
		t.Fatal(err)
	}
	for _, options := range []KelpOptions{
		{Species: "macrocystis-pyrifera", Condition: "slack"},
		{Species: "macrocystis-pyrifera", Condition: "moderate", Forest: true},
		{Species: "macrocystis-pyrifera", Condition: "surge"},
	} {
		row, err := book.NewKelp(options)
		if err != nil {
			t.Fatalf("%+v: %v", options, err)
		}
		if _, err := Make(row, Sow(5)); err != nil {
			t.Fatalf("%+v: %v", options, err)
		}
	}
	for _, options := range []KelpOptions{
		{Species: "unknown", Condition: "slack"},
		{Species: "macrocystis-pyrifera", Condition: "unknown"},
	} {
		if _, err := book.NewKelp(options); err == nil {
			t.Errorf("accepted unknown variant %+v", options)
		}
	}
	var missing *KelpProfiles
	if _, err := missing.NewKelp(KelpOptions{}); err == nil {
		t.Fatal("accepted a missing profile book")
	}
	if _, err := ReadKelpProfiles("broken.yaml", []byte("species: [")); err == nil {
		t.Fatal("accepted malformed profile data")
	}
	for _, raw := range [][]byte{nil, make([]byte, (1<<20)+1)} {
		if _, err := ReadKelpProfiles("bounded.yaml", raw); err == nil {
			t.Fatal("accepted an empty or oversized kelp book")
		}
	}
	withoutSpecies := bytes.Replace(kelpRaw, []byte("species: macrocystis-pyrifera\n"), nil, 1)
	if bytes.Equal(withoutSpecies, kelpRaw) {
		t.Fatal("species source not found")
	}
	if _, err := ReadKelpProfiles("no-species.yaml", withoutSpecies); err == nil {
		t.Fatal("accepted a book without species identity")
	}
}

func TestKelpRetiresPublicVariantsButPreservesClusterPopulation(t *testing.T) {
	for _, prefix := range []string{"giant-kelp", "kelp-forest"} {
		for _, suffix := range []string{"", "-slack", "-moderate", "-surge"} {
			name := prefix + suffix
			if _, err := KindNamed(name); err == nil {
				t.Errorf("retired public kind %q remains", name)
			}
			if _, err := scavengedModel(name); err != nil {
				t.Errorf("historical cluster variant %q: %v", name, err)
			}
		}
	}
}

func TestKelpProfilesRefuseNonfiniteWater(t *testing.T) {
	for _, pair := range [][2]string{
		{"current: 0.15", "current: .inf"},
		{"swell-height: 0.6", "swell-height: .inf"},
	} {
		bad := bytes.Replace(kelpRaw, []byte(pair[0]), []byte(pair[1]), 1)
		if bytes.Equal(bad, kelpRaw) {
			t.Fatalf("source does not contain %q", pair[0])
		}
		if _, err := ReadKelpProfiles("bad-water.yaml", bad); err == nil {
			t.Errorf("accepted %s", pair[1])
		}
	}
}

func TestKelpProfilesRefuseUngrowableFigures(t *testing.T) {
	for _, pair := range [][2]string{
		{"node-spacing: {least: 0.15", "node-spacing: {least: 0"},
		{"segment: {least: 0.5", "segment: {least: 0"},
		{"variants: {least: 6, most: 6", "variants: {least: 0, most: 6"},
		{"variants: {least: 6, most: 6", "variants: {least: 6.5, most: 6.5"},
		{"variants: {least: 6, most: 6", "variants: {least: 65, most: 65"},
		{"depth: {least: 12, most: 12", "depth: {least: 0, most: 12"},
		{"blade-length: {least: 0.4", "blade-length: {least: 0"},
		{"drop: {least: 0, most: 0.8", "drop: {least: 0, most: 12"},
	} {
		bad := bytes.Replace(kelpRaw, []byte(pair[0]), []byte(pair[1]), 1)
		if bytes.Equal(bad, kelpRaw) {
			t.Fatalf("source does not contain %q", pair[0])
		}
		if _, err := ReadKelpProfiles("ungrowable.yaml", bad); err == nil {
			t.Errorf("accepted %s", pair[1])
		}
	}
}

// A book owns its checked values, including the shared mesh kit, even if the
// caller changes its source bytes or reads a different book later.
func TestKelpProfilesFreezeTheirSource(t *testing.T) {
	raw := bytes.Clone(kelpRaw)
	first, err := ReadKelpProfiles("first.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	options := KelpOptions{Species: "macrocystis-pyrifera", Condition: "moderate"}
	row, err := first.NewKelp(options)
	if err != nil {
		t.Fatal(err)
	}
	before, err := Make(row, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte("depth: {least: 12, most: 12"), []byte("depth: {least: 8, most: 8"), 1)
	if bytes.Equal(changed, raw) {
		t.Fatal("depth source not found")
	}
	second, err := ReadKelpProfiles("second.yaml", changed)
	if err != nil {
		t.Fatal(err)
	}
	otherRow, err := second.NewKelp(options)
	if err != nil {
		t.Fatal(err)
	}
	other, err := Make(otherRow, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	wider := bytes.Replace(raw, []byte("blade-width: {least: 0.06, most: 0.12"), []byte("blade-width: {least: 0.12, most: 0.24"), 1)
	if bytes.Equal(wider, raw) {
		t.Fatal("blade width source not found")
	}
	third, err := ReadKelpProfiles("third.yaml", wider)
	if err != nil {
		t.Fatal(err)
	}
	widerRow, err := third.NewKelp(options)
	if err != nil {
		t.Fatal(err)
	}
	widerPlant, err := Make(widerRow, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw {
		raw[i] = 0
	}
	again, err := Make(row, Sow(5))
	if err != nil {
		t.Fatal(err)
	}
	if referenceDigest(t, before) != referenceDigest(t, again) {
		t.Fatal("changing source or reading a second book changed the first")
	}
	if referenceDigest(t, before) == referenceDigest(t, other) {
		t.Fatal("changed depth did not change constructed geometry")
	}
	if referenceDigest(t, before) == referenceDigest(t, widerPlant) {
		t.Fatal("changed blade geometry did not change the shared kit")
	}
}
