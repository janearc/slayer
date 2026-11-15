package hillside

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSequenceAssetIdentityIncludesActualBytesAndMissingFiles(test *testing.T) {
	path := filepath.Join(test.TempDir(), "data")
	sequenceWrite(test, path, []byte("original"))
	first, err := sequenceAssetHash([]string{path})
	if err != nil {
		test.Fatal(err)
	}
	sequenceWrite(test, path, []byte("changed"))
	second, err := sequenceAssetHash([]string{path})
	if err != nil || first == second {
		test.Fatal("changing input bytes did not change identity", err)
	}
	if err := os.Remove(path); err != nil {
		test.Fatal(err)
	}
	if _, err := sequenceAssetHash([]string{path}); err == nil {
		test.Fatal("missing input retained its identity")
	}
}

func TestSequenceSchemaRequiresExactlyOneCompleteSceneRoute(test *testing.T) {
	raw, err := os.ReadFile("../doc/camera-smoke-sequence.yaml")
	if err != nil {
		test.Fatal(err)
	}
	for _, change := range []struct{ from, to string }{
		{"smoke: 1", "smoke: 1\nscene: scene.yaml"},
		{"film: {width: 640, height: 360, rays: 1}", ""},
		{"smoke: 1", "scene: scene.yaml"},
		{"smoke: 1", "smoke: 99999"},
	} {
		text := strings.Replace(string(raw), change.from, change.to, sequenceOne)
		path := filepath.Join(test.TempDir(), "invalid.yaml")
		sequenceWrite(test, path, []byte(text))
		var out, wrong bytes.Buffer
		args := []string{"sequence", "--plan", path, "--dry-run"}
		if code := sequenceRun(args, &out, &wrong); code == sequenceZero {
			test.Fatal("accepted incomplete or conflicting scene route", change.to)
		}
	}
}
