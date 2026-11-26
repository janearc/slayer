// Package regionref verifies a retained reference's identity and linear light.
package regionref

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/render/frame"
)

// Identity is the caller's digest of the reference's complete provenance.
type Identity [32]byte

// Read verifies and decodes one reference without exceeding maxBytes.
func Read(path string, identity Identity, width, height int, maxBytes int64) (*frame.Frame, error) {
	size, err := sizeOf(width, height)
	if err != nil || maxBytes < int64(size) {
		return nil, fmt.Errorf("regionref: invalid dimensions or size limit")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() != int64(size) {
		return nil, fmt.Errorf("regionref: reference size differs")
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, err
	}
	var extra [1]byte
	if n, err := f.Read(extra[:]); n != 0 || err != io.EOF {
		return nil, fmt.Errorf("regionref: reference has extra data")
	}
	if string(data[:8]) != "REGREF01" || string(data[8:40]) != string(identity[:]) ||
		binary.LittleEndian.Uint32(data[72:76]) != uint32(width) ||
		binary.LittleEndian.Uint32(data[76:80]) != uint32(height) {
		return nil, fmt.Errorf("regionref: identity or dimensions differ")
	}
	light := data[80:]
	digest := sha256.Sum256(light)
	if string(data[40:72]) != string(digest[:]) {
		return nil, fmt.Errorf("regionref: content digest differs")
	}
	return decode(light, width, height)
}

// Publish installs a validated reference without replacing any existing path.
// Atomic visibility does not guarantee durability across a system failure.
func Publish(path string, identity Identity, width, height int, light []byte) (result error) {
	size, err := sizeOf(width, height)
	if err != nil || len(light) != size-80 {
		return fmt.Errorf("regionref: invalid dimensions or light length")
	}
	if _, err := decode(light, width, height); err != nil {
		return err
	}
	data := make([]byte, size)
	copy(data[:8], "REGREF01")
	copy(data[8:40], identity[:])
	digest := sha256.Sum256(light)
	copy(data[40:72], digest[:])
	binary.LittleEndian.PutUint32(data[72:76], uint32(width))
	binary.LittleEndian.PutUint32(data[76:80], uint32(height))
	copy(data[80:], light)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".regionref-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(tmp.Name()); err != nil {
			result = errors.Join(result, fmt.Errorf("regionref: remove temporary file (destination may be installed): %w", err))
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		return fmt.Errorf("regionref: install without replacement: %w", err)
	}
	return nil
}

// sizeOf checks dimensions and computes the encoded reference size.
func sizeOf(width, height int) (int, error) {
	if width <= 0 || height <= 0 || uint64(width) > math.MaxUint32 ||
		uint64(height) > math.MaxUint32 || width > (math.MaxInt-80)/12/height ||
		width > (math.MaxInt/20)/height {
		return 0, fmt.Errorf("regionref: invalid dimensions")
	}
	return 80 + width*height*12, nil
}

// decode reads finite linear-light channels into a frame.
func decode(light []byte, width, height int) (*frame.Frame, error) {
	made := frame.New(width, height)
	for i := range made.Pix {
		at := i * 12
		r := math.Float32frombits(binary.LittleEndian.Uint32(light[at:]))
		g := math.Float32frombits(binary.LittleEndian.Uint32(light[at+4:]))
		b := math.Float32frombits(binary.LittleEndian.Uint32(light[at+8:]))
		for _, value := range []float32{r, g, b} {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return nil, fmt.Errorf("regionref: nonfinite light at pixel %d", i)
			}
		}
		made.Pix[i].R, made.Pix[i].G, made.Pix[i].B = r, g, b
	}
	return made, nil
}
