package cheeptune

import (
	"bufio"
	"encoding/binary"
	"math"
	"os"
)

// WriteWAV writes the piece as 16-bit stereo PCM, the format afplay and
// everything else reads.
func (p *Piece) WriteWAV(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	n := len(p.Left)
	data := uint32(n * 4)
	le := binary.LittleEndian
	hdr := make([]byte, 44)
	copy(hdr[0:], "RIFF")
	le.PutUint32(hdr[4:], 36+data)
	copy(hdr[8:], "WAVEfmt ")
	le.PutUint32(hdr[16:], 16)
	le.PutUint16(hdr[20:], 1) // pcm
	le.PutUint16(hdr[22:], 2) // channels
	le.PutUint32(hdr[24:], uint32(p.Rate))
	le.PutUint32(hdr[28:], uint32(p.Rate*4))
	le.PutUint16(hdr[32:], 4)
	le.PutUint16(hdr[34:], 16)
	copy(hdr[36:], "data")
	le.PutUint32(hdr[40:], data)
	if _, err := w.Write(hdr); err != nil {
		f.Close()
		return err
	}
	buf := make([]byte, 4)
	for i := range n {
		le.PutUint16(buf[0:], uint16(pcm16(p.Left[i])))
		le.PutUint16(buf[2:], uint16(pcm16(p.Right[i])))
		if _, err := w.Write(buf); err != nil {
			f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// pcm16 is a sample as a signed 16-bit integer, clamped to full scale.
func pcm16(x float32) int16 {
	return int16(math.Round(math.Max(-1, math.Min(1, float64(x))) * 32767))
}
