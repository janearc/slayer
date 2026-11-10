package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRigKeepsTheActualShutterRefusalBeforeExposure(t *testing.T) {
	given := mustParse(t)
	given.set = onFixtureRigged(t, "0.5", false)
	world, _, given, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	given.width, given.height = 4, 3
	given.glimpses = 0
	_, cause := instants(given, *given.eye)
	if cause == nil || !strings.HasPrefix(cause.Error(), "take: a camera is cranked") {
		t.Fatalf("fixture lost the actual shutter constructor refusal: %v", cause)
	}
	var panicValue interface{}
	func() {
		defer func() { panicValue = recover() }()
		_, err = exposed(world, given, *given.eye)
	}()
	if panicValue != nil || !errors.Is(err, cause) {
		t.Fatalf("shutter failure became a different exposure failure: %v, panic %v, want %v", err, panicValue, cause)
	}
}
