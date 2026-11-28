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

const playgroundExample = "ferruccio-playground.yaml"
const campExample = "larson-camp.yaml"

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
		if work.currentLibraryEntry(entry.name) {
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
	painter := appearance.DropdownPainter{Theme: work.theme, Size: work.size, TriggerPrefix: "Examples · "}
	var control *dropdown.Widget
	control = dropdown.New(dropdown.Items(labels...), dropdown.Selected(selected), dropdown.PainterOpt(painter), dropdown.OnChange(func(i int, _ string) {
		if i >= 0 && i < len(work.library) {
			// Close first: upstream close removes overlays above the menu too.
			control.Close(work.ui.Window().Context())
			work.chooseExample(work.library[i].name)
		}
	}))
	control.SetID("Scene library")
	return control
}

// currentLibraryEntry recognizes both the bundled filename and its editable copy.
func (work *Workspace) currentLibraryEntry(name string) bool {
	if work.exampleDraft {
		if study, ok := studyForExample(name); ok {
			return work.study == study.id
		}
		return work.study == "" && work.state.path == filepath.Join(work.exampleFolder, name)
	}
	return work.state.path != "" && filepath.Clean(work.state.path) == filepath.Join(work.examplesFolder(), name)
}

// readSceneLibrary lists scenes and the manifest entry of the bundled study.
func readSceneLibrary(directory string) []sceneEntry {
	entries := []sceneEntry{}
	studyPrefix := oaklandLarsonStudy.folder + string(filepath.Separator)
	_ = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return nil
		}
		inStudy := strings.HasPrefix(relative, studyPrefix)
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") || inStudy {
				return filepath.SkipDir
			}
			return nil
		}
		study, isStudy := studyForExample(relative)
		if inStudy && (!isStudy || !entry.Type().IsRegular()) {
			return nil
		}
		if scene, ok := readSceneEntry(path, relative); ok {
			if isStudy {
				scene.label = study.label
			}
			entries = append(entries, scene)
		}
		return nil
	})
	return sortedLibrary(entries)
}

// readSceneEntry reads the display name from a scene, with its filename as fallback.
func readSceneEntry(path, relative string) (sceneEntry, bool) {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	if (ext != ".yaml" && ext != ".yml") || name == "sources.yaml" {
		return sceneEntry{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return sceneEntry{}, false
	}
	document, err := draft.Open(name, raw)
	if err != nil || document.Kind() == "catalogue" {
		return sceneEntry{}, false
	}
	label, err := document.Value("scene", "name")
	if err != nil || label == "" {
		label = strings.TrimSuffix(name, ext)
	}
	return sceneEntry{name: relative, label: strings.ReplaceAll(label, "-", " ")}, true
}

// sortedLibrary keeps the starter scenes first, then sorts the rest by label.
// Entries that share a label show their relative paths.
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
	sort.SliceStable(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if leftStart, rightStart := starterOrder(left.name), starterOrder(right.name); leftStart != rightStart {
			return leftStart < rightStart
		}
		return left.label < right.label
	})
	return entries
}

// starterOrder keeps first-run scenes visible in the initial menu.
func starterOrder(name string) int {
	if name == playgroundExample {
		return 0
	}
	if name == campExample {
		return 1
	}
	if _, ok := studyForExample(name); ok {
		return 2
	}
	return 3
}

// resetDocumentView prevents old pixels or gestures from being attributed to a new set.
func (work *Workspace) resetDocumentView() {
	work.stopPreview()
	work.play.disclosure = ""
	work.working.cameraName, work.working.cameraWide, work.working.cameraPath = "", 0, ""
	work.working.cameraPathAt = 0
	work.working.detail, work.working.detailFallback = "", nil
	work.working.detailRetry = nil
	if work.working.gpu != nil {
		if err := work.working.gpu.DiscardScene(); err != nil {
			work.log.Write([]byte("discard working scene: " + err.Error() + "\n"))
		}
	}
	if work.working.stager != nil {
		work.working.stager.discard()
		work.working.staging = false
	}
	work.cancelViewportGesture()
	work.viewer.set(nil)
	work.viewer.SetNavigable(false)
	work.working.photograph = nil
	work.working.scene = nil
	work.working.inspection = nil
	work.working.parsed = nil
	work.working.stageAuthoring = nil
	work.working.rollback = nil
	work.working.preview = nil
	work.working.poseHistory = nil
	work.working.provisional = false
	work.working.plan, work.working.planFraming, work.working.fullScene = nil, nil, false
	work.working.source = ""
	work.working.path = ""
	work.working.failure = ""
	work.working.notice = ""
	work.resetWorkingFrame()
	work.working.displayed = false
	work.working.waiting = false
	work.working.dirty = true
	work.play.selected = ""
	work.play.renderedText = ""
	work.output = ""
	work.acceptedPath = ""
	work.acceptedText = ""
	work.caption = "Preparing this scene"
}
