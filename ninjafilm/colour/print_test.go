package colour_test

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/grain"
	"pgregory.net/rapid"
)

// the tables the print tests are run over
const (
	masksFile   = "testdata/masks.csv"
	grainsFile  = "testdata/grains.csv"
	noGrainFile = "testdata/no-grain.csv"
)

// what the print tests count in, each defined
const (
	opaque = 255 // defined: the alpha of a pixel that hides what is behind it
)

// what the print tests choose, each picked in picked-numbers.yml
var (
	maskAcross  = data.Whole("colour-mask-across")
	maskDown    = data.Whole("colour-mask-down")
	grainAcross = data.Whole("colour-grain-across")
	grainDown   = data.Whole("colour-grain-down")
	grainSeed   = uint64(data.Whole("colour-grain-seed"))
	softSize    = data.Picked("colour-grain-size")
	softAmount  = data.Picked("colour-grain-amount")
	softness    = data.Picked("colour-grain-soft")
	levels      = data.Whole("colour-grain-levels")
	greyByte    = uint8(data.Whole("colour-grey"))
	noiseBound  = data.Picked("grain-bound")
	noiseHeight = data.Picked("grain-height")
)

// marked is a picture whose every pixel is its own. Red is the column
// and green is the row. Blue is the same everywhere. Nothing is black.
// So a bar shows. Pixels the mask touched that it should not have are
// not the ones they were.
func marked(across, down int) *image.RGBA {
	img := image.NewRGBA(image.Rect(none, none, across, down))
	for y := range down {
		for x := range across {
			img.SetRGBA(x, y, color.RGBA{uint8(whole + x), uint8(whole + y),
				greyByte, opaque})
		}
	}
	return img
}

// Masks keep a rectangle of the picture to the pixel. What is in it is
// as it was and what is out of it is black. The rectangle is worked by
// integer arithmetic on the shape. Have is the width of the picture
// times the pixel over its height.
//
// Wider frames keep rows of the height times have over aspect. Half the
// bars are above. Narrower ones keep columns of the width times aspect
// over have. Half the bars are beside. Each row of the table is worked
// by hand on a picture of 100 by 50.
func TestAMaskKeepsTheRectangleTheShapeLeaves(t *testing.T) {
	for _, row := range rows(t, masksFile) {
		img := marked(maskAcross, maskDown)
		out := colour.Masked(img, amount(t, row, "aspect"), amount(t, row,
			"pixel"))
		keep := image.Rect(count(t, row, "x0"), count(t, row, "y0"),
			count(t, row, "x1"), count(t, row, "y1"))
		for y := range maskDown {
			for x := range maskAcross {
				want := img.RGBAAt(x, y)
				if !(image.Point{X: x, Y: y}).In(keep) {
					want = color.RGBA{A: opaque}
				}
				if got := out.RGBAAt(x, y); got != want {
					t.Fatalf("%s: (%d, %d) is %v, want %v", text(t, row,
						"label"), x, y, got, want)
				}
			}
		}
	}
}

// pictureOf draws a picture of drawn bytes, cut from a larger one so its
// own corner is not the origin. Each pixel is its own.
func pictureOf(t *rapid.T) *image.RGBA {
	size := rapid.IntRange(pair, pictureMost)
	across, down := size.Draw(t, "across"), size.Draw(t, "down")
	left, top := size.Draw(t, "left"), size.Draw(t, "top")
	big := image.NewRGBA(image.Rect(none, none, across+left+whole,
		down+top+whole))
	for index := range big.Pix {
		big.Pix[index] = rapid.Byte().Draw(t, "byte")
	}
	return big.SubImage(image.Rect(left, top, left+across,
		top+down)).(*image.RGBA)
}

// Whatever the shape asked for, a mask leaves the bars even to a pixel.
// It keeps every pixel it keeps as it was and makes the rest black. It
// leaves the picture it was given as it was. Pictures cut from larger
// ones keep their own pixels. They do not run the rows of the larger one
// together.
func TestAMaskKeepsPixelsAndBarsEvenly(t *testing.T) {
	rapid.Check(t, maskKeepsPixelsAndBarsEvenly)
}

func maskKeepsPixelsAndBarsEvenly(t *rapid.T) {
	img := pictureOf(t)
	was := append([]uint8(nil), img.Pix...)
	aspect := rapid.Float64Range(radiusLeast, lightMost).Draw(t, "aspect")
	pixel := rapid.Float64Range(radiusLeast, pair).Draw(t, "pixel")
	out := colour.Masked(img, aspect, pixel)
	bounds := out.Bounds()
	if bounds != img.Bounds() {
		t.Fatalf("a mask of %v made a picture %v", img.Bounds(), bounds)
	}
	var first, last image.Point
	kept := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			got := out.RGBAAt(x, y)
			if got == img.RGBAAt(x, y) {
				if !kept {
					first = image.Point{X: x, Y: y}
				}
				kept, last = true, image.Point{X: x, Y: y}
				continue
			}
			if got != (color.RGBA{A: opaque}) {
				t.Fatalf("(%d, %d) is %v, neither as it was nor black", x, y,
					got)
			}
		}
	}
	if kept && (first.X-bounds.Min.X < none || bounds.Max.X-whole-last.X <
		none) {
		t.Fatalf("the bars %v %v lie outside %v", first, last, bounds)
	}
	for place := range was {
		if img.Pix[place] != was[place] {
			t.Fatalf("the mask changed the picture it was given")
		}
	}
}

// flat is a flat grey picture.
func flat(across, down int) *image.RGBA {
	img := image.NewRGBA(image.Rect(none, none, across, down))
	for y := range down {
		for x := range across {
			img.SetRGBA(x, y, color.RGBA{greyByte, greyByte, greyByte, opaque})
		}
	}
	return img
}

// distance is how far apart two bytes are, as a number.
func distance(first, second uint8) float64 {
	return math.Abs(float64(first) - float64(second))
}

// coloured is a picture whose pixels differ. Its three parts differ in
// each. Grain taken to the wrong part or the wrong pixel shows. It is
// cut from a larger one. So its own corner is not the origin.
func coloured(across, down int) *image.RGBA {
	shade := int(greyByte)
	big := image.NewRGBA(image.Rect(none, none, across+pair*pair,
		down+pair*pair))
	for y := range down + pair*pair {
		for x := range across + pair*pair {
			red, green := shade/pair+x*pair, shade/pair+y*pair
			blue := shade - x*pair - y
			big.SetRGBA(x, y, color.RGBA{uint8(red), uint8(green), uint8(blue),
				opaque})
		}
	}
	return big.SubImage(image.Rect(pair, pair, pair+across,
		pair+down)).(*image.RGBA)
}

// grained is the picture that grain makes of a picture, worked from the
// doc. It is the noise at the centre of each pixel in units of the size
// of the clump at the height of the grain. It is scaled by the amount
// about one. It is applied to each part in linear light.
func grained(img *image.RGBA, size, amount float64, seed uint64) *image.RGBA {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			across, down := (float64(x)+half)/size, (float64(y)+half)/size
			noise := grain.Gradient(seed, across, down, noiseHeight)
			shift := whole + amount*noise
			seen := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			red := colour.Decode(seen.R) * shift
			green := colour.Decode(seen.G) * shift
			blue := colour.Decode(seen.B) * shift
			out.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.RGBA{
				colour.Encode(red), colour.Encode(green),
				colour.Encode(blue), opaque})
		}
	}
	return out
}

// differ is how far apart two bytes are.
func differ(first, second uint8) int {
	return max(int(first)-int(second), int(second)-int(first))
}

// agree says two pictures are the same to a byte in each part. The grain
// is worked twice, and a fused multiply on one machine may round a byte
// over where another does not.
func agree(t *testing.T, name string, got, want *image.RGBA) {
	t.Helper()
	if got.Bounds() != want.Bounds() {
		t.Fatalf("%s: bounds %v, want %v", name, got.Bounds(), want.Bounds())
	}
	bounds := want.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			have, need := got.RGBAAt(x, y), want.RGBAAt(x, y)
			if differ(have.R, need.R) > whole || differ(have.G, need.G) >
				whole || differ(have.B, need.B) > whole || have.A != need.A {
				t.Errorf("%s: (%d, %d) is %v, want %v", name, x, y, have,
					need)
				return
			}
		}
	}
}

// Grain is the noise at the centre of each pixel in clumps of so many
// pixels. It moves the light by up to the amount. For clumps of a pixel
// or more the amount is as asked. For finer ones the clump is a pixel and
// the amount falls by the size. So half a pixel at 0.4 is a pixel at 0.2.
// Sizes of one and a half keep their own amount and clump.
func TestGrainFollowsTheNoiseAtEachPixelsCentre(t *testing.T) {
	img := coloured(grainAcross, grainDown)
	for _, row := range rows(t, grainsFile) {
		got := colour.Grained(img, amount(t, row, "size"), amount(t, row,
			"amount"), grainSeed)
		sizeIs, amountIs := amount(t, row, "size_is"), amount(t, row, "amount_is")
		want := grained(img, sizeIs, amountIs, grainSeed)
		agree(t, text(t, row, "label"), got, want)
	}
	first := colour.Grained(img, pair, softAmount, grainSeed)
	second := colour.Grained(img, pair, softAmount, grainSeed+whole)
	if string(first.Pix) == string(second.Pix) {
		t.Error("two seeds made one grain")
	}
}

// Grain lightens and darkens in soft clumps of the size asked. It moves
// the light by no more than the amount. It is the same for a seed.
// Pixels are nearer their neighbour than the one three away.
func TestGrainIsSoftAndBounded(t *testing.T) {
	img := flat(maskAcross, maskDown)
	made := colour.Grained(img, softSize, softAmount, grainSeed)
	seen := map[uint8]bool{}
	var near, far float64
	across := int(softSize)
	for y := range maskDown {
		for x := across; x < maskAcross; x++ {
			code := made.RGBAAt(x, y).R
			seen[code] = true
			beside, apart := made.RGBAAt(x-whole, y).R,
				made.RGBAAt(x-across, y).R
			near += distance(code, beside)
			far += distance(code, apart)
			ratio := colour.Decode(code) / colour.Decode(greyByte)
			if math.Abs(ratio-whole) > noiseBound*softAmount {
				t.Fatalf("grain moved a pixel to %v of what it was", ratio)
			}
		}
	}
	if len(seen) < levels || !(near < softness*far) {
		t.Errorf("%d levels, neighbours differ %.0f, three apart %.0f",
			len(seen), near, far)
	}
	again := colour.Grained(img, softSize, softAmount, grainSeed)
	if string(again.Pix) != string(made.Pix) {
		t.Error("the grain changed between two asks")
	}
}

// anti-tests: no grain is the picture as it was. That is every part of every
// pixel with alpha among them. Sizes or amounts of zero or less give back what
// they were given. Grain draws every pixel opaque. The picture given is not
// changed by the grain made of it.
func TestNoGrainIsThePictureWithItsAlpha(t *testing.T) {
	img := image.NewRGBA(image.Rect(none, none, pair*pair, pair+whole))
	for y := range pair + whole {
		for x := range pair * pair {
			img.SetRGBA(x, y, color.RGBA{greyByte / pair, greyByte,
				opaque / pair, greyByte})
		}
	}
	kept := append([]uint8(nil), img.Pix...)
	for _, row := range rows(t, noGrainFile) {
		out := colour.Grained(img, amount(t, row, "size"), amount(t, row,
			"amount"), grainSeed)
		if string(out.Pix) != string(kept) {
			t.Fatalf("%s changed the picture", text(t, row, "label"))
		}
	}
	colour.Grained(img, pair, softAmount, grainSeed)
	if string(img.Pix) != string(kept) {
		t.Fatal("grain changed the picture it was given")
	}
}
