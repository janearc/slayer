package main

import (
	"flag"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeStudyEntryPoints(t *testing.T) {
	if srgb(0) != 0 || srgb(1) != 255 {
		t.Fatal("output transfer endpoints")
	}
	oldArgs, oldFlags := os.Args, flag.CommandLine
	defer func() { os.Args = oldArgs; flag.CommandLine = oldFlags }()
	for _, args := range [][]string{{"-form", "shallow", "-undertone", "red", "-legacy-contour", "-lateral-attachment", "-clay", "-view", "90"}, {"-form", "full", "-undertone", "golden", "-chest-detail", "-tissue", "-arm", "90", "-lateral"}, {"-form", "asymmetric", "-undertone", "olive", "-age", "68"}, {"-form", "reference", "-undertone", "cool", "-proximal-arms", "-pigment-only"}} {
		path := filepath.Join(t.TempDir(), "native.png")
		os.Args = append([]string{"torso", "-width", "48", "-out", path}, args...)
		flag.CommandLine = flag.NewFlagSet("torso", flag.PanicOnError)
		main()
		f, e := os.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		cfg, e := png.DecodeConfig(f)
		f.Close()
		if e != nil || cfg.Width != 48 || cfg.Height != 60 {
			t.Fatal(cfg, e)
		}
	}
	os.Args = []string{"torso", "-sections"}
	flag.CommandLine = flag.NewFlagSet("torso", flag.PanicOnError)
	main()
	os.Args = []string{"torso", "-vertical-sections"}
	flag.CommandLine = flag.NewFlagSet("torso", flag.PanicOnError)
	main()
	os.Args = []string{"torso", "-measure"}
	flag.CommandLine = flag.NewFlagSet("torso", flag.PanicOnError)
	main()
	for _, args := range [][]string{{"-undertone", "none"}, {"-form", "none"}, {"-age", "9"}, {"-out", filepath.Join(t.TempDir(), "missing", "x.png")}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid run accepted")
				}
			}()
			os.Args = append([]string{"torso", "-width", "20"}, args...)
			flag.CommandLine = flag.NewFlagSet("torso", flag.PanicOnError)
			main()
		}()
	}
}
