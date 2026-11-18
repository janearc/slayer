package compare

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

func TestCompareImages(t *testing.T) {
	a := image.NewRGBA(image.Rect(0, 0, 2, 1))
	b := image.NewRGBA(image.Rect(0, 0, 2, 1))
	a.SetRGBA(0, 0, color.RGBA{R: 20, G: 30, B: 40, A: 255})
	a.SetRGBA(1, 0, color.RGBA{R: 10, G: 10, B: 10, A: 255})
	b.SetRGBA(0, 0, color.RGBA{R: 21, G: 32, B: 37, A: 255})
	b.SetRGBA(1, 0, color.RGBA{R: 12, G: 10, B: 10, A: 255})

	result, err := CompareImages(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if result.MaxDifference != [3]uint8{2, 2, 3} {
		t.Fatalf("maximum difference = %v", result.MaxDifference)
	}
	if result.FractionWithinOne != 0 {
		t.Fatalf("fraction within one = %g", result.FractionWithinOne)
	}
	if result.Histogram[0] != 2 || result.Histogram[1] != 1 || result.Histogram[2] != 2 || result.Histogram[3] != 1 {
		t.Fatalf("histogram = %v", result.Histogram)
	}
	wantPSNR := 10 * math.Log10(255*255/3.0)
	if math.Abs(result.PSNR-wantPSNR) > 1e-12 {
		t.Fatalf("PSNR = %.15g, want %.15g", result.PSNR, wantPSNR)
	}
	if result.Difference == nil || result.Difference.Bounds() != a.Bounds() {
		t.Fatalf("difference image = %v", result.Difference)
	}
	if got := result.Difference.RGBAAt(0, 0); got.R != 85 || got.G != 170 || got.B != 255 {
		t.Fatalf("scaled difference pixel = %v", got)
	}
}

func TestCompareImagesIdentical(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	result, err := CompareImages(img, img)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsInf(result.PSNR, 1) || result.FractionWithinOne != 1 {
		t.Fatalf("identical image metrics = PSNR %g, within one %g", result.PSNR, result.FractionWithinOne)
	}
	for difference, count := range result.Histogram {
		if difference == 0 && count != 12 || difference != 0 && count != 0 {
			t.Fatalf("histogram bin %d = %d", difference, count)
		}
	}
	if result.Difference.RGBAAt(0, 0) != (color.RGBA{A: 255}) {
		t.Fatalf("identical difference pixel = %v", result.Difference.RGBAAt(0, 0))
	}
}

func TestCompareImagesRejectsOversizedImage(t *testing.T) {
	oversized := image.NewUniform(color.RGBA{A: 255})
	if _, err := CompareImages(oversized, oversized); err == nil {
		t.Fatal("accepted an oversized image")
	}
}

func TestCompareImagesRejectsDifferentSizes(t *testing.T) {
	_, err := CompareImages(image.NewRGBA(image.Rect(0, 0, 1, 1)), image.NewRGBA(image.Rect(0, 0, 2, 1)))
	if err == nil {
		t.Fatal("expected size mismatch error")
	}
}

func TestComparePNGWritesDifference(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")
	difference := filepath.Join(dir, "difference.png")
	writePNG(t, a, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	changed := image.NewRGBA(image.Rect(0, 0, 1, 1))
	changed.SetRGBA(0, 0, color.RGBA{R: 3, A: 255})
	writePNG(t, b, changed)
	result, err := ComparePNG(a, b, difference)
	if err != nil {
		t.Fatal(err)
	}
	if result.MaxDifference != [3]uint8{3, 0, 0} {
		t.Fatalf("maximum difference = %v", result.MaxDifference)
	}
	file, err := os.Open(difference)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	pixel := color.RGBAModel.Convert(decoded.At(0, 0)).(color.RGBA)
	if pixel.R != 255 || pixel.G != 0 || pixel.B != 0 {
		t.Fatalf("difference image pixel = %v", pixel)
	}
}

func TestComparePNGDoesNotOverwriteInputs(t *testing.T) {
	directory := t.TempDir()
	oracle := filepath.Join(directory, "oracle.png")
	candidate := filepath.Join(directory, "candidate.png")
	writePNG(t, oracle, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	writePNG(t, candidate, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	original, err := os.ReadFile(oracle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ComparePNG(oracle, candidate, oracle); err == nil {
		t.Fatal("accepted an input as the difference image")
	}
	after, err := os.ReadFile(oracle)
	if err != nil || string(after) != string(original) {
		t.Fatalf("oracle changed: %v", err)
	}
}

func TestComparePNGRejectsOversizedHeader(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "large.png")
	data := []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 'I', 'H', 'D', 'R', 0, 1, 0, 0, 0, 1, 0, 0, 8, 2, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ComparePNG(path, path, ""); err == nil || !strings.Contains(err.Error(), "ceiling") {
		t.Fatalf("oversized header error: %v", err)
	}
}

func TestComparePNGRejectsDirectory(t *testing.T) {
	directory := t.TempDir()
	if _, err := ComparePNG(directory, directory, ""); err == nil || !strings.Contains(err.Error(), "regular") {
		t.Fatalf("directory error: %v", err)
	}
}

func TestComparePNGRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.png")
	bad := filepath.Join(dir, "bad.png")
	writePNG(t, valid, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err := os.WriteFile(bad, []byte("not png"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, paths := range [][2]string{{filepath.Join(dir, "missing.png"), valid}, {valid, bad}} {
		if _, err := ComparePNG(paths[0], paths[1], ""); err == nil {
			t.Fatalf("ComparePNG(%q, %q) succeeded", paths[0], paths[1])
		}
	}
}

func TestCompareRaw(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.pfm")
	b := filepath.Join(dir, "b.pfm")
	first := frame.New(2, 1)
	first.Set(0, 0, frame.Pixel{R: 1, G: 2, B: 3})
	first.Set(1, 0, frame.Pixel{R: -1, G: 0, B: 0})
	second := frame.New(2, 1)
	second.Set(0, 0, frame.Pixel{R: 1.5, G: 1, B: 3})
	second.Set(1, 0, frame.Pixel{R: -1, G: 0, B: 4})
	if err := raw.Write(a, first); err != nil {
		t.Fatal(err)
	}
	if err := raw.Write(b, second); err != nil {
		t.Fatal(err)
	}
	result, err := CompareRaw(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 2 || result.Height != 1 || result.Pixels != 2 {
		t.Fatalf("raw result dimensions = %dx%d, pixels %d", result.Width, result.Height, result.Pixels)
	}
	if result.MaxDifference != [3]float64{0.5, 1, 4} {
		t.Fatalf("raw max difference = %v", result.MaxDifference)
	}
	if result.MaxAt != [3]image.Point{{X: 0, Y: 0}, {X: 0, Y: 0}, {X: 1, Y: 0}} {
		t.Fatalf("raw maximum locations = %v", result.MaxAt)
	}
	if math.Abs(result.PSNR-10*math.Log10(16/(17.25/6))) > 1e-12 {
		t.Fatalf("raw PSNR = %g", result.PSNR)
	}
	if result.FractionWithinOne != 0 || result.Histogram[255] != 1 || result.Difference.RGBAAt(1, 0).B != 255 {
		t.Fatalf("raw difference metrics: within=%v histogram=%v difference=%v", result.FractionWithinOne, result.Histogram[255], result.Difference.RGBAAt(1, 0))
	}
}

func TestCompareRawReadsBigEndianAndScale(t *testing.T) {
	directory := t.TempDir()
	oracle := filepath.Join(directory, "oracle.pfm")
	candidate := filepath.Join(directory, "candidate.pfm")
	contents := []byte("PF\n1 2\n2\n")
	for _, value := range []float32{1, 2, 3, 4, 5, 6} {
		contents = binary.BigEndian.AppendUint32(contents, math.Float32bits(value))
	}
	if err := os.WriteFile(oracle, contents, 0600); err != nil {
		t.Fatal(err)
	}
	wanted := frame.New(1, 2)
	wanted.Set(0, 0, frame.Pixel{R: 8, G: 10, B: 12})
	wanted.Set(0, 1, frame.Pixel{R: 2, G: 4, B: 6})
	if err := raw.Write(candidate, wanted); err != nil {
		t.Fatal(err)
	}
	result, err := CompareRaw(oracle, candidate)
	if err != nil || !math.IsInf(result.PSNR, 1) {
		t.Fatalf("big-endian scaled PFM did not match: %v, %v", result.PSNR, err)
	}
}

func TestCompareRawIdentical(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "same.pfm")
	if err := raw.Write(path, frame.New(1, 1)); err != nil {
		t.Fatal(err)
	}
	result, err := CompareRaw(path, path)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsInf(result.PSNR, 1) || result.Pixels != 1 {
		t.Fatalf("identical raw metrics = PSNR %g, pixels %d", result.PSNR, result.Pixels)
	}
}

func TestCompareRawRejectsDifferentSizes(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.pfm"), filepath.Join(dir, "b.pfm")
	if err := raw.Write(a, frame.New(1, 1)); err != nil {
		t.Fatal(err)
	}
	if err := raw.Write(b, frame.New(2, 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := CompareRaw(a, b); err == nil {
		t.Fatal("expected size mismatch error")
	}
}

func TestCompareRawRejectsInvalidValues(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.pfm"), filepath.Join(dir, "b.pfm")
	if err := raw.Write(a, frame.New(1, 1)); err != nil {
		t.Fatal(err)
	}
	writePFMValues(t, b, 1, 1, []float32{float32(math.NaN()), 0, 0})
	if _, err := CompareRaw(a, b); err == nil {
		t.Fatal("expected non-finite raw value error")
	}
}

func TestCompareRawScalesBeforeNarrowing(t *testing.T) {
	directory := t.TempDir()
	oracle := filepath.Join(directory, "scale.pfm")
	candidate := filepath.Join(directory, "expected.pfm")
	contents := []byte("PF\n1 1\n1e40\n")
	for range 3 {
		contents = binary.BigEndian.AppendUint32(contents, math.Float32bits(1e-40))
	}
	if err := os.WriteFile(oracle, contents, 0600); err != nil {
		t.Fatal(err)
	}
	wanted := frame.New(1, 1)
	value := float32(float64(float32(1e-40)) * 1e40)
	wanted.Set(0, 0, frame.Pixel{R: value, G: value, B: value})
	if err := raw.Write(candidate, wanted); err != nil {
		t.Fatal(err)
	}
	result, err := CompareRaw(oracle, candidate)
	if err != nil || !math.IsInf(result.PSNR, 1) {
		t.Fatalf("scaled raw differs: PSNR=%v error=%v", result.PSNR, err)
	}
}

func TestCompareRawRejectsLargeTruncatedImage(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "short.pfm")
	if err := os.WriteFile(path, []byte("PF\n4096 4096\n-1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompareRaw(path, path); err == nil {
		t.Fatal("accepted a truncated PFM")
	}
}

func TestCompareRawRejectsHugeClaimedImage(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "huge.pfm")
	if err := os.WriteFile(path, []byte("PF\n1000000000 1000000000\n-1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompareRaw(path, path); err == nil {
		t.Fatal("accepted an oversized PFM")
	}
}

func TestCompareRawRejectsOversizedHeader(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "long.pfm")
	contents := []byte("PF\n" + strings.Repeat(" ", 4096) + "1 1\n-1\n")
	contents = append(contents, make([]byte, 12)...)
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompareRaw(path, path); err == nil {
		t.Fatal("accepted an oversized PFM header")
	}
}

func TestCompareRawRejectsMalformedHeaders(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.pfm"), filepath.Join(dir, "b.pfm")
	if err := raw.Write(a, frame.New(1, 1)); err != nil {
		t.Fatal(err)
	}
	for _, malformed := range [][]byte{[]byte("Pf\n1 1\n-1\n"), []byte("PF\n0 1\n-1\n"), []byte("PF\n1 1\n0\n")} {
		if err := os.WriteFile(b, malformed, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := CompareRaw(a, b); err == nil {
			t.Fatalf("accepted malformed PFM %q", malformed)
		}
	}
}

func TestCompareRawRejectsTruncatedData(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.pfm"), filepath.Join(dir, "b.pfm")
	if err := raw.Write(a, frame.New(1, 1)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("PF\n1 1\n-1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompareRaw(a, b); err == nil {
		t.Fatal("expected truncated raw error")
	}
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writePFMValues(t *testing.T, path string, width, height int, values []float32) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(file, "PF\n%d %d\n-1\n", width, height); err != nil {
		file.Close()
		t.Fatal(err)
	}
	for _, value := range values {
		if err := binary.Write(file, binary.LittleEndian, value); err != nil {
			file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestCompareImagesRefusesNil checks that a nil image on either side is an
// error and not a panic.
func TestCompareImagesRefusesNil(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	for _, pair := range [][2]image.Image{{nil, img}, {img, nil}} {
		if _, err := CompareImages(pair[0], pair[1]); err == nil || !strings.Contains(err.Error(), "nil image") {
			t.Errorf("nil image error: %v", err)
		}
	}
}

// TestComparePNGRefusesDifferentSizes checks that two readable PNGs of
// different sizes are refused by name, and that no difference file is made.
func TestComparePNGRefusesDifferentSizes(t *testing.T) {
	dir := t.TempDir()
	one, two, out := filepath.Join(dir, "one.png"), filepath.Join(dir, "two.png"), filepath.Join(dir, "diff.png")
	writePNG(t, one, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	writePNG(t, two, image.NewRGBA(image.Rect(0, 0, 2, 1)))
	if _, err := ComparePNG(one, two, out); err == nil || !strings.Contains(err.Error(), "sizes differ: 1x1 and 2x1") {
		t.Fatalf("different sizes error: %v", err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a difference file was written for a refused comparison")
	}
}

// TestComparePNGRefusesABodyThatDoesNotDecode checks a file whose header
// reads but whose pixels are missing.
func TestComparePNGRefusesABodyThatDoesNotDecode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "headless.png")
	data := []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ComparePNG(path, path, ""); err == nil || !strings.Contains(err.Error(), "decode PNG") {
		t.Fatalf("body error: %v", err)
	}
}

// TestCompareRawRefusesAMissingFile checks that a PFM that is not there is
// refused by name, on either side.
func TestCompareRawRefusesAMissingFile(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.pfm")
	missing := filepath.Join(dir, "missing.pfm")
	if err := raw.Write(good, frame.New(1, 1)); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{missing, good}, {good, missing}} {
		if _, err := CompareRaw(pair[0], pair[1]); err == nil || !strings.Contains(err.Error(), "open raw") {
			t.Errorf("CompareRaw(%q, %q): %v", pair[0], pair[1], err)
		}
	}
}

// TestPSNR checks the three cases of the ratio, with a hand-worked value:
// a peak of 10 and a mean square of 4 is 10 log10(100/4), about 13.98 dB.
func TestPSNR(t *testing.T) {
	if got := psnr(10, 0, 3); !math.IsInf(got, 1) {
		t.Errorf("no difference = %g, want +Inf", got)
	}
	if got := psnr(0, 5, 3); !math.IsInf(got, -1) {
		t.Errorf("zero peak = %g, want -Inf", got)
	}
	if got := psnr(10, 12, 3); math.Abs(got-13.9794) > 1e-4 {
		t.Errorf("psnr = %g, want 13.9794", got)
	}
}
