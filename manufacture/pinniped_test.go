package manufacture

import (
	"bytes"
	"testing"
)

func TestPinnipedProfileRefusals(t *testing.T) {
	for _, edit := range [][2]string{
		{"thoracic: 15", "thoracic: null"},
		{"thoracic: 15", "thoracic: 0"},
		{"sternal-ribs: 10", "sternal-ribs: 99"},
		{"ankle-forward: 0", "ankle-forward: .nan"},
		{"form: harbor-loft", "form: unknown"},
		{"  date: 2026-11-24\n", ""},
	} {
		raw := bytes.Replace(pinnipedProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(raw, pinnipedProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		if _, err := ReadPinnipedProfiles("bad.yaml", raw); err == nil {
			t.Fatalf("accepted %q", edit[1])
		}
	}
	book, err := ReadPinnipedProfiles("pinnipeds.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	renamed := bytes.Replace(pinnipedProfilesRaw, []byte("kind: harbor-seal"), []byte("kind: renamed-seal"), 1)
	book, err = ReadPinnipedProfiles("renamed.yaml", renamed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewHarborSeal(); err == nil {
		t.Fatal("accepted absent harbor profile")
	}
}

func TestPinnipedWalrusRefusesAnatomyThatCannotBeRendered(t *testing.T) {
	for _, edit := range [][2]string{
		{"thoracic: 14", "thoracic: 32"},
		{"ankle-forward: 2.6", "ankle-forward: 0"},
	} {
		raw := bytes.Replace(pinnipedProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if edit[0] == "thoracic: 14" {
			raw = bytes.Replace(raw, []byte(`thoracic: {basis: read, source: "Pierce, Clack and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755", value: 14}`), []byte(`thoracic: {basis: picked}`), 1)
		}
		if bytes.Equal(raw, pinnipedProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		book, err := ReadPinnipedProfiles("bad-walrus.yaml", raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := book.NewWalrus(); err == nil {
			t.Fatalf("accepted %q", edit[1])
		}
	}
}

func TestPinnipedSpeciesAndPoseSelection(t *testing.T) {
	book, err := ReadPinnipedProfiles("pinnipeds.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ species, pose string }{
		{"harbor-seal", ""},
		{"harbor-seal", "hauled"},
		{"harbor-seal", "swimming"},
		{"hawaiian-monk-seal", ""},
		{"hawaiian-monk-seal", "hauled"},
		{"walrus", ""},
		{"walrus", "lying"},
		{"walrus", "hauled-out"},
		{"walrus", "swimming"},
	} {
		made, err := book.NewPinniped(tc.species, tc.pose)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.species, tc.pose, err)
		}
		if made.name != tc.species {
			t.Fatalf("%s %s: got %q", tc.species, tc.pose, made.name)
		}
	}
	for _, tc := range [][2]string{
		{"harbor-seal", "lying"}, {"hawaiian-monk-seal", "swimming"},
		{"walrus", "hauled"}, {"unknown", ""},
	} {
		if _, err := book.NewPinniped(tc[0], tc[1]); err == nil {
			t.Fatalf("accepted %q %q", tc[0], tc[1])
		}
	}
	var missing *PinnipedProfiles
	if _, err := missing.NewPinniped("walrus", ""); err == nil {
		t.Fatal("accepted a nil book")
	}
}

func TestPinnipedFormIsProfileData(t *testing.T) {
	raw := bytes.Replace(pinnipedProfilesRaw, []byte("kind: harbor-seal"), []byte("kind: authored-seal"), 1)
	raw = bytes.Replace(raw, []byte("kind: walrus"), []byte("kind: authored-walrus"), 1)
	book, err := ReadPinnipedProfiles("authored.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := book.NewPinniped("authored-seal", "swimming")
	if err != nil {
		t.Fatal(err)
	}
	if seal.name != "authored-seal" {
		t.Fatalf("got name %q", seal.name)
	}
	if row, ok := seal.how.(sealKind); !ok || row.form != sealLoft {
		t.Fatal("profile did not select the seal form")
	}
	walrus, err := book.NewPinniped("authored-walrus", "lying")
	if err != nil {
		t.Fatal(err)
	}
	if walrus.name != "authored-walrus" {
		t.Fatalf("got name %q", walrus.name)
	}
	if row, ok := walrus.how.(sealKind); !ok || row.form != sealWalrus {
		t.Fatal("profile did not select the walrus form")
	}
}

func TestPinnipedEvidenceDistinguishesReadFromPicked(t *testing.T) {
	book, err := ReadPinnipedProfiles("pinnipeds.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		species string
		read    bool
	}{
		{"harbor-seal", true}, {"hawaiian-monk-seal", false}, {"walrus", true},
	} {
		profile, err := book.profile(tc.species)
		if err != nil {
			t.Fatal(err)
		}
		for name, evidence := range profile.Evidence {
			want := "picked"
			if tc.read && (name == "thoracic" || name == "lumbar") {
				want = "read"
			}
			if evidence.Basis != want {
				t.Errorf("%s %s basis %q, want %q", tc.species, name, evidence.Basis, want)
			}
			if want == "read" && (evidence.Source == "" || evidence.Value == nil) {
				t.Errorf("%s %s lacks value-specific source", tc.species, name)
			}
		}
	}
}

func TestPinnipedEvidenceRejectsMissingOrStaleClaims(t *testing.T) {
	base := string(pinnipedProfilesRaw)
	const readThoracic = `thoracic: {basis: read, source: "Pierce, Clack and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755", value: 15}`
	for _, tc := range []struct{ name, from, to string }{
		{"missing evidence", "    cervical: {basis: picked}\n", ""},
		{"unknown evidence", "    cervical: {basis: picked}", "    unknown: {basis: picked}"},
		{"invalid basis", "    cervical: {basis: picked}", "    cervical: {basis: guessed}"},
		{"source missing", readThoracic, "thoracic: {basis: read, value: 15}"},
		{"value missing", readThoracic, `thoracic: {basis: read, source: "Pierce"}`},
		{"stale sourced figure", "  thoracic: 15\n", "  thoracic: 16\n"},
		{"stale cited value", readThoracic, `thoracic: {basis: read, source: "Pierce", value: 16}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Contains([]byte(base), []byte(tc.from)) {
				t.Fatal("fixture absent")
			}
			raw := bytes.Replace([]byte(base), []byte(tc.from), []byte(tc.to), 1)
			if _, err := ReadPinnipedProfiles("pinnipeds.yaml", raw); err == nil {
				t.Fatal("accepted misleading or missing evidence")
			}
		})
	}
	changed := bytes.Replace([]byte(base), []byte("  thoracic: 15\n"), []byte("  thoracic: 16\n"), 1)
	changed = bytes.Replace(changed, []byte(readThoracic), []byte("thoracic: {basis: picked}"), 1)
	if _, err := ReadPinnipedProfiles("picked.yaml", changed); err != nil {
		t.Fatalf("valid newly picked count: %v", err)
	}
}

// An external profile changes the actual loft, its bone holds and its face,
// rather than merely selecting one of the built-in species by name.
func TestPinnipedRejectsInvalidLoftOutlines(t *testing.T) {
	for _, edit := range [][2]string{
		{"length: 1.75", "length: 0"},
		{"length: 1.75", "length: .nan"},
		{"[-.9200, .0060, .0080, -.015]", "[-.9100, .0060, .0080, -.015]"},
		{"[-.8600, .0260, .0300, -.012]", "[-.9300, .0260, .0300, -.012]"},
		{"[.7500, .0690, .0790, .067]", "[.7500, 0, .0790, .067]"},
		{"[.7500, .0690, .0790, .067]", "[.7500, .0690, .0790, .nan]"},
	} {
		raw := bytes.Replace(pinnipedProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(raw, pinnipedProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		if _, err := ReadPinnipedProfiles("bad.yaml", raw); err == nil {
			t.Errorf("accepted %q", edit[1])
		}
	}
}

func TestPinnipedRejectsOutlineOutsideSharedRig(t *testing.T) {
	for _, edit := range [][2]string{
		{"[.7500, .0690, .0790, .067]", "[.7500, .20, .0790, .067]"},
		{"[.7500, .0690, .0790, .067]", "[.7510, .0690, .0790, .067]"},
	} {
		raw := bytes.Replace(pinnipedProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		book, err := ReadPinnipedProfiles("changed.yaml", raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := book.NewHarborSeal(); err == nil {
			t.Errorf("accepted incompatible station %q", edit[1])
		}
	}
}

// Independent profile books retain their own production volume fields.

func TestWalrusRejectsOutlineOutsideItsRig(t *testing.T) {
	for _, edit := range [][2]string{
		{"[1.30, .502, .915, -.080, 2.2]", "[1.30, .75, .915, -.080, 2.2]"},
		{"[0.05, .190, .795, .470, 3.4]", "[0.05, .191, .795, .470, 3.4]"},
		{"[1.30, .502, .915, -.080, 2.2]", "[1.30, .nan, .915, -.080, 2.2]"},
	} {
		raw := bytes.Replace(pinnipedProfilesRaw, []byte(edit[0]), []byte(edit[1]), 1)
		if bytes.Equal(raw, pinnipedProfilesRaw) {
			t.Fatalf("missing fixture %q", edit[0])
		}
		book, err := ReadPinnipedProfiles("bad.yaml", raw)
		if err == nil {
			_, err = book.NewWalrus()
		}
		if err == nil {
			t.Errorf("accepted incompatible outline %q", edit[1])
		}
	}
}
