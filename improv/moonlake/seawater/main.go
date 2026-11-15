// Command seawater is improv/moonlake's: the flooded
// crater's water as oceantex sees it from moonlake's camera, lit by the
// moon as its sun, reflecting a raw of the scene as its sky, written as
// a pfm of oceantex's radiance (per unit of the moon's irradiance on the
// water). MOONREF names a plain plate to take the moon's light from when
// the sky raw has lights in it.
//
//	seawater out.pfm width height seconds wind sky.pfm
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/oceantex"
)

// moonlake's numbers: the moon's bearing and the 15/70 aperture's width.
//
// From the environment, the first view's when unset: the frame's height
// (MOONLAKE_FRAME_HEIGHT in mm, 39.61 for the 16:9 crop, 51.61 for the
// whole aperture), the camera (MOONLAKE_EYE metres over the lake at 1076
// m, MOONLAKE_LOOK its tilt, MOONLAKE_FOCAL in mm) and the moon
// (MOONLAKE_MOON_ALT its height in degrees, MOONLAKE_MOON_SCALE its disc
// in true radii of 0.259 degrees)
const (
	moonAzimuth = 109.65
	gaugeWidth  = 70.41
)

var (
	gaugeHeight  = numberFrom("MOONLAKE_FRAME_HEIGHT", 39.61)
	eyeOverLake  = numberFrom("MOONLAKE_EYE", 1129.6-1076)
	lookDown     = numberFrom("MOONLAKE_LOOK", -0.5)
	focal        = numberFrom("MOONLAKE_FOCAL", 150)
	moonAltitude = numberFrom("MOONLAKE_MOON_ALT", 3.0)
	moonRadius   = numberFrom("MOONLAKE_MOON_SCALE", 6) * 0.259
)

// numberFrom is a number from the environment, or the default.
func numberFrom(key string, otherwise float64) float64 {
	if raw := os.Getenv(key); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
		fmt.Fprintf(os.Stderr, "%s=%q is not a number\n", key, raw)
		os.Exit(2)
	}
	return otherwise
}

// main draws oceantex's water from moonlake's camera and writes the raw.
func main() {
	if len(os.Args) != 7 {
		fmt.Fprintln(os.Stderr, "seawater out.pfm width height seconds wind sky.pfm")
		os.Exit(2)
	}
	sky, err := skyFrom(os.Args[6])
	if err != nil {
		fail(err)
	}
	width, err1 := strconv.Atoi(os.Args[2])
	height, err2 := strconv.Atoi(os.Args[3])
	seconds, err3 := strconv.ParseFloat(os.Args[4], 64)
	wind, err4 := strconv.ParseFloat(os.Args[5], 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		fail(fmt.Errorf("width and height are whole, seconds and wind numbers"))
	}
	moon := toward(moonAzimuth, moonAltitude)
	scene := oceantex.Scene{
		Sea: oceantex.Sea{Wind: wind, Fetch: 1500,
			// toward the camera: the moon's bearing less half a turn,
			// in radians from east, counterclockwise
			Toward: (90 - (moonAzimuth + 180)) * math.Pi / 180},
		Salinity: 0, Celsius: 15, Chlorophyll: 2,
		Sun: moon,
		Sky: sky.at,
	}
	gauge, err := oceantex.NewGauge("moonlake 16:9", gaugeWidth, gaugeHeight)
	if err != nil {
		fail(err)
	}
	eye := oceantex.Vec{Z: eyeOverLake}
	look := toward(moonAzimuth, lookDown)
	camera := oceantex.Camera{
		Eye: oceantex.Subspace{Origin: eye,
			Orientation: oceantex.LookAt(eye, oceantex.Vec{
				X: eye.X + 1000*look.X, Y: eye.Y + 1000*look.Y,
				Z: eye.Z + 1000*look.Z})},
		Lens:  oceantex.NewLens(focal, gauge, 0),
		Width: width, Height: height, Seconds: seconds, Seed: 3,
	}
	picture, err := oceantex.View(scene, camera)
	if err != nil {
		fail(err)
	}
	if err := write(os.Args[1], picture); err != nil {
		fail(err)
	}
}

// fail says what went wrong and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "seawater:", err)
	os.Exit(1)
}

// rendered is moonlake's own render as the sky the water reflects: a
// direction is looked up in the camera's frame, and the raw's light
// there is given in oceantex's units, each channel over the moon disc's
// own, so the disc carries the moon's irradiance over its solid angle.
type rendered struct {
	width, height int
	light         []float32
	moon          [3]float64
	forward       oceantex.Vec
	right, up     oceantex.Vec
	focal         float64
	disc          float64
	// above is the sky past the frame's top: its top rows' mean, so near
	// water, which reflects the sky overhead, is dark at night and bright
	// under the overhead
	above [3]float64
}

// aboveOf is the mean light of the sky's top rows, those that are sky.
func (sky *rendered) aboveOf() [3]float64 {
	var sum [3]float64
	count := 0.0
	rows := max(1, sky.height/20)
	for r := 0; r < rows; r++ {
		// pfm rows run bottom up: the top row is the last
		base := (sky.height - 1 - r) * sky.width * 3
		for x := 0; x < sky.width; x++ {
			for c := range 3 {
				sum[c] += float64(sky.light[base+x*3+c])
			}
			count++
		}
	}
	for c := range sum {
		sum[c] /= count
	}
	return sum
}

// skyFrom reads a raw as a sky.
func skyFrom(path string) (*rendered, error) {
	width, height, light, err := readPFM(path)
	if err != nil {
		return nil, err
	}
	forward := toward(moonAzimuth, lookDown)
	right := unit(cross(forward, oceantex.Vec{Z: 1}))
	sky := &rendered{width: width, height: height, light: light,
		forward: forward, right: right, up: cross(right, forward),
		// pixels a unit of tangent spans: half the width over the
		// tangent of half the lens's sight
		focal: float64(width) / 2 / (gaugeWidth / 2 / focal),
		disc:  math.Pi * math.Pow(moonRadius*math.Pi/180, 2)}
	sky.moon = sky.brightest()
	sky.above = sky.aboveOf()
	// with lights in the sky, the moon is taken from the plain plate
	if ref := os.Getenv("MOONREF"); ref != "" {
		w, h, light, err := readPFM(ref)
		if err != nil {
			return nil, err
		}
		plain := &rendered{width: w, height: h, light: light}
		sky.moon = plain.brightest()
	}
	return sky, nil
}

// brightest is the mean of the raw's pixels at least half as bright as
// its brightest, which is the moon's disc.
func (sky *rendered) brightest() [3]float64 {
	most := 0.0
	for i := 0; i < len(sky.light); i += 3 {
		most = math.Max(most, float64(sky.light[i]+sky.light[i+1]+
			sky.light[i+2]))
	}
	var sum [3]float64
	count := 0.0
	for i := 0; i < len(sky.light); i += 3 {
		if float64(sky.light[i]+sky.light[i+1]+sky.light[i+2]) < most/2 {
			continue
		}
		for c := range 3 {
			sum[c] += float64(sky.light[i+c])
		}
		count++
	}
	for c := range sum {
		sum[c] /= count
	}
	return sum
}

// at is the light arriving from a direction.
func (sky *rendered) at(from oceantex.Vec) swatch.Swatch {
	d := unit(from)
	// over the moon disc's brightness, not its colour: the sky keeps its
	// own colours, and oceantex's light (the moon's, as its sun) stays a
	// neutral white, not the low moon's amber
	lum := (sky.moon[0] + sky.moon[1] + sky.moon[2]) / 3
	light := func(raw [3]float64) swatch.Swatch {
		return srgb.FromLight(raw[0]/lum/sky.disc, raw[1]/lum/sky.disc,
			raw[2]/lum/sky.disc)
	}
	ahead := dot(d, sky.forward)
	x, y := -1.0, -1.0
	if ahead > 0 {
		x = float64(sky.width)/2 + dot(d, sky.right)/ahead*sky.focal
		y = float64(sky.height)/2 - dot(d, sky.up)/ahead*sky.focal
	}
	if ahead <= 0 || x < 0 || y < 0 || x >= float64(sky.width) ||
		y >= float64(sky.height) {
		// past the frame: the sky above it carries on, the ground below
		// it is dark
		if d.Z > 0 {
			return light(sky.above)
		}
		return srgb.FromLight(0, 0, 0)
	}
	// pfm rows run bottom up
	i := ((sky.height-1-int(y))*sky.width + int(x)) * 3
	return light([3]float64{float64(sky.light[i]), float64(sky.light[i+1]),
		float64(sky.light[i+2])})
}

// cross is the cross product of a and b.
func cross(a, b oceantex.Vec) oceantex.Vec {
	return oceantex.Vec{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X}
}

// readPFM reads a colour pfm, rows bottom up.
func readPFM(path string) (int, int, []float32, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, nil, err
	}
	defer file.Close()
	in := bufio.NewReader(file)
	var kind string
	var width, height int
	var scale float64
	if _, err := fmt.Fscan(in, &kind, &width, &height, &scale); err != nil {
		return 0, 0, nil, err
	}
	in.ReadByte()
	if kind != "PF" {
		return 0, 0, nil, fmt.Errorf("%s is not a colour pfm", path)
	}
	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}
	light := make([]float32, width*height*3)
	err = binary.Read(in, order, light)
	return width, height, light, err
}

// toward is the unit way to an azimuth from north and an altitude, in
// degrees, with x east, y north and z up.
func toward(azimuth, altitude float64) oceantex.Vec {
	a, h := azimuth*math.Pi/180, altitude*math.Pi/180
	return oceantex.Vec{X: math.Sin(a) * math.Cos(h),
		Y: math.Cos(a) * math.Cos(h), Z: math.Sin(h)}
}

// dot is the dot product of a and b.
func dot(a, b oceantex.Vec) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// unit is a the same way, a unit long.
func unit(a oceantex.Vec) oceantex.Vec {
	n := math.Sqrt(dot(a, a))
	return oceantex.Vec{X: a.X / n, Y: a.Y / n, Z: a.Z / n}
}

// write keeps the picture as a little-endian pfm; both run bottom up.
func write(path string, picture *oceantex.Texture) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	out := bufio.NewWriter(file)
	fmt.Fprintf(out, "PF\n%d %d\n-1\n", picture.Width, picture.Height)
	row := make([]float32, 0, picture.Width*picture.Height*3)
	for _, each := range picture.Light {
		row = append(row, float32(each[0]), float32(each[1]),
			float32(each[2]))
	}
	if err := binary.Write(out, binary.LittleEndian, row); err != nil {
		return err
	}
	return out.Flush()
}
