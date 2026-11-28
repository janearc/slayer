package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const savingImport = `  world:
    unit: 1
    materials:
      matte:
        colour: {srgb_linear: [0.32, 0.38, 0.42]}
        source: eye
    imports:
      figure:
        file: meshes/figure.gltf # keep the asset comment
        at: [7, 3, 5]
        materials: {material/0/human-fixture-slot: matte}
        source: "Self-authored CC0 mannequin"
        license: CC0-1.0
`

func TestSaveAsRetainsTheImportedAssetAndBuffer(t *testing.T) {
	work, original, target := importedSaveFixture(t)
	edited := strings.Replace(work.state.document.Text(), "at: [7, 3, 5]", "at: [5, 3, 5]", 1)
	work.editor.ApplyText(edited)
	if !work.saveAt(target) {
		t.Fatal(work.diagnostic)
	}
	reopened, _, err := ReadDocument(target)
	if err != nil || reopened.Text() != edited {
		t.Fatalf("imported draft lost authored text: %v", err)
	}
	for _, name := range []string{"meshes/figure.gltf", "meshes/two-metre-human.bin"} {
		old, err := os.ReadFile(filepath.Join(filepath.Dir(original), name))
		if err != nil {
			t.Fatal(err)
		}
		copied, err := os.ReadFile(filepath.Join(filepath.Dir(target), name))
		if err != nil || !bytes.Equal(old, copied) {
			t.Fatalf("copied resource %s differs: %v", name, err)
		}
	}
	before, err := os.ReadFile(original)
	if err != nil || !strings.Contains(string(before), "at: [7, 3, 5]") {
		t.Fatalf("save changed the original set: %v", err)
	}
}

func TestSaveAsImportConflictLeavesExistingFilesAndDraft(t *testing.T) {
	work, original, target := importedSaveFixture(t)
	meshFolder := filepath.Join(filepath.Dir(target), "meshes")
	if err := os.Mkdir(meshFolder, 0700); err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(meshFolder, "two-metre-human.bin")
	if err := os.WriteFile(conflict, []byte("destination asset"), 0600); err != nil {
		t.Fatal(err)
	}
	before := work.state.document.Text()
	if work.saveAt(target) {
		t.Fatal("save accepted a conflicting buffer")
	}
	got, err := os.ReadFile(conflict)
	if err != nil || string(got) != "destination asset" {
		t.Fatalf("existing asset changed: %q, %v", got, err)
	}
	for _, path := range []string{target, filepath.Join(meshFolder, "figure.gltf")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("failed save left %s: %v", path, err)
		}
	}
	if work.state.path != original || work.state.document.Text() != before {
		t.Fatal("failed save changed the working document")
	}
}

func TestSaveAsImportWriteFailureRemovesNewResources(t *testing.T) {
	work, original, target := importedSaveFixture(t)
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if work.saveAt(target) {
		t.Fatal("scene write succeeded over a directory")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(target), "meshes")); !os.IsNotExist(err) {
		t.Fatalf("failed save retained a new asset directory: %v", err)
	}
	if work.state.path != original {
		t.Fatal("failed save changed the working path")
	}
}

func TestSaveAsImportRefusesLinkedDestinationFolder(t *testing.T) {
	work, _, target := importedSaveFixture(t)
	outside := t.TempDir()
	parent := filepath.Dir(target)
	if err := os.Symlink(outside, filepath.Join(parent, "meshes")); err != nil {
		t.Fatal(err)
	}
	if work.saveAt(target) {
		t.Fatal("saved imports through a link to another folder")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("save touched the linked folder: %v, %v", entries, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("failed save left the destination file: %v", err)
	}
}

func TestSaveAsKeepsUnfinishedImportDraftAndReportsUncheckedAssets(t *testing.T) {
	for _, test := range []struct{ name, old, replacement string }{
		{"syntax", "at: [7, 3, 5]", "at: [7, 3,"},
		{"producer", "human-fixture-slot: matte", "human-fixture-slot: missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			work, original, target := importedSaveFixture(t)
			before := work.state.document.Text()
			edited := strings.Replace(before, test.old, test.replacement, 1)
			work.editor.ApplyText(edited)
			if !work.saveAt(target) {
				t.Fatal(work.diagnostic)
			}
			raw, err := os.ReadFile(target)
			if err != nil || string(raw) != edited || work.state.document.Dirty() {
				t.Fatalf("unfinished draft changed on save: %v", err)
			}
			if !strings.Contains(work.diagnostic, "could not be checked or copied") {
				t.Fatalf("save gave no dependency notice: %q", work.diagnostic)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(target), "meshes")); !os.IsNotExist(err) {
				t.Fatalf("recovery save left unverified assets: %v", err)
			}
			raw, err = os.ReadFile(original)
			if err != nil || string(raw) != before {
				t.Fatalf("recovery save changed the original: %v", err)
			}
		})
	}
}

func importedSaveFixture(t *testing.T) (*Workspace, string, string) {
	t.Helper()
	sourceFolder := t.TempDir()
	meshFolder := filepath.Join(sourceFolder, "meshes")
	if err := os.Mkdir(meshFolder, 0700); err != nil {
		t.Fatal(err)
	}
	for fixture, name := range map[string]string{
		"two-metre-human.gltf": "figure.gltf",
		"two-metre-human.bin":  "two-metre-human.bin",
	} {
		source := filepath.Join("../../../../oakland/testdata", fixture)
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(meshFolder, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(sourceFolder, "scene.yaml")
	source := strings.Replace(checkingSource, "  film:", savingImport+"  film:", 1)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.openPath(path)
	if work.state.path != path {
		t.Fatal(work.diagnostic)
	}
	return work, path, filepath.Join(t.TempDir(), "copy.yaml")
}
