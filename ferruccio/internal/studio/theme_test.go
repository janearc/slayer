package studio

import (
	"fmt"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/janearc/libtheme-css/dialects/nvim"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

func colorHex(value color.Color) string {
	red, green, blue, _ := value.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", red>>8, green>>8, blue>>8)
}

func TestVaporwaveSourceRolesStayRecognizable(t *testing.T) {
	picked := Theme()
	want := map[fyne.ThemeColorName]string{
		theme.ColorNameBackground:  groundHex,
		theme.ColorNameForeground:  inkHex,
		theme.ColorNameButton:      panelHex,
		theme.ColorNamePrimary:     primaryHex,
		theme.ColorNameHyperlink:   linkHex,
		theme.ColorNameFocus:       focusHex,
		theme.ColorNameInputBorder: borderHex,
	}
	for role, hex := range want {
		if got := colorHex(picked.Color(role, theme.VariantDark)); got != hex {
			t.Errorf("%s = %s, want source token %s", role, got, hex)
		}
	}
}

// Every prose-bearing surface uses ink whose contrast and hue both remain legible.
func TestProsePaletteStaysWithinMeasuredBand(t *testing.T) {
	picked := Theme()
	surfaces := []fyne.ThemeColorName{
		theme.ColorNameBackground, theme.ColorNameInputBackground,
		theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground,
		theme.ColorNameHeaderBackground, theme.ColorNameButton,
		theme.ColorNameHover, theme.ColorNameSelection, theme.ColorNameDisabledButton,
		theme.ColorNameFocus,
	}
	roles := []fyne.ThemeColorName{
		theme.ColorNameForeground, theme.ColorNameDisabled,
		theme.ColorNamePlaceHolder, theme.ColorNameHyperlink, theme.ColorNamePrimary,
	}
	for _, surface := range surfaces {
		for _, role := range roles {
			checkPair(t, picked.Color(role, theme.VariantDark), picked.Color(surface, theme.VariantDark), string(role)+" on "+string(surface))
		}
	}
	checkPair(t, picked.Color(theme.ColorNameForegroundOnPrimary, theme.VariantDark), picked.Color(theme.ColorNamePrimary, theme.VariantDark), "primary action label")
}

func checkPair(t *testing.T, foreground, background color.Color, label string) {
	t.Helper()
	ink, ground := srgb.MustHex(colorHex(foreground)), srgb.MustHex(colorHex(background))
	ratio := nvim.Contrast(ink.Swatch(), ground.Swatch())
	if ratio < 10 || ratio > 12.5 {
		t.Errorf("%s: contrast %.4f outside 10–12.5", label, ratio)
	}
	front, back := ink.HSL(), ground.HSL()
	gap := math.Abs(front.H - back.H)
	gap = math.Min(gap, 360-gap)
	if back.S >= 0.35 && gap < 40 {
		t.Errorf("%s: hue gap %.2f and background saturation %.3f", label, gap, back.S)
	}
	t.Logf("%s: %s/%s ratio %.4f, hue gap %.2f, background saturation %.3f", label, ink.Hex(), ground.Hex(), ratio, gap, back.S)
}

func TestThemeTextSizeVariantsAndMinimumCaption(t *testing.T) {
	for _, base := range []float32{16, 18, 20, 24} {
		picked := ThemeWithSize(base)
		if got := picked.Size(theme.SizeNameText); got != base {
			t.Errorf("size %g body = %g", base, got)
		}
		if got := picked.Size(theme.SizeNameCaptionText); got < 16 {
			t.Errorf("size %g caption = %g, below minimum 16", base, got)
		}
		if got := picked.Size(theme.SizeNameSubHeadingText); got != base+4 {
			t.Errorf("size %g subheading = %g, want %g", base, got, base+4)
		}
		if got := picked.Size(theme.SizeNameHeadingText); got != base+8 {
			t.Errorf("size %g heading = %g, want %g", base, got, base+8)
		}
	}
	if got := Theme().Size(theme.SizeNameText); got != 18 {
		t.Errorf("default body size = %g, want 18", got)
	}
	application := test.NewApp()
	defer application.Quit()
	for _, base := range []float32{16, 24} {
		application.Settings().SetTheme(ThemeWithSize(base))
		want := max(float32(16), base*0.9)
		if got := (codeTheme{}).Size(theme.SizeNameText); got != want {
			t.Errorf("code theme body size at %g = %g, want %g", base, got, want)
		}
	}
	if got := ThemeWithSize(42).Size(theme.SizeNameText); got != 18 {
		t.Errorf("unsupported body size = %g, want default 18", got)
	}
}
