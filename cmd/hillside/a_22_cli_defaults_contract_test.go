package main

import (
	"io"
	"testing"
)

// mustParse intentionally shrinks frames for rendering tests. Read the real
// empty command line here so those overrides cannot hide changed defaults.
func TestDefaultCommandLineKeepsItsPublicRenderSettings(t *testing.T) {
	given, err := parse(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if given.width != 1280 || given.height != 800 || given.samples != 2 {
		t.Fatalf("default film changed: %dx%d, %d samples per side", given.width, given.height, given.samples)
	}
	if given.faces != 1500000 || given.exposure != 2000 || given.apart != 2.5 {
		t.Fatalf("default budget/exposure/spacing changed: %d/%g/%g", given.faces, given.exposure, given.apart)
	}
	if given.wide != 55 || given.tilt != 90 || given.seed != 1 || given.variety != 0 || given.at != 0 {
		t.Fatal("default view, replay seed or time changed")
	}
	if given.night || given.onGPU || given.above || given.earth || given.sky != "painted" {
		t.Fatal("default lighting, backend or scenery mode changed")
	}
}
