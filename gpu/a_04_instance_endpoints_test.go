package gpu

import (
	"strings"
	"testing"
)

// Even an empty scene keeps one hierarchy root, but no instance records.
func TestEmptyInstanceHierarchyFitsItsExactRootBudget(t *testing.T) {
	rootBytes := nodeStruct.size()
	for _, test := range []struct {
		name    string
		bytes   int
		refused bool
	}{
		{"below one root", rootBytes - 1, true},
		{"exactly one root", rootBytes, false},
		{"one byte above the root", rootBytes + 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded := newScene()
			err := encoded.instancesWithin(test.bytes)
			if test.refused {
				if err == nil || !strings.Contains(err.Error(), "instance hierarchy") {
					t.Fatalf("short root budget lost its refusal: %v", err)
				}
				if len(encoded.tops.bytes) != 0 || len(encoded.instances.bytes) != 0 {
					t.Fatal("refused root partially packed buffers")
				}
				return
			}
			if err != nil {
				t.Fatalf("empty root did not fit its actual storage: %v", err)
			}
			if len(encoded.tops.bytes) != rootBytes || len(encoded.instances.bytes) != 0 ||
				encoded.counts.tops != 1 || encoded.counts.instances != 0 {
				t.Fatal("empty scene packed something other than one root and zero instances")
			}
		})
	}
}
