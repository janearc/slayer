package mist

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// read is a colour pfm: its size, scale and light, rows from the bottom.
func Read(path string) (image Image, err error) {
	file, err := os.Open(path)
	if err != nil {
		return Image{}, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	in := bufio.NewReader(file)
	var kind string
	if _, err := fmt.Fscan(in, &kind, &image.Width, &image.Height, &image.Scale); err != nil {
		return Image{}, err
	}
	if _, err := in.ReadByte(); err != nil {
		return Image{}, err
	}
	if kind != "PF" {
		return Image{}, fmt.Errorf("%s is %s, not a colour pfm", path, kind)
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
	if err := binary.Read(in, order(image.Scale), image.Pixels); err != nil {
		return Image{}, err
	}
	return image, nil
}

// order is a pfm's byte order by its scale's sign: above nought, big
// endian.
func order(scale float64) binary.ByteOrder {
	if scale > 0 {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// write writes light as a colour pfm.
func Write(path string, image Image) (err error) {
	if !valid(image) {
		return fmt.Errorf("invalid mist image")
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
