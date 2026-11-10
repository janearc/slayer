package manufacture

import (
	"errors"
	"testing"
)

func TestEveryRepresentationPreservesTheLegacyScaffoldRefusal(t *testing.T) {
	wrong := spruceRow()
	wrong.lowest = span{}
	row := kind{name: "a malformed legacy spruce", how: wrong}
	for _, detail := range Complexities() {
		if _, err := MakeWithComplexity(row, Sow(5), detail); !errors.Is(err, errWhorls) {
			t.Fatalf("%s lost the whorl-count cause: %v", detail, err)
		}
	}
}
