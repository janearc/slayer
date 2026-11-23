package lights

import (
	"fmt"
	"math"
)

// skyOf marks the open sky pixels of mask: those whose channel sum is over 2
// percent of the largest sum in the image. An image with no light has no sky.
func skyOf(mask Image) ([]bool, error) {
	if err := mask.validate(); err != nil {
		return nil, err
	}
	lum := make([]float64, mask.Width*mask.Height)
	most := 0.0
	for i := range lum {
		lum[i] = mask.Light[3*i] + mask.Light[3*i+1] + mask.Light[3*i+2]
		most = math.Max(most, lum[i])
	}
	open := make([]bool, len(lum))
	for i, light := range lum {
		open[i] = light > most*0.02
	}
	return open, nil
}

// coveredBy marks the pixels of mask whose channel sum is over 0.3 times the
// largest single channel value in the image. An image with no light covers
// nothing.
func coveredBy(mask Image) ([]bool, error) {
	if err := mask.validate(); err != nil {
		return nil, err
	}
	most := 0.0
	for _, value := range mask.Light {
		most = math.Max(most, value)
	}
	covered := make([]bool, mask.Width*mask.Height)
	if most == 0 {
		return covered, nil
	}
	for i := range covered {
		covered[i] = mask.Light[3*i]+mask.Light[3*i+1]+mask.Light[3*i+2] > 0.3*most
	}
	return covered, nil
}

// rimOf returns a function giving the rim row for column x: one below the
// lowest open pixel in the column, then the largest such row within 30
// columns either side. A column with no open pixel counts as 0, and x outside
// the picture is held to the nearest column.
func rimOf(open []bool, width, height int) (func(int) int, error) {
	if err := validateMask(open, width, height); err != nil {
		return nil, err
	}
	rows := make([]int, width)
	for x := range width {
		for y := height - 1; y >= 0; y-- {
			if open[y*width+x] {
				rows[x] = y + 1
				break
			}
		}
	}
	smoothed := make([]int, width)
	for x := range width {
		for k := max(0, x-30); k <= min(width-1, x+30); k++ {
			smoothed[x] = max(smoothed[x], rows[k])
		}
	}
	return func(x int) int { return smoothed[max(0, min(width-1, x))] }, nil
}

// nearOf marks pixels that are not open and can be reached from not-open
// pixels in the top 40 rows by steps up, down, left or right through not-open
// pixels, never entering a row that is 2 above the rim of its column or
// lower. It refuses a nil rim.
func nearOf(open []bool, rim func(int) int, width, height int) ([]bool, error) {
	if err := validateMask(open, width, height); err != nil {
		return nil, err
	}
	if rim == nil {
		return nil, fmt.Errorf("nil rim")
	}
	found := make([]bool, len(open))
	var stack []int
	for i := range min(40, height) * width {
		if !open[i] {
			found[i] = true
			stack = append(stack, i)
		}
	}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := i%width, i/width
		for _, delta := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+delta[0], y+delta[1]
			if nx < 0 || ny < 0 || nx >= width || ny >= height || ny >= rim(nx)-2 {
				continue
			}
			j := ny*width + nx
			if !open[j] && !found[j] {
				found[j] = true
				stack = append(stack, j)
			}
		}
	}
	return found, nil
}

// brightest is the mean colour of the pixels whose channel sum is at least
// half the largest sum in the image.
func brightest(image Image) ([3]float64, error) {
	if err := image.validate(); err != nil {
		return [3]float64{}, err
	}
	most := 0.0
	for i := 0; i < len(image.Light); i += 3 {
		most = math.Max(most, image.Light[i]+image.Light[i+1]+image.Light[i+2])
	}
	var sum [3]float64
	count := 0.0
	for i := 0; i < len(image.Light); i += 3 {
		if image.Light[i]+image.Light[i+1]+image.Light[i+2] < most/2 {
			continue
		}
		for c := range 3 {
			sum[c] += image.Light[i+c]
		}
		count++
	}
	for c := range sum {
		sum[c] /= count
	}
	return sum, nil
}

// validateMask refuses an open mask that does not hold exactly width times
// height entries, or whose size is not valid.
func validateMask(open []bool, width, height int) error {
	if !dimensions(width, height) || len(open) != width*height {
		return fmt.Errorf("invalid mask dimensions or pixel count")
	}
	return nil
}
