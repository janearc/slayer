//go:build draw

package gpudevelop

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"testing"

	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

func TestSignedCancellationMatchesCPU(t *testing.T) {
	tone := colour.Tone{Exposure: 1, Knee: 0.5}
	d, err := New(1, 1, tone)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	rgb := []float32{1e8, -29725950, 0}
	got, err := d.Develop(rgb)
	if err != nil {
		t.Fatal(err)
	}
	f := frame.New(1, 1)
	f.Pix[0] = frame.Pixel{R: rgb[0], G: rgb[1], B: rgb[2]}
	want := tone.Develop(f)
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatalf("signed cancellation: Metal %v CPU %v", got.Pix, want.Pix)
	}
	t.Logf("signed cancellation: Metal %v CPU %v", got.Pix, want.Pix)
}

func TestNearbySignedCancellationMatchesCPU(t *testing.T) {
	d, err := New(1, 1, colour.Tone{Exposure: 1, Knee: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for delta := -8; delta <= 8; delta++ {
		rgb := []float32{1e8, -29725950 + float32(delta)*2, 0}
		got, err := d.Develop(rgb)
		if err != nil {
			t.Fatal(err)
		}
		f := frame.New(1, 1)
		f.Pix[0] = frame.Pixel{R: rgb[0], G: rgb[1], B: rgb[2]}
		want := (colour.Tone{Exposure: 1, Knee: 0.5}).Develop(f)
		for i := range got.Pix {
			difference := int(got.Pix[i]) - int(want.Pix[i])
			if difference < -1 || difference > 1 {
				t.Fatalf("delta %d: Metal %v CPU %v", delta, got.Pix, want.Pix)
			}
		}
	}
}

func TestGeneratedPreludeAndIntersectionBounds(t *testing.T) {
	text := prelude()
	for _, declaration := range []string{
		fmt.Sprintf("constant float red = %.17ef;", 0.2126),
		fmt.Sprintf("constant float third = %.17ef;", 1.0/3),
		"constant int four = 4;",
	} {
		if !bytes.Contains([]byte(text), []byte(declaration)) {
			t.Fatalf("missing generated constant %q", declaration)
		}
	}
	for _, bound := range []string{"for (int k=first; k<=last; k++)", "int(p.w)-one-int(at.x)", "int(p.h)-one-int(at.y)"} {
		if !bytes.Contains([]byte(kernelBody), []byte(bound)) {
			t.Fatalf("missing image intersection %q", bound)
		}
	}
	if bytes.Contains([]byte(kernelBody), []byte("k<=two*int(p.reach)")) {
		t.Fatal("full-reach blur loop survived")
	}
}

func TestTotalAllocationBudget(t *testing.T) {
	if _, err := allocationBytes(4096, 3002, 300000); err != nil {
		t.Fatalf("4K stage should fit: %v", err)
	}
	for _, size := range [][3]int{{8192, 8192, 0}, {math.MaxInt, 1, 0}, {1, 1, math.MaxInt}, {1, 1, -1}} {
		if _, err := allocationBytes(size[0], size[1], size[2]); err == nil {
			t.Fatalf("accepted unsafe allocation %v", size)
		}
	}
	if got, err := allocationBytes(2, 3, 4); err != nil || got != 6*68+9*12 {
		t.Fatalf("total byte accounting = %d, %v", got, err)
	}
	if d, err := New(8192, 8192, colour.Tone{}); err == nil {
		d.Close()
		t.Fatal("aggregate budget not enforced")
	}
}

func TestBroadBloomTinyImageMatchesCPU(t *testing.T) {
	tone := colour.Tone{Bloom: 100, Knee: 0.5}
	d, err := New(2, 1, tone)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	got, err := d.Develop([]float32{20, 10, 5, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	f := frame.New(2, 1)
	f.Pix[0] = frame.Pixel{R: 20, G: 10, B: 5}
	want := tone.Develop(f)
	for i := range got.Pix {
		delta := int(got.Pix[i]) - int(want.Pix[i])
		if delta < -1 || delta > 1 {
			t.Fatalf("broad bloom byte %d: GPU %d CPU %d", i, got.Pix[i], want.Pix[i])
		}
	}
}

func TestTinyMetalMatchesCPUAndRepeats(t *testing.T) {
	const w, h = 7, 5
	rgb := make([]float32, w*h*3)
	for i := range rgb {
		rgb[i] = float32((i*37)%101) / 97
	}
	rgb[0], rgb[1], rgb[2] = 25, 2, 0.1
	rgb[51], rgb[52], rgb[53] = 0.1, 30, 5
	rgb[len(rgb)-3] = 40
	rgb[15] = -0.2
	for _, tone := range []colour.Tone{
		{},
		{Exposure: 0.3},
		{Exposure: 2, Knee: 0.7, Top: 8},
		{Exposure: 0.25, Knee: 0.7, Top: 0, Bloom: 0.8},
		{Exposure: 3, Knee: 0.4, Top: 4, Bloom: 2},
		{Exposure: 1, Knee: 1, Bloom: -1},
	} {
		t.Run(stringName(tone), func(t *testing.T) {
			d, err := New(w, h, tone)
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			got, err := d.Develop(rgb)
			if err != nil {
				t.Fatal(err)
			}
			f := frame.New(w, h)
			for i := range f.Pix {
				f.Pix[i].R, f.Pix[i].G, f.Pix[i].B = rgb[3*i], rgb[3*i+1], rgb[3*i+2]
			}
			want := tone.Develop(f)
			worst := 0
			for i, v := range got.Pix {
				diff := int(v) - int(want.Pix[i])
				if diff < 0 {
					diff = -diff
				}
				worst = max(worst, diff)
				if diff > 1 {
					t.Fatalf("byte %d: GPU %d CPU %d", i, v, want.Pix[i])
				}
			}
			t.Logf("tiny numeric fixture: maximum byte difference %d", worst)
			again, err := d.Develop(rgb)
			if err != nil || !bytes.Equal(got.Pix, again.Pix) {
				t.Fatalf("repeat changed: %v", err)
			}
			black := make([]float32, len(rgb))
			zero, err := d.Develop(black)
			if err != nil {
				t.Fatal(err)
			}
			for i, v := range zero.Pix {
				if (i%4 == 3 && v != 255) || (i%4 != 3 && v != 0) {
					t.Fatalf("stale frame byte %d = %d", i, v)
				}
			}
			if d.GPUTime() <= 0 {
				t.Fatal("no command-buffer GPU time recorded")
			}
		})
	}
}

func TestInvalidDimensionsAndTonesAreRefused(t *testing.T) {
	for _, size := range [][2]int{{0, 1}, {1, 0}, {-1, 2}, {math.MaxInt, 2}, {1 << 30, 1}} {
		if d, err := New(size[0], size[1], colour.Tone{}); err == nil {
			d.Close()
			t.Fatalf("accepted dimensions %v", size)
		}
	}
	for _, tone := range []colour.Tone{
		{Exposure: math.NaN()}, {Exposure: math.Inf(1)},
		{Exposure: -1}, {Exposure: 1e-100},
		{Knee: math.NaN()}, {Knee: math.Nextafter(1, 0)},
		{Top: math.Inf(1)}, {Bloom: math.NaN()},
		{Bloom: math.Inf(1)}, {Bloom: 100001},
	} {
		if d, err := New(1, 1, tone); err == nil {
			d.Close()
			t.Fatalf("accepted tone %+v", tone)
		}
	}
}

func TestBadInputAndClosedStageAreRefused(t *testing.T) {
	d, err := New(1, 1, colour.Tone{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	for _, rgb := range [][]float32{nil, {1, 2}, {1, 2, 3, 4},
		{float32(math.NaN()), 0, 0}, {0, float32(math.Inf(1)), 0},
		{0, 0, float32(math.Inf(-1))}, {math.MaxFloat32, 0, 0}} {
		if _, err := d.Develop(rgb); err == nil {
			t.Fatalf("accepted input %v", rgb)
		}
	}
	if _, err := d.Develop([]float32{0, 0, 0}); err != nil {
		t.Fatalf("refusal poisoned stage: %v", err)
	}
	d.Close()
	d.Close()
	if _, err := d.Develop([]float32{0, 0, 0}); err == nil {
		t.Fatal("closed stage accepted work")
	}
}

func TestKernelBodyHasNoNumericalLiterals(t *testing.T) {
	literal := regexp.MustCompile(`\b[0-9]+(?:\.[0-9]*)?\b`)
	if literal.MatchString(kernelBody) {
		t.Fatal("Metal kernel body has a numerical literal")
	}
	if !literal.MatchString("float value = 0.5;") {
		t.Fatal("literal guard did not reject a mutant")
	}
}

func TestBloomSpillsWithZeroPaddedEdges(t *testing.T) {
	tone := colour.Tone{Exposure: 0.5, Bloom: 1}
	d, err := New(5, 1, tone)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	rgb := make([]float32, 15)
	rgb[0], rgb[1], rgb[2] = 9, 9, 9
	got, err := d.Develop(rgb)
	if err != nil {
		t.Fatal(err)
	}
	f := frame.New(5, 1)
	f.Pix[0] = frame.Pixel{R: 9, G: 9, B: 9}
	want := tone.Develop(f)
	for i := range got.Pix {
		delta := int(got.Pix[i]) - int(want.Pix[i])
		if delta < -1 || delta > 1 {
			t.Fatalf("edge byte %d: GPU %d CPU %d", i, got.Pix[i], want.Pix[i])
		}
	}
	if got.Pix[4] == 0 || got.Pix[8] == 0 || got.Pix[16] != 0 {
		t.Fatalf("missing glow or beyond-reach contamination: %v", got.Pix)
	}
	plain := colour.Tone{Exposure: 0.5}.Develop(f)
	if bytes.Equal(got.Pix, plain.Pix) || plain.Pix[4] != 0 {
		t.Fatal("bloom-disabled mutant was not distinguished")
	}
}

func TestManyNumericPixelsAgainstCPU(t *testing.T) {
	const w, h = 19, 13
	f := frame.New(w, h)
	rgb := make([]float32, w*h*3)
	state := uint32(197)
	for i := range rgb {
		state = state*1664525 + 1013904223
		rgb[i] = float32(math.Exp2(float64(state%1400)/100-9) - 0.001)
	}
	for i := range f.Pix {
		f.Pix[i].R, f.Pix[i].G, f.Pix[i].B = rgb[3*i], rgb[3*i+1], rgb[3*i+2]
	}
	for _, tone := range []colour.Tone{
		{Exposure: 0.01, Knee: 0.7, Top: 8, Bloom: 1.3},
		{Exposure: 2, Knee: 0.7, Top: 8, Bloom: 1.3},
		{Exposure: 1, Knee: 0.3, Top: 100},
	} {
		d, err := New(w, h, tone)
		if err != nil {
			t.Fatal(err)
		}
		got, err := d.Develop(rgb)
		d.Close()
		if err != nil {
			t.Fatal(err)
		}
		want := tone.Develop(f)
		changed, worst := 0, 0
		for i := range got.Pix {
			delta := int(got.Pix[i]) - int(want.Pix[i])
			if delta < 0 {
				delta = -delta
			}
			if delta != 0 {
				changed++
			}
			worst = max(worst, delta)
			if delta > 1 {
				t.Fatalf("tone %+v byte %d: GPU %d CPU %d", tone, i, got.Pix[i], want.Pix[i])
			}
		}
		t.Logf("tone %+v: changed bytes %d/%d; maximum difference %d", tone, changed, len(got.Pix), worst)
	}
}

func BenchmarkDevelopPreview(b *testing.B) {
	benchmarkDevelop(b, 1024, 751)
}

func BenchmarkDevelop4K(b *testing.B) {
	benchmarkDevelop(b, 4096, 3002)
}

func benchmarkDevelop(b *testing.B, w, h int) {
	d, err := New(w, h, colour.Tone{Exposure: 1, Knee: 0.7, Top: 8, Bloom: 1})
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()
	rgb := make([]float32, w*h*3)
	for i := range rgb {
		rgb[i] = float32(i%251) / 79
	}
	if _, err := d.Develop(rgb); err != nil {
		b.Fatal(err)
	}
	gpuStart := d.GPUTime()
	b.SetBytes(int64(w) * int64(h) * 16)
	b.ResetTimer()
	for b.Loop() {
		if _, err := d.Develop(rgb); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64((d.GPUTime()-gpuStart).Nanoseconds())/1e6/float64(b.N), "batch-gpu-ms/op")
	b.ReportMetric(float64(w)*float64(h)*float64(b.N)/b.Elapsed().Seconds(), "pixels/s")
}

func stringName(tone colour.Tone) string {
	return fmt.Sprintf("exposure=%g/knee=%g/top=%g/bloom=%g", tone.Exposure, tone.Knee, tone.Top, tone.Bloom)
}
