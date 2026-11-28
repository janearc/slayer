// Portrait stages an SMPL-X head using Slayer's rest-space material and skeleton.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/janearc/slayer/bioforms/diffusion"
	"github.com/janearc/slayer/bioforms/pigment"
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/film"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	framebuffer "github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

var v = maths.V
var rgb = maths.RGB

type paint func(maths.Vec) maths.Light

func (f paint) At(p maths.Vec) maths.Light        { return f(p) }
func mix(a, b maths.Light, t float64) maths.Light { return a.Scale(1 - t).Add(b.Scale(t)) }
func gauss(x float64) float64                     { return math.Exp(-x * x) }
func must(err error) {
	if err != nil {
		panic(err)
	}
}

var freckles = pigment.Spots{Seed: 347, Spacing: .0036, Radius: .00125}

func skin(p maths.Vec) maths.Light { return skinWithMarks(p, 1) }
func skinWithMarks(p maths.Vec, markStrength float64) maths.Light {
	x, y, z := math.Abs(p.X), p.Y, p.Z
	grain := func(seed uint64, f float64) float64 { return noise.Gradient(seed, p.X*f, y*f, z*f) }
	tone := rgb(.57, .305, .218).Scale(1 + .042*grain(21, 150) + .022*grain(23, 1800))
	front := maths.Smooth(.017, .052, z)
	cheek := gauss((x-.041)/.026) * gauss((y-1.524)/.024)
	nose := gauss(x/.014) * gauss((y-1.522)/.012)
	tone = mix(tone, rgb(.58, .220, .176), front*(.39*cheek+.45*nose))
	spot := freckles.At(p) * (.62 + .48*grain(10, 85)) * front * math.Max(cheek, gauss((y-1.535)/.011)*gauss(x/.048))
	tone = mix(tone, rgb(.19, .070, .035), spot*markStrength)
	lipX := x / .024
	if lipX < 1 && z > .072 {
		line := 1.4882 - .0008*gauss((x-.012)/.008)
		top := .0070 * (1 - lipX*lipX) * (.74 + .28*gauss((x-.006)/.004))
		bottom := .0068 * (1 - lipX*lipX)
		lip := maths.Smooth(line-bottom-.0013, line-bottom+.0011, y) * (1 - maths.Smooth(line+top-.0016, line+top+.0016, y))
		tone = mix(tone, rgb(.47, .160, .144).Scale(1+.1*grain(14, 850)), lip)
		tone = mix(tone, rgb(.12, .034, .028), .65*lip*gauss((y-line)/.0007))
	}
	brow := 1.5697 + .0040*gauss((x-.038)/.013) - .0024*gauss((x-.014)/.008)
	reach := maths.Smooth(.010, .017, x) * (1 - maths.Smooth(.042, .054, x))
	hair := .81 + .16*math.Sin(11900*x-4100*y+grain(40, 720)*6)
	tone = mix(tone, rgb(.058, .026, .019), front*reach*gauss((y-brow)/.0021)*hair)
	return tone
}
func eye(p maths.Vec) maths.Light {
	x, y := math.Abs(p.X)-.03075, p.Y-1.5535
	r := math.Hypot(x, y)
	a := math.Atan2(y, x)
	col := rgb(.85, .76, .70)
	if p.Z > .06 {
		iris := rgb(.135, .21, .13).Scale(.85 + .19*math.Sin(83*a+380*r) + .20*math.Sin(119*a-540*r))
		iris = mix(iris, rgb(.27, .146, .043), .70*gauss((r-.0025)/.0012))
		iris = mix(iris, rgb(.027, .035, .026), maths.Smooth(.0046, .0056, r))
		col = mix(iris, col, maths.Smooth(.0056, .0061, r))
		col = mix(rgb(.011, .010, .009), col, maths.Smooth(.0021, .0024, r))
	}
	return col
}

type geom struct {
	pts   []maths.Vec
	faces [][3]int
	polys [][]int
}

func readObj(path string) (geom, geom) {
	f, e := os.Open(path)
	must(e)
	defer f.Close()
	var body, eyes geom
	s := bufio.NewScanner(f)
	for s.Scan() {
		t := strings.Fields(s.Text())
		if len(t) == 0 {
			continue
		}
		switch t[0] {
		case "v":
			var xyz [3]float64
			for i := range xyz {
				xyz[i], e = strconv.ParseFloat(t[i+1], 64)
				must(e)
			}
			body.pts = append(body.pts, v(xyz[0], xyz[1], xyz[2]))
		case "f":
			ix := []int{}
			for _, k := range t[1:] {
				n, e := strconv.Atoi(strings.Split(k, "/")[0])
				must(e)
				ix = append(ix, n-1)
			}
			g := &body
			if ix[0] >= 9383 {
				g = &eyes
			}
			g.polys = append(g.polys, ix)
			for j := 1; j+1 < len(ix); j++ {
				g.faces = append(g.faces, [3]int{ix[0], ix[j], ix[j+1]})
			}
		}
	}
	must(s.Err())
	eyes.pts = body.pts
	return body, eyes
}
func hair(t, a float64) maths.Vec {
	// Polar tracks flow down from a left side part. The bob sweeps past the ears.
	front := (1 + math.Cos(a)) * .5
	edge := 1.437 + .174*math.Pow(front, 2.9) + .019*math.Sin(a)*front
	centre := 1.548
	limit := math.Acos(math.Max(-.98, math.Min(.98, (edge-centre)/.123)))
	polar := .004 + t*limit
	angle := a + .19*(1-t)*math.Sin(a+.8)
	widen := 1 + .17*math.Max(0, polar-1.55)
	return v(.095*widen*math.Sin(polar)*math.Sin(angle)-.009*math.Pow(1-t, 3), centre+.123*math.Cos(polar), -.021+.118*math.Sin(polar)*math.Cos(angle))
}
func bob() geom {
	var g geom
	const rows = 46
	const cols = 360
	for j := 0; j <= rows; j++ {
		t := float64(j) / rows
		for i := 0; i < cols; i++ {
			a := 2 * math.Pi * float64(i) / cols
			g.pts = append(g.pts, hair(t, a))
			if j > 0 {
				k := (j-1)*cols + i
				l := (j-1)*cols + (i+1)%cols
				g.faces = append(g.faces, [3]int{k, k + cols, l}, [3]int{l, k + cols, l + cols})
			}
		}
	}
	return g
}
func hairPaint(p maths.Vec) maths.Light {
	a := math.Atan2(p.X/.095, (p.Z+.021)/.118)
	n := noise.Gradient(23, 125*p.X, 33*p.Y, 118*p.Z)
	phase := a + .13*math.Sin(17*p.Y+2*a) + .10*math.Sin(a+.7)*(p.Y-1.53)*10
	strands := .5 + .5*math.Sin(457*phase+2*n)
	return rgb(.068, .024, .012).Scale(.75 + .24*strands + .16*math.Sin(38*phase+1.5*n) + .28*n)
}

// locks are tapered ribbons combed from the part toward the ends of the bob.
func locks() geom {
	rng := rand.New(rand.NewSource(61))
	var g geom
	const steps = 28
	for i := 0; i < 720; i++ {
		a := rng.Float64() * 2 * math.Pi
		start := .015 + .8*rng.Float64()
		end := .98 + .095*rng.Float64()
		width := .00013 + .00040*rng.Float64()
		off := .00010 + .0016*math.Pow(rng.Float64(), 3)
		sweep := (rng.Float64() - .5) * .15
		for j := 0; j <= steps; j++ {
			u := float64(j) / steps
			t := start + u*(end-start)
			theta := a + sweep*math.Sin(math.Pi*u) + .025*math.Sin(5*t+a*7)
			p := hair(t, theta)
			n := v(p.X/.095, (p.Y-1.548)/.123, (p.Z+.021)/.118).Unit()
			p = p.Add(n.Scale(off))
			side := v(math.Cos(theta), 0, -math.Sin(theta)).Scale(width * (.02 + .98*math.Sin(math.Pi*math.Pow(u, .77))))
			k := len(g.pts)
			g.pts = append(g.pts, p.Sub(side), p.Add(side))
			if j > 0 {
				g.faces = append(g.faces, [3]int{k - 2, k - 1, k}, [3]int{k - 1, k + 1, k})
			}
		}
	}
	return g
}

type studio struct{ sky.Gradient }

func (s studio) Light(d maths.Vec) maths.Light {
	l := s.Gradient.Light(d)
	if d.Z > 0 && math.Abs(d.X/d.Z+.40) < .15 && math.Abs(d.Y/d.Z-.48) < .19 {
		return rgb(5, 4.8, 4.6)
	}
	return l
}
func main() {
	w := flag.Int("width", 420, "frame width")
	out := flag.String("out", "portrait.png", "output PNG")
	obj := flag.String("obj", "studies/portrait/SMPLX-female.obj", "pinned sample OBJ")
	camp := flag.String("camp-mesh", "", "write the static camp GLB and exit")
	samples := flag.Int("samples", 1, "rays per pixel axis")
	scatter := flag.Bool("sss", true, "apply depth-aware screen-space subsurface diffusion")
	alley := flag.Bool("alley", false, "stage the full standing figure in a service alley")
	height := flag.Float64("height-scale", 1, "standing height multiplier")
	shoulders := flag.Float64("shoulder-scale", 1, "shoulder breadth multiplier")
	hips := flag.Float64("hip-scale", 1, "hip breadth multiplier")
	hair := flag.Bool("hair", true, "include scalp and lock geometry")
	pigment := flag.Float64("mark-strength", 1, "facial freckle strength multiplier (zero to one)")
	profile := flag.String("profile", "reference", "reference, compact, long-legged or broad-shouldered")
	style := flag.String("hairstyle", "swept", "rooted, layered, swept, curls or legacy-bob")
	look := flag.String("look-at", "", "world-space gaze target in metres, x,y,z")
	fall := flag.Float64("crown-fall", 0, "additional length in metres for strands from the frontal crown")
	frontLimit := flag.Float64("follicle-front", .98, "frontal follicle boundary down from crown in radians")
	templeLimit := flag.Float64("follicle-temple", 1.52, "temporal follicle boundary in radians")
	napeLimit := flag.Float64("follicle-nape", 1.93, "occipital follicle boundary in radians")
	poseName := flag.String("pose", "rest", "named full-figure stance")
	shot := flag.String("camera", "", "plate, figure, alley or portrait; an authored camera takes priority")
	identifier := flag.String("figure-name", "study", "figure name for its generated camera")
	origin := flag.String("at", "0,0,0", "figure placement in world metres, x,y,z")
	heading := flag.Float64("heading", 0, "figure heading in degrees about scene up")
	naturalSkin := flag.Bool("natural-skin", false, "soften the shading normals around broad skin planes")
	lightStudy := flag.String("light-study", "authored", "authored, bounce or albedo illumination for material diagnosis")
	softStudio := flag.Bool("soft-studio", false, "broaden the studio fill and rim sources")
	skinSheen := flag.String("skin-sheen", "regional", "study pass 2 surface response: regional, original, or matte")
	studyPass := flag.Int("study-pass", 0, "research comparison: 0 baseline, 1 pose, 2 pose, skin and hair")
	flag.Parse()
	if *camp != "" {
		must(writeCampMesh(*obj, *camp))
		return
	}
	if *studyPass < 0 || *studyPass > 2 {
		panic("study pass must be 0, 1 or 2")
	}
	if *skinSheen != "matte" && *skinSheen != "regional" && *skinSheen != "original" {
		panic("unknown skin sheen")
	}
	originPoint, e := vector(*origin)
	must(e)
	actor := maths.NewSubspace(originPoint, maths.Rotation(maths.YAxis, *heading*maths.Degree), 1)
	body, eyes := readObj(*obj)
	if *studyPass == 2 {
		body = refine(body)
		eyes = refine(eyes)
	}
	world := thing.NewWorld()
	rig := skeleton.New()
	must(rig.Add("neck", "", v(0, 1.36, -.02), v(0, 1.44, -.025), .03, .03))
	must(rig.Add("head", "neck", v(0, 1.44, -.025), v(0, 1.655, -.02), .04, .04))
	frames := rig.Frames(skeleton.Pose{}.With("head", skeleton.NewTurn(maths.YAxis, -.095)))
	carry := func(p maths.Vec) maths.Vec {
		head := maths.Smooth(1.38, 1.447, p.Y)
		return skeleton.Carry(p, frames, []skeleton.Influence{skeleton.NewInfluence(0, 1-head), skeleton.NewInfluence(1, head)})
	}
	gesture, ok := repertoire[*poseName]
	if !ok {
		panic("unknown pose")
	}
	figure := standardFigure()
	figure.SoftNormals = *naturalSkin
	figure.Anchors = map[string]func(maths.Vec) maths.Vec{"head": frames[1].Apply}
	if *alley || *shot == "figure" || *shot == "plate" {
		standingRig := articulated(body, *studyPass >= 1, gesture)
		carry = standingRig.Carry
		figure.Anchors = standingRig.Anchors
	}
	figure.Layered = *style == "layered"
	figure.Swept = *style == "swept" || figure.Layered
	if *style == "rooted" {
		if !(*frontLimit >= .6 && *frontLimit <= 1.18 && *templeLimit >= 1.26 && *templeLimit <= 1.64 && *napeLimit >= 1.65 && *napeLimit <= 2.08) {
			panic("follicle boundaries outside scalp")
		}
		if !(*fall >= 0 && *fall <= .11) {
			panic("crown fall must be between zero and .11m")
		}
		figure.CrownFall = *fall
		figure.Rooted = &follicleRegion{front: *frontLimit, temple: *templeLimit, nape: *napeLimit}
	}
	figure.Curls = *style == "curls"
	if figure.Rooted == nil && !figure.Curls && !figure.Swept && *style != "legacy-bob" {
		panic("unknown hairstyle")
	}
	var profileError error
	figure.Shape, profileError = bodyProfile(*profile)
	must(profileError)
	figure.Height, figure.Shoulders, figure.Hips, figure.Hair = *height, *shoulders, *hips, *hair
	if !(figure.Height > 0 && figure.Shoulders > 0 && figure.Hips > 0 && *pigment >= 0 && *pigment <= 1) {
		panic("positive proportions and mark strength in [0,1] required")
	}
	figure.Skin = func(p maths.Vec) maths.Light {
		c := skinWithMarks(p, *pigment)
		if *studyPass == 2 {
			c = complexion(p, *pigment)
		}
		if figure.Hair && figure.Rooted != nil {
			c = figure.Rooted.roots(p, c)
		}
		if *alley || *shot == "figure" || *shot == "plate" {
			return figureSkin(p, c)
		}
		return c
	}
	if *look != "" {
		target, e := vector(*look)
		must(e)
		figure.Gaze, e = solveGaze(actor.In(target), func(p maths.Vec) maths.Vec { return figure.fit(p, carry(p)) })
		must(e)
	}
	parts, err := figure.build(body, eyes, carry)
	must(err)
	placement := placeFigure(world, *identifier, parts, .8, actor)
	if !*alley {
		bg := rgb(.069, .094, .103)
		world.Sky = studio{sky.Gradient{Horizon: bg, Zenith: bg.Scale(2), Ground: bg}}
		for index, lamp := range []struct {
			dir maths.Vec
			c   maths.Light
		}{{v(-.65, .81, 1), rgb(2.3, 2.12, 1.98)}, {v(.55, .24, 1), rgb(.92, 1.02, 1.13)}, {v(.8, .7, -.5), rgb(1.1, .8, .57)}} {
			if index == 0 {
				world.Lamp(film.Sun{Toward: lamp.dir.Unit(), Strength: lamp.c, Disc: true, Size: .72})
			} else {
				if *softStudio {
					world.Lamp(film.Sun{Toward: lamp.dir.Unit(), Strength: lamp.c, Disc: true, Size: .95})
				} else {
					world.Lamp(thing.Sun(lamp.dir.Unit(), lamp.c))
				}
			}
		}
	}
	h := *w * 5 / 4
	portraitCamera := figureCamera{name: "portrait", from: v(.016, 1.543, .75), at: v(0, 1.532, 0), wide: .303}
	var explicit *figureCamera
	if *alley {
		from, at, wide := serviceAlley(world)
		if *shot == "alley" {
			explicit = &figureCamera{name: "alley", from: from, at: at, wide: wide}
		}
	}
	if *shot == "plate" {
		explicit = &figureCamera{name: "plate", from: v(.40, 1.13, 3.1), at: v(0, .94, 0), wide: .55}
		world.Lamps = nil
		world.Lamp(film.Flood{Strength: rgb(.34, .34, .34)})
		world.Lamp(film.Sun{Toward: v(-.38, .72, 1).Unit(), Strength: rgb(1.35, 1.29, 1.24), Disc: true, Size: .85})
		world.Lamp(film.Sun{Toward: v(.62, .82, .45).Unit(), Strength: rgb(.29, .33, .36), Disc: true, Size: 1.12})
		world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: rgb(.20, .20, .20)}})
	}
	if *shot == "portrait" || (*shot == "" && !*alley) {
		explicit = &portraitCamera
	}
	if *shot == "alley" && !*alley {
		panic("the alley camera requires the alley set")
	}
	if *shot != "" && *shot != "alley" && *shot != "portrait" && *shot != "figure" && *shot != "plate" {
		panic("unknown camera")
	}
	switch *lightStudy {
	case "authored":
	case "albedo":
		world.Lamps = nil
		world.Lamp(film.Flood{Strength: maths.Grey(.75)})
		world.Lamp(film.Flood{Up: maths.YAxis.Neg(), Strength: maths.Grey(.75)})
	case "bounce":
		world.Lamps = nil
		world.Lamp(film.Flood{Strength: rgb(.48, .485, .49)})
		world.Lamp(film.Sun{Toward: v(-.38, .75, 1).Unit(), Strength: rgb(.94, .90, .86), Disc: true, Size: .80})
	default:
		panic("unknown lighting study")
	}
	selected := placement.view(explicit)
	camera := selected.eye()
	fmt.Printf("figure camera %s; selected %s, from %v towards %v\n", placement.Camera.name, selected.name, selected.from, selected.at)
	lens, e := film.PinholeOf(selected.wide)
	must(e)
	if *studyPass == 2 {
		m := *world.Things[0].Material
		m.Gloss = .045
		m.Satin = .62
		m.Index = 1.40
		if *skinSheen != "original" {
			m.Gloss = 0
			m.Satin = .79
		}
		world.Things[0].Material = &m
	}
	frame, e := trace.Sampled(world, camera, lens, 0, *w, h, *samples)
	must(e)
	var moist *framebuffer.Frame
	if *studyPass == 2 && *skinSheen != "matte" {
		m := *world.Things[0].Material
		m.Gloss = .43
		m.Satin = .395
		world.Things[0].Material = &m
		moist, e = trace.Sampled(world, camera, lens, 0, *w, h, *samples)
		must(e)
	}
	if *scatter || moist != nil {

		// A white diffuse tissue and black occluders isolate incident diffuse light.
		for _, part := range world.Things {
			part.Material = &thing.Material{}
		}
		world.Things[0].Material = &thing.Material{Colour: maths.Grey(1)}
		irradiance, e := trace.Render(world, camera, lens, 0, *w, h)
		must(e)
		// Use the corresponding rest point of the visible skin intersection.
		// The front depth is from the same single-sample irradiation rays.
		albedo := framebuffer.New(*w, h)
		tissue := world.Things[0].Shape.(*mesh.Mesh)
		for y := 0; y < h; y++ {
			for x := 0; x < *w; x++ {
				ray := world.Things[0].At(0).RayIn(film.Look(camera, lens, x, y, *w, h, maths.Band{}))
				front := irradiance.At(x, y)
				if hit, ok := tissue.Meet(ray, 0, float64(front.Depth)+.000001); ok && math.Abs(hit.Along()-float64(front.Depth)) < .000001 {
					rest, _ := tissue.RestPointAt(hit)
					c := figure.Skin(rest)
					if moist != nil {
						index := y*(*w) + x
						dry := &frame.Pix[index]
						wet := moist.Pix[index]
						mix := float32(oilFilm(rest))
						if *skinSheen == "regional" {
							mix *= float32(.14 + .52*gauss(rest.X/.014)*gauss((rest.Y-1.522)/.017) + .36*gauss(rest.X/.020)*gauss((rest.Y-1.488)/.006))
						}
						dry.R += mix * (wet.R - dry.R)
						dry.G += mix * (wet.G - dry.G)
						dry.B += mix * (wet.B - dry.B)
					}
					albedo.Set(x, y, framebuffer.Pixel{R: float32(c.R), G: float32(c.G), B: float32(c.B), Cover: 1})
				}
			}
		}
		if *scatter {
			save(strings.TrimSuffix(*out, ".png")+"-surface.png", frame)
			profile := diffusion.Profile{Radius: [3]float64{.0031, .0014, .0007}, Share: .47}
			if *studyPass == 2 {
				profile = diffusion.Profile{Radius: [3]float64{.0025, .00105, .00053}, Share: .32}
			}
			must(diffusion.Apply(frame, irradiance, albedo, 2*math.Tan(lens.Wide/2), profile))
		}
	}
	save(*out, frame)
}
func save(path string, frame *framebuffer.Frame) {
	w, h := frame.W, frame.H
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	srgb := func(l float64) uint8 {
		if l <= .0031308 {
			return uint8(255*math.Max(0, 12.92*l) + .5)
		}
		return uint8(255*math.Min(1, 1.055*math.Pow(l, 1/2.4)-.055) + .5)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			l := frame.At(x, y).Light()
			img.SetNRGBA(x, y, color.NRGBA{srgb(l.R), srgb(l.G), srgb(l.B), 255})
		}
	}
	f, e := os.Create(path)
	must(e)
	must(png.Encode(f, img))
	must(f.Close())
	fmt.Println(path)
}

// lashes samples the actual aperture of each eye. Its upper and lower
// margins are the last visible eyeball hits before the lid occludes it.
func lashes(body, eyes geom) geom {
	skinMesh, e := mesh.New(body.pts, body.faces, nil, false)
	must(e)
	eyeMesh, e := mesh.New(eyes.pts, eyes.faces, nil, false)
	must(e)
	rng := rand.New(rand.NewSource(317))
	var hair geom
	for _, side := range []float64{-1, 1} {
		for _, upper := range []bool{false, true} {
			var last maths.Vec
			started := false
			for i := 0; i <= 95; i++ {
				x := .015 + .032*float64(i)/95
				origin := v(side*x, 0, .18)
				var anchor maths.Vec
				for k := 0; k < 100; k++ {
					offset := float64(k) * .00015
					if upper {
						origin.Y = 1.562 - offset
					} else {
						origin.Y = 1.543 + offset
					}
					ray := maths.Ray{At: origin, Dir: maths.ZAxis.Neg()}
					eyeHit, ok := eyeMesh.Meet(ray, 0, .20)
					if !ok {
						continue
					}
					if _, occluded := skinMesh.Meet(ray, 0, eyeHit.Along()); occluded {
						continue
					}
					anchor = ray.Point(eyeHit.Along())
					break
				}
				if anchor == (maths.Vec{}) {
					continue
				}
				outward := (x - .015) / .032
				// The wet, pigmented upper lid is continuous underneath the lashes.
				if upper && started {
					k := len(hair.pts)
					lift := v(0, .00042, .00034)
					in := v(0, -.00038, .00042)
					hair.pts = append(hair.pts, last.Add(lift), last.Add(in), anchor.Add(lift), anchor.Add(in))
					hair.faces = append(hair.faces, [3]int{k, k + 1, k + 2}, [3]int{k + 1, k + 3, k + 2})
				}
				started = true
				last = anchor
				if rng.Float64() > .84 || (!upper && i%2 == 1) {
					continue
				}
				length := (.0016 + .0019*rng.Float64()) * (.44 + .68*math.Sin(math.Pi*outward))
				dir := 1.
				if !upper {
					dir = -1
					length *= .60
				}
				root := anchor.Add(v(0, dir*.00012, .00042))
				for j := 0; j <= 5; j++ {
					u := float64(j) / 5
					lean := .1 + .7*outward
					p := root.Add(v(side*length*lean*u, dir*length*u*u, length*(1.6*u-.85*u*u)))
					thick := v(.00010*(1.01-u), 0, 0)
					k := len(hair.pts)
					hair.pts = append(hair.pts, p.Sub(thick), p.Add(thick))
					if j > 0 {
						hair.faces = append(hair.faces, [3]int{k - 2, k, k - 1}, [3]int{k - 1, k, k + 1})
					}
				}
			}
		}
	}
	return hair
}
