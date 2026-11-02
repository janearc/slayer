package typography_test

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/typography"
	"github.com/janearc/slayer/typography/mocks"
	"go.uber.org/mock/gomock"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// glyph is a glyph's number in its font.
type glyph = typography.Glyph

// a font that kerns "AV" by minus 150 of its units, as a font with a
// kerning table would: go's own fonts have none, so the font is a mock,
// told what it is asked and in what order.
func kerning(t *testing.T, kern float64) *mocks.MockMetrics {
	control := gomock.NewController(t)
	font := mocks.NewMockMetrics(control)
	gomock.InOrder(
		font.EXPECT().Index('A').Return(glyph(36), nil),
		font.EXPECT().Advance(glyph(36)).Return(1366.0, nil),
		font.EXPECT().Index('V').Return(glyph(57), nil),
		font.EXPECT().Kern(glyph(36), glyph(57)).
			Return(kern, nil),
		font.EXPECT().Advance(glyph(57)).Return(1366.0, nil),
	)
	return font
}

// "AV" in a font that kerns it is set narrower than "A" and "V" each
// alone: the V's pen is the A's advance less the kerning, and so is the
// line's width less it.
func TestALineFollowsTheFontsKerning(t *testing.T) {
	line, err := typography.Layout(kerning(t, -150), "AV")
	if err != nil {
		t.Fatal(err)
	}
	if got := line.Letters[1].Pen; got != 1366-150 {
		t.Errorf("the V's pen at %v, want 1216", got)
	}
	if line.Width != 2*1366-150 || !(line.Width < 2*1366) {
		t.Errorf("AV is %v wide, A and V alone 2732", line.Width)
	}
}

// with no kerning the pair is as wide as its letters: the test above
// would see a line that ignored the font's kerning.
func TestALineWithNoKerningIsItsAdvances(t *testing.T) {
	line, err := typography.Layout(kerning(t, 0), "AV")
	if err != nil {
		t.Fatal(err)
	}
	if line.Width != 2*1366 || line.Letters[1].Pen != 1366 {
		t.Errorf("AV unkerned: %+v", line)
	}
}

// a font that cannot say a glyph, an advance or a kerning stops the
// line with what it said.
func TestALineStopsAtWhatTheFontCannotSay(t *testing.T) {
	broken := errors.New("broken")
	for name, expect := range map[string]func(*mocks.MockMetrics){
		"index": func(f *mocks.MockMetrics) {
			f.EXPECT().Index('A').Return(glyph(0), broken)
		},
		"advance": func(f *mocks.MockMetrics) {
			f.EXPECT().Index('A').Return(glyph(36), nil)
			f.EXPECT().Advance(glyph(36)).Return(0.0, broken)
		},
		"kern": func(f *mocks.MockMetrics) {
			f.EXPECT().Index('A').Return(glyph(36), nil)
			f.EXPECT().Advance(glyph(36)).Return(1366.0, nil)
			f.EXPECT().Index('V').Return(glyph(57), nil)
			f.EXPECT().Kern(gomock.Any(), gomock.Any()).
				Return(0.0, broken)
		},
	} {
		font := mocks.NewMockMetrics(gomock.NewController(t))
		expect(font)
		_, err := typography.Layout(font, "AV")
		if !errors.Is(err, broken) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// in go's regular, "carrie" is set at sfnt's own advances, each letter
// where the ones before it leave the pen, and go's fonts kern nothing:
// "AV" is as wide as "A" and "V".
func TestGosRegularIsSetAtItsAdvances(t *testing.T) {
	face, err := typography.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := sfnt.Parse(goregular.TTF)
	var work sfnt.Buffer
	em := fixed.I(int(raw.UnitsPerEm()))
	line, err := typography.Layout(face, "carrie")
	if err != nil {
		t.Fatal(err)
	}
	pen := 0.0
	for i, letter := range "carrie" {
		glyph, _ := raw.GlyphIndex(&work, letter)
		advance, _ := raw.GlyphAdvance(&work, glyph, em,
			font.HintingNone)
		if got := line.Letters[i].Pen; got != pen {
			t.Errorf("%q at %v, want %v", letter, got, pen)
		}
		pen += float64(advance) / 64
	}
	if line.Width != pen {
		t.Errorf("carrie is %v wide, want %v", line.Width, pen)
	}
	pair, _ := typography.Layout(face, "AV")
	a, _ := typography.Layout(face, "A")
	v, _ := typography.Layout(face, "V")
	if pair.Width != a.Width+v.Width {
		t.Errorf("AV %v, A and V %v", pair.Width, a.Width+v.Width)
	}
}
