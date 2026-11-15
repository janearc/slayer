package identity

import (
	"os"
	"strings"
	"testing"
)

func TestIdentityPrehashed(t *testing.T) {
	s := fixture(t)
	want := build(t, s)
	binary, err := HashFile(s.BinaryPath)
	if err != nil {
		t.Fatal(err)
	}
	input, err := HashFile(s.Inputs["fire.steady"].Path)
	if err != nil {
		t.Fatal(err)
	}
	p := PrehashedSpec{BinarySHA256: binary, Settings: s.Settings, Inputs: map[string]HashedInput{"fire.steady": {SHA256: input, Choice: "000017.pfm"}}, FPS: s.FPS, Frame: s.Frame, Device: s.Device, Backend: s.Backend, Width: s.Width, Height: s.Height}
	if err := os.Remove(s.BinaryPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(s.Inputs["fire.steady"].Path); err != nil {
		t.Fatal(err)
	}
	got, err := BuildPrehashed(p)
	if err != nil || got != want {
		t.Fatalf("prehashed differs: %v", err)
	}
	p.Inputs["fire.steady"] = HashedInput{SHA256: strings.Repeat("0", 64), Choice: "000017.pfm"}
	changed, err := BuildPrehashed(p)
	if err != nil || changed == got {
		t.Fatalf("digest ignored: %v", err)
	}
	for _, bad := range []string{"", "abc", strings.Repeat("z", 64), strings.Repeat("A", 64)} {
		p.BinarySHA256 = bad
		if _, err := BuildPrehashed(p); err == nil {
			t.Fatalf("accepted digest %q", bad)
		}
	}
}
