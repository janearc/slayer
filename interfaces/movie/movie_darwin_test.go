//go:build darwin && cgo

package movie

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests encode real movies through AVFoundation. On macOS with
// cgo they always run: none of them skips.

// near is whether two durations are within one AAC packet at that rate.
func near(got, want time.Duration, rate int) bool {
	packet := time.Duration(picked("aac-packet")) * time.Second /
		time.Duration(rate)
	return (got - want).Abs() <= packet
}

// monoWav writes a mono 16-bit wav of the test tone at that rate.
func monoWav(t *testing.T, rate int, seconds float64) string {
	t.Helper()
	const (
		pcm       = 1  // defined: the wav format code for plain PCM
		depth     = 16 // defined: bits a sample in the file written
		chunkSize = 16 // defined: the bytes of a PCM fmt chunk
		bits      = 8  // defined: bits in a byte
		header    = 36 // defined: a PCM wav's header after RIFF's size
	)
	count := int(seconds * float64(rate))
	angle := radiansInTurn * pickedValue("tone-hertz") / float64(rate)
	loud := pickedValue("tone-loudness") * math.MaxInt16
	var data []byte
	for i := range count {
		phase := angle * float64(i)
		sample := int16(loud * math.Sin(phase))
		data = binary.LittleEndian.AppendUint16(data, uint16(sample))
	}
	bytesPer := depth / bits
	riffSize := uint32(header + len(data))
	dataSize := uint32(len(data))
	var out []byte
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, riffSize)
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, chunkSize)
	out = binary.LittleEndian.AppendUint16(out, pcm)
	out = binary.LittleEndian.AppendUint16(out, mono)
	out = binary.LittleEndian.AppendUint32(out, uint32(rate))
	out = binary.LittleEndian.AppendUint32(out, uint32(rate*bytesPer))
	out = binary.LittleEndian.AppendUint16(out, uint16(bytesPer))
	out = binary.LittleEndian.AppendUint16(out, depth)
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, dataSize)
	out = append(out, data...)
	path := filepath.Join(t.TempDir(), "tone.wav")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// a mono wav at another rate comes back at 44100, on both channels.
func TestMonoComesBackOnBothChannels(t *testing.T) {
	rate := picked("mono-rate")
	read, err := ReadSound(monoWav(t, rate, one))
	if err != nil {
		t.Fatal(err)
	}
	if read.Rate() != readRate ||
		!near(read.Duration(), time.Second, readRate) {
		t.Errorf("read %v at %d", read.Duration(), read.Rate())
	}
	var louder float64
	for left := zero; left+one < len(read.samples); left += channels {
		right := left + one
		if read.samples[left] != read.samples[right] {
			t.Fatalf("instant %d: left %v, right %v", left/channels,
				read.samples[left], read.samples[right])
		}
		louder = math.Max(louder, float64(read.samples[left]))
	}
	if louder < pickedValue("tone-loudness")*pickedValue("tone-kept") {
		t.Errorf("the tone came back at %v", louder)
	}
}

// Probe and ReadSound refuse what is not there and what is not a movie.
func TestReadingRefusesWhatIsNotAMovie(t *testing.T) {
	folder := t.TempDir()
	notes := filepath.Join(folder, notMovieName)
	if err := os.WriteFile(notes, []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{notes, folder,
		filepath.Join(folder, movieName)} {
		if _, err := Probe(path); err == nil {
			t.Errorf("probed %s", path)
		}
		if _, err := ReadSound(path); err == nil {
			t.Errorf("read sound from %s", path)
		}
	}
}

// the bridge says why when AVFoundation will not start: here, because
// a file is already where the movie would be written.
func TestTheBridgeSaysWhyItWillNotOpen(t *testing.T) {
	request := smallRequest(t)
	settings := mustSettings(t, request)
	if err := os.WriteFile(request.Path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openPipe(settings, request.Path); err == nil {
		t.Errorf("the bridge opened over an existing file")
	}
}

// a temporary file that cannot be removed is said so, by its name.
func TestAFileThatWillNotGoIsSaidSo(t *testing.T) {
	folder := t.TempDir()
	inside := filepath.Join(folder, notMovieName)
	if err := os.WriteFile(inside, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	err := removeQuietly(folder)
	if err == nil || !strings.Contains(err.Error(), folder) {
		t.Errorf("removing a full directory: %v", err)
	}
	if removeQuietly(filepath.Join(folder, movieName)) != nil {
		t.Errorf("removing what was never made is an error")
	}
}
