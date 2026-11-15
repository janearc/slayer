package finish

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/render/frame"
)

// Mean is the mean of glimpses in linear light, pixel by pixel and
// channel by channel, as a shutter open across them would gather. The
// frame is new; coverage and depth are the first glimpse's. One glimpse
// is copied. None, a nil one, or glimpses of two sizes are refused, the
// odd one by its place.
func Mean(glimpses []*frame.Frame) (*frame.Frame, error) {
	if len(glimpses) == none {
		return nil, errors.New("finish: a frame is developed from one " +
			"glimpse or more; none were given")
	}
	first := glimpses[none]
	if first == nil {
		return nil, errors.New("finish: glimpse 0 is nil; every " +
			"glimpse is a rendered frame")
	}
	total := make([][3]float64, len(first.Pix))
	for place, each := range glimpses {
		if err := added(total, first, each, place); err != nil {
			return nil, err
		}
	}
	mean := frame.New(first.W, first.H)
	copy(mean.Pix, first.Pix)
	for index, sum := range total {
		mean.Pix[index].R = float32(sum[0] / float64(len(glimpses)))
		mean.Pix[index].G = float32(sum[1] / float64(len(glimpses)))
		mean.Pix[index].B = float32(sum[2] / float64(len(glimpses)))
	}
	return mean, nil
}

// added puts a glimpse's light into the totals, if it is a frame of the
// first's size.
func added(total [][3]float64, first, each *frame.Frame, place int) error {
	if each == nil {
		return fmt.Errorf("finish: glimpse %d is nil; every glimpse "+
			"is a rendered frame", place)
	}
	if each.W != first.W || each.H != first.H {
		return fmt.Errorf("finish: glimpse %d is %dx%d among glimpses "+
			"%dx%d; render every glimpse at one size", place, each.W,
			each.H, first.W, first.H)
	}
	for index, pixel := range each.Pix {
		total[index][0] += float64(pixel.R)
		total[index][1] += float64(pixel.G)
		total[index][2] += float64(pixel.B)
	}
	return nil
}
