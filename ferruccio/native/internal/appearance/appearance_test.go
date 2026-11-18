package appearance

import (
	"crypto/sha256"
	hexutil "encoding/hex"
	"fmt"
	"image"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/dialog"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/theme"
	"github.com/gogpu/ui/theme/devtools"
	"github.com/gogpu/ui/widget"
)

var (
	_ button.Painter       = ButtonPainter{}
	_ button.LayoutMetrics = ButtonPainter{}
	_ dropdown.Painter     = DropdownPainter{}
	_ dialog.Painter       = DialogPainter{}
	_ dialog.LayoutMetrics = DialogPainter{}
	_ widget.ThemeProvider = (*devtools.Theme)(nil)
)

func TestThemeUsesVaporwaveRolesAndReadablePairs(t *testing.T) {
	previous := SelectedTheme()
	t.Cleanup(func() { _ = SelectTheme(previous) })
	if err := SelectTheme("vaporwave"); err != nil {
		t.Fatal(err)
	}
	got := Theme()
	if got == nil || !got.IsDark() {
		t.Fatal("Vaporwave must return a dark DevTools theme")
	}
	for role, pair := range map[string]struct {
		got  widget.Color
		want string
	}{
		"ground": {Ground, "#160d2b"}, "panel": {DecorativePanel, "#2a1a4d"},
		"raised": {DecorativeRaised, "#3a2260"}, "readable ink": {Ink, "#94c3ff"},
		"pink": {Pink, "#ffa1ff"}, "cyan": {Cyan, "#00dff9"}, "border": {Border, "#432c6b"},
	} {
		if hex(pair.got) != pair.want {
			t.Errorf("%s = %s, want %s", role, hex(pair.got), pair.want)
		}
	}
	cs := got.Colors
	if cs.Background != Ground || cs.Surface != Ground || cs.SurfaceElevated != Ground ||
		cs.OnSurface != Ink || cs.OnSurfaceSecondary != Cyan || cs.OnSurfaceDisabled != selected.dim ||
		cs.Primary != selected.primaryFill || cs.PrimaryHover != selected.primaryHoverFill ||
		cs.PrimaryPress != selected.primaryPressedFill || cs.OnPrimary != selected.primaryInk ||
		cs.BorderFocus != Cyan || cs.BorderStrong != selected.borderStrong ||
		cs.InputBackground != Ground || cs.ControlFill != Ground || cs.ControlHover != Ground {
		t.Fatal("DevTools theme roles do not match the Vaporwave widget palette")
	}
	if InkSource() != "--blue" || !InkAdjusted() {
		t.Errorf("readable Vaporwave ink source = %q adjusted=%t, want --blue adjusted", InkSource(), InkAdjusted())
	}
}

func TestEmbeddedColourwayFilesMatchChromaSourceBytes(t *testing.T) {
	expected := map[string]string{
		"aqua":            "72e85072677b00df6b8e195bad630864d74e87b1ec793cad7b3c9cdbeea457b1",
		"fermi-long":      "7ac313ed7269f07227a8bd082f4c7d2c319ff2420bc69e5deb17130b0b613a3b",
		"fermi-midnight":  "7bfff458e0f19a12ee78b87511d168185ab33996285b213c11fbc7b4f004833d",
		"neptune-voyager": "b240b8a88d28cbf534e3d9bdc8aee7b22c2d9511f8ca7bdc9213d51b3f1a0a34",
		"pastel-invert":   "b45eee51021f41e837e113f4cbdfec45df0e47214b3fd8c13052046f1c10b2a5",
		"pastel-rose":     "cc800485e73663c912d9b4bb5244a76f750f5eb65c1489dc69f7eee95966059c",
		"pastel":          "4f6f08bbd74897b391c4f94fd4dbc980499ab01385d1ab0dd1615fcb3a0db29b",
		"pluto-coffee":    "35e03f9be7057a2ea995af7657bcfe24bcd30306842860cf56f8737833789bfc",
		"pluto-night":     "e3feee7f7014b2cfdfccb3f2b755fedc48f17cc6a1d97637704c5e33829d7617",
		"pluto":           "583a6d8c68d5cdab01c99659af2405a9b15a50e5d1c177272ce33fba9424bbf2",
		"twilight-burnt":  "311f2d0734196db4a541be973aeaecd619c2ea7f2913ee341b4b4b6145a89380",
		"twilight-deep":   "c259d1c598c844023719ab4f96b33de20b6721b77b97198edeff499d9887df7b",
		"twilight-low":    "2be32bcd30d657404dc9cc345517d50fa201eed033c9903da24035d8975c201b",
		"twilight":        "c5aac56355cfab0f425131ed3a7d0c753a162b930c2f4e8dbe271b00b0cdadc6",
		"vaporwave-dusk":  "d984b4094a5fea053c3626348a082ce38649de429afdcda55b1d32ea9db889b3",
		"vaporwave":       "1dbf52c864aca50a9ecdd91df00765d26fb6ca378511fc8d904a27bcb8c76d86",
	}
	for name, want := range expected {
		raw, err := colourwayFiles.ReadFile("themes/" + name + ".css")
		if err != nil {
			t.Fatalf("read embedded %s: %v", name, err)
		}
		sum := sha256.Sum256(raw)
		if got := hexutil.EncodeToString(sum[:]); got != want {
			t.Errorf("embedded Chroma source %s SHA-256 = %s, want %s", name, got, want)
		}
	}
}

func TestAllChromaThemesKeepTextOnGroundWithinContrastBand(t *testing.T) {
	previous := SelectedTheme()
	t.Cleanup(func() { _ = SelectTheme(previous) })
	sourceNames := []string{
		"aqua", "fermi-long", "fermi-midnight", "neptune-voyager", "pastel", "pastel-invert",
		"pastel-rose", "pluto", "pluto-coffee", "pluto-night", "twilight", "twilight-burnt",
		"twilight-deep", "twilight-low", "vaporwave", "vaporwave-dusk",
	}
	available := AvailableThemes()
	unavailable := UnavailableThemes()
	wantAvailable := []string{"aqua", "fermi-long", "fermi-midnight", "neptune-voyager", "pluto-coffee", "pluto-night", "vaporwave"}
	if !reflect.DeepEqual(available, wantAvailable) {
		t.Fatalf("AvailableThemes() = %v, want %v", available, wantAvailable)
	}
	wantUnavailable := []string{"pastel", "pastel-invert", "pastel-rose", "pluto", "twilight", "twilight-burnt", "twilight-deep", "twilight-low", "vaporwave-dusk"}
	unavailableNames := make([]string, 0, len(unavailable))
	covered := append([]string(nil), available...)
	for name, reason := range unavailable {
		t.Logf("Chroma theme %s unavailable: %s", name, reason)
		unavailableNames = append(unavailableNames, name)
		covered = append(covered, name)
	}
	sort.Strings(covered)
	sort.Strings(unavailableNames)
	if !reflect.DeepEqual(unavailableNames, wantUnavailable) {
		t.Fatalf("unavailable Chroma themes = %v, want %v", unavailableNames, wantUnavailable)
	}
	if !reflect.DeepEqual(covered, sourceNames) {
		t.Fatalf("available and unavailable Chroma sources = %v, want %v", covered, sourceNames)
	}
	for _, name := range available {
		if err := SelectTheme(name); err != nil {
			t.Fatalf("SelectTheme(%q): %v", name, err)
		}
		for role, color := range map[string]widget.Color{
			"ink": Ink, "heading": Pink, "link": Cyan, "dim": selected.dim, "code": selected.code,
		} {
			assertPair(t, name+" "+role, color, Ground)
		}
		assertPair(t, name+" primary label", selected.primaryInk, selected.primaryFill)
		assertPair(t, name+" primary hover label", selected.primaryInk, selected.primaryHoverFill)
		assertPair(t, name+" primary pressed label", selected.primaryInk, selected.primaryPressedFill)
		isDark := linearLuminance(Ground) < linearLuminance(Ink)
		if Theme().IsDark() != isDark {
			t.Errorf("%s IsDark() = %t, want %t", name, Theme().IsDark(), isDark)
		}
	}
}

func TestUnavailableThemeRefusalKeepsActiveSelection(t *testing.T) {
	previous := SelectedTheme()
	t.Cleanup(func() { _ = SelectTheme(previous) })
	for name, reason := range UnavailableThemes() {
		beforeName, beforeGround, beforeInk := SelectedTheme(), Ground, Ink
		if err := SelectTheme(name); err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("SelectTheme(%q) error = %v; want reason %q", name, err, reason)
		}
		if SelectedTheme() != beforeName || Ground != beforeGround || Ink != beforeInk {
			t.Errorf("refusing %q changed the active Chroma theme", name)
		}
	}
}

func TestSelectThemeRejectsUnknownNameWithoutChangingSelection(t *testing.T) {
	previous := SelectedTheme()
	t.Cleanup(func() { _ = SelectTheme(previous) })
	if err := SelectTheme("vaporwave"); err != nil {
		t.Fatal(err)
	}
	beforeName, beforeGround, beforeInk := SelectedTheme(), Ground, Ink
	if err := SelectTheme("bronze"); err == nil {
		t.Fatal("SelectTheme accepted a non-Chroma name")
	}
	if SelectedTheme() != beforeName || Ground != beforeGround || Ink != beforeInk {
		t.Error("rejected selection partially changed the active theme")
	}
}

func TestThemeForSizeSetsReadableTypeScale(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 24} {
		got := ThemeForSize(size)
		if got == nil {
			t.Fatalf("ThemeForSize(%g) is nil", size)
		}
		styles := []theme.TextStyle{
			got.Typography.DisplayLarge, got.Typography.DisplayMedium, got.Typography.DisplaySmall,
			got.Typography.HeadlineLarge, got.Typography.HeadlineMedium, got.Typography.HeadlineSmall,
			got.Typography.TitleLarge, got.Typography.TitleMedium, got.Typography.TitleSmall,
			got.Typography.BodyLarge, got.Typography.BodyMedium, got.Typography.BodySmall,
			got.Typography.LabelLarge, got.Typography.LabelMedium, got.Typography.LabelSmall,
		}
		for _, style := range styles {
			if style.Size < 16 {
				t.Errorf("size %g has text style below 16: %+v", size, style)
			}
		}
		if got.Typography.BodyMedium.Size != size {
			t.Errorf("body size = %g, want %g", got.Typography.BodyMedium.Size, size)
		}
		painter := ButtonPainter{Size: size}
		if painter.ButtonFontSize(button.Medium) != size {
			t.Errorf("button font size = %g, want %g", painter.ButtonFontSize(button.Medium), size)
		}
	}
	if got := ThemeForSize(42).Typography.BodyMedium.Size; got != 18 {
		t.Errorf("unsupported body size = %g, want default 18", got)
	}
}

func TestButtonPainterHasVisibleStatefulRoles(t *testing.T) {
	for _, primary := range []bool{true, false} {
		painter := ButtonPainter{Size: 18, Primary: primary}
		states := []button.PaintState{
			{Variant: button.Filled},
			{Variant: button.Filled, Hovered: true},
			{Variant: button.Filled, Pressed: true},
			{Variant: button.Filled, Focused: true},
			{Variant: button.Filled, Disabled: true},
		}
		for _, state := range states {
			style := painter.resolveButton(state)
			if style.borderWidth < 1 || style.border.A < 1 {
				t.Errorf("primary=%t state=%+v has no visible border: %+v", primary, state, style)
			}
			assertPair(t, "button label", style.foreground, style.background)
			if state.Focused && !state.Disabled && style.focusRing != Cyan {
				t.Errorf("focus ring = %s, want cyan %s", hex(style.focusRing), hex(Cyan))
			}
			if state.Disabled && style.foreground != Ink {
				t.Errorf("disabled foreground = %s, want readable ink %s", hex(style.foreground), hex(Ink))
			}
		}
		style := painter.resolveButton(button.PaintState{Variant: button.Filled})
		if primary && style.background != selected.primaryFill {
			t.Errorf("primary fill = %s, want selected Chroma fill %s", hex(style.background), hex(selected.primaryFill))
		}
		if !primary && (style.background != Ground || style.foreground != Cyan) {
			t.Errorf("secondary fill/text = %s/%s, want ground/link %s/%s", hex(style.background), hex(style.foreground), hex(Ground), hex(Cyan))
		}
	}
}

// TestAquaAndVaporwaveButtonStatesStayDistinctAndReadable verifies interaction colors.
func TestAquaAndVaporwaveButtonStatesStayDistinctAndReadable(t *testing.T) {
	previous := SelectedTheme()
	t.Cleanup(func() { _ = SelectTheme(previous) })
	for _, name := range []string{"aqua", "vaporwave"} {
		if err := SelectTheme(name); err != nil {
			t.Fatalf("SelectTheme(%q): %v", name, err)
		}
		for _, primary := range []bool{false, true} {
			painter := ButtonPainter{Size: 18, Primary: primary}
			idle := painter.resolveButton(button.PaintState{Variant: button.Filled})
			hover := painter.resolveButton(button.PaintState{Variant: button.Filled, Hovered: true})
			pressed := painter.resolveButton(button.PaintState{Variant: button.Filled, Pressed: true})
			t.Logf("theme=%s primary=%t fillSource=%s primaryFill=%s hoverFill=%s pressedFill=%s", name, primary, selected.primaryFillSource, hex(selected.primaryFill), hex(selected.primaryHoverFill), hex(selected.primaryPressedFill))
			for state, style := range map[string]buttonAppearance{
				"idle": idle, "hover": hover, "pressed": pressed,
			} {
				assertPair(t, name+" button label "+state, style.foreground, style.background)
			}
			if primary && idle.background == Ground {
				t.Errorf("%s primary fill is indistinguishable from its ground", name)
			}
			if name == "aqua" && primary && selected.primaryFillSource != "--selection-ink" {
				t.Errorf("Aqua primary fill source = %s, want readable Chroma --selection-ink", selected.primaryFillSource)
			}
			if hover.background == idle.background && hover.border == idle.border {
				t.Errorf("%s primary=%t hover has no visible state change", name, primary)
			}
			if pressed.background == hover.background && pressed.border == hover.border {
				t.Errorf("%s primary=%t pressed is indistinguishable from hover", name, primary)
			}
		}
		if Panel != Ground || Raised != Ground || DecorativePanel != selected.panel || DecorativeRaised != selected.raised {
			t.Errorf("%s must keep prose surfaces readable and preserve exact decorative Chroma roles", name)
		}
		assertPair(t, name+" prose", Ink, Ground)
	}
}

func TestDropdownAndDialogSizeAdaptersRenderAtSelectedSizes(t *testing.T) {
	trigger16, menu16 := renderDropdownText(16)
	trigger24, menu24 := renderDropdownText(24)
	if trigger24 <= trigger16 {
		t.Errorf("rendered dropdown trigger glyph height 16pt=%d, 24pt=%d", trigger16, trigger24)
	}
	if menu24 <= menu16 {
		t.Errorf("rendered dropdown menu glyph height 16pt=%d, 24pt=%d", menu16, menu24)
	}
	title16, action16 := renderDialogText(16)
	title24, action24 := renderDialogText(24)
	if title24 <= title16 {
		t.Errorf("rendered dialog title glyph height 16pt=%d, 24pt=%d", title16, title24)
	}
	if action24 <= action16 {
		t.Errorf("rendered dialog action glyph height 16pt=%d, 24pt=%d", action16, action24)
	}
}

func renderDropdownText(size float32) (triggerHeight, menuHeight int) {
	const width, height = 360, 100
	dc := gg.NewContext(width, height)
	canvas := render.NewCanvas(dc, width, height)
	canvas.DrawRect(geometry.NewRect(0, 0, width, height), Ground)
	painter := DropdownPainter{Size: size, Theme: Theme()}
	painter.PaintTrigger(canvas, &dropdown.TriggerPaintState{
		Bounds: geometry.NewRect(0, 0, width, 40), SelectedText: "Reference source",
	})
	painter.PaintMenu(canvas, &dropdown.MenuPaintState{
		Bounds: geometry.NewRect(0, 42, width, 40), Items: []dropdown.ItemDef{{Value: "Venus reference"}},
		VisibleCount: 1, ItemHeight: 40,
	})
	img := dc.Image()
	return glyphHeight(img, image.Rect(8, 2, width-36, 38), Ground),
		glyphHeight(img, image.Rect(8, 44, width-8, 80), Raised)
}

func renderDialogText(size float32) (titleHeight, actionHeight int) {
	const width, height = 420, 180
	dc := gg.NewContext(width, height)
	canvas := render.NewCanvas(dc, width, height)
	canvas.DrawRect(geometry.NewRect(0, 0, width, height), Ground)
	painter := DialogPainter{Size: size, Theme: Theme()}
	painter.PaintDialog(canvas, dialog.PaintState{
		Title: "Save changes", Bounds: geometry.NewRect(10, 10, 400, 160),
		Actions:     []dialog.Action{{Label: "Cancel"}},
		ActionRects: []geometry.Rect{geometry.NewRect(290, 110, 90, 36)},
	})
	img := dc.Image()
	return glyphHeight(img, image.Rect(28, 26, 380, 60), Panel),
		glyphHeight(img, image.Rect(292, 112, 378, 144), Panel)
}

func glyphHeight(img image.Image, area image.Rectangle, background widget.Color) int {
	minY, maxY := area.Max.Y, area.Min.Y
	bg := [3]uint8{uint8(background.R * 255), uint8(background.G * 255), uint8(background.B * 255)}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			pixel := [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
			if pixel != bg {
				minY = min(minY, y)
				maxY = max(maxY, y)
			}
		}
	}
	if maxY < minY {
		return 0
	}
	return maxY - minY + 1
}

func assertPair(t *testing.T, label string, foreground, background widget.Color) {
	t.Helper()
	front, back := linearLuminance(foreground), linearLuminance(background)
	ratio := (math.Max(front, back) + 0.05) / (math.Min(front, back) + 0.05)
	if ratio < 10 || ratio > 12.5 {
		t.Errorf("%s contrast %.3f outside 10–12.5", label, ratio)
	}
	frontHue, _ := hueSaturation(foreground)
	backHue, backgroundSaturation := hueSaturation(background)
	gap := math.Abs(frontHue - backHue)
	gap = math.Min(gap, 360-gap)
	if backgroundSaturation >= 0.35 && gap < 40 {
		t.Errorf("%s hue gap %.1f with background saturation %.3f", label, gap, backgroundSaturation)
	}
}

func hex(c widget.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", uint8(math.Round(float64(c.R*255))), uint8(math.Round(float64(c.G*255))), uint8(math.Round(float64(c.B*255))))
}

func linearLuminance(c widget.Color) float64 {
	channel := func(v float32) float64 {
		x := float64(v)
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

func hueSaturation(c widget.Color) (float64, float64) {
	r, g, b := float64(c.R), float64(c.G), float64(c.B)
	maxC, minC := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	delta := maxC - minC
	hue, saturation := 0.0, 0.0
	lightness := (maxC + minC) / 2
	if delta != 0 {
		saturation = delta / (1 - math.Abs(2*lightness-1))
		switch maxC {
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
	}
	return hue, saturation
}
