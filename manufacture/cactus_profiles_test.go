package manufacture

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/plants/cacti"
	"gopkg.in/yaml.v3"
)

func TestCactusProfilesConstructSaguaro(t *testing.T) {
	book, err := ReadCactusProfiles("cacti.yaml", cactusProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewSaguaro()
	if err != nil {
		t.Fatal(err)
	}
	if row.name != "saguaro" {
		t.Fatalf("kind %q", row.name)
	}
	if _, err := Make(row, Sow(5)); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewCactus("missing"); err == nil {
		t.Fatal("accepted missing species")
	}
	if _, err := (*CactusProfiles)(nil).NewCactus("saguaro"); err == nil {
		t.Fatal("accepted nil book")
	}
}

func TestCactusProfilesRejectInvalidAndExcessiveInputs(t *testing.T) {
	base := string(cactusProfilesRaw)
	for name, raw := range map[string]string{
		"empty": "", "unknown": base + "      unknown: 1\n",
		"missing value":      strings.Replace(base, "      ribs: 12\n", "", 1),
		"invalid source":     strings.Replace(base, "status: picked", "status: read", 1),
		"invalid dimensions": strings.Replace(base, "height-minimum: 4.0", "height-minimum: -4.0", 1),
		"invalid color":      strings.Replace(base, "green-r: 0.19", "green-r: 2", 1),
		"spine overload":     strings.Replace(base, "spine-spacing: 0.18", "spine-spacing: 0.000001", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadCactusProfiles("cacti.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted invalid cactus book")
			}
		})
	}
	if _, err := ReadCactusProfiles("cacti.yaml", make([]byte, (1<<20)+1)); err == nil {
		t.Fatal("accepted oversized book")
	}
}

func TestCactusProfileBooksKeepTheirOwnSpineMeshes(t *testing.T) {
	first, err := ReadCactusProfiles("first.yaml", cactusProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(cactusProfilesRaw), "spine-length: 0.03", "spine-length: 0.06", 1)
	second, err := ReadCactusProfiles("second.yaml", []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := first.NewSaguaro()
	b, _ := second.NewSaguaro()
	one, err := a.how.(columnarCactus).spine("quick")
	if err != nil {
		t.Fatal(err)
	}
	two, err := b.how.(columnarCactus).spine("quick")
	if err != nil {
		t.Fatal(err)
	}
	if one == two || one.Bounds() == two.Bounds() {
		t.Fatal("distinct profile dimensions shared a spine")
	}
}

func TestCactusProfileRequiresEveryNumericValue(t *testing.T) {
	base := string(cactusProfilesRaw)
	var document struct {
		Profiles []struct{ Values map[string]float64 }
	}
	if err := yaml.Unmarshal(cactusProfilesRaw, &document); err != nil {
		t.Fatal(err)
	}
	for field := range document.Profiles[0].Values {
		t.Run(field, func(t *testing.T) {
			lines := strings.Split(base, "\n")
			prefix := "      " + field + ":"
			at := -1
			for i, line := range lines {
				if strings.HasPrefix(line, prefix) {
					if at >= 0 {
						t.Fatalf("duplicate fixture field %s", field)
					}
					at = i
				}
			}
			if at < 0 {
				t.Fatalf("fixture missing %s", field)
			}
			for _, value := range []string{"null", "~"} {
				changed := append([]string(nil), lines...)
				changed[at] = prefix + " " + value
				if _, err := ReadCactusProfiles("cacti.yaml", []byte(strings.Join(changed, "\n"))); err == nil {
					t.Fatalf("accepted %s: %s", field, value)
				}
			}
			missing := append(append([]string(nil), lines[:at]...), lines[at+1:]...)
			if _, err := ReadCactusProfiles("cacti.yaml", []byte(strings.Join(missing, "\n"))); err == nil {
				t.Fatalf("accepted missing %s", field)
			}
		})
	}
	zero := strings.Replace(base, "green-r: 0.19", "green-r: 0", 1)
	if zero == base {
		t.Fatal("fixture missing green-r")
	}
	if _, err := ReadCactusProfiles("cacti.yaml", []byte(zero)); err != nil {
		t.Fatalf("valid zero colour: %v", err)
	}
}

func TestCactusProfilesRejectTrailingDocumentsAndMergeKeys(t *testing.T) {
	base := string(cactusProfilesRaw)
	for name, raw := range map[string]string{
		"second document":           base + "\n---\nprofiles: []\n",
		"malformed second document": base + "\n---\nprofiles: [\n",
		"merge key":                 strings.Replace(base, "height-minimum: 4.0", "<<: {height-minimum: 4.0}", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if raw == base {
				t.Fatal("fixture was not changed")
			}
			if _, err := ReadCactusProfiles("cacti.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted trailing document or merge")
			}
		})
	}
}

func TestCactusProfileRejectsUnmeshableSpines(t *testing.T) {
	base := string(cactusProfilesRaw)
	for _, tc := range []struct{ name, from, to string }{
		{"tiny radius", "spine-radius: 0.0015", "spine-radius: 1e-300"},
		{"tiny length", "spine-length: 0.03", "spine-length: 1e-300"},
		{"tiny stem radius", "radius-minimum: 0.22", "radius-minimum: 1e-300"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.from) {
				t.Fatal("fixture absent")
			}
			if _, err := ReadCactusProfiles("bad.yaml", []byte(strings.Replace(base, tc.from, tc.to, 1))); err == nil {
				t.Fatal("accepted unmeshable cactus")
			}
		})
	}
	book, err := ReadCactusProfiles("cactus.yaml", cactusProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewSaguaro()
	if err != nil {
		t.Fatal(err)
	}
	for _, detail := range []string{"quick", "demo"} {
		if _, err := row.how.(columnarCactus).spine(detail); err != nil {
			t.Fatalf("%s spine: %v", detail, err)
		}
	}
}

func TestCactusFamilySnapshotsConstructionPolicy(t *testing.T) {
	policy := cactusPolicy()
	book, err := cacti.Read("cactus.yaml", cactusProfilesRaw, policy)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.New("saguaro")
	if err != nil {
		t.Fatal(err)
	}
	before := row.Value("curve-quick")
	policy.Lookup = nil
	policy.Details[0] = "changed"
	if row.Value("curve-quick") != before {
		t.Fatal("book retained mutable policy callback")
	}
	if err := row.Validate(); err != nil {
		t.Fatalf("book retained mutable policy details: %v", err)
	}
	if _, err := cacti.Read("bad.yaml", cactusProfilesRaw, policy); err == nil {
		t.Fatal("accepted missing policy lookup")
	}
}
