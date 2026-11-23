package lights

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
)

func pfmBytes(t *testing.T, header string, order binary.ByteOrder, values []float32) []byte {
	t.Helper()
	var data bytes.Buffer
	data.WriteString(header)
	if err := binary.Write(&data, order, values); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestDecodePFMUsesTopDownFloat64WithoutScaling(t *testing.T) {
	for _, test := range []struct {
		name, header string
		order        binary.ByteOrder
		scale        float64
	}{
		{"little", "PF\n2 2\n-2.5\n", binary.LittleEndian, -2.5},
		{"big", "PF\n2 2\n2.5\n", binary.BigEndian, 2.5},
		{"zero", "PF\n2 2\n0\n", binary.LittleEndian, 0},
		{"crlf", "PF\r\n2 2\r\n-1\r\n", binary.LittleEndian, -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := []float32{7, 8, 9, 10, 11, 12, 1, 2, 3, 4, 5, 6}
			encoded := pfmBytes(t, test.header, test.order, raw)
			image, scale, err := DecodePFM(bytes.NewReader(encoded))
			if err != nil {
				t.Fatal(err)
			}
			want := Image{Width: 2, Height: 2, Light: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
			if !reflect.DeepEqual(image, want) || scale != test.scale || image.At(1, 1) != 9 {
				t.Fatalf("image=%+v scale=%v, want %+v scale=%v", image, scale, want, test.scale)
			}
		})
	}
}

func TestEncodePFMExactBottomUpFloat32Bytes(t *testing.T) {
	image := Image{Width: 1, Height: 2, Light: []float64{1.0 / 3, -2, 3, 4, 5, 6}}
	before := append([]float64(nil), image.Light...)
	for _, test := range []struct {
		scale  float64
		header string
		order  binary.ByteOrder
	}{
		{-2, "PF\n1 2\n-2\n", binary.LittleEndian},
		{2, "PF\n1 2\n2\n", binary.BigEndian},
		{0, "PF\n1 2\n0\n", binary.LittleEndian},
	} {
		var got bytes.Buffer
		if err := encodePFM(&got, image, test.scale); err != nil {
			t.Fatal(err)
		}
		want := pfmBytes(t, test.header, test.order, []float32{4, 5, 6, float32(1.0 / 3), -2, 3})
		if !bytes.Equal(got.Bytes(), want) {
			t.Fatalf("scale=%v bytes=%x want=%x", test.scale, got.Bytes(), want)
		}
	}
	if !reflect.DeepEqual(image.Light, before) {
		t.Fatal("encoding mutated float64 input")
	}
}

func TestPFMRoundTripPreservesFloat32SpecialValues(t *testing.T) {
	image := Image{Width: 1, Height: 1, Light: []float64{math.Inf(1), math.NaN(), math.Copysign(0, -1)}}
	var data bytes.Buffer
	if err := encodePFM(&data, image, -1); err != nil {
		t.Fatal(err)
	}
	got, _, err := DecodePFM(&data)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsInf(got.Light[0], 1) || !math.IsNaN(got.Light[1]) || !math.Signbit(got.Light[2]) {
		t.Fatalf("special values changed: %v", got.Light)
	}
}

func TestDecodePFMRejectsInvalidAndTruncatedInput(t *testing.T) {
	for _, input := range []string{
		"", "PF\n", "Pf\n1 1\n-1\n", "PF\n0 1\n-1\n", "PF\n-1 1\n-1\n",
		"PF\n1 0\n-1\n", "PF\n9223372036854775807 2\n-1\n",
		"PF\n1 1\nNaN\n", "PF\n1 1\n+Inf\n", "PF\n1 1\n-Inf\n",
		"PF\n1 1\n-1", "PF\n1 1\n-1x", "PF\n1 1\n-1\n", "PF\n1 1\n-1\n12345",
		"PF\n1000000000 1\n-1\n",
	} {
		t.Run(input, func(t *testing.T) {
			if _, _, err := DecodePFM(bytes.NewBufferString(input)); err == nil {
				t.Fatal("invalid PFM accepted")
			}
		})
	}
}

func TestPFMExpectedDimensionsAndBoundedInput(t *testing.T) {
	valid := pfmBytes(t, "PF\n1 1\n-1\n", binary.LittleEndian, []float32{1, 2, 3})
	copied := append([]byte(nil), valid...)
	trailing := append(copied, 0)
	for _, input := range [][]byte{
		[]byte("PF\n1000000000 1000000000\n-1\n"),
		trailing,
		[]byte("PF 1 1\n-1\njunk\nabcdefghijkl"),
		[]byte(strings.Repeat(" ", 4097) + "PF\n1 1\n-1\n"),
	} {
		if _, _, err := DecodePFMExpected(bytes.NewReader(input), 1, 1); err == nil {
			t.Fatal("invalid expected-dimension input accepted")
		}
	}
	image, _, err := DecodePFMExpected(bytes.NewReader(valid), 1, 1)
	if err != nil || !reflect.DeepEqual(image.Light, []float64{1, 2, 3}) {
		t.Fatalf("valid bounded PFM: %+v, %v", image, err)
	}
	if _, _, err := DecodePFMExpected(bytes.NewReader(valid), 2, 2); err == nil {
		t.Fatal("wrong expected dimensions accepted")
	}
}

func TestPFMFileIOAndValidationBeforeCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tiny.pfm")
	image := Image{Width: 1, Height: 1, Light: []float64{1, 2, 3}}
	if err := Write(path, image, -1); err != nil {
		t.Fatal(err)
	}
	got, scale, err := Read(path)
	if err != nil || scale != -1 || !reflect.DeepEqual(got, image) {
		t.Fatalf("read=%+v scale=%v err=%v", got, scale, err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		image Image
		scale float64
	}{
		{Image{}, -1},
		{Image{Width: 1, Height: 1, Light: []float64{1, 2}}, -1},
		{image, math.NaN()},
		{image, math.Inf(1)},
	} {
		if err := Write(path, test.image, test.scale); err == nil {
			t.Fatal("invalid write accepted")
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("invalid write damaged existing file")
		}
		var buffer bytes.Buffer
		if err := encodePFM(&buffer, test.image, test.scale); err == nil || buffer.Len() != 0 {
			t.Fatal("invalid encode emitted output")
		}
	}
	if _, _, err := Read(path + ".missing"); err == nil {
		t.Fatal("missing file accepted")
	}
	if err := Write(filepath.Join(path, "child"), image, -1); err == nil {
		t.Fatal("invalid output path accepted")
	}
}

// failingWriter takes nothing and says why.
func failingWriter(t *testing.T, why error) *mocks.MockWriter {
	writer := mocks.NewMockWriter(gomock.NewController(t))
	writer.EXPECT().Write(gomock.Any()).Return(0, why).AnyTimes()
	return writer
}

func TestPFMPropagatesIOErrors(t *testing.T) {
	want := errors.New("synthetic write failure")
	image := Image{Width: 1, Height: 1, Light: []float64{1, 2, 3}}
	if err := encodePFM(failingWriter(t, want), image, -1); !errors.Is(err, want) {
		t.Fatalf("write error=%v", err)
	}
	_, _, err := DecodePFM(bytes.NewBufferString("PF\n1 1\n-1\n123"))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncation error=%v", err)
	}
}
