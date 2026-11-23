package lights

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

// Read decodes the colour PFM file at path and returns the image and the scale
// from its header. A failure to close the file is added to any other error.
func Read(path string) (image Image, scale float64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return Image{}, 0, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return DecodePFM(file)
}

// DecodePFM decodes a colour PFM from reader and returns the image and the
// scale from its header. Rows are turned so the first row is the top. Any
// size is accepted.
func DecodePFM(reader io.Reader) (Image, float64, error) {
	return decodePFM(reader, 0, 0)
}

// DecodePFMExpected is DecodePFM for a file that must be exactly width by
// height, with a scale that is not zero, a header of three lines within 4096
// bytes, and no bytes after the pixels.
func DecodePFMExpected(reader io.Reader, width, height int) (Image, float64, error) {
	if !dimensions(width, height) {
		return Image{}, 0, fmt.Errorf("invalid expected PFM dimensions")
	}
	return decodePFM(reader, width, height)
}

// decodePFM does the work of both decoders. Expected sizes of 0 mean any
// size. The header must say PF, and the scale must end in a newline. The
// scale's sign picks the byte order of the float32 pixels, which are stored
// bottom row first and widened to float64.
func decodePFM(reader io.Reader, expectedWidth, expectedHeight int) (Image, float64, error) {
	in := bufio.NewReader(reader)
	var header strings.Builder
	if expectedWidth != 0 {
		for lines := 0; lines < 3; lines++ {
			line, err := in.ReadSlice('\n')
			if err != nil {
				return Image{}, 0, err
			}
			if header.Len()+len(line) > 4096 {
				return Image{}, 0, fmt.Errorf("PFM header exceeds 4096 bytes")
			}
			header.Write(line)
		}
	}
	var scanner io.Reader = in
	if expectedWidth != 0 {
		scanner = strings.NewReader(header.String())
	}
	var kind string
	var width, height int
	var scale float64
	if _, err := fmt.Fscan(scanner, &kind, &width, &height, &scale); err != nil {
		return Image{}, 0, err
	}
	if kind != "PF" || !dimensions(width, height) || !finite(scale) {
		return Image{}, 0, fmt.Errorf("invalid colour pfm header")
	}
	if expectedWidth != 0 && (width != expectedWidth || height != expectedHeight || scale == 0) {
		return Image{}, 0, fmt.Errorf("PFM dimensions %dx%d, expected %dx%d colour PF with nonzero scale", width, height, expectedWidth, expectedHeight)
	}
	separatorReader := io.ByteReader(in)
	if expectedWidth != 0 {
		separatorReader = scanner.(*strings.Reader)
	}
	separator, err := separatorReader.ReadByte()
	if err != nil {
		return Image{}, 0, err
	}
	if separator == '\r' {
		separator, err = separatorReader.ReadByte()
		if err != nil {
			return Image{}, 0, err
		}
	}
	if separator != '\n' {
		return Image{}, 0, fmt.Errorf("pfm scale must end with a newline")
	}
	if expectedWidth != 0 && scanner.(*strings.Reader).Len() != 0 {
		return Image{}, 0, fmt.Errorf("invalid PFM header: extra header bytes")
	}
	size := width * height * 12
	raw, err := io.ReadAll(io.LimitReader(in, int64(size)))
	if err != nil {
		return Image{}, 0, err
	}
	if len(raw) != size {
		if len(raw) == 0 {
			return Image{}, 0, io.EOF
		}
		return Image{}, 0, io.ErrUnexpectedEOF
	}
	if expectedWidth != 0 {
		if _, err := in.ReadByte(); err != io.EOF {
			if err != nil {
				return Image{}, 0, err
			}
			return Image{}, 0, fmt.Errorf("PFM has excess pixel bytes")
		}
	}
	image := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	order := byteOrder(scale)
	for y := range height {
		from := (height - 1 - y) * width * 12
		for i := range width * 3 {
			image.Light[y*width*3+i] = float64(math.Float32frombits(order.Uint32(raw[from+4*i:])))
		}
	}
	return image, scale, nil
}

// Write encodes image as a colour PFM into the file at path, creating or
// replacing it. The image and scale are checked before the file is created. A
// failure to close the file is added to any other error.
func Write(path string, image Image, scale float64) (err error) {
	if err := validatePFM(image, scale); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return encodePFM(file, image, scale)
}

// encodePFM writes image to writer as a colour PFM, bottom row first, as
// float32. The byte order follows the sign of scale, as byteOrder says.
func encodePFM(writer io.Writer, image Image, scale float64) error {
	if err := validatePFM(image, scale); err != nil {
		return err
	}
	out := bufio.NewWriter(writer)
	if _, err := fmt.Fprintf(out, "PF\n%d %d\n%g\n", image.Width, image.Height, scale); err != nil {
		return err
	}
	row := make([]float32, image.Width*3)
	for y := image.Height - 1; y >= 0; y-- {
		for i := range row {
			row[i] = float32(image.Light[y*image.Width*3+i])
		}
		if err := binary.Write(out, byteOrder(scale), row); err != nil {
			return err
		}
	}
	return out.Flush()
}

// validatePFM refuses an image that is not valid or a scale that is not
// finite.
func validatePFM(image Image, scale float64) error {
	if err := image.validate(); err != nil {
		return err
	}
	if !finite(scale) {
		return fmt.Errorf("nonfinite pfm scale")
	}
	return nil
}

// finite reports whether value is neither not-a-number nor infinite.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// byteOrder is the byte order a PFM scale picks: big-endian when it is
// positive, little-endian when it is zero or negative.
func byteOrder(scale float64) binary.ByteOrder {
	if scale > 0 {
		return binary.BigEndian
	}
	return binary.LittleEndian
}
