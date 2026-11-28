//go:build cgo

package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
)

// The producer schema lists both source forms and named-scene modifiers.
// Check every form with the real producer reader before dispatch.
func TestWorkingJobCoverage(t *testing.T) {
	var schema struct {
		Properties map[string]struct {
			Const      string
			Properties map[string]struct {
				Type string
			}
		}
	}
	if err := json.Unmarshal(job.Schema(), &schema); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"set": "", "named": "", "smoke": "",
		"frames": "already-rendered frames",
	}
	path := filepath.Join(t.TempDir(), "job.yaml")
	for name, property := range schema.Properties["scene"].Properties {
		if name == "with" || name == "play" {
			continue
		}
		expected, listed := want[name]
		if !listed {
			t.Errorf("producer scene key %q needs a working-view disposition", name)
			continue
		}
		value := `"fixture"`
		switch property.Type {
		case "integer":
			value = "1"
		case "object":
			value = "{}"
		}
		source := fmt.Sprintf("$schema: %s\nscene: {%s: %s}\n", schema.Properties["$schema"].Const, name, value)
		if _, err := job.ReadSource(path, []byte(source)); err != nil {
			t.Errorf("producer scene fixture %q was refused: %v", name, err)
			continue
		}
		delete(want, name)
		_, err := stagingJob(context.Background(), path, source, "job")
		switch {
		case expected == "" && err != nil:
			t.Errorf("producer scene form %q refused at dispatch: %v", name, err)
		case expected != "" && (err == nil || !strings.Contains(err.Error(), expected)):
			t.Errorf("producer scene form %q missing its limitation: %v", name, err)
		}
	}
	if len(want) != 0 {
		t.Errorf("producer schema no longer admits these tested scene forms: %v", want)
	}
}
