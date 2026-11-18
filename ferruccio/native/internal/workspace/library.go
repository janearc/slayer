package workspace

import (
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sceneEntry struct{ name, label string }

// scenePicker discovers the bundled library rather than maintaining a second catalogue.
func (work *Workspace) scenePicker() widget.Widget {
	if work.library == nil {
		work.library = readSceneLibrary(work.options.Examples)
	}
	labels := make([]string, len(work.library))
	selected := -1
	for i, entry := range work.library {
		labels[i] = entry.label
		if filepath.Base(entry.name) == work.state.document.Name() {
			selected = i
		}
	}
	if len(labels) == 0 {
		return work.text(work.state.document.Name())
	}
	var control *dropdown.Widget
	control = dropdown.New(dropdown.Items(labels...), dropdown.Selected(selected), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(i int, _ string) {
		if i >= 0 && i < len(work.library) {
			// Close first: upstream close removes overlays above the menu too.
			control.Close(work.ui.Window().Context())
			work.chooseExample(work.library[i].name)
		}
	}))
	control.SetID("Scene library")
	return control
}

// readSceneLibrary lists the scene entries in directory, without duplicates.
func readSceneLibrary(directory string) []sceneEntry {
	entries := []sceneEntry{}
	seen := map[string]bool{}
	_ = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "oakland-larson" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}
		if entry.Name() == "sources.yaml" {
			return nil
		}
		if seen[entry.Name()] {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		document, err := draft.Open(entry.Name(), raw)
		if err != nil || document.Kind() == "catalogue" {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return nil
		}
		label, err := document.Value("scene", "name")
		if err != nil || label == "" {
			label = strings.TrimSuffix(entry.Name(), filepath.Ext(path))
		}
		label = strings.ReplaceAll(label, "-", " ")
		entries = append(entries, sceneEntry{name: relative, label: label})
		seen[entry.Name()] = true
		return nil
	})
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].label < entries[j].label })
	return entries
}

// resetDocumentView prevents old pixels or gestures from being attributed to a new set.
func (work *Workspace) resetDocumentView() {
	work.stopPreview()
	work.viewer.dragging = false
	if work.viewer.ball != nil {
		work.viewer.ball.dragging = false
	}
	if work.viewer.travel != nil {
		work.viewer.travel.dragging = false
	}
	if work.ui != nil {
		work.ui.Window().Context().ReleasePointer(work.viewer)
	}
	work.viewer.set(nil)
	work.viewer.SetNavigable(false)
	work.working.photograph = nil
	work.working.scene = nil
	work.working.gesture = nil
	work.working.source = ""
	work.working.path = ""
	work.working.failure = ""
	work.working.displayed = false
	work.working.waiting = false
	work.working.dirty = true
	work.play.geometry = nil
	work.play.selected = ""
	work.play.renderedText = ""
	work.output = ""
	work.acceptedPath = ""
	work.acceptedText = ""
	work.caption = "Preparing this scene"
}
