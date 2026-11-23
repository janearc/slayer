package workspace

import (
	"os"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
)

// TestExampleFolderIsRemovedWhenTheDocumentCloses checks that choosing a
// second example, and then closing, leaves none of the temporary copies.
func TestExampleFolderIsRemovedWhenTheDocumentCloses(t *testing.T) {
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
		Examples: "../../../examples", Composition: "observe",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(work.releaseExample)
	first := work.exampleFolder
	if first == "" {
		t.Fatal("the opening example kept no folder to release")
	}
	if err := work.example("fizz.yaml"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("the replaced example's folder was kept")
	}
	second := work.exampleFolder
	work.releaseExample()
	if _, err := os.Stat(second); !os.IsNotExist(err) || work.exampleFolder != "" {
		t.Fatal("closing the document kept its example folder")
	}
}

// TestExampleFolderStaysWhileTheDocumentIsOpen is the anti-test: the open
// example's copy is still there until the document closes.
func TestExampleFolderStaysWhileTheDocumentIsOpen(t *testing.T) {
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{
		Examples: "../../../examples", Composition: "observe",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(work.releaseExample)
	if _, err := os.Stat(work.state.path); err != nil {
		t.Fatal("the open example's file is gone", err)
	}
}
