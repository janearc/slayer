package studio

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// outlined gives actions a visible boundary, including at their resting state.
func outlined(button *widget.Button) fyne.CanvasObject {
	edge := canvas.NewRectangle(color.Transparent)
	edge.StrokeColor = Theme().Color(theme.ColorNameHyperlink, theme.VariantDark)
	edge.StrokeWidth = 1.2
	edge.CornerRadius = 8
	return container.NewStack(container.NewThemeOverride(button, actionTheme{button: button}), edge)
}

type actionTheme struct{ button *widget.Button }

// Color keeps a primary action's dark label on a light fill during focus too.
func (selected actionTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	base := fyne.CurrentApp().Settings().Theme()
	if selected.button.Importance == widget.HighImportance && (name == theme.ColorNameFocus || name == theme.ColorNameHover) {
		return base.Color(theme.ColorNamePrimary, variant)
	}
	return base.Color(name, variant)
}

// Font follows the configured application typeface.
func (actionTheme) Font(style fyne.TextStyle) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Font(style)
}

// Icon uses the current application icon vocabulary.
func (actionTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(name)
}

// Size forwards the adjustable widget scale.
func (actionTheme) Size(name fyne.ThemeSizeName) float32 {
	return fyne.CurrentApp().Settings().Theme().Size(name)
}

// SetTextSize changes both theme-sized widgets and explicit canvas type.
func (workspace *Workspace) SetTextSize(size float32) {
	if size == workspace.textSize || workspace.content == nil {
		return
	}
	selected := ThemeWithSize(size)
	actual := selected.Size(theme.SizeNameText)
	if workspace.canvasSizes == nil {
		workspace.canvasSizes = map[*canvas.Text]float32{}
	}
	visited := map[fyne.CanvasObject]bool{}
	var resize func(fyne.CanvasObject)
	resize = func(object fyne.CanvasObject) {
		if object == nil || visited[object] {
			return
		}
		visited[object] = true
		if label, ok := object.(*canvas.Text); ok {
			ratio, known := workspace.canvasSizes[label]
			if !known {
				ratio = label.TextSize / workspace.textSize
				workspace.canvasSizes[label] = ratio
			}
			label.TextSize = max(float32(16), ratio*actual)
			label.Refresh()
		}
		if group, ok := object.(*fyne.Container); ok {
			for _, child := range group.Objects {
				resize(child)
			}
		}
	}
	resize(workspace.content)
	resize(workspace.setContent)
	workspace.textSize = actual
	fyne.CurrentApp().Settings().SetTheme(selected)
	workspace.content.Refresh()
}
