package cheeptune

import (
	"errors"
	"strings"
	"testing"
)

func TestWhyIsOneLine(t *testing.T) {
	err := errors.New("first thing\nsecond thing\nthird thing")
	got := why(err)
	if strings.Contains(got, "\n") {
		t.Errorf("why kept a newline: %q", got)
	}
	if !strings.HasPrefix(got, "first thing") ||
		!strings.HasSuffix(got, "(and more)") {
		t.Errorf("why gave %q; want the first line and a mark that "+
			"there is more", got)
	}
}
