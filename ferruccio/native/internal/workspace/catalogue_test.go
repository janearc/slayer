package workspace

import (
	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"os"
	"path/filepath"
	"testing"
)

// TestCatalogueWorkspaceSelectsReceiptSpecimen keeps settings renderable on open.
func TestCatalogueWorkspaceSelectsReceiptSpecimen(t *testing.T) {
	source, err := os.ReadFile("../../../examples/subaru-settings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	path := filepath.Join(folder, "settings.yaml")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "render.json"), []byte(`{"specimen":{"entry":{"kind":"subaru"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{File: path})
	if err != nil {
		t.Fatal(err)
	}
	if work.specimen != "subaru" || work.state.document.Kind() != "catalogue" || work.diagnostic != "" {
		t.Fatalf("catalogue load: specimen=%q kind=%q diagnostic=%q", work.specimen, work.state.document.Kind(), work.diagnostic)
	}
	if work.editor.Text() != string(source) {
		t.Fatal("loading changed settings")
	}
}

// TestCatalogueWithoutReceiptKeepsSelectionEmpty never guesses from a folder name.
func TestCatalogueWithoutReceiptKeepsSelectionEmpty(t *testing.T) {
	source, err := os.ReadFile("../../../examples/subaru-settings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(t.TempDir(), "subaru-demo")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(folder, "settings.yaml")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{File: path})
	if err != nil {
		t.Fatal(err)
	}
	if work.specimen != "" {
		t.Fatal("folder name selected a specimen")
	}
	if work.state.document.Text() != string(source) {
		t.Fatal("missing receipt changed settings")
	}
}
