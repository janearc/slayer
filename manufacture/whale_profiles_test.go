package manufacture

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"
)

// These digests were captured from the three registered orca recipes before
// their profile migration, on Darwin/ARM64 at the committed checkout.
func TestWhaleProfilesRetainOrcaGeometryAndAnimation(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("geometry witnesses were captured on Darwin/ARM64")
	}
	book, err := ReadWhaleProfiles("animals/vertebrates/mammals/whales/profiles.yaml", whaleProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		pose, representation, digest string
		timed                        bool
	}{
		{"swim", "", "540dc692bbb6430cbfe0b45de2abd7f78dc325cd3f2c1cad64820d718fc1ce55", true},
		{"spy-hop", "", "e061c88a810be524982a0476ebd28b74fd0b34bb0dacf5bf5b6dd27980b747de", false},
		{"", "bones", "eb7a0757e679eaa12d9f068ff6a0bfde81db4a274b5c886e7839e4817653173e", false},
	} {
		row, err := book.NewWhale("orca", tc.pose, tc.representation)
		if err != nil {
			t.Fatal(err)
		}
		if Deforms(row) != tc.timed {
			t.Fatalf("%s: incorrect animation capability", tc.pose)
		}
		made, err := MakeWithComplexity(row, Sow(17), "quick")
		if err != nil {
			t.Fatal(err)
		}
		if got := referenceDigest(t, made); got != tc.digest {
			t.Errorf("%s %s: %s", tc.pose, tc.representation, got)
		}
		if tc.timed {
			later, err := MakeAtTime(row, Sow(17), "quick", 750*time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			if got := referenceDigest(t, later); got != "4c30ca1ee5759b67dac76d96efb67f9f225b00411dda328a16e3ba7dd1ddb5cc" {
				t.Errorf("swim at 750 ms: %s", got)
			}
		}
	}
	for _, old := range []string{"orca-clean-05", "orca-clean-05-bones", "orca-clean-05-spy-hop"} {
		if _, err := KindNamed(old); err == nil {
			t.Errorf("retired kind %q remains public", old)
		}
	}
}

func TestWhaleProfilesRejectInvalidAnatomyAndOptions(t *testing.T) {
	for _, bad := range [][]byte{nil, make([]byte, (1<<20)+1)} {
		if _, err := ReadWhaleProfiles("invalid.yaml", bad); err == nil {
			t.Error("accepted invalid whale book")
		}
	}
	for _, tc := range []struct{ from, to string }{
		{"cycle-seconds: 3", "cycle-seconds: 0"}, {"cycle-seconds: 3", "cycle-seconds: 10001"},
		{"cycle-seconds: 3", "cycle-seconds: .nan"}, {"cycle-seconds: 3", "cycle-seconds: null"},
		{"cycle-seconds: 3", "unexpected: 1"}, {"satin: 0.12", "satin: -1"},
		{"satin: 0.12", "satin: 2"}, {"satin: 0.12", "satin: .inf"},
		{"form: orca", "form: mysticete"}, {"kind: orca", "kind: ''"},
		{"status: picked", "status: read"}, {"by: Orca clean study", "by: ' '"},
		{"date: '2026-11-24'", "date: ''"},
	} {
		bad := bytes.Replace(whaleProfilesRaw, []byte(tc.from), []byte(tc.to), 1)
		if bytes.Equal(bad, whaleProfilesRaw) {
			t.Fatalf("missing fixture %q", tc.from)
		}
		if _, err := ReadWhaleProfiles("invalid.yaml", bad); err == nil {
			t.Errorf("accepted %q", tc.to)
		}
	}
	entry := whaleProfilesRaw[bytes.Index(whaleProfilesRaw, []byte("  - kind:")):]
	if _, err := ReadWhaleProfiles("duplicate.yaml", append(bytes.Clone(whaleProfilesRaw), entry...)); err == nil || !strings.Contains(err.Error(), "duplicate whale") {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := ReadWhaleProfiles("empty.yaml", []byte("profiles: []\n")); err == nil {
		t.Fatal("accepted no profiles")
	}
	book, err := ReadWhaleProfiles("animals/vertebrates/mammals/whales/profiles.yaml", whaleProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ species, pose, representation string }{
		{"unknown", "", ""}, {"orca", "glide", ""}, {"orca", "spy-hop", "bones"}, {"orca", "", "volumes"},
	} {
		if _, err := book.NewWhale(tc.species, tc.pose, tc.representation); err == nil {
			t.Errorf("accepted %+v", tc)
		}
	}
	var absent *WhaleProfiles
	if _, err := absent.NewOrca("", ""); err == nil {
		t.Fatal("nil whale book accepted")
	}
	for _, f := range []float64{math.Inf(1), math.NaN()} {
		bad := []byte(strings.Replace(string(whaleProfilesRaw), "satin: 0.12", "satin: "+fmt.Sprint(f), 1))
		if _, err := ReadWhaleProfiles("bad.yaml", bad); err == nil {
			t.Fatal("nonfinite satin accepted")
		}
	}
}

func TestWhaleProfileControlsSwimAndSheen(t *testing.T) {
	raw := bytes.Replace(whaleProfilesRaw, []byte("cycle-seconds: 3"), []byte("cycle-seconds: 6"), 1)
	raw = bytes.Replace(raw, []byte("satin: 0.12"), []byte("satin: 0.4"), 1)
	changed, err := ReadWhaleProfiles("changed.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	row, err := changed.NewOrca("swim", "")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeAtTime(row, Sow(17), "quick", 1500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got := referenceDigest(t, made); got != "4c30ca1ee5759b67dac76d96efb67f9f225b00411dda328a16e3ba7dd1ddb5cc" {
		t.Errorf("longer cycle did not change phase: %s", got)
	}
	if len(made.native) != 1 || made.native[0].Material.Satin != .4 {
		t.Fatal("authored sheen was not applied")
	}
	for i := range raw {
		raw[i] = 0
	}
	again, err := MakeAtTime(row, Sow(17), "quick", 1500*time.Millisecond)
	if err != nil || again.native[0].Material.Satin != .4 {
		t.Fatalf("book did not snapshot profile: %v", err)
	}
}

func TestWhaleMakersRefuseInvalidSeedAndDetail(t *testing.T) {
	profile, _ := defaultWhaleProfiles.book.Get("orca")
	swim, still := whaleSwim{profile}, whaleStill{profile: profile}
	if _, err := swim.made(seed{}); err == nil {
		t.Fatal("swim accepted an unmade seed")
	}
	if _, err := swim.madeAtTime(seed{}, "quick", 0); err == nil {
		t.Fatal("timed swim accepted an unmade seed")
	}
	if _, err := still.made(seed{}); err == nil {
		t.Fatal("still pose accepted an unmade seed")
	}
	if _, err := swim.madeAtTime(Sow(1), "hull", 0); err == nil {
		t.Fatal("swim accepted hull detail")
	}
	want := errors.New("skin construction")
	if _, err := orcaMade(nil, want); !errors.Is(err, want) {
		t.Fatalf("skin failure: %v", err)
	}
}
