package colour_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/janearc/slayer/ninjafilm/colour"
	"pgregory.net/rapid"
)

// anti-tests: a mask of another shape cuts something. Some pixels of the
// picture are black after it, and some are kept.
func TestAMaskOfAnotherShapeCutsSomething(t *testing.T) {
	for _, row := range rows(t, masksFile) {
		img := marked(maskAcross, maskDown)
		out := colour.Masked(img, amount(t, row, "aspect"), amount(t, row,
			"pixel"))
		cut, kept := none, none
		for y := range maskDown {
			for x := range maskAcross {
				if out.RGBAAt(x, y) == (color.RGBA{A: opaque}) {
					cut++
				} else {
					kept++
				}
			}
		}
		full := cut == none
		if kept == none || (full && count(t, row, "x1")-count(t, row,
			"x0") != maskAcross) {
			t.Errorf("%s cut %d and kept %d", text(t, row, "label"), cut,
				kept)
		}
	}
}

// anti-tests: a mask of another shape is not the picture it was given.
// Where the shape differs the pixels differ.
func TestAMaskOfAnotherShapeIsNotThePicture(t *testing.T) {
	rapid.Check(t, maskOfAnotherShapeDiffers)
}

func maskOfAnotherShapeDiffers(t *rapid.T) {
	img := pictureOf(t)
	bounds := img.Bounds()
	have := float64(bounds.Dx()) / float64(bounds.Dy())
	aspect := have * (pair + rapid.Float64Range(none, lightMost).Draw(t,
		"more"))
	out := colour.Masked(img, aspect, whole)
	if string(out.Pix) == string(img.Pix) {
		t.Skip("bars that cut nothing of a picture of one pixel")
	}
	if out.Bounds() != bounds {
		t.Fatalf("the mask changed the bounds")
	}
}

// anti-tests: another size of clump is another grain, and so is another
// amount. Grain of two sizes or two amounts does not match.
func TestAnotherSizeOrAmountOfGrainIsAnotherGrain(t *testing.T) {
	img := coloured(grainAcross, grainDown)
	base := colour.Grained(img, pair, softAmount, grainSeed)
	size := colour.Grained(img, pair+pair, softAmount, grainSeed)
	more := colour.Grained(img, pair, softAmount+softAmount, grainSeed)
	if string(base.Pix) == string(size.Pix) ||
		string(base.Pix) == string(more.Pix) {
		t.Error("grain of another size or amount matched")
	}
}

// anti-tests: more amount of grain does not move the light less. The
// grain of twice the amount strays further from the picture.
func TestMoreGrainDoesNotMoveTheLightLess(t *testing.T) {
	img := flat(maskAcross, maskDown)
	small := colour.Grained(img, softSize, softAmount, grainSeed)
	large := colour.Grained(img, softSize, pair*softAmount, grainSeed)
	var near, far float64
	for y := range maskDown {
		for x := range maskAcross {
			near += distance(small.RGBAAt(x, y).R, greyByte)
			far += distance(large.RGBAAt(x, y).R, greyByte)
		}
	}
	if !(far > near) {
		t.Errorf("twice the amount strayed %v, the plain amount %v", far, near)
	}
}

// anti-tests: grain of a picture is never a picture of another size. The
// bounds of the grain are the bounds it was given, a cut picture's too.
func TestGrainNeverChangesTheBoundsOfAPicture(t *testing.T) {
	rapid.Check(t, grainKeepsTheBounds)
}

func grainKeepsTheBounds(t *rapid.T) {
	img := pictureOf(t)
	got := colour.Grained(img, rapid.Float64Range(radiusLeast, bloomRadius).Draw(
		t, "size"), softAmount, grainSeed)
	if got.Bounds() != (image.Rectangle{Min: img.Bounds().Min,
		Max: img.Bounds().Max}) {
		t.Fatalf("grain made %v of %v", got.Bounds(), img.Bounds())
	}
}
