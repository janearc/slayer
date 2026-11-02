package hud

import (
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// themed is a theme of roles given as hex, by name.
func themed(roles map[string]string) Theme {
	sheet := css.New()
	for name, hex := range roles {
		sheet.Set(name, srgb.MustHex(hex).Swatch())
	}
	return NewTheme(sheet)
}

// dusk and dawn are two themes with the same roles in other colours.
var (
	dusk = themed(map[string]string{"ink": "#e8dcc8", "ground": "#1d1b26",
		"ok": "#7fb069", "red": "#c94f4f"})
	dawn = themed(map[string]string{"ink": "#2b2622", "ground": "#f4efe6",
		"ok": "#3c7a3a", "red": "#a33b3b"})
)

// rgba is a hex colour as a screen's bytes.
func rgba(hex string) color.RGBA {
	r, g, b := srgb.MustHex(hex).Bytes()
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// a role's colour is the theme's: ink in dusk is dusk's ink to the byte,
// and in dawn is dawn's; a role the theme does not set but resolves,
// err from red, is there too.
func TestARolesColourIsTheThemes(t *testing.T) {
	for theme, want := range map[*Theme]string{&dusk: "#e8dcc8",
		&dawn: "#2b2622"} {
		got, err := theme.Colour("ink")
		if err != nil || got != rgba(want) {
			t.Errorf("ink %v, %v; want %s", got, err, want)
		}
	}
	got, err := dusk.Colour("err")
	if err != nil || got != rgba("#c94f4f") {
		t.Errorf("err resolved from red: %v, %v", got, err)
	}
}

// the same hud drawn in another theme is drawn in that theme's colours,
// with nothing of the hud changed: every cell's ink and ground follow.
func TestAHudChangesWithItsTheme(t *testing.T) {
	made := Hud{Elements: []Element{
		Text{Words: Words("ready"), At: Point{0.5, 0.5}, Align: Centre,
			Ink: "ink", Ground: "ground"},
		Bar{Box: Box{0, 0, 1, 0.1}, Value: Fixed(0.5), Fill: "ok",
			Empty: "ground"}}}
	rect := image.Rect(0, 0, 40, 10)
	one, err := made.Cover(rect, 0, dusk, View{})
	if err != nil {
		t.Fatal(err)
	}
	two, err := made.Cover(rect, 0, dawn, View{})
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Mask()) != len(two.Mask()) {
		t.Fatalf("%d cells in dusk, %d in dawn", len(one.Mask()),
			len(two.Mask()))
	}
	for _, at := range one.Mask() {
		a, _ := one.At(at.X, at.Y)
		b, _ := two.At(at.X, at.Y)
		if a.Rune != b.Rune || a.Fg == b.Fg || a.Bg == b.Bg {
			t.Errorf("cell %v: dusk %+v, dawn %+v", at, a, b)
		}
	}
	text, _ := one.At(20, 5)
	if text.Fg != rgba("#e8dcc8") || text.Bg != rgba("#1d1b26") {
		t.Errorf("dusk's words are %+v", text)
	}
}

// a role the theme has not, and no theme at all, are errors naming the
// role; a hud that names one is not drawn.
func TestARoleTheThemeHasNotIsAnError(t *testing.T) {
	if _, err := dusk.Colour("chartreuse"); !errors.Is(err, ErrHud) {
		t.Errorf("an unknown role: %v", err)
	}
	if _, err := (Theme{}).Colour("ink"); !errors.Is(err, ErrHud) {
		t.Errorf("no theme: %v", err)
	}
	made := Hud{Elements: []Element{Text{Words: Words("x"),
		Ink: "chartreuse"}}}
	if _, err := made.Cover(image.Rect(0, 0, 4, 4), 0, dusk,
		View{}); !errors.Is(err, ErrHud) {
		t.Errorf("a hud in an unknown role: %v", err)
	}
}

// a colourway file is read as its roles, and one not there is an error.
func TestAThemeIsReadFromItsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dusk.css")
	os.WriteFile(path, []byte(":root {\n  --ink: #e8dcc8;\n"+
		"  --ground: #1d1b26;\n}\n"), 0o644)
	theme, err := ReadTheme(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := theme.Colour("ground"); err != nil ||
		got != rgba("#1d1b26") {
		t.Errorf("ground %v, %v", got, err)
	}
	if _, err := ReadTheme(path + ".gone"); !errors.Is(err, ErrHud) {
		t.Errorf("a file not there: %v", err)
	}
}

// a ramp is its from before it starts, its to once it is over, and the
// straight line between; with no time to run it steps at its start. a
// count writes its gauge out by its format.
func TestValuesChangeWithTheFrame(t *testing.T) {
	ramp := Ramp{From: 0, To: 1, Start: time.Second, Over: 2 * time.Second}
	for at, want := range map[time.Duration]float64{0: 0, time.Second: 0,
		2 * time.Second: 0.5, 3 * time.Second: 1, 9 * time.Second: 1} {
		if got := ramp.At(at); got != want {
			t.Errorf("the ramp at %v is %v, want %v", at, got, want)
		}
	}
	step := Ramp{From: 2, To: 5, Start: time.Second}
	if step.At(999*time.Millisecond) != 2 || step.At(time.Second) != 5 {
		t.Error("a ramp with no time does not step at its start")
	}
	percent := Ramp{From: 0, To: 100, Start: time.Second,
		Over: 2 * time.Second}
	count := Count{Of: percent, Format: "%.0f%%"}
	if got := count.At(2 * time.Second); got != "50%" {
		t.Errorf("the count at two seconds says %q", got)
	}
	if got := (Count{Of: Fixed(42.5), Format: "%05.1f"}).At(0); got !=
		"042.5" {
		t.Errorf("a fixed count says %q", got)
	}
	if Words("hi").At(time.Hour) != "hi" {
		t.Error("words change")
	}
}
