package census

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/janearc/slayer/render/frame"
)

// ReadFrame reads the census's linear RGB PFM and its separate coverage/depth
// sidecar. Both artifacts are required; a missing or damaged sidecar must not
// be mistaken for a valid image with default coverage and depth.
func ReadFrame(pfmPath, detailPath string) (*frame.Frame, error) {
	light, err := os.Open(pfmPath)
	if err != nil {
		return nil, fmt.Errorf("open linear light: %w", err)
	}
	defer light.Close()
	detail, err := os.Open(detailPath)
	if err != nil {
		return nil, fmt.Errorf("open coverage/depth: %w", err)
	}
	defer detail.Close()
	return readFrame(light, detail)
}

// readFrame accepts exactly the census writer's PFM dialect: little-endian,
// unit scale and bottom row first. The sidecar is top row first, two floats
// per pixel. Size limits match the writer's documented dimensions.
func readFrame(light, detail *os.File) (*frame.Frame, error) {
	in := bufio.NewReader(light)
	var magic string
	var width, height int
	var scale float64
	if _, err := fmt.Fscanln(in, &magic); err != nil || magic != "PF" {
		return nil, fmt.Errorf("invalid PFM magic")
	}
	if _, err := fmt.Fscanln(in, &width, &height); err != nil || width < 1 || width > 8192 || height < 1 || height > 8192 {
		return nil, fmt.Errorf("invalid PFM dimensions")
	}
	if _, err := fmt.Fscanln(in, &scale); err != nil || scale != -1 {
		return nil, fmt.Errorf("PFM must use little-endian unit scale")
	}
	// Check the actual files before allocation: a corrupt header naming an
	// 8192-square image must not allocate its full frame from a tiny file.
	position, err := light.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("inspect PFM: %w", err)
	}
	info, err := light.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect PFM: %w", err)
	}
	expected := int64(width) * int64(height)
	if info.Size()-(position-int64(in.Buffered())) != expected*12 {
		return nil, fmt.Errorf("PFM payload has the wrong length")
	}
	info, err = detail.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect coverage/depth: %w", err)
	}
	if info.Size() != expected*8 {
		return nil, fmt.Errorf("coverage/depth payload has the wrong length")
	}
	made := frame.New(width, height)
	row := make([]byte, width*12)
	for y := height - 1; y >= 0; y-- {
		if _, err := io.ReadFull(in, row); err != nil {
			return nil, fmt.Errorf("truncated PFM: %w", err)
		}
		for x := 0; x < width; x++ {
			i := x * 12
			p := &made.Pix[y*width+x]
			p.R = math.Float32frombits(binary.LittleEndian.Uint32(row[i:]))
			p.G = math.Float32frombits(binary.LittleEndian.Uint32(row[i+4:]))
			p.B = math.Float32frombits(binary.LittleEndian.Uint32(row[i+8:]))
		}
	}
	if err := noMore(in); err != nil {
		return nil, fmt.Errorf("PFM: %w", err)
	}
	side := bufio.NewReader(detail)
	row = make([]byte, width*8)
	for y := 0; y < height; y++ {
		if _, err := io.ReadFull(side, row); err != nil {
			return nil, fmt.Errorf("truncated coverage/depth: %w", err)
		}
		for x := 0; x < width; x++ {
			i := x * 8
			p := &made.Pix[y*width+x]
			p.Cover = math.Float32frombits(binary.LittleEndian.Uint32(row[i:]))
			p.Depth = math.Float32frombits(binary.LittleEndian.Uint32(row[i+4:]))
			if err := validPixel(*p); err != nil {
				return nil, fmt.Errorf("pixel %d,%d: %w", x, y, err)
			}
		}
	}
	if err := noMore(side); err != nil {
		return nil, fmt.Errorf("coverage/depth: %w", err)
	}
	return made, nil
}

// noMore reports an error when r still holds a byte past what was read.
func noMore(r io.Reader) error {
	var extra [1]byte
	n, err := r.Read(extra[:])
	if n > 0 {
		return fmt.Errorf("unexpected trailing data")
	}
	if err != io.EOF {
		return fmt.Errorf("cannot check end of file: %w", err)
	}
	return nil
}
