package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// Fetch refuses with no shelf named, and says how to name one.
func TestFetchRefusesWithNoShelfNamed(t *testing.T) {
	at, made := verifying("6.84", "6.84")
	defer made.done()
	at.unnamed = errors.New("no shelf; set SLAYER_SHELF")
	var out bytes.Buffer
	code := tool(at).Run([]string{"fetch"}, &out, &out)
	if code != 1 || !strings.Contains(out.String(), "SLAYER_SHELF") {
		t.Errorf("fetch with no shelf gave %d: %s", code, out.String())
	}
}
