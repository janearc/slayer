package hillside

import (
	"io"
	"strings"
	"testing"
)

// -set says the world and the film, so a flag that says them too is
// refused, by its name; -w, -h and -n are settings of the picture and are
// not refused (see TestSizeAndRaysAreSettingsOfASet).
func TestASetIsRefusedBesideTheFlagsWorld(t *testing.T) {
	for _, flag := range []string{"-spruce", "-seed", "-exposure", "-alight"} {
		_, err := parse([]string{"-set", "x.yaml", flag, "2"}, io.Discard)
		if err == nil || !strings.Contains(err.Error(), flag) {
			t.Errorf("%s beside -set: %v", flag, err)
		}
	}
	if _, err := parse([]string{"-set", "x.yaml", "-raw", "r.pfm"},
		io.Discard); err != nil {
		t.Errorf("-raw beside -set: %v", err)
	}
}
