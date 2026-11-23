package session

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// an argument that is no flag is refused by name, with what to say
// instead: a set's path given without -set used to be ignored, with every
// flag after it, and the default hillside drawn.
func TestAnArgumentThatIsNoFlagIsRefused(t *testing.T) {
	_, err := parse([]string{"-w", "96", "sets/melancholia.yaml", "-gpu"},
		io.Discard)
	if !errors.Is(err, errAsked) {
		t.Fatalf("a bare path was taken: %v", err)
	}
	for _, want := range []string{"sets/melancholia.yaml", "-set"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not say %q", err, want)
		}
	}
	if _, err := parse([]string{"-w", "96", "-h", "60"}, io.Discard); err != nil {
		t.Errorf("flags alone were refused: %v", err)
	}
}
