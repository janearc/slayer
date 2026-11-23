package lights

import (
	"fmt"
	"math"
)

type OverheadConfig struct {
	Gain   float64
	Colour [3]float64
}

type MoonDiscConfig struct {
	Shift, Level float64
}

// addOverhead adds the overhead pass to sky, each channel times
// config.Gain, level and the config colour. It refuses images that are not
// valid or differ in size, and then changes nothing.
func addOverhead(sky, pass Image, config OverheadConfig, level float64) error {
	if err := sky.validate(); err != nil {
		return err
	}
	if err := pass.validate(); err != nil {
		return err
	}
	if pass.Width != sky.Width || pass.Height != sky.Height {
		return fmt.Errorf("overhead is %dx%d, the frame %dx%d", pass.Width, pass.Height, sky.Width, sky.Height)
	}
	amount := config.Gain * level
	for i := 0; i < len(sky.Light); i += 3 {
		for c := range 3 {
			sky.Light[i+c] += amount * config.Colour[c] * pass.Light[i+c]
		}
	}
	return nil
}

// AddMoonDisc adds the moon disc to the open pixels of sky, moved down by
// config.Shift rows (blending two rows when it is not whole) and scaled by
// config.Level. Rows whose source falls outside the disc are skipped. It
// refuses a size mismatch, a bad mask or a Shift that is not finite.
func AddMoonDisc(sky, disc Image, open []bool, config MoonDiscConfig) error {
	if err := sky.validate(); err != nil {
		return err
	}
	if err := disc.validate(); err != nil {
		return err
	}
	if disc.Width != sky.Width || disc.Height != sky.Height {
		return fmt.Errorf("the moon disc is %dx%d, the plate %dx%d", disc.Width, disc.Height, sky.Width, sky.Height)
	}
	if err := validateMask(open, sky.Width, sky.Height); err != nil {
		return err
	}
	if !finite(config.Shift) {
		return fmt.Errorf("invalid moon disc shift")
	}
	skyOnly := func(x, y, c int) float64 {
		if !open[y*disc.Width+x] {
			return 0
		}
		return disc.Light[disc.At(x, y)+c]
	}
	for y := range sky.Height {
		source := float64(y) - config.Shift
		low := int(math.Floor(source))
		if low < 0 || low+1 >= disc.Height {
			continue
		}
		f := source - float64(low)
		for x := range sky.Width {
			if !open[y*sky.Width+x] {
				continue
			}
			i := sky.At(x, y)
			for c := range 3 {
				sky.Light[i+c] += config.Level * ((1-f)*skyOnly(x, low, c) + f*skyOnly(x, low+1, c))
			}
		}
	}
	return nil
}
