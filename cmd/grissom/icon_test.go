package main

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/internal/appicon"
)

func TestApplicationIconsLoadAsNativeImages(test *testing.T) {
	if err := installIcons(); err != nil {
		test.Fatal(err)
	}
}

func TestApplicationIconRefusalsNameTheMissingAppearance(test *testing.T) {
	light, err := appicon.PNG(false)
	if err != nil {
		test.Fatal(err)
	}
	dark, err := appicon.PNG(true)
	if err != nil {
		test.Fatal(err)
	}
	if err := nativeIcons(nil, dark); err == nil ||
		!strings.Contains(err.Error(), "light") {
		test.Fatal("missing light artwork was not explained", err)
	}
	if err := nativeIcons(light, nil); err == nil ||
		!strings.Contains(err.Error(), "dark") {
		test.Fatal("missing dark artwork was not explained", err)
	}
}
