// Torso renders a native manufactured adult anatomical section under controlled lights.
package main

import (
	"flag"
	"fmt"
	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	framebuffer "github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}
func main() {
	transport := flag.Bool("tissue", false, "whole-surface dermal irradiance diffusion and regional roughness")
	chroma := flag.String("undertone", "neutral", "neutral, red, golden, olive or cool")
	detail := flag.Bool("chest-detail", false, "frame the chest from upper abdomen to clavicles")
	legacy := flag.Bool("legacy-contour", false, "reproduce the earlier masked pectoral-envelope geometry")
	arms := flag.Bool("proximal-arms", false, "continue both shoulder sockets into proximal upper arms")
	lateralRoot := flag.Bool("lateral-attachment", false, "project laterally from the curved chest wall")
	albedo := flag.Bool("pigment-only", false, "show the rest-space diffuse pigmentation without shadows")
	age := flag.Float64("age", 38, "adult age in years for illustrative elastic-response control")
	sections := flag.Bool("sections", false, "write anterior transverse section depths as CSV")
	vertical := flag.Bool("vertical-sections", false, "sample actual sagittal anterior contours at four lateral positions")
	measure := flag.Bool("measure", false, "write modeled landmark coordinates in metres and exit")
	out := flag.String("out", "torso.png", "output path")
	view := flag.Float64("view", 25, "azimuth in degrees")
	width := flag.Int("width", 400, "width in pixels")
	form := flag.String("form", "reference", "reference, shallow, full or asymmetric")
	tone := flag.Float64("melanin", .36, "epidermal pigment fraction")
	arm := flag.Float64("arm", 0, "left arm elevation in degrees")
	pitch := flag.Float64("pitch", 0, "forward trunk pitch in degrees")
	twist := flag.Float64("twist", 0, "axial twist in degrees")
	gravity := flag.Float64("gravity", 1, "downward loading share")
	clay := flag.Bool("clay", false, "neutral diffuse geometry control")
	sideLight := flag.Bool("lateral", false, "light grazing across the chest wall")
	flag.Parse()
	s := primates.Adult()
	switch *chroma {
	case "neutral":
	case "red":
		s.Undertone = primates.Chroma{Red: .90}
	case "golden":
		s.Undertone = primates.Chroma{Gold: .90}
	case "olive":
		s.Undertone = primates.Chroma{Olive: .90}
	case "cool":
		s.Undertone = primates.Chroma{Gold: -.70, Red: .26}
	default:
		panic("unknown undertone")
	}
	s.LegacyContour = *legacy
	s.ProximalArms = *arms
	s.LateralAttachment = *lateralRoot
	s.AgeYears = *age
	s.Melanin = *tone
	s.LeftArm = *arm * maths.Degree
	s.Pitch = *pitch * maths.Degree
	s.Twist = *twist * maths.Degree
	s.Gravity = *gravity
	switch *form {
	case "reference":
	case "shallow":
		s.Left.Projection = .032
		s.Right.Projection = .034
		s.Width = .93
		s.Depth = .92
		s.Compliance = .25
	case "full":
		s.Left.Projection = .086
		s.Right.Projection = .083
		s.Left.Width = .097
		s.Right.Width = .095
		s.Compliance = .76
		s.Width = 1.08
		s.Depth = 1.07
	case "asymmetric":
		s.Left.Projection = .080
		s.Left.Height = -.010
		s.Right.Projection = .048
		s.Right.Height = .005
		s.Compliance = .62
	default:
		panic("unknown torso morphology")
	}
	k, e := manufacture.AdultTorso(s)
	must(e)
	if *measure {
		fmt.Println("landmark,x_m,y_m,z_m")
		for _, p := range s.Landmarks(431) {
			fmt.Printf("%s,%.6f,%.6f,%.6f\n", p.Name, p.Position.X, p.Position.Y, p.Position.Z)
		}
		return
	}
	made, e := manufacture.Make(k, manufacture.Sow(431))
	must(e)
	parts, e := made.Things(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))
	must(e)
	if *sections {
		fmt.Println("height_m,lateral_m,anterior_m")
		m := parts[0].Shape.(*mesh.Mesh)
		for _, y := range []float64{1.18, 1.22, 1.26} {
			for offset := -60; offset <= 60; offset++ {
				x := float64(offset) * .002
				r := maths.Ray{At: maths.V(x, y, .6), Dir: maths.ZAxis.Neg()}
				if h, ok := m.Meet(r, 0, 1); ok {
					fmt.Printf("%.3f,%.3f,%.7f\n", y, x, r.Point(h.Along()).Z)
				}
			}
		}
		return
	}
	if *vertical {
		fmt.Println("lateral_m,height_m,anterior_m")
		m := parts[0].Shape.(*mesh.Mesh)
		for _, x := range []float64{0, .040, .080, .115} {
			for mm := 1040; mm <= 1320; mm += 2 {
				y := float64(mm) / 1000
				r := maths.Ray{At: maths.V(x, y, .6), Dir: maths.ZAxis.Neg()}
				if h, ok := m.Meet(r, 0, 1); ok {
					fmt.Printf("%.3f,%.3f,%.7f\n", x, y, r.Point(h.Along()).Z)
				}
			}
		}
		return
	}
	w := thing.NewWorld()
	background := maths.RGB(.105, .12, .125)
	w.Sky = sky.Gradient{Horizon: background, Zenith: background, Ground: background}
	for _, part := range parts {
		if *clay {
			part.Material = &thing.Material{Colour: maths.RGB(.48, .48, .48)}
		}
		w.Add(part)
	}
	key := maths.V(-.62, .73, 1)
	if *sideLight {
		key = maths.V(-1, .32, .22)
	}
	w.Lamp(film.Sun{Toward: key.Unit(), Strength: maths.RGB(1.55, 1.49, 1.45), Disc: true, Size: .72})
	for _, up := range []maths.Vec{maths.YAxis, maths.YAxis.Neg()} {
		w.Lamp(film.Flood{Up: up, Strength: maths.Grey(.32)})
	}
	if *albedo {
		m := *w.Things[0].Material
		m.Gloss = 0
		m.Satin = 0
		w.Things[0].Material = &m
		w.Lamps = nil
		w.Sky = nil
		for _, up := range []maths.Vec{maths.YAxis, maths.YAxis.Neg()} {
			w.Lamp(film.Flood{Up: up, Strength: maths.Grey(1)})
		}
	}
	at := maths.V(0, 1.125, .004)
	field := .41
	if *detail {
		at = maths.V(0, 1.19, .035)
		field = .232
	}
	a := *view * maths.Degree
	from := at.Add(maths.V(1.55*math.Sin(a), .028, 1.55*math.Cos(a)))
	move, e := camera.Timed(from, from, time.Second, "linear")
	must(e)
	path, e := camera.New([]camera.Move{move}, at, maths.YAxis)
	must(e)
	eye, e := path.At(0)
	must(e)
	lens, e := film.PinholeOf(field)
	must(e)
	h := *width * 5 / 4
	var frame *framebuffer.Frame
	if *transport && !*clay && !*albedo {
		frame, e = materialStudy(w, *eye, lens, *width, h, s, key)
	} else {
		frame, e = traceFrame(w, *eye, lens, *width, h)
	}
	must(e)
	img := image.NewNRGBA(image.Rect(0, 0, *width, h))

	for y := 0; y < h; y++ {
		for x := 0; x < *width; x++ {
			p := frame.At(x, y).Light()
			img.SetNRGBA(x, y, color.NRGBA{srgb(p.R), srgb(p.G), srgb(p.B), 255})
		}
	}
	f, e := os.Create(*out)
	must(e)
	must(png.Encode(f, img))
	must(f.Close())
	fmt.Println(*out)
}

func traceFrame(w *thing.World, eye maths.Subspace, lens film.Sensor, width, height int) (*framebuffer.Frame, error) {
	return trace.Sampled(w, eye, lens, 0, width, height, 2)
}

func srgb(v float64) uint8 {
	if v <= .0031308 {
		return uint8(math.Max(0, 12.92*v)*255 + .5)
	}
	return uint8(math.Min(1, 1.055*math.Pow(v, 1/2.4)-.055)*255 + .5)
}
