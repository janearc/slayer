package searching

import (
	"github.com/janearc/slayer/interfaces/movie"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"math"
)

// readSpectrum measures the actual revised WAV using Hann-windowed
// Goertzel bins. Only the first eight seconds are retained; the frequency
// axis is logarithmic, 40 Hz to 8 kHz. No invented signal is drawn.
func (s *heardStage) readSpectrum(path string) error {
	sound, err := movie.ReadSound(path)
	if err != nil {
		return err
	}
	excerpt, err := sound.Slice(0, min(sound.Length(), sound.Rate()*8))
	if err != nil {
		return err
	}
	samples := excerpt.Samples()
	rate := excerpt.Rate()
	const columns, bins, n = 240, 96, 2048
	s.spectrum = make([][]float64, columns)
	for col := range columns {
		start := int(float64(col) * 8 / columns * float64(rate))
		s.spectrum[col] = make([]float64, bins)
		for bin := range bins {
			frequency := 40 * math.Pow(200, float64(bin)/float64(bins-1))
			coefficient := 2 * math.Cos(2*math.Pi*frequency/float64(rate))
			a, b := 0.0, 0.0
			for j := range n {
				sample := 0.0
				if 2*(start+j)+1 < len(samples) {
					sample = .5 * float64(samples[2*(start+j)]+samples[2*(start+j)+1])
				}
				sample *= .5 - .5*math.Cos(2*math.Pi*float64(j)/(n-1))
				next := sample + coefficient*a - b
				b = a
				a = next
			}
			power := math.Sqrt(math.Max(0, a*a+b*b-coefficient*a*b)) / n
			s.spectrum[col][bin] = math.Max(0, math.Min(1, (20*math.Log10(power+1e-10)+85)/65))
		}
	}
	return nil
}

// listening draws the measured soundtrack spectrogram at the clock at.
func (s *heardStage) listening(width, height int, at float64) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 3; i < len(out.Pix); i += 4 {
		out.Pix[i] = 255
	}
	fade := smooth(at / 1.4)
	if at > 6.2 {
		fade *= 1 - smooth((at-6.2)/1.07)
	}
	for x := 0; x < width; x++ {
		t := at - 5 + 5*float64(x)/float64(width)
		if t < 0 {
			continue
		}
		col := min(len(s.spectrum)-1, int(t/8*float64(len(s.spectrum))))
		for y := height / 5; y < height*4/5; y++ {
			if y > height/2-3 && y < height/2+3 {
				continue
			}
			bin := 0
			if y < height/2 {
				bin = int(float64(height/2-y) / float64(height*3/10) * 58)
			} else {
				bin = 45 + int(float64(height*4/5-y)/float64(height*3/10)*50)
			}
			v := s.spectrum[col][bin] * fade
			out.SetRGBA(x, y, color.RGBA{uint8(150 * v * v), uint8(95 * v), uint8(55 * v), 255})
		}
	}
	if at >= 240.0/66+120.0/66 { // bar 2, beat 3
		// Scale the small bitmap face by two, retaining the exact lowercase title.
		text := image.NewRGBA(image.Rect(0, 0, 50, 18))
		d := font.Drawer{Dst: text, Src: image.NewUniform(color.RGBA{235, 226, 203, 255}), Face: basicfont.Face7x13, Dot: fixed.P(3, 13)}
		d.DrawString("heard.")
		for y := range 36 {
			for x := range 100 {
				c := text.RGBAAt(x/2, y/2)
				if c.A > 0 {
					out.SetRGBA((width-100)/2+x, (height-36)/2+y, c)
				}
			}
		}
	}
	return out
}
