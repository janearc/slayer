package draft

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestRebasedJobTextKeepsFrameAndCueInputs(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "original", "job.yaml")
	to := filepath.Join(root, "copy", "job.yaml")
	for _, row := range []struct{ key, scene string }{
		{"frames", "{frames: plates}"},
		{"play", "{named: zappy, play: plates}"},
	} {
		t.Run(row.key, func(t *testing.T) {
			source := fmt.Sprintf("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: %s\n", row.scene)
			document, err := Open("job.yaml", []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			rebased, err := document.RebasedJobText(source, from, to)
			if err != nil {
				t.Fatal(err)
			}
			document.Replace(rebased)
			relative, err := document.Value("scene", row.key)
			if err != nil || filepath.Join(filepath.Dir(to), relative) != filepath.Join(filepath.Dir(from), "plates") {
				t.Fatalf("relocated %s = %q, %v", row.key, relative, err)
			}
		})
	}
}

func TestRebasedJobTextKeepsAbsoluteAndInvalidInputsVerbatim(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "original", "job.yaml")
	to := filepath.Join(root, "copy", "job.yaml")
	for _, source := range []string{
		"$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: /somewhere/set.yaml}\n",
		"$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: missing bracket\n",
	} {
		document, err := Open("job.yaml", []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		rebased, err := document.RebasedJobText(source, from, to)
		if err != nil || rebased != source {
			t.Fatalf("source changed to %q: %v", rebased, err)
		}
	}
}
