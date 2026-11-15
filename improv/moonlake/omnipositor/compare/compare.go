package compare

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"syscall"

	"github.com/janearc/slayer/render/frame"
)

const pixelLimit = 4096 * 4096

type Result struct {
	Width             int
	Height            int
	Pixels            int
	MaxDifference     [3]uint8
	FractionWithinOne float64
	PSNR              float64
	Histogram         [256]uint64
	Difference        *image.RGBA
}

type RawResult struct {
	Width             int
	Height            int
	Pixels            int
	MaxDifference     [3]float64
	MaxAt             [3]image.Point
	FractionWithinOne float64
	PSNR              float64
	Histogram         [256]uint64
	Difference        *image.RGBA
}

// CompareImages compares two images of the same size, red, green and blue
// only, as 8-bit values. It refuses nil images, differing sizes and more than
// 16777216 pixels. The result has the largest difference per channel, the
// fraction of pixels within one code value in every channel, PSNR against a
// peak of 255, a histogram of the differences, and a difference image scaled
// so the largest difference is 255.
func CompareImages(oracle, candidate image.Image) (Result, error) {
	if oracle == nil || candidate == nil {
		return Result{}, errors.New("compare: nil image")
	}
	bounds := oracle.Bounds()
	other := candidate.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 1 || height < 1 || width != other.Dx() || height != other.Dy() {
		return Result{}, fmt.Errorf("compare: image sizes differ: %dx%d and %dx%d", width, height, other.Dx(), other.Dy())
	}
	if height > pixelLimit || width > pixelLimit/height {
		return Result{}, errors.New("compare: image exceeds the image ceiling")
	}

	result := Result{Width: width, Height: height, Pixels: width * height, Difference: image.NewRGBA(image.Rect(0, 0, width, height))}
	var squared float64
	within := 0
	maxAbs := uint8(0)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			a := color.NRGBAModel.Convert(oracle.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
			b := color.NRGBAModel.Convert(candidate.At(other.Min.X+x, other.Min.Y+y)).(color.NRGBA)
			av := [3]uint8{a.R, a.G, a.B}
			bv := [3]uint8{b.R, b.G, b.B}
			pixelWithin := true
			var diff [3]uint8
			for channel := range av {
				d := int(av[channel]) - int(bv[channel])
				if d < 0 {
					d = -d
				}
				diff[channel] = uint8(d)
				result.Histogram[d]++
				squared += float64(d * d)
				if diff[channel] > result.MaxDifference[channel] {
					result.MaxDifference[channel] = diff[channel]
				}
				if diff[channel] > 1 {
					pixelWithin = false
				}
				if diff[channel] > maxAbs {
					maxAbs = diff[channel]
				}
			}
			if pixelWithin {
				within++
			}
			result.Difference.SetRGBA(x, y, color.RGBA{R: diff[0], G: diff[1], B: diff[2], A: 255})
		}
	}
	result.FractionWithinOne = float64(within) / float64(result.Pixels)
	result.PSNR = psnr(255, squared, float64(result.Pixels*3))
	if maxAbs > 0 {
		for i, pixel := range result.Difference.Pix {
			if i%4 == 3 {
				continue
			}
			result.Difference.Pix[i] = uint8(uint16(pixel) * 255 / uint16(maxAbs))
		}
	}
	return result, nil
}

// ComparePNG reads two PNG files and compares them with CompareImages. If
// differencePath is not empty it writes the difference image there as a new
// file, refusing one that exists and removing it if writing fails.
func ComparePNG(oraclePath, candidatePath, differencePath string) (Result, error) {
	oracle, err := readPNG(oraclePath)
	if err != nil {
		return Result{}, err
	}
	candidate, err := readPNG(candidatePath)
	if err != nil {
		return Result{}, err
	}
	result, err := CompareImages(oracle, candidate)
	if err != nil {
		return Result{}, err
	}
	if differencePath != "" {
		file, err := os.OpenFile(differencePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return Result{}, fmt.Errorf("compare: create difference image: %w", err)
		}
		if err := png.Encode(file, result.Difference); err != nil {
			closeErr := file.Close()
			removeErr := os.Remove(differencePath)
			return Result{}, fmt.Errorf("compare: encode difference image: %w", errors.Join(err, closeErr, removeErr))
		}
		if err := file.Close(); err != nil {
			removeErr := os.Remove(differencePath)
			return Result{}, fmt.Errorf("compare: close difference image: %w", errors.Join(err, removeErr))
		}
	}
	return result, nil
}

// CompareRaw compares two colour PFM files of the same size, in float values,
// and refuses samples that are not finite. MaxAt gives the pixel of each
// channel's largest difference. PSNR takes the largest absolute sample as the
// peak, within one means within a 255th of it, and the histogram and the
// difference image are scaled to the largest difference.
func CompareRaw(oraclePath, candidatePath string) (RawResult, error) {
	oracle, err := readRawFile(oraclePath)
	if err != nil {
		return RawResult{}, err
	}
	candidate, err := readRawFile(candidatePath)
	if err != nil {
		return RawResult{}, err
	}
	if oracle.W != candidate.W || oracle.H != candidate.H {
		return RawResult{}, fmt.Errorf("compare: raw sizes differ: %dx%d and %dx%d", oracle.W, oracle.H, candidate.W, candidate.H)
	}
	result := RawResult{Width: oracle.W, Height: oracle.H, Pixels: oracle.W * oracle.H}
	result.Difference = image.NewRGBA(image.Rect(0, 0, oracle.W, oracle.H))
	var squared, peak float64
	for i := range oracle.Pix {
		a := [3]float64{float64(oracle.Pix[i].R), float64(oracle.Pix[i].G), float64(oracle.Pix[i].B)}
		b := [3]float64{float64(candidate.Pix[i].R), float64(candidate.Pix[i].G), float64(candidate.Pix[i].B)}
		for channel := range a {
			if math.IsNaN(a[channel]) || math.IsInf(a[channel], 0) || math.IsNaN(b[channel]) || math.IsInf(b[channel], 0) {
				return RawResult{}, fmt.Errorf("compare: non-finite raw sample at pixel %d channel %d", i, channel)
			}
			if math.Abs(a[channel]) > peak {
				peak = math.Abs(a[channel])
			}
			if math.Abs(b[channel]) > peak {
				peak = math.Abs(b[channel])
			}
			difference := math.Abs(a[channel] - b[channel])
			if difference > result.MaxDifference[channel] {
				result.MaxDifference[channel] = difference
				result.MaxAt[channel] = image.Pt(i%oracle.W, i/oracle.W)
			}
			squared += difference * difference
		}
	}
	result.PSNR = psnr(peak, squared, float64(result.Pixels*3))
	maximum := math.Max(result.MaxDifference[0], math.Max(result.MaxDifference[1], result.MaxDifference[2]))
	within := 0
	for index := range oracle.Pix {
		first := [3]float64{float64(oracle.Pix[index].R), float64(oracle.Pix[index].G), float64(oracle.Pix[index].B)}
		second := [3]float64{float64(candidate.Pix[index].R), float64(candidate.Pix[index].G), float64(candidate.Pix[index].B)}
		var scaled [3]uint8
		pixelWithin := true
		for channel := range first {
			difference := math.Abs(first[channel] - second[channel])
			if maximum > 0 {
				scaled[channel] = uint8(math.Round(255 * difference / maximum))
			}
			result.Histogram[scaled[channel]]++
			if difference > peak/255 {
				pixelWithin = false
			}
		}
		if pixelWithin {
			within++
		}
		result.Difference.SetRGBA(index%oracle.W, index/oracle.W, color.RGBA{R: scaled[0], G: scaled[1], B: scaled[2], A: 255})
	}
	result.FractionWithinOne = float64(within) / float64(result.Pixels)
	return result, nil
}

// readRawFile reads a colour PFM file into a frame. It needs the PF marker,
// a size and a scale; a negative scale means little-endian samples, and each
// sample is multiplied by the size of the scale. Rows are stored bottom first.
// It refuses files that are not regular, too big or short of pixels.
func readRawFile(path string) (*frame.Frame, error) {
	file, err := openRegular(path)
	if err != nil {
		return nil, fmt.Errorf("compare: open raw %q: %w", path, err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for line := 0; line < 3; line++ {
		if _, err := reader.ReadSlice('\n'); err != nil {
			return nil, fmt.Errorf("compare: raw %q has an invalid or oversized header: %w", path, err)
		}
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("compare: inspect raw %q: %w", path, err)
	}
	reader.Reset(file)
	var marker string
	if _, err := fmt.Fscanln(reader, &marker); err != nil || marker != "PF" {
		return nil, fmt.Errorf("compare: raw %q is not a colour PFM", path)
	}
	var width, height int
	if _, err := fmt.Fscanln(reader, &width, &height); err != nil || width < 1 || height < 1 {
		return nil, fmt.Errorf("compare: raw %q has invalid dimensions", path)
	}
	var scale float64
	if _, err := fmt.Fscanln(reader, &scale); err != nil || scale == 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return nil, fmt.Errorf("compare: raw %q has invalid scale", path)
	}
	factor := math.Abs(scale)
	if height > pixelLimit || width > pixelLimit/height {
		return nil, fmt.Errorf("compare: raw %q exceeds the image ceiling", path)
	}
	position, err := file.Seek(0, 1)
	if err != nil {
		return nil, fmt.Errorf("compare: inspect raw %q: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("compare: inspect raw %q: %w", path, err)
	}
	headerBytes := position - int64(reader.Buffered())
	if !info.Mode().IsRegular() || info.Size()-headerBytes < int64(width)*int64(height)*12 {
		return nil, fmt.Errorf("compare: raw %q ends before its pixels do", path)
	}
	var order binary.ByteOrder = binary.BigEndian
	if scale < 0 {
		order = binary.LittleEndian
	}
	result := frame.New(width, height)
	row := make([]byte, width*12)
	for y := height - 1; y >= 0; y-- {
		if _, err := io.ReadFull(reader, row); err != nil {
			return nil, fmt.Errorf("compare: raw %q ends before its pixels do: %w", path, err)
		}
		for x := 0; x < width; x++ {
			index := x * 12
			result.Set(x, y, frame.Pixel{
				R: float32(float64(math.Float32frombits(order.Uint32(row[index:]))) * factor),
				G: float32(float64(math.Float32frombits(order.Uint32(row[index+4:]))) * factor),
				B: float32(float64(math.Float32frombits(order.Uint32(row[index+8:]))) * factor),
			})
		}
	}
	return result, nil
}

// readPNG decodes the PNG file at path, which must be regular. It checks the
// size in the header against the pixel limit before decoding the pixels.
func readPNG(path string) (image.Image, error) {
	file, err := openRegular(path)
	if err != nil {
		return nil, fmt.Errorf("compare: open PNG %q: %w", path, err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("compare: decode PNG %q: %w", path, err)
	}
	if config.Width < 1 || config.Height < 1 || config.Height > pixelLimit || config.Width > pixelLimit/config.Height {
		return nil, fmt.Errorf("compare: PNG %q exceeds the image ceiling", path)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("compare: seek PNG %q: %w", path, err)
	}
	img, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("compare: decode PNG %q: %w", path, err)
	}
	return img, nil
}

// openRegular opens path for reading without blocking, so a pipe cannot stall
// it, and returns it if it is a regular file. Otherwise it closes it and fails.
func openRegular(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, errors.New("not a regular file")
	}
	return file, nil
}

// psnr is the peak signal to noise ratio in dB for squared, the sum of squared
// differences over samples samples, against peak. It is positive infinity when
// there is no difference, and negative infinity when peak is zero.
func psnr(peak, squared, samples float64) float64 {
	if squared == 0 {
		return math.Inf(1)
	}
	if peak == 0 {
		return math.Inf(-1)
	}
	return 10 * math.Log10(peak*peak/(squared/samples))
}
