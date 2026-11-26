package workspace

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/ferruccio/native/internal/settings"
)

func TestShortenKeepsTheEndOfALongPathAndMarksTheCut(t *testing.T) {
	got := shorten("/a/very/long/path/to/the/cache", 12)
	if utf8.RuneCountInString(got) != 12 || !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "o/the/cache") {
		t.Fatalf("got %q", got)
	}
	if shorten("/short", 12) != "/short" {
		t.Fatal("a path that fits was cut")
	}
}

func TestShortenNeverReturnsMoreThanItIsGivenRoomFor(t *testing.T) {
	for limit := 2; limit < 30; limit++ {
		if got := shorten(strings.Repeat("x", 60), limit); utf8.RuneCountInString(got) > limit {
			t.Fatalf("limit %d gave %d runes", limit, utf8.RuneCountInString(got))
		}
	}
}

func TestEverySettingsLineFitsTheSheetAtEveryTypeSize(t *testing.T) {
	for _, size := range []float32{16, 18, 20, 24} {
		work := settingsWorkspace(t, "cache :: /a/very/long/path/that/goes/on/and/on/and/on/for/a/long/while/and/more/and/more\n")
		work.size, work.theme = size, appearance.Theme()
		ctx := uitest.NewMockContext()
		ctx.WindowSizeVal = geometry.Sz(1600, 1000)
		made := newSheet(work.settingsPage(), nil)
		made.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
		canvas := &uitest.MockCanvas{}
		made.Draw(ctx, canvas)
		edge := (ctx.WindowSizeVal.Width + settingsSheetWidth) / 2
		for _, drawn := range canvas.Texts {
			if drawn.Bounds.Max.X > edge+1 {
				t.Fatalf("size %v: %q reaches %v, past the panel's edge %v", size, drawn.Text, drawn.Bounds.Max.X, edge)
			}
		}
	}
}

func TestAParagraphIsOneTextPerWrappedLine(t *testing.T) {
	work := settingsWorkspace(t, "")
	work.size = 16
	long := strings.Repeat("word ", 80)
	ctx := uitest.NewMockContext()
	panel := work.paragraph(long, appearance.Ink)
	panel.Layout(ctx, geometry.Loose(geometry.Sz(2000, 2000)))
	canvas := &uitest.MockCanvas{}
	panel.Draw(ctx, canvas)
	if len(canvas.Texts) < 2 {
		t.Fatalf("a long paragraph was drawn as %d line", len(canvas.Texts))
	}
	limit := work.runesIn(settingsSheetWidth - 2*settingsSheetPadding)
	for _, drawn := range canvas.Texts {
		if utf8.RuneCountInString(drawn.Text) > limit {
			t.Fatalf("a line of %d runes exceeds %d", utf8.RuneCountInString(drawn.Text), limit)
		}
	}
}

func TestALockedRowHasNoControlAndSaysWhy(t *testing.T) {
	work := settingsWorkspace(t, "")
	work.size, work.theme = 16, appearance.Theme()
	work.settings.Flag = func(name string) string {
		if name == "cache" {
			return "/from/the/flag"
		}
		return ""
	}
	ctx := uitest.NewMockContext()
	ctx.WindowSizeVal = geometry.Sz(1600, 1000)
	made := newSheet(work.settingsPage(), nil)
	made.Layout(ctx, geometry.Loose(ctx.WindowSizeVal))
	canvas := &uitest.MockCanvas{}
	made.Draw(ctx, canvas)
	joined := ""
	for _, drawn := range canvas.Texts {
		joined += drawn.Text + "\n"
	}
	if !strings.Contains(joined, "/from/the/flag") || !strings.Contains(joined, "cannot be changed here") {
		t.Fatalf("a locked row does not show its value and why:\n%s", joined)
	}
	key, _ := settings.KeyOf(settings.Cache)
	if work.valueRunes(settings.Entry{Key: key, Source: settings.FromFlag}) <= work.valueRunes(settings.Entry{Key: key, Source: settings.FromFile}) {
		t.Fatal("a row with no control was given no more room for its value")
	}
}
