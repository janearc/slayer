// Package appearance loads the checked Chroma colourways used by Atelier.
package appearance

import (
	"embed"
	"fmt"
	"io/fs"
	"math"
	"path"
	"sort"
	"strings"

	uitheme "github.com/gogpu/ui/theme"
	"github.com/gogpu/ui/theme/devtools"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/chroma/libtheme/colourway"
	"github.com/janearc/chroma/libtheme/css"
	"github.com/janearc/chroma/libtheme/primitives/swatch"
	"github.com/janearc/chroma/libtheme/spaces/srgb"
)

//go:embed themes/*.css
var colourwayFiles embed.FS

const (
	defaultTheme         = "vaporwave"
	defaultSize  float32 = 18
	minimumSize  float32 = 16
)

type palette struct {
	ground, panel, raised, ink, dim, heading, link, cursorInk      widget.Color
	magenta, brightMagenta, cyan, brightCyan, border, borderStrong widget.Color
	error, warning, success, info, code, stringColor, hint         widget.Color
	inkSource, headingSource, linkSource, dimSource, codeSource    string
	primaryFillSource                                              string
	primaryFill, primaryHoverFill, primaryPressedFill, primaryInk  widget.Color
}

type colourwayResult struct {
	palette palette
	err     error
}

var (
	colourways   = loadColourways()
	selectedName string
	selected     palette

	// Exported colors are the selected Chroma role tokens used by widgets.
	Ground widget.Color
	Panel  widget.Color
	Raised widget.Color
	Ink    widget.Color
	Pink   widget.Color
	Cyan   widget.Color
	Border widget.Color

	// Decorative colors should only be used behind non-text elements.
	DecorativePanel  widget.Color
	DecorativeRaised widget.Color

	primaryHover   widget.Color
	primaryPressed widget.Color
	secondaryHover widget.Color
	secondaryPress widget.Color
)

// init activates the checked default source colourway after loading embedded themes.
func init() {
	if err := SelectTheme(defaultTheme); err != nil {
		panic(err)
	}
}

// loadColourways parses each embedded source through Chroma's CSS reader.
func loadColourways() map[string]colourwayResult {
	files, err := fs.Glob(colourwayFiles, "themes/*.css")
	if err != nil {
		panic(fmt.Errorf("list Chroma colourways: %w", err))
	}
	sort.Strings(files)
	loaded := make(map[string]colourwayResult, len(files))
	for _, name := range files {
		raw, err := colourwayFiles.ReadFile(name)
		if err != nil {
			panic(fmt.Errorf("read Chroma colourway %q: %w", name, err))
		}
		parsed, parseErr := parseColourway(raw)
		loaded[strings.TrimSuffix(path.Base(name), path.Ext(name))] = colourwayResult{
			palette: parsed,
			err:     parseErr,
		}
	}
	return loaded
}

// parseColourway maps Chroma's measured roles without inventing colour values.
func parseColourway(source []byte) (palette, error) {
	resolved := colourway.Resolve(css.Read(string(source)).Roles)
	read := func(name string) (widget.Color, error) {
		value, ok := resolved.Of(name)
		if !ok {
			return widget.Color{}, fmt.Errorf("Chroma cannot resolve role --%s", name)
		}
		rgb, inGamut := srgb.FromSwatch(value)
		if !inGamut {
			return widget.Color{}, fmt.Errorf("Chroma role --%s is outside sRGB", name)
		}
		return widget.RGBA(float32(rgb.R), float32(rgb.G), float32(rgb.B), 1), nil
	}
	readAll := func(names ...string) ([]widget.Color, error) {
		values := make([]widget.Color, len(names))
		for index, name := range names {
			value, err := read(name)
			if err != nil {
				return nil, err
			}
			values[index] = value
		}
		return values, nil
	}
	values, err := readAll(
		"ground", "panel", "surface-1", "ink", "dim", "heading", "link", "cursor-ink",
		"magenta", "bright-magenta", "cyan", "bright-cyan", "border", "bright-black",
		"err", "warn", "hint", "info", "code", "string",
	)
	if err != nil {
		return palette{}, err
	}
	ground := values[0]
	ink, inkSource, err := readableRole("ink", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma ink: %w", err)
	}
	dim, dimSource, err := readableRole("dim", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma dim: %w", err)
	}
	heading, headingSource, err := readableRole("heading", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma heading: %w", err)
	}
	link, linkSource, err := readableRole("link", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma link: %w", err)
	}
	code, codeSource, err := readableRole("code", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma code: %w", err)
	}
	primaryFill, primaryInk, primaryFillSource, err := readableButtonPair("heading", "cursor-ink", resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma primary button pair: %w", err)
	}
	primaryHoverFill, err := readableButtonFill("bright-magenta", primaryInk, primaryFill, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma hover fill: %w", err)
	}
	primaryPressedFill, err := readableButtonFill("magenta", primaryInk, primaryFill, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma pressed fill: %w", err)
	}
	errorColor, _, err := readableRole("err", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma error color: %w", err)
	}
	warning, _, err := readableRole("warn", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma warning color: %w", err)
	}
	success, _, err := readableRole("hint", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma success color: %w", err)
	}
	info, _, err := readableRole("info", []widget.Color{ground}, resolved)
	if err != nil {
		return palette{}, fmt.Errorf("select readable Chroma info color: %w", err)
	}
	return palette{
		ground: values[0], panel: values[1], raised: values[2], ink: ink,
		dim: dim, heading: heading, link: link, cursorInk: values[7], primaryInk: primaryInk,
		magenta: values[8], brightMagenta: values[9], cyan: values[10], brightCyan: values[11],
		border: values[12], borderStrong: values[13], error: errorColor, warning: warning,
		success: success, info: info, code: code, stringColor: values[19], hint: success,
		inkSource: inkSource, headingSource: headingSource, linkSource: linkSource,
		dimSource: dimSource, codeSource: codeSource,
		primaryFillSource: primaryFillSource, primaryFill: primaryFill,
		primaryHoverFill: primaryHoverFill, primaryPressedFill: primaryPressedFill,
	}, nil
}

// readableRole selects a Chroma role that meets Atelier's fixed ink gate.
func readableRole(preferred string, backgrounds []widget.Color, resolved colourway.Resolved) (widget.Color, string, error) {
	priority := []string{
		preferred, "selection-ink", "ink", "blue", "code", "cyan", "green", "yellow", "red",
		"magenta", "white", "black", "bright-blue", "bright-cyan", "bright-green",
		"bright-yellow", "bright-red", "bright-magenta", "bright-white", "bright-black",
		"cursor-ink", "ground",
	}
	names := append([]string(nil), priority...)
	names = append(names, resolved.Roles.Names()...)
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		value, ok := resolved.Of(name)
		if !ok {
			continue
		}
		color := colorFromSwatch(value)
		readable := true
		for _, background := range backgrounds {
			if !meetsTextGate(color, background) {
				readable = false
				break
			}
		}
		if !readable {
			continue
		}
		if from := resolved.From[name]; from != "" {
			return color, fmt.Sprintf("--%s (derived from %s)", name, from), nil
		}
		return color, "--" + name, nil
	}
	return widget.Color{}, "", fmt.Errorf("no Chroma role meets the 10–12.5 contrast and hue gates")
}

// readableButtonPair selects two existing Chroma swatches for a filled action.
func readableButtonPair(fillRole, labelRole string, resolved colourway.Resolved) (widget.Color, widget.Color, string, error) {
	groundSwatch, hasGround := resolved.Of("ground")
	ground := widget.Color{}
	if hasGround {
		ground = colorFromSwatch(groundSwatch)
	}
	fills := []string{fillRole, "heading", "bright-magenta", "magenta", "link", "blue", "green", "red", "yellow", "cyan", "ground"}
	labels := []string{labelRole, "cursor-ink", "ground", "ink", "selection-ink", "black", "bright-black", "white", "bright-white"}
	fills = append(fills, resolved.Roles.Names()...)
	labels = append(labels, resolved.Roles.Names()...)
	for _, fillName := range unique(fills) {
		fillSwatch, ok := resolved.Of(fillName)
		if !ok {
			continue
		}
		fill := colorFromSwatch(fillSwatch)
		for _, labelName := range unique(labels) {
			labelSwatch, ok := resolved.Of(labelName)
			if !ok {
				continue
			}
			label := colorFromSwatch(labelSwatch)
			if !meetsTextGate(label, fill) {
				continue
			}
			if fillName == "ground" || (hasGround && fill == ground) {
				continue
			}
			return fill, label, sourceRole(fillName, resolved), nil
		}
	}
	return widget.Color{}, widget.Color{}, "", fmt.Errorf("no Chroma fill and label meet the 10–12.5 contrast and hue gates")
}

// readableButtonFill uses a preferred source role or the chosen primary fill.
func readableButtonFill(preferred string, foreground, fallback widget.Color, resolved colourway.Resolved) (widget.Color, error) {
	if value, ok := resolved.Of(preferred); ok {
		candidate := colorFromSwatch(value)
		if meetsTextGate(foreground, candidate) {
			return candidate, nil
		}
	}
	if meetsTextGate(foreground, fallback) {
		return fallback, nil
	}
	for _, name := range resolved.Roles.Names() {
		value, _ := resolved.Of(name)
		candidate := colorFromSwatch(value)
		if name != "ground" && meetsTextGate(foreground, candidate) {
			return candidate, nil
		}
	}
	return widget.Color{}, fmt.Errorf("no Chroma fill meets the 10–12.5 contrast and hue gates")
}

// unique preserves Chroma role order while discarding repeated candidate names.
func unique(names []string) []string {
	seen := make(map[string]bool, len(names))
	result := make([]string, 0, len(names))
	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}

// sourceRole retains Chroma's direct or resolved role provenance.
func sourceRole(name string, resolved colourway.Resolved) string {
	if from := resolved.From[name]; from != "" {
		return fmt.Sprintf("--%s (derived from %s)", name, from)
	}
	return "--" + name
}

// colorFromSwatch converts a Chroma swatch without changing its screen color.
func colorFromSwatch(value swatch.Swatch) widget.Color {
	rgb, _ := srgb.FromSwatch(value)
	return widget.RGBA(float32(rgb.R), float32(rgb.G), float32(rgb.B), 1)
}

// meetsTextGate applies Atelier's fixed contrast and hue thresholds.
func meetsTextGate(foreground, background widget.Color) bool {
	front, back := themeLuminance(foreground), themeLuminance(background)
	ratio := (max(front, back) + 0.05) / (min(front, back) + 0.05)
	if ratio < 10 || ratio > 12.5 {
		return false
	}
	frontHue, _ := themeHueSaturation(foreground)
	backHue, saturation := themeHueSaturation(background)
	if saturation < 0.35 {
		return true
	}
	gap := math.Abs(frontHue - backHue)
	return min(gap, 360-gap) >= 40
}

// themeHueSaturation returns the sRGB hue and HSL saturation for the hue gate.
func themeHueSaturation(color widget.Color) (float64, float64) {
	r, g, b := float64(color.R), float64(color.G), float64(color.B)
	maxColor, minColor := max(r, g, b), min(r, g, b)
	delta := maxColor - minColor
	hue, saturation := 0.0, 0.0
	lightness := (maxColor + minColor) / 2
	if delta == 0 {
		return hue, saturation
	}
	saturation = delta / (1 - math.Abs(2*lightness-1))
	switch maxColor {
	case r:
		hue = math.Mod((g-b)/delta, 6) * 60
	case g:
		hue = ((b-r)/delta + 2) * 60
	case b:
		hue = ((r-g)/delta + 4) * 60
	}
	if hue < 0 {
		hue += 360
	}
	return hue, saturation
}

// AvailableThemes returns the names of embedded Chroma colourways.
func AvailableThemes() []string {
	names := make([]string, 0, len(colourways))
	for name, result := range colourways {
		if result.err == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// UnavailableThemes reports source themes that fail Atelier's unchanged readability gate.
func UnavailableThemes() map[string]string {
	result := make(map[string]string)
	for name, colourway := range colourways {
		if colourway.err != nil {
			result[name] = colourway.err.Error()
		}
	}
	return result
}

// SelectedTheme returns the active Chroma colourway name.
func SelectedTheme() string { return selectedName }

// InkSource reports the Chroma role currently used for readable body text.
func InkSource() string { return selected.inkSource }

// InkAdjusted reports whether the active Chroma ink required a readable alias.
func InkAdjusted() bool { return selected.inkSource != "--ink" }

// SelectTheme activates an embedded Chroma colourway without partial changes.
func SelectTheme(name string) error {
	result, ok := colourways[name]
	if !ok {
		return fmt.Errorf("unknown Chroma colourway %q", name)
	}
	if result.err != nil {
		return fmt.Errorf("Chroma colourway %q cannot meet Atelier's readable text contract: %w", name, result.err)
	}
	value := result.palette
	selectedName, selected = name, value
	Ground, Panel, Raised = value.ground, value.ground, value.ground
	DecorativePanel, DecorativeRaised = value.panel, value.raised
	Ink, Pink, Cyan, Border = value.ink, value.heading, value.link, value.border
	primaryHover, primaryPressed = value.primaryHoverFill, value.primaryPressedFill
	secondaryHover, secondaryPress = value.brightCyan, value.cyan
	return nil
}

// Theme returns the selected Chroma colors in the GoGPU DevTools roles.
func Theme() *devtools.Theme {
	value := selected
	var t *devtools.Theme
	if themeLuminance(value.ground) > themeLuminance(value.ink) {
		t = devtools.NewTheme()
	} else {
		t = devtools.NewDarkTheme()
	}
	t.Colors = devtools.ColorScheme{
		Background:          value.ground,
		Surface:             value.ground,
		SurfaceElevated:     value.ground,
		OnSurface:           value.ink,
		OnSurfaceSecondary:  value.link,
		OnSurfaceDisabled:   value.dim,
		Primary:             value.primaryFill,
		PrimaryHover:        value.primaryHoverFill,
		PrimaryPress:        value.primaryPressedFill,
		OnPrimary:           value.primaryInk,
		Selection:           value.ground,
		Border:              value.border,
		BorderStrong:        value.borderStrong,
		BorderFocus:         value.link,
		InputBackground:     value.ground,
		ControlFill:         value.ground,
		ControlHover:        value.ground,
		HeaderBackground:    value.ground,
		HeaderForeground:    value.ink,
		Error:               value.error,
		ErrorBackground:     value.ground,
		Warning:             value.warning,
		Success:             value.success,
		Info:                value.info,
		Backdrop:            withOpacity(value.ground, 0.78),
		Shadow:              withOpacity(value.ground, 0.45),
		ScrollbarThumb:      value.border,
		ScrollbarThumbHover: value.link,
		ScrollbarTrack:      widget.ColorTransparent,
	}
	return t
}

// ThemeForSize returns the selected Chroma theme with the requested type size.
// It supports 16, 18, 20 and 24 point body text; other values use 18.
func ThemeForSize(size float32) *uitheme.Theme {
	size = normalizedSize(size)
	t := Theme().AsTheme()
	t.Name = "Atelier " + selectedName
	t.Typography = typography(size)
	return t
}

// typography builds a readable hierarchy at the selected body size.
func typography(base float32) uitheme.Typography {
	style := func(size, line float32, weight uitheme.FontWeight) uitheme.TextStyle {
		return uitheme.TextStyle{Font: "System", Size: size, Weight: weight, LineHeight: line}
	}
	regular, medium := uitheme.FontWeightNormal, uitheme.FontWeightMedium
	caption := max(minimumSize, base-2)
	return uitheme.Typography{
		FontFamily:     "System",
		DisplayLarge:   style(base*2, base*2.4, regular),
		DisplayMedium:  style(base*1.75, base*2.15, regular),
		DisplaySmall:   style(base*1.5, base*1.9, regular),
		HeadlineLarge:  style(base*1.4, base*1.75, regular),
		HeadlineMedium: style(base*1.3, base*1.65, regular),
		HeadlineSmall:  style(base*1.2, base*1.55, regular),
		TitleLarge:     style(base*1.1, base*1.45, regular),
		TitleMedium:    style(base, base*1.4, regular),
		TitleSmall:     style(caption, caption*1.35, regular),
		BodyLarge:      style(base, base*1.6, regular),
		BodyMedium:     style(base, base*1.55, regular),
		BodySmall:      style(caption, caption*1.5, regular),
		LabelLarge:     style(base, base*1.35, medium),
		LabelMedium:    style(caption, caption*1.3, medium),
		LabelSmall:     style(caption, caption*1.25, medium),
	}
}

// normalizedSize accepts supported settings and falls back to the default.
func normalizedSize(size float32) float32 {
	switch size {
	case 16, 18, 20, 24:
		return size
	default:
		return defaultSize
	}
}

// withOpacity keeps a Chroma color's RGB while applying a UI-layer alpha.
func withOpacity(color widget.Color, alpha float32) widget.Color {
	color.A = alpha
	return color
}

// themeLuminance estimates relative sRGB luminance to classify light themes.
func themeLuminance(color widget.Color) float64 {
	linear := func(channel float32) float64 {
		value := float64(channel)
		if value <= 0.04045 {
			return value / 12.92
		}
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(color.R) + 0.7152*linear(color.G) + 0.0722*linear(color.B)
}
