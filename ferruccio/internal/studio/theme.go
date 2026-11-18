package studio

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// These source tokens come from Chroma's Vaporwave CSS. Its prose-bearing
// layers are derived, desaturated violet shades so muted lavender prose keeps
// the owner's hue-separation rule as well as the contrast band.
const (
	groundHex       = "#14111a" // desaturated violet derived from --ground
	inkHex          = "#cdc3d8" // --ink
	panelHex        = "#1b1623" // desaturated violet derived from --cursor-line
	raisedHex       = "#1f192a" // desaturated violet derived from --surface-1
	primaryHex      = "#ffb5ff" // --bright-magenta
	linkHex         = "#00dff9" // --link
	borderHex       = "#432c6b" // --border
	focusHex        = panelHex
	separatorHex    = "#432c6b" // --border
	shadowHex       = "#160d2b" // --ground
	defaultTextSize = float32(18)
	minimumTextSize = float32(16)
)

type atelierTheme struct{ textSize float32 }

// Theme supplies the Vaporwave presentation at the compact default size.
func Theme() fyne.Theme { return ThemeWithSize(defaultTextSize) }

// ThemeWithSize sets the base text size. Supported UI settings are 16, 18, 20,
// and 24; unknown or non-positive values use the default size of 18.
func ThemeWithSize(size float32) fyne.Theme {
	switch size {
	case 16, 18, 20, 24:
		return atelierTheme{textSize: size}
	default:
		return atelierTheme{textSize: defaultTextSize}
	}
}

// Color supplies dark, quiet layers and saturated source accents.
func (atelierTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	colors := map[fyne.ThemeColorName]string{
		theme.ColorNameBackground:          groundHex,
		theme.ColorNameInputBackground:     groundHex,
		theme.ColorNameMenuBackground:      panelHex,
		theme.ColorNameOverlayBackground:   panelHex,
		theme.ColorNameHeaderBackground:    panelHex,
		theme.ColorNameButton:              panelHex,
		theme.ColorNameDisabledButton:      panelHex,
		theme.ColorNameHover:               raisedHex,
		theme.ColorNameSelection:           raisedHex,
		theme.ColorNamePrimary:             primaryHex,
		theme.ColorNameForegroundOnPrimary: groundHex,
		theme.ColorNameFocus:               focusHex,
		theme.ColorNameInputBorder:         borderHex,
		theme.ColorNameSeparator:           separatorHex,
		theme.ColorNameScrollBar:           borderHex,
		theme.ColorNameScrollBarBackground: groundHex,
		theme.ColorNameShadow:              shadowHex,
		theme.ColorNameHyperlink:           linkHex,
	}
	if hex, ok := colors[name]; ok {
		return pigment(hex)
	}
	return pigment(inkHex)
}

// Font keeps the owner's regular-weight preference across Fyne controls.
func (atelierTheme) Font(style fyne.TextStyle) fyne.Resource {
	style.Bold = false
	return theme.DefaultTheme().Font(style)
}

// Icon uses Fyne's familiar controls rather than a parallel icon vocabulary.
func (atelierTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size scales the type hierarchy while keeping every text role at least 16.
func (selected atelierTheme) Size(name fyne.ThemeSizeName) float32 {
	base := selected.textSize
	if base < minimumTextSize {
		base = defaultTextSize
	}
	values := map[fyne.ThemeSizeName]float32{
		theme.SizeNameText:            base,
		theme.SizeNameLineSpacing:     base * 0.55,
		theme.SizeNameCaptionText:     max(minimumTextSize, base-2),
		theme.SizeNameSubHeadingText:  base + 4,
		theme.SizeNameHeadingText:     base + 8,
		theme.SizeNamePadding:         max(8, base*0.45),
		theme.SizeNameInnerPadding:    max(8, base*0.45),
		theme.SizeNameInputRadius:     8,
		theme.SizeNameSelectionRadius: 8,
		theme.SizeNameScrollBar:       8,
	}
	if size, ok := values[name]; ok {
		return size
	}
	return theme.DefaultTheme().Size(name)
}

// pigment uses libtheme's screen colour conversion rather than a second model.
func pigment(hex string) color.Color {
	lamps := srgb.MustHex(hex)
	return color.NRGBA{R: uint8(math.Round(lamps.R * 255)), G: uint8(math.Round(lamps.G * 255)), B: uint8(math.Round(lamps.B * 255)), A: 255}
}
