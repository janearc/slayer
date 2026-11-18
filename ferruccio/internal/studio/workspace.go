// Package studio is the Fyne interface for one document workspace.
package studio

import (
	"context"
	"fmt"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"github.com/janearc/slayer/ferruccio/internal/draft"
)

// Options identifies the owning window and the reference study to open.
type Options struct {
	Window   fyne.Window
	Study    string
	Tools    preview.Tools
	Examples string
}

// Workspace keeps document state separate from its view objects.
type Workspace struct {
	window                           fyne.Window
	document                         *draft.Document
	editor                           *widget.Entry
	quality, renderer, preset, study *widget.Select
	content                          fyne.CanvasObject
	updating                         bool
	status, diagnostic, fileLabel    *widget.Label
	worldTitle, worldSubtitle        *canvas.Text
	caption, creditDetail            *canvas.Text
	picture                          *canvas.Image
	view                             *fyne.Container
	setContent                       fyne.CanvasObject
	inspectorTabs                    *container.AppTabs
	navigationButtons                map[string]*widget.Button
	reference                        string
	sourcePath, examples             string
	tools                            preview.Tools
	renderButton, stopButton         *widget.Button
	renderState                      *widget.Label
	progress                         *widget.ProgressBarInfinite
	renderLog                        *widget.Entry
	cancelRender                     context.CancelFunc
	renderDone                       chan struct{}
	textSize                         float32
	lastPreview                      string
	previewError                     error
	canvasSizes                      map[*canvas.Text]float32
}

// New prepares an independent editor window, without starting a render.
func New(options Options) (*Workspace, error) {
	if options.Window == nil {
		return nil, fmt.Errorf("a workspace needs a window")
	}
	if options.Study == "" {
		options.Study = "venus"
	}
	if options.Study != "venus" && options.Study != "titan" {
		return nil, fmt.Errorf("unknown reference study %q", options.Study)
	}
	schema, err := assets.Read("job.schema.json")
	if err != nil {
		return nil, err
	}
	document, err := draft.New(draft.Options{Name: options.Study + ".job.yaml", Source: []byte(initialJob(options.Study)), Schema: schema})
	if err != nil {
		return nil, err
	}
	workspace := &Workspace{window: options.Window, document: document, reference: options.Study, tools: options.Tools, examples: options.Examples, textSize: fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)}
	workspace.makeControls()
	workspace.makeRenderControls()
	workspace.content = workspace.compose()
	workspace.showStudy(options.Study)
	workspace.syncDocument()
	workspace.protectClose()
	return workspace, nil
}

// Content is the complete workspace for the application's window.
func (workspace *Workspace) Content() fyne.CanvasObject { return workspace.content }

// fyneColor keeps colour names typed at the theme boundary.
func fyneColor(name string) fyne.ThemeColorName { return fyne.ThemeColorName(name) }

// initialJob is an authored starting point using the existing Slayer contract.
func initialJob(study string) string {
	return "# A planetary study for Slayer.\n# Terrain and render submission are not connected in this prototype.\n" +
		"$schema: https://github.com/janearc/slayer/schemas/job-v1.json\n" +
		"scene:\n  set: scenes/" + study + ".yaml\nfilm:\n  preset: 720\n  wide: 62\n  rays: 1\n" +
		"renderer: gpu\nquality: preview\nat: {value: 0, unit: s}\n"
}

// makeControls binds actual widgets to the document, not a second settings model.
func (workspace *Workspace) makeControls() {
	workspace.status = widget.NewLabel("Ready to explore")
	workspace.diagnostic = widget.NewLabel("")
	workspace.diagnostic.Wrapping = fyne.TextWrapWord
	workspace.fileLabel = widget.NewLabel(workspace.document.Name())
	workspace.editor = widget.NewMultiLineEntry()
	workspace.editor.TextStyle = fyne.TextStyle{Monospace: true}
	workspace.editor.Wrapping = fyne.TextWrapOff
	workspace.editor.OnChanged = func(text string) {
		if workspace.updating {
			return
		}
		workspace.document.Replace(text)
		workspace.syncDocument()
	}
	workspace.quality = workspace.choice([]string{"preview", "exact"}, []string{"quality"})
	workspace.renderer = workspace.choice([]string{"gpu", "cpu", "gpu-native-shadows", "gpu-native"}, []string{"renderer"})
	workspace.preset = workspace.choice([]string{"720", "320", "portrait", "portrait-preview", "2k", "4k", "imax", "imax-2k", "video-preview", "video-preview-720"}, []string{"film", "preset"})
	workspace.study = widget.NewSelect([]string{"Venus", "Titan"}, func(name string) {
		if !workspace.updating {
			workspace.chooseStudy(strings.ToLower(name))
		}
	})
}

// choice updates a scalar atomically and leaves a broken text draft intact.
func (workspace *Workspace) choice(options []string, path []string) *widget.Select {
	return widget.NewSelect(options, func(value string) {
		if workspace.updating {
			return
		}
		chosenPath := path
		if workspace.document.Kind() != "job" {
			chosenPath = append([]string{"scene"}, path...)
		}
		err := workspace.document.Set(draft.Change{Path: chosenPath, Value: value, Tag: "!!str"})
		if err != nil {
			workspace.diagnostic.SetText("Keep editing the YAML; this control could not apply: " + err.Error())
		}
		workspace.syncDocument()
	})
}

// syncDocument reconciles the widgets while callbacks are suppressed.
func (workspace *Workspace) syncDocument() {
	workspace.updating = true
	defer func() { workspace.updating = false }()
	if workspace.editor.Text != workspace.document.Text() {
		workspace.editor.SetText(workspace.document.Text())
	}
	for control, path := range map[*widget.Select][]string{workspace.quality: {"quality"}, workspace.renderer: {"renderer"}, workspace.preset: {"film", "preset"}} {
		if workspace.document.Kind() != "job" {
			path = append([]string{"scene"}, path...)
		}
		if value, err := workspace.document.Value(path...); err == nil {
			control.SetSelected(value)
		} else {
			control.ClearSelected()
		}
	}
	name := workspace.document.Name()
	if workspace.document.Dirty() {
		name += " · unsaved"
	}
	workspace.fileLabel.SetText(name)
	if err := workspace.document.Check(); err != nil {
		workspace.quality.Disable()
		workspace.renderer.Disable()
		workspace.preset.Disable()
		workspace.status.SetText("Draft needs attention")
		workspace.diagnostic.SetText(err.Error())
		workspace.renderButton.Disable()
	} else {
		workspace.quality.Enable()
		workspace.renderer.Enable()
		workspace.preset.Enable()
		if workspace.document.Kind() != "job" {
			workspace.quality.Disable()
			workspace.preset.Disable()
		}
		workspace.status.SetText("Ready · " + workspace.document.Kind())
		workspace.diagnostic.SetText("Slayer " + workspace.document.Kind() + " · local validation")
		if workspace.document.Kind() == "legacy-set" {
			workspace.diagnostic.SetText("Legacy set · syntax checked; Slayer checks the scene")
		}
		if workspace.cancelRender == nil && workspace.sourcePath != "" && workspace.tools.Game != "" && workspace.tools.Slayer != "" {
			workspace.renderButton.Enable()
		} else {
			workspace.renderButton.Disable()
		}
	}
}

// chooseStudy preserves unrelated film, camera and music settings.
func (workspace *Workspace) chooseStudy(study string) {
	if workspace.sourcePath != "" {
		workspace.showStudy(study)
		workspace.renderState.SetText("Reference only · your YAML has not changed")
		return
	}
	if err := workspace.document.Set(draft.Change{Path: []string{"scene", "set"}, Value: "scenes/" + study + ".yaml", Tag: "!!str"}); err != nil {
		workspace.status.SetText("Study not changed · fix the draft first")
		return
	}
	workspace.reference = study
	workspace.showStudy(study)
	workspace.syncDocument()
}

// compose keeps motion and music as future spaces rather than fake controls.
func (workspace *Workspace) compose() fyne.CanvasObject {
	workspace.view = container.NewStack()
	workspace.setContent = workspace.setView()
	workspace.view.Objects = []fyne.CanvasObject{workspace.setContent}
	columns := container.New(&columnLayout{}, workspace.navigation(), workspace.view, workspace.inspector())
	return container.NewBorder(workspace.toolbar(), workspace.footer(), nil, nil, pad(22, columns))
}

// toolbar is deliberately quiet; the landscape is the centre of the workspace.
func (workspace *Workspace) toolbar() fyne.CanvasObject {
	brand := container.NewVBox(display("Atelier", 38), text("Worlds for Slayer", 18))
	open := widget.NewButtonWithIcon("Open", theme.FolderOpenIcon(), workspace.openDocument)
	save := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), workspace.saveDocument)
	examples := widget.NewButtonWithIcon("Examples", theme.HomeIcon(), workspace.openExamples)
	appearance := widget.NewSelect([]string{"Text 16", "Text 18", "Text 20", "Text 24"}, func(value string) {
		sizes := map[string]float32{"Text 16": 16, "Text 18": 18, "Text 20": 20, "Text 24": 24}
		workspace.SetTextSize(sizes[value])
	})
	appearance.SetSelected(fmt.Sprintf("Text %.0f", workspace.textSize))
	return pad(16, container.NewHBox(brand, layout.NewSpacer(), appearance, outlined(examples), outlined(open), outlined(save)))
}

// footer reports document state; rendering has no fabricated progress display.
func (workspace *Workspace) footer() fyne.CanvasObject {
	return pad(14, container.NewHBox(workspace.status, layout.NewSpacer(), text("Camera planning and music have a place here.", 18)))
}
