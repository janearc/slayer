package hillside

import (
	"os"
	"strings"
	"testing"
)

// changed is the set file at path with each pair of changes made to it,
// the first of each pair replaced by the second; and its path.
func changed(t *testing.T, path string, changes ...string) string {
	t.Helper()
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scene := string(written)
	for i := 0; i+1 < len(changes); i += 2 {
		if !strings.Contains(scene, changes[i]) {
			t.Fatalf("the scene has no %q to change", changes[i])
		}
		scene = strings.Replace(scene, changes[i], changes[i+1], 1)
	}
	os.WriteFile(path, []byte(scene), 0o644)
	return path
}
