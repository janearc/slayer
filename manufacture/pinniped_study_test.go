package manufacture

import (
	"bytes"
	"testing"
)

func TestPinnipedStudiesSelectProfiledDrawnRigs(t *testing.T) {
	base, err := ReadPinnipedProfiles("base.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		wet, nostrils float64
	}{{"detailed-wet", .85, .8}, {"detailed-dry", 0, 1}} {
		row, err := base.NewPinnipedStudy("harbor-seal", tc.name)
		if err != nil {
			t.Fatal(err)
		}
		study, ok := row.how.(sealKind)
		if !ok || study.form != sealDetail || study.look.Wet != tc.wet || study.look.Nostrils != tc.nostrils {
			t.Fatalf("%s: %#v", tc.name, row.how)
		}
	}
	row, err := base.NewPinnipedStudy("harbor-seal", "bay")
	if err != nil {
		t.Fatal(err)
	}
	if study, ok := row.how.(sealKind); !ok || study.form != sealBay {
		t.Fatal("bay did not select its original rig")
	}
	for _, tc := range [][2]string{{"harbor-seal", "unknown"}, {"walrus", "bay"}, {"missing", "bay"}} {
		if _, err := base.NewPinnipedStudy(tc[0], tc[1]); err == nil {
			t.Fatalf("accepted %v", tc)
		}
	}
	if _, err := (*PinnipedProfiles)(nil).NewPinnipedStudy("harbor-seal", "bay"); err == nil {
		t.Fatal("accepted nil book")
	}

	// The scaffold comes from the profile, not an embedded species-name test.
	raw := bytes.Replace(pinnipedProfilesRaw, []byte("kind: harbor-seal"), []byte("kind: authored-seal"), 1)
	raw = bytes.Replace(raw, []byte("wet: 0.85"), []byte("wet: 0.4"), 1)
	changed, err := ReadPinnipedProfiles("changed.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	custom, err := changed.NewPinnipedStudy("authored-seal", "detailed-wet")
	if err != nil {
		t.Fatal(err)
	}
	if custom.how.(sealKind).look.Wet != .4 {
		t.Fatal("profile look ignored")
	}
	original, err := base.NewPinnipedStudy("harbor-seal", "detailed-wet")
	if err != nil || original.how.(sealKind).look.Wet != .85 {
		t.Fatal("another book changed the original")
	}
}

func TestPinnipedStudiesRefuseInvalidScaffoldsAndLooks(t *testing.T) {
	for _, tc := range []struct{ from, to string }{
		{"scaffold: detailed-rig", "scaffold: unknown-rig"},
		{"wet: 0.85", "wet: null"},
		{"wet: 0.85", "wet: -0.1"},
		{"wet: 0.85", "wet: 1.1"},
		{"nostrils: 0.8", "other: 0.8"},
		{"status: picked, by: Pinniped study, date: 2026-11-24, wet: 0.85", "status: read, by: Pinniped study, date: 2026-11-24, wet: 0.85"},
		{"scaffold: bay-rig, status: picked, by: Pinniped study, date: 2026-11-24}", "scaffold: bay-rig, status: picked, by: Pinniped study, date: 2026-11-24, wet: 0.8}"},
	} {
		if !bytes.Contains(pinnipedProfilesRaw, []byte(tc.from)) {
			t.Fatalf("missing fixture %q", tc.from)
		}
		raw := bytes.Replace(pinnipedProfilesRaw, []byte(tc.from), []byte(tc.to), 1)
		if _, err := ReadPinnipedProfiles("bad.yaml", raw); err == nil {
			t.Errorf("accepted %q", tc.to)
		}
	}
}
