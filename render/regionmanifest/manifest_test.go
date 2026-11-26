package regionmanifest

import (
	"math"
	"testing"
)

func example() Manifest {
	zero := int64(0)
	full := false
	return Manifest{ClosureAsserted: true,
		Inputs: []Input{{Name: "set.yaml", Bytes: 2, SHA256: [32]byte{1}},
			{Name: "shelf/terrain.tif", Bytes: 4, SHA256: [32]byte{2}}},
		Executable: [32]byte{3}, Width: 8, Height: 6,
		Region:   Rectangle{Left: 1, Top: 2, Width: 3, Height: 4},
		Renderer: "cpu", Rays: 4, Camera: "street", TimeNS: &zero,
		Settings: [32]byte{4}, Development: [32]byte{5}, Scrooge: &full}
}

func TestDigestDoesNotDependOnInputOrder(t *testing.T) {
	first := example()
	second := example()
	second.Inputs[0], second.Inputs[1] = second.Inputs[1], second.Inputs[0]
	a, err := Digest(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Digest(second)
	if err != nil || a != b || a == ([32]byte{}) {
		t.Fatalf("digests %x, %x, %v", a, b, err)
	}
}

func TestChangingAFieldChangesIdentity(t *testing.T) {
	first, err := Digest(example())
	if err != nil {
		t.Fatal(err)
	}
	changes := map[string]func(*Manifest){
		"input bytes":  func(m *Manifest) { m.Inputs[0].Bytes++ },
		"input digest": func(m *Manifest) { m.Inputs[0].SHA256[0]++ },
		"executable":   func(m *Manifest) { m.Executable[0]++ },
		"frame":        func(m *Manifest) { m.Width++ },
		"rectangle":    func(m *Manifest) { m.Region.Left++ },
		"renderer":     func(m *Manifest) { m.Renderer = "gpu" },
		"rays":         func(m *Manifest) { m.Rays++ },
		"camera":       func(m *Manifest) { m.Camera = "other" },
		"time":         func(m *Manifest) { later := int64(1); m.TimeNS = &later },
		"settings":     func(m *Manifest) { m.Settings[0]++ },
		"development":  func(m *Manifest) { m.Development[0]++ },
		"detail mode":  func(m *Manifest) { on := true; m.Scrooge = &on },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			changed := example()
			change(&changed)
			got, err := Digest(changed)
			if err != nil || got == first {
				t.Fatalf("identity %x, %v", got, err)
			}
		})
	}
}

func TestInvalidManifestIsRefused(t *testing.T) {
	cases := map[string]func(*Manifest){
		"no closure assertion": func(m *Manifest) { m.ClosureAsserted = false },
		"no inputs":            func(m *Manifest) { m.Inputs = nil },
		"duplicate":            func(m *Manifest) { m.Inputs[1].Name = m.Inputs[0].Name },
		"escaping name":        func(m *Manifest) { m.Inputs[0].Name = "../set" },
		"unclean name":         func(m *Manifest) { m.Inputs[0].Name = "sub/../set" },
		"negative size":        func(m *Manifest) { m.Inputs[0].Bytes = -1 },
		"missing input digest": func(m *Manifest) { m.Inputs[0].SHA256 = [32]byte{} },
		"missing executable":   func(m *Manifest) { m.Executable = [32]byte{} },
		"invalid frame":        func(m *Manifest) { m.Width = 0 },
		"overflow rectangle":   func(m *Manifest) { m.Region.Left = math.MaxInt },
		"outside rectangle":    func(m *Manifest) { m.Region.Width = 8 },
		"empty renderer":       func(m *Manifest) { m.Renderer = "" },
		"invalid rays":         func(m *Manifest) { m.Rays = 0 },
		"empty camera":         func(m *Manifest) { m.Camera = "" },
		"missing time":         func(m *Manifest) { m.TimeNS = nil },
		"missing settings":     func(m *Manifest) { m.Settings = [32]byte{} },
		"missing development":  func(m *Manifest) { m.Development = [32]byte{} },
		"missing detail mode":  func(m *Manifest) { m.Scrooge = nil },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			m := example()
			change(&m)
			got, err := Digest(m)
			if err == nil || got != ([32]byte{}) {
				t.Fatalf("accepted invalid manifest: %x, %v", got, err)
			}
		})
	}
}
