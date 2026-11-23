package thing

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv tables the Fresnel tests read, each beside numbers.yaml.
const (
	mirrorsSchlickGlassFile       = "testdata/mirrors-schlick-glass.csv"
	mirrorsSchlickFaceOnFile      = "testdata/mirrors-schlick-face-on.csv"
	mirrorsSchlickSlantFile       = "testdata/mirrors-schlick-slant.csv"
	mirrorsSchlickNoIndexFile     = "testdata/mirrors-schlick-no-index.csv"
	mirrorsSchlickDullFile        = "testdata/mirrors-schlick-dull.csv"
	mirrorsSchlickDullCosinesFile = "testdata/mirrors-schlick-dull-cosines.csv"
	mirrorsSchlickSlopeFile       = "testdata/mirrors-schlick-slope.csv"
	mirrorsSchlickSplitsFile      = "testdata/mirrors-schlick-splits.csv"
)

// mirrorsSchlickSideInside is the side column's word for light that
// arrives from inside the glass; any other word is the air.
const mirrorsSchlickSideInside = "inside"

// mirrorsSchlickTolerance is how close a share must be to the one worked
// by hand.
func mirrorsSchlickTolerance(t *testing.T) float64 {
	t.Helper()
	return picked(t, "mirrors-schlick-tolerance")
}

// mirrorsSchlickGlass is glass at a full gloss.
func mirrorsSchlickGlass(t *testing.T) *Material {
	t.Helper()
	index := picked(t, "mirrors-schlick-glass-index")
	return &Material{Gloss: unity, Index: index}
}

// mirrorsSchlickCheckIndex holds Mirrors, at the row's index and cosine and
// a gloss of one, to the row's numerator over its denominator.
func mirrorsSchlickCheckIndex(t *testing.T, row map[string]string) {
	t.Helper()
	index, cosine := cell(t, row, "index"), cell(t, row, "cosine")
	want := cell(t, row, "numerator") / cell(t, row, "denominator")
	material := &Material{Gloss: unity, Index: index}
	if got := material.Mirrors(cosine); !nearly(got, want,
		mirrorsSchlickTolerance(t)) {
		t.Errorf("an index of %v at cos %v mirrors %v, want %v",
			index, cosine, got, want)
	}
}

// schlick's fresnel is f0 + (1-f0)(1-cos)^5, with f0 = ((n-1)/(n+1))^2,
// scaled by Gloss. worked by hand: at n = 1.5, f0 = (0.5/2.5)^2 = 0.04.
//
//	cos 1:   0.04
//	cos 0.5: 0.04 + 0.96 * (0.5)^5 = 0.04 + 0.03 = 0.07
//	cos 0.2: 0.04 + 0.96 * (0.8)^5 = 0.04 + 0.96 * 0.32768 = 0.3545728
//	cos 0:   0.04 + 0.96 = 1
func TestMirrorsIsSchlicksFresnel(t *testing.T) {
	tolerance := mirrorsSchlickTolerance(t)
	glass := mirrorsSchlickGlass(t)
	for _, row := range readSheet(t, mirrorsSchlickGlassFile) {
		cosine, want := cell(t, row, "cosine"), cell(t, row, "want")
		if got := glass.Mirrors(cosine); !nearly(got, want, tolerance) {
			t.Errorf("glass at cos %v mirrors %v, want %v", cosine, got, want)
		}
	}
	// a gloss of a half is half of it
	middle := picked(t, "mirrors-schlick-half")
	matte := &Material{Gloss: middle, Index: glass.Index}
	mirrored := picked(t, "mirrors-schlick-half-gloss-mirror")
	if got := matte.Mirrors(middle); !nearly(got, mirrored, tolerance) {
		t.Errorf("a gloss of a half at cos 0.5 mirrors %v, want %v",
			got, mirrored)
	}
}

// f0 is ((n-1)/(n+1))^2, worked by hand face on at a gloss of one:
// n = 2 is (1/3)^2 = 1/9, n = 3 is (2/4)^2 = 0.25, and n = 1, no
// change from the air, mirrors nothing face on.
func TestMirrorsFaceOnIsTheIndexsFresnel(t *testing.T) {
	for _, row := range readSheet(t, mirrorsSchlickFaceOnFile) {
		mirrorsSchlickCheckIndex(t, row)
	}
	// n = 2 at cos 0.5 is 1/9 + (8/9)/32 = 1/9 + 1/36 = 5/36
	slant := readSheet(t, mirrorsSchlickSlantFile)
	mirrorsSchlickCheckIndex(t,
		slant.row(t, mirrorsSchlickSlantFile, "index two at a half"))
}

// no index, or one that is not greater than zero, is glass's 1.5: 0.04 face on,
// where an index of 4 would be 0.36 and of zero would be 1.
func TestNoIndexIsGlass(t *testing.T) {
	tolerance := mirrorsSchlickTolerance(t)
	want := picked(t, "mirrors-schlick-glass-face-on")
	for _, row := range readSheet(t, mirrorsSchlickNoIndexFile) {
		index := cell(t, row, "index")
		material := &Material{Gloss: unity, Index: index}
		if got := material.Mirrors(unity); !nearly(got, want, tolerance) {
			t.Errorf("an index of %v face on mirrors %v, want 0.04", index, got)
		}
	}
}

// a cos past one or less than zero is the nearest of face on and grazing: 0.04
// and 1 at glass.
func TestMirrorsClampsTheCosine(t *testing.T) {
	tolerance := mirrorsSchlickTolerance(t)
	glass := mirrorsSchlickGlass(t)
	past := picked(t, "mirrors-schlick-cosine-past-one")
	faceOn := picked(t, "mirrors-schlick-glass-face-on")
	if got := glass.Mirrors(past); !nearly(got, faceOn, tolerance) {
		t.Errorf("a cos of 1.5 mirrors %v, want 0.04", got)
	}
	below := picked(t, "mirrors-schlick-cosine-below-zero")
	if got := glass.Mirrors(below); !nearly(got, unity, tolerance) {
		t.Errorf("a cos of -0.5 mirrors %v, want 1", got)
	}
}

// a surface with no gloss, or a gloss less than zero, mirrors nothing, at any
// look: zero is zero even for a look of no number, which the formula alone
// would carry through as no number.
func TestADullSurfaceMirrorsNothing(t *testing.T) {
	index := mirrorsSchlickGlass(t).Index
	for _, dull := range readSheet(t, mirrorsSchlickDullFile) {
		gloss := cell(t, dull, "gloss")
		material := &Material{Gloss: gloss, Index: index}
		for _, look := range readSheet(t, mirrorsSchlickDullCosinesFile) {
			cosine := cell(t, look, "cosine")
			if got := material.Mirrors(cosine); got != nought {
				t.Errorf("a gloss of %v at cos %v mirrors %v",
					gloss, cosine, got)
			}
		}
	}
}

// roughness is one without a texture, and with one is the texture's
// luminance at the point asked, in the material's own space. a grey of
// v has luminance v: the weights 0.2126 + 0.7152 + 0.0722 come to one.
func TestRoughnessIsTheTexturesLuminance(t *testing.T) {
	tolerance := mirrorsSchlickTolerance(t)
	anywhere := maths.V(picked(t, "mirrors-schlick-anywhere-x"),
		picked(t, "mirrors-schlick-anywhere-y"),
		picked(t, "mirrors-schlick-anywhere-z"))
	if got := (&Material{}).Roughness(anywhere); got != unity {
		t.Errorf("a surface rough everywhere alike is %v, want 1", got)
	}
	material := &Material{Rough: slopeTexture{}}
	for _, row := range readSheet(t, mirrorsSchlickSlopeFile) {
		at := maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
		want := cell(t, row, "want")
		if got := material.Roughness(at); !nearly(got, want, tolerance) {
			t.Errorf("roughness at %v is %v, want %v", at, got, want)
		}
	}
	// a colour that is not grey: red alone is 0.2126
	scarlet := &Material{Rough: flatTexture{maths.RGB(unity, nought, nought)}}
	origin := maths.V(nought, nought, nought)
	luminance := picked(t, "mirrors-schlick-red-luminance")
	if got := scarlet.Roughness(origin); !nearly(got, luminance, tolerance) {
		t.Errorf("roughness of red is %v, want 0.2126", got)
	}
}

// slopeTexture is a grey as light as the point is far along x.
type slopeTexture struct{}

// At is the grey the point's x makes.
func (slopeTexture) At(point maths.Vec) maths.Light {
	return maths.Grey(point.X)
}

// flatTexture is the same light everywhere.
type flatTexture struct{ light maths.Light }

// At is the one light, wherever the point is.
func (flat flatTexture) At(maths.Vec) maths.Light { return flat.light }

// relativeIndex is the index of what the light goes into over the index of
// what it comes from, for light that arrives from the air onto the material,
// or from inside it when within.
func relativeIndex(material *Material, within bool) float64 {
	relative := material.Dense()
	if within {
		relative = unity / relative
	}
	return relative
}

// SplitsBetween is Fresnel's share for a surface that transmits, by the side
// the light arrives from, worked by hand at n = 1.5. Square on it is
// (0.5 / 2.5)^2 = 0.04 from either side. Inside the glass at cosine 0.8
// the sine is 0.6, the leaving sine is 1.5 * 0.6 = 0.9 and its cosine
// t is 0.435890, so for light going from index 1.5 to 1:
//
//	rs = (1.5 i - t) / (1.5 i + t) = (1.2 - 0.435890) / (1.2 + 0.435890)
//	   = 0.764110 / 1.635890 = 0.467091
//	rp = (i - 1.5 t) / (i + 1.5 t) = (0.8 - 0.653835) / (0.8 + 0.653835)
//	   = 0.146165 / 1.453835 = 0.100537
//	R  = (rs^2 + rp^2) / 2 = (0.218173 + 0.010108) / 2 = 0.114140
//
// At 60 degrees inside, cosine 0.5, the sine 0.866 is past the critical
// angle, asin(1 / 1.5) = 41.81 degrees, and the share is one at any
// gloss. A gloss of a half halves a share that is not total.
func TestSplitsIsFresnelBySide(t *testing.T) {
	tolerance := mirrorsSchlickTolerance(t)
	glass := mirrorsSchlickGlass(t)
	for _, row := range readSheet(t, mirrorsSchlickSplitsFile) {
		within := row["side"] == mirrorsSchlickSideInside
		want := cell(t, row, "want")
		got := glass.SplitsBetween(cell(t, row, "cosine"),
			relativeIndex(glass, within))
		if !nearly(got, want, tolerance) {
			t.Errorf("%s: share %v, want %v", row["name"], got, want)
		}
	}
	under := picked(t, "mirrors-schlick-under-critical-cosine")
	share := picked(t, "mirrors-schlick-under-critical-share")
	loose := picked(t, "mirrors-schlick-loose-tolerance")
	if got := glass.SplitsBetween(under, relativeIndex(glass, true)); !nearly(
		got, share, loose) {
		t.Errorf("inside under the critical angle: share %v, want %v",
			got, share)
	}
	middle := picked(t, "mirrors-schlick-half")
	matte := &Material{Gloss: middle, Index: glass.Index}
	squareOn := picked(t, "mirrors-schlick-half-gloss-square-on-share")
	if got := matte.SplitsBetween(unity, relativeIndex(matte, false)); !nearly(
		got, squareOn, tolerance) {
		t.Errorf("a gloss of a half square on: share %v, want 0.02", got)
	}
	if got := matte.SplitsBetween(middle, relativeIndex(matte, true)); got != unity {
		t.Errorf("a gloss of a half past the critical angle: share %v, want 1",
			got)
	}
}
