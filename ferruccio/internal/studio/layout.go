package studio

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/janearc/slayer/ferruccio/internal/assets"
	"strings"
)

// The three columns remain independent so a future timeline can span the centre.
type columnLayout struct{}

// MinSize reserves enough width for legible controls at the chosen type scale.
func (columnLayout) MinSize(objects []fyne.CanvasObject) fyne.Size { return fyne.NewSize(1190, 760) }

// Layout gives the landscape the space left by the two tool columns.
func (columnLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	left, right, gap := float32(210), float32(420), float32(24)
	if size.Width < 1500 {
		left, right, gap = 175, 360, 16
	}
	centre := size.Width - left - right - 2*gap
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(left, size.Height))
	objects[1].Move(fyne.NewPos(left+gap, 0))
	objects[1].Resize(fyne.NewSize(centre, size.Height))
	objects[2].Move(fyne.NewPos(left+centre+2*gap, 0))
	objects[2].Resize(fyne.NewSize(right, size.Height))
}

// pad keeps spacing independent of the system's widget padding.
func pad(amount float32, content fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewCustomPaddedLayout(amount, amount, amount, amount), content)
}

// text is a single-line label on one of the measured neutral surfaces.
func text(value string, size float32) *canvas.Text {
	label := canvas.NewText(value, pigment(inkHex))
	base := float32(18)
	if fyne.CurrentApp() != nil {
		base = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
	}
	label.TextSize = max(float32(16), size*base/24)
	return label
}

// display uses the bundled licensed face for place names and the studio mark.
func display(value string, size float32) *canvas.Text {
	label := text(value, size)
	label.Color = Theme().Color(theme.ColorNamePrimary, theme.VariantDark)
	font, err := assets.Read("CormorantGaramond-Regular.ttf")
	if err == nil {
		label.FontSource = fyne.NewStaticResource("CormorantGaramond-Regular.ttf", font)
	}
	return label
}

// paragraph wraps explanatory copy instead of reducing its type size.
func paragraph(value string) *widget.Label {
	label := widget.NewLabel(value)
	label.Wrapping = fyne.TextWrapWord
	return label
}

// card groups related controls on the measured panel surface.
func card(content fyne.CanvasObject) *fyne.Container {
	background := canvas.NewRectangle(pigment(panelHex))
	background.CornerRadius = 12
	return container.NewStack(background, pad(18, content))
}

// navigation separates authoring areas without adding still-only assumptions.
func (workspace *Workspace) navigation() fyne.CanvasObject {
	set := widget.NewButtonWithIcon("Set", theme.HomeIcon(), func() { workspace.ShowArea("set") })
	set.Importance = widget.HighImportance
	cameras := widget.NewButtonWithIcon("Cameras", theme.VisibilityIcon(), func() { workspace.ShowArea("cameras") })
	sequence := widget.NewButtonWithIcon("Sequence", theme.MediaVideoIcon(), func() { workspace.ShowArea("sequence") })
	music := widget.NewButtonWithIcon("Music", theme.MediaMusicIcon(), func() { workspace.ShowArea("music") })
	workspace.navigationButtons = map[string]*widget.Button{"set": set, "cameras": cameras, "sequence": sequence, "music": music}
	studies := container.NewVBox(text("References", 18), outlined(widget.NewButton("Venus", func() { workspace.chooseStudy("venus") })), outlined(widget.NewButton("Titan", func() { workspace.chooseStudy("titan") })))
	return container.NewBorder(container.NewVBox(text("Workspace", 18), outlined(set), outlined(cameras), outlined(sequence), outlined(music), widget.NewSeparator(), studies),
		pad(12, container.NewVBox(display("A room", 30), display("for worlds.", 30))), nil, nil)
}

// setView keeps the reference landscape central while its document is edited.
func (workspace *Workspace) setView() fyne.CanvasObject {
	workspace.worldTitle = display("Maat Mons", 58)
	workspace.worldSubtitle = text("Venus · surface study", 22)
	raw, _ := assets.Read("venus.jpg")
	workspace.picture = canvas.NewImageFromResource(fyne.NewStaticResource("venus.jpg", raw))
	workspace.picture.FillMode = canvas.ImageFillCover
	workspace.picture.CornerRadius = 12
	imageArea := container.NewStack(workspace.picture)
	reference := widget.NewButtonWithIcon("Open image", theme.FolderOpenIcon(), workspace.openImage)
	log := widget.NewButtonWithIcon("Output", theme.InfoIcon(), workspace.showRenderLog)
	actions := container.NewHBox(outlined(workspace.renderButton), outlined(workspace.stopButton), outlined(log), layout.NewSpacer(), outlined(reference))
	top := container.NewVBox(workspace.worldSubtitle, workspace.worldTitle, actions, workspace.progress)
	caption := text("Reference visualization · NASA/JPL · not a Slayer render", 18)
	workspace.caption = caption
	credits := container.NewVBox(caption, text("Magellan radar + altimetry · simulated colour · vertical scale 10×", 18))
	workspace.creditDetail = credits.Objects[1].(*canvas.Text)
	ingredients := container.NewGridWithColumns(3,
		card(container.NewVBox(text("Terrain", 24), text("Source selection next", 18))),
		card(container.NewVBox(text("Atmosphere", 24), text("Set authoring next", 18))),
		card(container.NewVBox(text("Objects", 24), text("Placement next", 18))))
	bottom := container.NewVBox(workspace.renderState, credits, widget.NewSeparator(), text("Build the place", 26), ingredients)
	return container.NewBorder(top, bottom, nil, nil, imageArea)
}

// showStudy changes only the reference; it makes no terrain or renderer claim.
func (workspace *Workspace) showStudy(study string) {
	imageName, title, subtitle := "venus.jpg", "Maat Mons", "Venus · surface study"
	if study == "titan" {
		imageName, title, subtitle = "titan-colour.jpg", "After the descent", "Titan · surface study"
	}
	workspace.picture.FillMode = canvas.ImageFillCover
	if study == "titan" {
		workspace.picture.FillMode = canvas.ImageFillContain
	}
	raw, err := assets.Read(imageName)
	if err != nil {
		workspace.status.SetText(err.Error())
		return
	}
	workspace.picture.Resource = fyne.NewStaticResource(imageName, raw)
	workspace.picture.Image = nil
	workspace.picture.File = ""
	workspace.picture.Refresh()
	workspace.worldTitle.Text = title
	workspace.worldTitle.Refresh()
	workspace.worldSubtitle.Text = subtitle
	workspace.worldSubtitle.Refresh()
	if study == "titan" {
		workspace.caption.Text = "Reference photograph · Huygens · not a Slayer render"
		workspace.creditDetail.Text = "ESA / NASA / JPL / University of Arizona · PIA07232 · processed colour"
	} else {
		workspace.caption.Text = "Reference visualization · NASA/JPL · not a Slayer render"
		workspace.creditDetail.Text = "Magellan radar + altimetry · simulated colour · vertical scale 10×"
	}
	workspace.caption.Refresh()
	workspace.creditDetail.Refresh()
	workspace.updating = true
	if study == "titan" {
		workspace.study.SetSelected("Titan")
	} else {
		workspace.study.SetSelected("Venus")
	}
	workspace.updating = false
}

// inspector gives source and settings separate tabs so both can stay legible.
func (workspace *Workspace) inspector() fyne.CanvasObject {
	controls := container.NewVBox(text("Quality", 18), workspace.quality,
		text("Renderer", 18), workspace.renderer)
	upper := container.NewVBox(text("Document", 28), workspace.fileLabel, widget.NewSeparator())
	settings := container.NewVBox(controls, text("Image preset", 18), workspace.preset,
		widget.NewSeparator(), paragraph("These controls edit the same YAML document. Existing camera, timing and sound fields stay in your draft."))
	tabs := container.NewAppTabs(container.NewTabItem("YAML", container.NewThemeOverride(workspace.editor, codeTheme{})),
		container.NewTabItem("Settings", container.NewVScroll(settings)))
	workspace.inspectorTabs = tabs
	validate := widget.NewButtonWithIcon("Validate", theme.ConfirmIcon(), workspace.syncDocument)
	guide := widget.NewButtonWithIcon("Schema guide", theme.InfoIcon(), workspace.schemaGuide)
	lower := container.NewVBox(workspace.diagnostic, outlined(validate), outlined(guide), text("Preview renders through Slayer and Game.", 16))
	return card(container.NewBorder(upper, lower, nil, nil, tabs))
}

// schemaGuide reads explanatory prose from the same pinned validation schema.
func (workspace *Workspace) schemaGuide() {
	if workspace.document.Kind() != "job" {
		if workspace.document.Kind() == "legacy-set" {
			dialog.ShowInformation("Legacy scene", "This scene does not declare a schema. The editor checks YAML syntax and duplicate keys; Slayer checks scene support when you render.", workspace.window)
			return
		}
		name := "ecosystem.schema.json"
		if identity, _ := workspace.document.Value("$schema"); identity == "https://github.com/janearc/slayer/schemas/ecosystem-v2.json" {
			name = "ecosystem-v2.schema.json"
		}
		raw, err := assets.Read(name)
		if err != nil {
			workspace.problem(err)
			return
		}
		reference := widget.NewMultiLineEntry()
		reference.TextStyle = fyne.TextStyle{Monospace: true}
		reference.SetText(string(raw))
		reference.Disable()
		guide := dialog.NewCustom("Slayer scene schema", "Close", container.NewThemeOverride(reference, codeTheme{}), workspace.window)
		guide.Resize(fyne.NewSize(900, 620))
		guide.Show()
		return
	}
	prose := paragraph("")
	choice := widget.NewSelect([]string{"scene", "film", "film.preset", "film.wide", "renderer", "quality", "camera", "at", "fps", "duration", "sound"}, func(key string) {
		value, err := assets.Guidance(strings.Split(key, ".")...)
		if err != nil {
			workspace.problem(err)
			return
		}
		prose.SetText(value)
	})
	choice.SetSelected("scene")
	guide := dialog.NewCustom("Slayer job schema", "Close", container.NewBorder(choice, text("Pinned schema · structural validation only", 18), nil, nil, container.NewVScroll(prose)), workspace.window)
	guide.Resize(fyne.NewSize(780, 420))
	guide.Show()
}

type codeTheme struct{ atelierTheme }

// Size gives code its own scale without changing prose and dialog copy.
func (codeTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return max(float32(16), fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)*0.9)
	}
	return atelierTheme{}.Size(name)
}

// ShowArea reserves distinct camera, sequence and music workspaces for review.
func (workspace *Workspace) ShowArea(area string) {
	if area == "set" {
		workspace.view.Objects = []fyne.CanvasObject{workspace.setContent}
		workspace.markArea(area)
		workspace.view.Refresh()
		return
	}
	titles := map[string]string{"cameras": "Cameras & movement", "sequence": "Time, given a shape", "music": "Music & sound"}
	copy := map[string]string{"cameras": "Camera views and movement paths will live here.", "sequence": "Shots, duration and transitions will live here.", "music": "Tracks, cues and musical timing will live here."}
	title, ok := titles[area]
	if !ok {
		return
	}
	workspace.markArea(area)
	contents := container.NewVBox(text("Reserved workspace", 20), display(title, 48), paragraph(copy[area]),
		paragraph("This space is intentionally reserved in the prototype. Planning, playback and editing arrive after the interface review."),
		widget.NewButton("Return to the set", func() { workspace.ShowArea("set") }))
	workspace.view.Objects = []fyne.CanvasObject{card(pad(24, contents))}
	workspace.view.Refresh()
}

// markArea makes the active navigation button agree with the displayed area.
func (workspace *Workspace) markArea(area string) {
	for name, button := range workspace.navigationButtons {
		button.Importance = widget.MediumImportance
		if name == area {
			button.Importance = widget.HighImportance
		}
		button.Refresh()
	}
}

// ShowInspector exposes the same two tabs used by the native window.
func (workspace *Workspace) ShowInspector(pane string) {
	if pane == "settings" {
		workspace.inspectorTabs.SelectIndex(1)
	} else {
		workspace.inspectorTabs.SelectIndex(0)
	}
}
