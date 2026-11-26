package workspace

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
)

// settingsSheetWidth is the width of the settings sheet's panel, which holds
// its padding.
const settingsSheetWidth = 820

// settingsSheetPadding is the space between the panel's edge and its rows.
const settingsSheetPadding = 24

// settingsTitleExtra is how much larger than the text the title is, in points.
const settingsTitleExtra = 8

// requestSettings asks the next tick to open the settings sheet. It is safe
// from any thread, which the menu's action may need.
func (work *Workspace) requestSettings() {
	work.wantSettings.Store(true)
	if work.host != nil {
		work.host.RequestRedraw()
	}
}

// OpenSettings opens the settings sheet on the next tick, as the menu and the
// key do. A proof uses it to draw the sheet.
func (work *Workspace) OpenSettings() { work.requestSettings() }

// InstallMenu gives the window the application menu: Settings on command and
// comma, and the standard hide and quit items. Both Settings roads, this and
// the key, open the same sheet.
func (work *Workspace) InstallMenu() {
	work.host.SetMenu(gogpu.NewMenu().
		AddItem(gogpu.MenuItem{Title: "Open…", Action: work.requestOpen}).
		AddItem(gogpu.MenuItem{Title: "Settings…", Role: gogpu.RolePreferences, Action: work.requestSettings}).
		AddItem(gogpu.MenuItem{Separator: true}).
		AddItem(gogpu.MenuItem{Title: "Hide Ferruccio", Role: gogpu.RoleHide}).
		AddItem(gogpu.MenuItem{Title: "Hide Others", Role: gogpu.RoleHideOthers}).
		AddItem(gogpu.MenuItem{Title: "Show All", Role: gogpu.RoleShowAll}).
		AddItem(gogpu.MenuItem{Separator: true}).
		AddItem(gogpu.MenuItem{Title: "Quit Ferruccio", Role: gogpu.RoleQuit}))
}

// noteMissingCache, at launch on the Slayer road, says in one plain line that
// there is no cache folder and opens Settings. It does not refuse to open,
// and it does not wait for the first render to say so. A proof, which has no
// person to answer a sheet, is left as it was.
func (work *Workspace) noteMissingCache() {
	if work.options.Tools.Transport != "slayer" || work.options.SuppressAutomaticPreview {
		return
	}
	if preview.RequireCache(work.renderTools()) == nil {
		return
	}
	work.autoPreview = false
	work.status, work.caption = preview.ErrNoCache.Error(), preview.ErrNoCache.Error()
	work.requestSettings()
}

// refuseWithoutCache stops a render on the Slayer road that has no cache
// folder, with the one plain line, and opens Settings.
func (work *Workspace) refuseWithoutCache(tools preview.Tools) error {
	if work.options.Tools.Transport != "slayer" {
		return nil
	}
	if err := preview.RequireCache(tools); err != nil {
		work.requestSettings()
		return err
	}
	return nil
}

// SettingsOpen reports whether the settings sheet is the window now.
func (work *Workspace) SettingsOpen() bool { return work.settingsOpen }

// openSettingsIfAsked shows the sheet, built again from the settings as they
// are, when something asked for it since the last tick. It waits for the
// window. The sheet is the window's root, so showing it is a new layout.
func (work *Workspace) openSettingsIfAsked() {
	if !work.wantSettings.Load() || work.host.PrimaryWindow() == nil {
		return
	}
	work.wantSettings.Store(false)
	work.settingsOpen = true
	work.rebuild()
}

// closeSettings gives the window its own widgets back on the next tick. It
// is called from a button on the sheet, which cannot replace its own root.
func (work *Workspace) closeSettings() {
	if !work.settingsOpen {
		return
	}
	work.settingsOpen = false
	work.refreshLayout = true
	work.invalidate()
}

// settingsSheet is the sheet as the window's root.
func (work *Workspace) settingsSheet() widget.Widget {
	return newSheet(work.settingsPage(), work.closeSettings)
}

// settingsPage is the sheet's content: one panel in the middle of the window.
// Its title, the file it saves to and its Close button stay where they are; the
// row for each setting, with its value, where the value came from and how to
// change it, scrolls between them when they do not fit the window.
func (work *Workspace) settingsPage() *primitives.BoxWidget {
	resolved := work.resolved()
	title := work.text("Settings").FontSize(work.size + settingsTitleExtra).Color(appearance.Pink)
	header := []widget.Widget{title, work.paragraph("Saved in "+resolved.Path+". A change is used by the next render.", appearance.Ink)}
	if resolved.Problem != "" {
		header = append(header, work.paragraph("The settings file could not be read, so its values are not used: "+resolved.Problem, appearance.Pink))
	}
	if work.settings.Path == "" {
		header = append(header, work.paragraph("There is no settings file: no home folder was found, so nothing can be saved.", appearance.Pink))
	}
	if preview.RequireCache(work.renderTools()) != nil && work.options.Tools.Transport == "slayer" {
		header = append(header, work.paragraph(preview.ErrNoCache.Error(), appearance.Pink))
	}
	var rows []widget.Widget
	for _, entry := range resolved.Entries {
		rows = append(rows, work.settingRow(entry))
	}
	scrolling := scrollview.New(primitives.VBox(rows...).Gap(settingsRowsGap).PaddingRight(settingsScrollbarRoom))
	closer := primitives.HBox(primitives.Expanded(primitives.Box()), work.action("Close", true, func() bool { return false }, work.closeSettings))
	panel := primitives.VBox(primitives.VBox(header...).Gap(settingsHeaderGap), primitives.Expanded(scrolling), closer).Gap(settingsRowsGap).Padding(settingsSheetPadding).Width(settingsSheetWidth).Background(appearance.Panel)
	return primitives.HBox(primitives.Expanded(primitives.Box()), panel, primitives.Expanded(primitives.Box())).CrossAlign(primitives.CrossAxisStretch).Background(appearance.Ground)
}

// settingsRowsGap is the space between the rows of the sheet and between its
// header, rows and Close button.
const settingsRowsGap = 14

// settingsScrollbarRoom is the space kept at the right of the rows so the
// scrollbar, which is drawn over the content, does not cover a control.
const settingsScrollbarRoom = 16

// settingsHeaderGap is the space between the lines of the sheet's header.
const settingsHeaderGap = 8

// settingRow is one setting: its label, its value and its control, then a
// line that says where it came from or why it cannot be changed.
func (work *Workspace) settingRow(entry settings.Entry) widget.Widget {
	label := work.text(entry.Key.Label).Color(appearance.Cyan).MaxLines(1)
	value := work.text(shorten(work.settingValue(entry), work.valueRunes(entry))).MaxLines(1)
	head := primitives.HBox(primitives.VBox(label).Width(settingsLabelWidth), primitives.Expanded(value), work.settingControl(entry)).Gap(12)
	note := settingNote(entry)
	if entry.Source == settings.Unset && entry.Key.Name == settings.Queue && work.options.Tools.Queue != "" {
		note = "Found the existing shared queue. Choose another queue to use it instead."
	}
	if entry.Source == settings.Unset && entry.Key.Name == settings.Evidence && work.options.Evidence != "" {
		note = "Using the default output folder. Choose another folder to use it instead."
	}
	if entry.Source == settings.Unset && entry.Key.Name == settings.Cache && work.options.Tools.Cache != "" {
		note = "Using the default cache folder. Choose another folder to use it instead."
	}
	if entry.Source == settings.Unset && entry.Key.Name == settings.Shelf && work.options.Tools.Shelf != "" {
		note = "Found the existing source shelf. Choose another folder to use it instead."
	}
	return primitives.VBox(head, work.paragraph(note, appearance.Ink)).Gap(4)
}

// settingsLabelWidth is the width of the column of setting names.
const settingsLabelWidth = 190

// settingsGlyphWidth is the width of an average glyph as a share of the text
// size. Text that is not wrapped here is not wrapped by the toolkit either, so
// lines are cut to a count of runes from it. Measured: 99 characters drew 816
// points at size 16, 0.51; this leaves room.
const settingsGlyphWidth = 0.55

// settingsControlWidth is the room kept for a row's controls beside its value,
// in points: a Choose and a Clear button, or one dropdown.
const settingsControlWidth = 210

// runesIn is how many average glyphs fit in width points at the window's text size.
func (work *Workspace) runesIn(width float32) int {
	return max(1, int(width/(work.size*settingsGlyphWidth)))
}

// valueRunes is how many runes of a row's value fit beside its label and its control.
func (work *Workspace) valueRunes(entry settings.Entry) int {
	room := float32(settingsSheetWidth-2*settingsSheetPadding) - settingsLabelWidth - settingsControlWidth - 2*settingsRowGap
	if entry.Locked() {
		room += settingsControlWidth
	}
	return work.runesIn(room)
}

// settingsRowGap is the space between a row's label, value and control.
const settingsRowGap = 12

// paragraph is text wrapped to the sheet's width, one line to a text widget,
// since the toolkit's own text widget draws a long line whole.
func (work *Workspace) paragraph(text string, colour widget.Color) widget.Widget {
	lines := wrapMessageLines(text, work.runesIn(settingsSheetWidth-2*settingsSheetPadding))
	children := make([]widget.Widget, len(lines))
	for index, line := range lines {
		children[index] = work.text(line).Color(colour)
	}
	return primitives.VBox(children...).Gap(2)
}

// shorten is text cut to at most limit runes, keeping its end and marking the
// cut with an ellipsis at the front, since the end of a path is what names it.
func shorten(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit || limit < 2 {
		return text
	}
	return "…" + string(runes[len(runes)-limit+1:])
}

// settingValue is the value to show: what is set, or what unset means.
func (work *Workspace) settingValue(entry settings.Entry) string {
	switch {
	case entry.Value != "":
		return entry.Value
	case entry.Key.Name == settings.Queue && work.options.Tools.Queue != "":
		return work.options.Tools.Queue + " (detected)"
	case entry.Key.Name == settings.Evidence && work.options.Evidence != "":
		return work.options.Evidence + " (default)"
	case entry.Key.Name == settings.Cache && work.options.Tools.Cache != "":
		return work.options.Tools.Cache + " (default)"
	case entry.Key.Name == settings.Shelf && work.options.Tools.Shelf != "":
		return work.options.Tools.Shelf + " (detected)"
	case entry.Key.Default != "":
		return entry.Key.Default + " (default)"
	case entry.Key.Name == settings.Theme:
		return appearance.SelectedTheme() + " (in use)"
	}
	return "not set"
}

// settingNote says where a value came from and, when a flag or the
// environment holds it, why the sheet cannot change it.
func settingNote(entry settings.Entry) string {
	switch entry.Source {
	case settings.FromFlag:
		return fmt.Sprintf("Set by the %s flag, so it cannot be changed here. Start Ferruccio without it to change it.", entry.Origin)
	case settings.FromEnvironment:
		return fmt.Sprintf("Set by the %s variable, so it cannot be changed here. Start Ferruccio without it to change it.", entry.Origin)
	case settings.FromFile:
		return "From the settings file. " + entry.Key.Why
	}
	return "Not set. " + entry.Key.Why
}

// settingControl is the control that changes a setting, or nothing when a
// flag or the environment holds the value.
func (work *Workspace) settingControl(entry settings.Entry) widget.Widget {
	if entry.Locked() {
		return primitives.Box().Height(0)
	}
	if entry.Key.Kind == settings.Choice {
		return work.settingChoice(entry)
	}
	idle := func() bool { return false }
	choose := work.action("Choose…", false, idle, func() { work.choosePath(entry.Key) })
	choose.SetID("Choose " + strings.ToLower(entry.Key.Label))
	if entry.Source != settings.FromFile {
		return choose
	}
	forget := work.action("Clear", false, idle, func() { work.saveSetting(entry.Key.Name, "") })
	forget.SetID("Clear " + strings.ToLower(entry.Key.Label))
	return primitives.HBox(choose, forget).Gap(8)
}

// settingChoice is the dropdown for a setting that takes one of a list.
func (work *Workspace) settingChoice(entry settings.Entry) widget.Widget {
	items := entry.Key.Choices
	current := entry.Effective()
	if entry.Key.Name == settings.Theme {
		items, current = appearance.AvailableThemes(), appearance.SelectedTheme()
	}
	choice := dropdown.New(dropdown.Items(items...),
		dropdown.Selected(slices.Index(items, current)),
		dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}),
		dropdown.OnChange(func(_ int, value string) { work.saveSetting(entry.Key.Name, value) }))
	choice.SetID("Setting " + string(entry.Key.Name))
	return choice
}

// choosePath asks for a folder or an executable in the system's own chooser
// and saves it.
func (work *Workspace) choosePath(key settings.Key) {
	paths, err := work.host.ShowOpenFileDialog(gogpu.FileDialogOptions{Title: "Choose " + strings.ToLower(key.Label), Directory: key.Kind == settings.Folder})
	if err != nil {
		work.problem(err)
		return
	}
	if len(paths) > 0 {
		work.saveSetting(key.Name, paths[0])
	}
}

// saveSetting writes one setting to the file, applies it where it can be
// applied at once, and asks for the sheet to be shown again with it.
func (work *Workspace) saveSetting(name settings.Name, value string) {
	if work.settings.Path == "" {
		work.problem(errors.New("there is no settings file: no home folder was found, so nothing can be saved"))
		return
	}
	if err := settings.Save(work.settings.Path, name, value); err != nil {
		work.problem(err)
		return
	}
	if name == settings.Theme && value != "" {
		if err := appearance.SelectTheme(value); err != nil {
			work.problem(err)
		} else {
			work.working.dirty = true
			work.rebuild()
		}
	}
	work.status = "Settings saved"
	if name == settings.Cache || name == settings.Shelf || name == settings.Data || name == settings.Processors {
		work.working.source = ""
		work.working.rollback = nil
	}
	work.refreshRenderBlocked(work.state.document.Check() == nil)
	work.requestSettings()
	work.invalidate()
}

// requestOpen defers the file chooser to the UI thread.
func (work *Workspace) requestOpen() {
	work.wantOpen.Store(true)
	if work.host != nil {
		work.host.RequestRedraw()
	}
}

// requestSave defers saving, including a native chooser when needed.
func (work *Workspace) requestSave() {
	work.wantSave.Store(true)
	if work.host != nil {
		work.host.RequestRedraw()
	}
}
