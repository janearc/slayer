package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
)

// TestPlayAutomaticPreviewOwnership separates interactive startup from explicit proof requests.
func TestPlayAutomaticPreviewOwnership(t *testing.T) {
	for _, fixture := range []struct {
		name     string
		suppress bool
	}{
		{"interactive", false}, {"manual proof", true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
				Examples: "../../../examples", Composition: "play", SuppressAutomaticPreview: fixture.suppress,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { work.clearNavigation(); os.RemoveAll(filepath.Dir(work.state.path)) })
			if work.autoPreview == fixture.suppress {
				t.Fatalf("automatic startup=%t; suppression=%t", work.autoPreview, fixture.suppress)
			}
			if work.Running() {
				t.Fatal("constructing workspace started preview work")
			}
			if fixture.suppress {
				work.Tick()
				if work.autoPreview || work.Running() {
					t.Fatal("manual proof tick started automatic preview work")
				}
			}
		})
	}
}
