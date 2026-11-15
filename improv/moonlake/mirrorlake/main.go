// Command mirrorlake is improv only: a matte painter's lake. It reads a
// raw, and below the far shore lays the picture mirrored about the
// horizon, at water's grazing reflectance, with a ripple that grows
// toward the camera. It writes a raw for develop.
//
//	mirrorlake in.pfm out.pfm top bottom shore
//
// top and bottom are the frame's edges in degrees of elevation, shore
// the far shore's.
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
)

// water's share mirrored at a grazing look, by schlick's fresnel for an
// index of 1.34 near 85 degrees from straight down
const mirrored = 0.6

// main lays the mirrored lake below the far shore.
func main() {
	if len(os.Args) != 6 && len(os.Args) != 7 {
		fmt.Println("mirrorlake in.pfm out.pfm top bottom shore [seconds]")
		return
	}
	seconds := 0.0
	if len(os.Args) == 7 {
		var err error
		if seconds, err = strconv.ParseFloat(os.Args[6], 64); err != nil {
			fmt.Println(err)
			return
		}
	}
	edges := make([]float64, 3)
	for i := range edges {
		v, err := strconv.ParseFloat(os.Args[3+i], 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		edges[i] = v
	}
	top, bottom, shore := edges[0], edges[1], edges[2]
	width, height, scale, pixels, err := read(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	// rows are stored bottom first; row r from the top is height-1-r
	at := func(x, fromTop int) int { return ((height-1-fromTop)*width + x) * 3 }
	span := top - bottom
	rowOf := func(elevation float64) float64 {
		return (top - elevation) / span * float64(height)
	}
	out := append([]float32(nil), pixels...)
	first := int(math.Ceil(rowOf(shore)))
	for r := first; r < height; r++ {
		elevation := top - (float64(r)+0.5)/float64(height)*span
		source := rowOf(-elevation)
		// nearer water, lower in the frame, ripples more
		near := float64(r-first) / float64(height-first)
		for x := 0; x < width; x++ {
			// the ripples run toward the camera, a little across
			wobble := (0.6 + 4*near) * math.Sin(float64(x)*0.09+
				float64(r)*0.7-seconds*2.1) *
				math.Sin(float64(r)*0.23-seconds*1.3)
			sx := clamp(int(math.Round(float64(x)+wobble)), width)
			sr := clamp(int(math.Round(source+wobble*0.5)), height)
			if sr >= first {
				continue
			}
			from, to := at(sx, sr), at(x, r)
			for c := 0; c < 3; c++ {
				out[to+c] = pixels[to+c] + float32(mirrored)*pixels[from+c]
			}
		}
	}
	if err := write(os.Args[2], width, height, scale, out); err != nil {
		fmt.Println(err)
	}
}

// clamp is v held between nought and one less than n.
func clamp(v, n int) int { return max(0, min(n-1, v)) }

// read is a colour pfm: its size, scale and light, rows from the bottom.
func read(path string) (int, int, float64, []float32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	defer file.Close()
	in := bufio.NewReader(file)
	var kind string
	var width, height int
	var scale float64
	if _, err := fmt.Fscan(in, &kind, &width, &height, &scale); err != nil {
		return 0, 0, 0, nil, err
	}
	in.ReadByte()
	if kind != "PF" {
		return 0, 0, 0, nil, fmt.Errorf("%s is %s, not a colour pfm", path, kind)
	}
	pixels := make([]float32, width*height*3)
	err = binary.Read(in, order(scale), pixels)
	return width, height, scale, pixels, err
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
func write(path string, width, height int, scale float64, pixels []float32) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	out := bufio.NewWriter(file)
	fmt.Fprintf(out, "PF\n%d %d\n%g\n", width, height, scale)
	if err := binary.Write(out, order(scale), pixels); err != nil {
		return err
	}
	return out.Flush()
}
