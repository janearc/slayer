//go:build draw

package main

import (
	"io"
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// mustAsk is what the command is asked by these arguments, drawn small.
func mustAsk(t *testing.T, arguments ...string) asked {
	t.Helper()
	given, err := parse(append([]string{"-w", "36", "-h", "25", "-n", "1",
		"-out", t.TempDir() + "/moonrise.pfm"}, arguments...), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return given
}

// mustScene is the scene as asked, built through the registry, or the
// test fails.
func mustScene(t *testing.T, given asked) *named.Built {
	t.Helper()
	scene, err := named.Get("moonrise")
	if err != nil {
		t.Fatal(err)
	}
	built, err := scene.Build(given.with(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return built
}

// at the middle of the frame the moon is lit: it is clear of the
// earth's shadow, which an exactly opposite sun would put it in.
func TestTheMoonIsLitNotEclipsed(t *testing.T) {
	given := mustAsk(t, "-moon", "1.5", "-sun", "2")
	picture, err := draw(mustScene(t, given), given)
	if err != nil {
		t.Fatal(err)
	}
	if middle := picture.At(given.width/2, given.height/2); middle.R <= 0 {
		t.Errorf("the moon's middle is %v", middle)
	}
}
