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
		work.library = readSceneLibrary(work.examplesFolder())
	}
	labels := make([]string, len(work.library))
	selected := -1
	for i, entry := range work.library {
		labels[i] = entry.label
		if work.state.path != "" && filepath.Clean(work.state.path) == filepath.Join(work.examplesFolder(), entry.name) {
			selected = i
		} else if work.exampleDraft && selected < 0 && strings.HasSuffix(work.state.path, string(filepath.Separator)+entry.name) {
			selected = i
		}
	}
	if len(labels) == 0 {
		return work.text(work.state.document.Name())
	}
	if selected < 0 {
		selected = len(labels)
		labels = append(labels, work.state.document.Name())
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

// readSceneLibrary lists every scene entry in directory. Two scenes that
// share a label are told apart by their path from the directory.
func readSceneLibrary(directory string) []sceneEntry {
	entries := []sceneEntry{}
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
		return nil
	})
	return sortedLibrary(entries)
}

// sortedLibrary gives entries that share a label the path that tells them
// apart, then orders all of them by label.
func sortedLibrary(entries []sceneEntry) []sceneEntry {
	count := map[string]int{}
	for _, entry := range entries {
		count[entry.label]++
	}
	for index, entry := range entries {
		if count[entry.label] > 1 {
			entries[index].label = entry.label + " (" + filepath.ToSlash(entry.name) + ")"
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].label < entries[j].label })
	return entries
}

// resetDocumentView prevents old pixels or gestures from being attributed to a new set.
func (work *Workspace) resetDocumentView() {
	work.stopPreview()
	if work.working.stager != nil {
		work.working.stager.discard()
		work.working.staging = false
	}
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
	work.working.parsed = nil
	work.working.gesture = nil
	work.working.source = ""
	work.working.path = ""
	work.working.failure = ""
	work.working.notice = ""
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
