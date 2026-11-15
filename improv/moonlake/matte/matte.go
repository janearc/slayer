package matte

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

type Image struct {
	Width, Height int
	Scale         float64
	Pixels        []float32
}

type Config struct {
	Top, Bottom, Shore, MoonRadius float64
	MoonRef, WaterMask, Front      *Image
}

const mostPixels = 16777216

// dimensions is true when width and height are positive, the picture holds
// no more than mostPixels pixels, and twelve bytes a pixel fits in an int.
func dimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= mostPixels/height && width <= int(^uint(0)>>1)/height/12
}

// finite is true when value is neither NaN nor an infinity.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// valid is true when image has good dimensions, a finite scale, three
// finite values per pixel, and no other length of pixel data.
func valid(image Image) bool {
	if !dimensions(image.Width, image.Height) || !finite(image.Scale) || len(image.Pixels) != image.Width*image.Height*3 {
		return false
	}
	for _, value := range image.Pixels {
		if !finite(float64(value)) {
			return false
		}
	}
	return true
}

// Apply blends sea into raw from the shore row down, easing over a few rows
// either side of it, or by the WaterMask where there is one. The sea is
// scaled by the moon's mean luminance times its disc area, MoonRadius being
// in degrees. The moon is taken from MoonRef, else from raw. Front is added
// last. The raw, sea, and any masks must be valid and the same size.
func Apply(raw, sea Image, config Config) (Image, error) {
	if !finite(config.Top) || !finite(config.Bottom) || !finite(config.Shore) || !finite(config.MoonRadius) {
		return Image{}, fmt.Errorf("nonfinite matte configuration")
	}
	if !valid(raw) || !valid(sea) {
		return Image{}, fmt.Errorf("invalid raw or sea image")
	}
	width, height := raw.Width, raw.Height
	if sea.Width != width || sea.Height != height {
		return Image{}, fmt.Errorf("the raw is %dx%d and the sea %dx%d", width, height, sea.Width, sea.Height)
	}
	moon := brightest(raw.Pixels)
	if config.MoonRef != nil {
		if !valid(*config.MoonRef) {
			return Image{}, fmt.Errorf("invalid moon reference image")
		}
		moon = brightest(config.MoonRef.Pixels)
	}
	disc := math.Pi * math.Pow(config.MoonRadius*math.Pi/180, 2)
	shoreRow := (config.Top - config.Shore) / (config.Top - config.Bottom) * float64(height)
	const feather = 4.0
	var mask []float32
	most := float32(0)
	if config.WaterMask != nil {
		if !valid(*config.WaterMask) || config.WaterMask.Width != width || config.WaterMask.Height != height {
			return Image{}, fmt.Errorf("invalid water mask image")
		}
		mask = config.WaterMask.Pixels
		for _, v := range mask {
			most = max(most, v)
		}
	}
	out := append([]float32(nil), raw.Pixels...)
	for r := 0; r < height; r++ {
		w := math.Max(0, math.Min(1, (float64(r)-shoreRow+feather)/(2*feather)))
		if w == 0 && mask == nil {
			continue
		}
		for x := 0; x < width; x++ {
			i := ((height-1-r)*width + x) * 3
			if mask != nil {
				w = math.Min(1, float64(mask[i]+mask[i+1]+mask[i+2])/float64(3*most)*1.25)
			}
			lum := (moon[0] + moon[1] + moon[2]) / 3
			for c := range 3 {
				water := float64(sea.Pixels[i+c]) * lum * disc
				out[i+c] = float32((1-w)*float64(raw.Pixels[i+c]) + w*water)
			}
		}
	}
	if config.Front != nil {
		if !valid(*config.Front) || config.Front.Width != width || config.Front.Height != height {
			return Image{}, fmt.Errorf("invalid front image")
		}
		for i := range out {
			out[i] += config.Front.Pixels[i]
		}
	}
	return Image{Width: width, Height: height, Scale: raw.Scale, Pixels: out}, nil
}

// brightest is the average colour of the pixels whose channel sum is at
// least half that of the brightest pixel.
func brightest(light []float32) [3]float64 {
	most := 0.0
	for i := 0; i < len(light); i += 3 {
		most = math.Max(most, float64(light[i]+light[i+1]+light[i+2]))
	}
	var sum [3]float64
	count := 0.0
	for i := 0; i < len(light); i += 3 {
		if float64(light[i]+light[i+1]+light[i+2]) < most/2 {
			continue
		}
		for c := range 3 {
			sum[c] += float64(light[i+c])
		}
		count++
	}
	for c := range sum {
		sum[c] /= count
	}
	return sum
}

// Read reads the colour PFM at path. It refuses a file that is not a regular
// file, that is not "PF", that has bad dimensions or a nonfinite scale, or
// that holds fewer pixels than the header says. It does not check the pixel
// values themselves.
func Read(path string) (result Image, err error) {
	file, err := os.Open(path)
	if err != nil {
		return Image{}, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	in := bufio.NewReader(file)
	var kind string
	var image Image
	if _, err := fmt.Fscan(in, &kind, &image.Width, &image.Height, &image.Scale); err != nil {
		return Image{}, err
	}
	if _, err := in.ReadByte(); err != nil {
		return Image{}, err
	}
	if kind != "PF" {
		return Image{}, fmt.Errorf("%s is not a colour pfm", path)
	}
	if !dimensions(image.Width, image.Height) {
		return Image{}, fmt.Errorf("%s has invalid dimensions", path)
	}
	if !finite(image.Scale) {
		return Image{}, fmt.Errorf("%s has nonfinite scale", path)
	}
	info, err := file.Stat()
	if err != nil {
		return Image{}, err
	}
	if !info.Mode().IsRegular() {
		return Image{}, fmt.Errorf("%s is not a regular pfm file", path)
	}
	position, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return Image{}, err
	}
	remaining := info.Size() - position + int64(in.Buffered())
	if remaining < int64(image.Width)*int64(image.Height)*12 {
		if remaining <= 0 {
			return Image{}, io.EOF
		}
		return Image{}, io.ErrUnexpectedEOF
	}
	image.Pixels = make([]float32, image.Width*image.Height*3)
	err = binary.Read(in, order(image.Scale), image.Pixels)
	return image, err
}

// order is the byte order a PFM with this scale holds: big-endian if the
// scale is positive, little-endian otherwise.
func order(scale float64) binary.ByteOrder {
	if scale > 0 {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// Write writes image to path as a colour PFM, with the scale as the header
// line and its sign choosing the byte order. It refuses an invalid image.
func Write(path string, image Image) (err error) {
	if !valid(image) {
		return fmt.Errorf("invalid output image")
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	out := bufio.NewWriter(file)
	if _, err := fmt.Fprintf(out, "PF\n%d %d\n%g\n", image.Width, image.Height, image.Scale); err != nil {
		return err
	}
	if err := binary.Write(out, order(image.Scale), image.Pixels); err != nil {
		return err
	}
	return out.Flush()
}
