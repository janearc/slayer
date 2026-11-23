package ocean

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/oceantex"
)

const (
	moonAzimuth = 109.65
	gaugeWidth  = 70.41
)

// GaugeWidth is the width of the gauge that Setup builds.
func GaugeWidth() float64 { return gaugeWidth }

type Config struct {
	FrameHeight, EyeOverLake, LookDown, Focal float64
	MoonAltitude, MoonScale                   float64
	Width, Height                             int
	Seconds, Wind                             float64
}

// DefaultConfig is the config with the frame, eye height, tilt, lens, and
// moon set, and with width, height, time, and wind left at zero.
func DefaultConfig() Config {
	return Config{FrameHeight: 39.61, EyeOverLake: 1129.6 - 1076, LookDown: -.5, Focal: 150, MoonAltitude: 3, MoonScale: 6}
}

type RGB struct {
	Width, Height int
	Light         []float32
}

type Sky struct {
	width, height int
	light         []float32
	moon          [3]float64
	forward       oceantex.Vec
	right, up     oceantex.Vec
	focal         float64
	disc          float64
	above         [3]float64
}

type Projection struct {
	Forward, Right, Up   oceantex.Vec
	Focal, Normalization float64
}

// Projection is the camera basis and focal length of the sky, with a
// normalization of the moon's average luminance times its disc size.
func (sky *Sky) Projection() (Projection, error) {
	if sky == nil {
		return Projection{}, fmt.Errorf("nil sky")
	}
	luminance := (sky.moon[0] + sky.moon[1] + sky.moon[2]) / 3
	return Projection{Forward: sky.forward, Right: sky.right, Up: sky.up, Focal: sky.focal, Normalization: luminance * sky.disc}, nil
}

// NewSky makes a sky from image, held bottom-up. It refuses a config that is
// not finite or out of range, an invalid image or moonReference, and a
// projection or moon brightness that is not positive. The moon colour comes
// from moonReference if it is given, else from the brightest part of image.
func NewSky(config Config, image RGB, moonReference *RGB) (*Sky, error) {
	for _, value := range []float64{config.FrameHeight, config.EyeOverLake, config.LookDown, config.Focal, config.MoonAltitude, config.MoonScale, config.Seconds, config.Wind} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("nonfinite sky configuration")
		}
	}
	if config.FrameHeight <= 0 || config.EyeOverLake <= 0 || config.Focal <= 0 || config.MoonScale <= 0 || config.LookDown <= -90 || config.LookDown >= 90 || config.MoonAltitude < -90 || config.MoonAltitude > 90 || config.Seconds < 0 || config.Wind < 0 || config.Width < 0 || config.Height < 0 {
		return nil, fmt.Errorf("invalid sky configuration")
	}
	if err := image.validate(); err != nil {
		return nil, err
	}
	if moonReference != nil {
		if err := moonReference.validate(); err != nil {
			return nil, err
		}
	}
	forward := toward(moonAzimuth, config.LookDown)
	right := unit(cross(forward, oceantex.Vec{Z: 1}))
	sky := &Sky{width: image.Width, height: image.Height, light: append([]float32(nil), image.Light...),
		forward: forward, right: right, up: cross(right, forward),
		focal: float64(image.Width) / 2 / (gaugeWidth / 2 / config.Focal),
		disc:  math.Pi * math.Pow(config.MoonScale*0.259*math.Pi/180, 2)}
	sky.moon = sky.brightest()
	sky.above = sky.aboveOf()
	if moonReference != nil {
		plain := &Sky{width: moonReference.Width, height: moonReference.Height, light: moonReference.Light}
		sky.moon = plain.brightest()
	}
	lum := (sky.moon[0] + sky.moon[1] + sky.moon[2]) / 3
	normalization := lum * sky.disc
	if sky.focal <= 0 || math.IsInf(sky.focal, 0) || sky.disc <= 0 || math.IsInf(sky.disc, 0) || lum <= 0 || math.IsNaN(lum) || math.IsInf(lum, 0) || normalization <= 0 || math.IsInf(normalization, 0) {
		return nil, fmt.Errorf("invalid sky projection or moon normalization")
	}
	return sky, nil
}

// validate refuses an image whose size is invalid, whose light is not three
// values per pixel, or whose light is not finite.
func (image RGB) validate() error {
	count, err := pfmSamples(image.Width, image.Height)
	if err != nil {
		return err
	}
	if len(image.Light) != count {
		return fmt.Errorf("invalid bottom-up RGB dimensions or light length")
	}
	for _, value := range image.Light {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return fmt.Errorf("nonfinite RGB light")
		}
	}
	return nil
}

// Setup makes the scene and camera for config, with image as the sky and the
// moon as the sun. The eye is EyeOverLake up, facing the moon's compass
// heading at altitude LookDown degrees, through a gauge GaugeWidth wide.
func Setup(config Config, image RGB, moonReference *RGB) (oceantex.Scene, oceantex.Camera, error) {
	sky, err := NewSky(config, image, moonReference)
	if err != nil {
		return oceantex.Scene{}, oceantex.Camera{}, err
	}
	moon := toward(moonAzimuth, config.MoonAltitude)
	scene := oceantex.Scene{
		Sea: oceantex.Sea{Wind: config.Wind, Fetch: 1500,
			Toward: (90 - (moonAzimuth + 180)) * math.Pi / 180},
		Salinity: 0, Celsius: 15, Chlorophyll: 2,
		Sun: moon, Sky: sky.at,
	}
	gauge, err := oceantex.NewGauge("moonlake 16:9", gaugeWidth, config.FrameHeight)
	if err != nil {
		return oceantex.Scene{}, oceantex.Camera{}, err
	}
	eye := oceantex.Vec{Z: config.EyeOverLake}
	look := toward(moonAzimuth, config.LookDown)
	camera := oceantex.Camera{
		Eye: oceantex.Subspace{Origin: eye,
			Orientation: oceantex.LookAt(eye, oceantex.Vec{
				X: eye.X + 1000*look.X, Y: eye.Y + 1000*look.Y,
				Z: eye.Z + 1000*look.Z})},
		Lens:  oceantex.NewLens(config.Focal, gauge, 0),
		Width: config.Width, Height: config.Height, Seconds: config.Seconds, Seed: 3,
	}
	return scene, camera, nil
}

// Render builds the scene and camera with Setup and returns the view of them.
func Render(config Config, image RGB, moonReference *RGB) (*oceantex.Texture, error) {
	scene, camera, err := Setup(config, image, moonReference)
	if err != nil {
		return nil, err
	}
	return oceantex.View(scene, camera)
}

// aboveOf is the average colour of the top twentieth of the picture, at
// least one row. The picture is held bottom-up, so those are the last rows.
func (sky *Sky) aboveOf() [3]float64 {
	var sum [3]float64
	count := 0.0
	rows := max(1, sky.height/20)
	for r := 0; r < rows; r++ {
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

// pixelSum is the sum of the three channels of the pixel at light index i.
// It adds in float64 if the float32 sum overflows.
func (sky *Sky) pixelSum(i int) float64 {
	sum := float64(sky.light[i] + sky.light[i+1] + sky.light[i+2])
	if math.IsInf(sum, 0) {
		return float64(sky.light[i]) + float64(sky.light[i+1]) + float64(sky.light[i+2])
	}
	return sum
}

// brightest is the average colour of the pixels whose channel sum is at
// least half that of the brightest pixel.
func (sky *Sky) brightest() [3]float64 {
	most := 0.0
	for i := 0; i < len(sky.light); i += 3 {
		most = math.Max(most, sky.pixelSum(i))
	}
	var sum [3]float64
	count := 0.0
	for i := 0; i < len(sky.light); i += 3 {
		if sky.pixelSum(i) < most/2 {
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

// at is the colour of the sky seen along the direction from. A direction
// that falls inside the picture gets that pixel; one outside it gets the
// average of the top rows if it points above the horizon, else black. Light
// is divided by the moon's luminance and disc size.
func (sky *Sky) at(from oceantex.Vec) swatch.Swatch {
	d := unit(from)
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
	if ahead <= 0 || x < 0 || y < 0 || x >= float64(sky.width) || y >= float64(sky.height) {
		if d.Z > 0 {
			return light(sky.above)
		}
		return srgb.FromLight(0, 0, 0)
	}
	i := ((sky.height-1-int(y))*sky.width + int(x)) * 3
	return light([3]float64{float64(sky.light[i]), float64(sky.light[i+1]),
		float64(sky.light[i+2])})
}

// cross is the cross product of a and b.
func cross(a, b oceantex.Vec) oceantex.Vec {
	return oceantex.Vec{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X}
}

// ReadPFM reads the colour PFM file at path. It refuses anything but a
// regular file, a header that is not "PF", dimensions, and scale, or a file
// whose payload is not exactly width by height by three float32 values. A
// negative scale means little-endian data, a positive one big-endian.
func ReadPFM(path string) (RGB, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return RGB{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return RGB{}, err
	}
	if !info.Mode().IsRegular() {
		return RGB{}, fmt.Errorf("%s is not a regular PFM file", path)
	}
	in := bufio.NewReader(file)
	lines := make([]string, 3)
	headerBytes := int64(0)
	for i := range lines {
		line, err := in.ReadSlice('\n')
		if err != nil {
			return RGB{}, fmt.Errorf("invalid PFM header: %w", err)
		}
		headerBytes += int64(len(line))
		lines[i] = strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
	}
	if lines[0] != "PF" {
		return RGB{}, fmt.Errorf("%s is not a colour pfm", path)
	}
	dimensions := strings.Fields(lines[1])
	if len(dimensions) != 2 {
		return RGB{}, fmt.Errorf("invalid PFM dimensions")
	}
	width, err := strconv.Atoi(dimensions[0])
	if err != nil {
		return RGB{}, err
	}
	height, err := strconv.Atoi(dimensions[1])
	if err != nil {
		return RGB{}, err
	}
	count, err := pfmSamples(width, height)
	if err != nil {
		return RGB{}, err
	}
	scale, err := strconv.ParseFloat(strings.TrimSpace(lines[2]), 64)
	if err != nil || scale == 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return RGB{}, fmt.Errorf("invalid PFM scale")
	}
	if info.Size()-headerBytes != int64(count)*4 {
		return RGB{}, fmt.Errorf("invalid PFM payload length")
	}
	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}
	light := make([]float32, count)
	if err := binary.Read(in, order, light); err != nil {
		return RGB{}, err
	}
	return RGB{Width: width, Height: height, Light: light}, nil
}

// pfmSamples is the number of float32 values in a colour picture width by
// height, three to a pixel. It refuses a size that is not positive or that
// would hold more than 256 MiB of float32 data.
func pfmSamples(width, height int) (int, error) {
	const maxSamples = (256 << 20) / 4
	if width <= 0 || height <= 0 || width > maxSamples/3 || height > maxSamples/3/width {
		return 0, fmt.Errorf("invalid or excessive PFM dimensions")
	}
	return width * height * 3, nil
}

// toward is the unit vector for a compass azimuth and an altitude above the
// horizon, both in degrees. Azimuth zero is +Y, ninety is +X, and +Z is up.
func toward(azimuth, altitude float64) oceantex.Vec {
	a, h := azimuth*math.Pi/180, altitude*math.Pi/180
	return oceantex.Vec{X: math.Sin(a) * math.Cos(h),
		Y: math.Cos(a) * math.Cos(h), Z: math.Sin(h)}
}

// dot is the dot product of a and b.
func dot(a, b oceantex.Vec) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// unit scales a to length one. A zero vector gives NaN components.
func unit(a oceantex.Vec) oceantex.Vec {
	n := math.Sqrt(dot(a, a))
	return oceantex.Vec{X: a.X / n, Y: a.Y / n, Z: a.Z / n}
}

// WritePFM writes picture to path as a little-endian colour PFM. It refuses
// a nil picture, a light slice that does not match the size, or a value that
// is not finite or does not fit a float32. Rows are written in the order held.
func WritePFM(path string, picture *oceantex.Texture) (err error) {
	if picture == nil {
		return fmt.Errorf("nil PFM texture")
	}
	count, err := pfmSamples(picture.Width, picture.Height)
	if err != nil {
		return err
	}
	if len(picture.Light) != count/3 {
		return fmt.Errorf("invalid PFM light length")
	}
	for _, pixel := range picture.Light {
		for _, value := range pixel {
			if math.IsNaN(value) || math.IsInf(value, 0) || value > math.MaxFloat32 || value < -math.MaxFloat32 {
				return fmt.Errorf("invalid PFM light value")
			}
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()
	out := bufio.NewWriter(file)
	if _, err := fmt.Fprintf(out, "PF\n%d %d\n-1\n", picture.Width, picture.Height); err != nil {
		return err
	}
	row := make([]float32, picture.Width*3)
	for y := 0; y < picture.Height; y++ {
		for x, pixel := range picture.Light[y*picture.Width : (y+1)*picture.Width] {
			row[x*3], row[x*3+1], row[x*3+2] = float32(pixel[0]), float32(pixel[1]), float32(pixel[2])
		}
		if err := binary.Write(out, binary.LittleEndian, row); err != nil {
			return err
		}
	}
	return out.Flush()
}
