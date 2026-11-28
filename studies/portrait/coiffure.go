// A side part is a line on the scalp. Each lock flows from one side of
// that line to a point on the hairline; its distal tips can hang freely.
package main

import (
	"github.com/janearc/slayer/bioforms/assembly"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"math/rand"
)

func comb(u, t, side float64) maths.Vec {
	theta := 1.03 - 2.02*u
	root := v(-.265, math.Cos(theta), math.Sin(theta)).Unit()
	a := -.28*math.Pow(1-u, 2) + side*3.135*u
	front := (1 + math.Cos(a)) * .5
	edge := 1.433 + .165*math.Pow(front, 3.2) + .016*math.Sin(a)*front + .003*math.Sin(19*a)
	limit := math.Acos(max(-.23, min(.98, (edge-1.551)/.122)))
	end := v(math.Sin(a)*math.Sin(limit), math.Cos(limit), math.Cos(a)*math.Sin(limit))
	// A front lock sweeps out from the off-centre part, rather than down
	// toward the nose. Volume at the roots falls away at the temple.
	arc := math.Acos(max(-1, min(1, root.Dot(end))))
	q := root.Scale(math.Sin((1 - t) * arc)).Add(end.Scale(math.Sin(t * arc))).Scale(1 / math.Sin(arc))
	q.X += side * .095 * maths.Smooth(0, .16, u) * math.Sin(math.Pi*t) * math.Cos(u*math.Pi*.8)
	q.Z += .10 * math.Sin(math.Pi*t) * math.Sin(u*math.Pi)
	q = q.Unit()
	loft := 1 + .054*math.Sin(math.Pi*t)*(1-.68*u)
	wave := maths.Smooth(.25, .96, t) * math.Sin(math.Pi*u)
	swing := .008 * wave * math.Sin(7.1*t+5.3*u+side)
	return v(.093*loft*q.X+side*swing, 1.551+.121*loft*q.Y+min(0, edge-(1.551-.23*.122))*maths.Smooth(.52, 1, t), -.021+.117*loft*q.Z+.009*wave*math.Sin(6.6*t+4.2*u))
}
func lockTexture(p maths.Vec) maths.Light {
	wave := noise.Gradient(72, p.X*9, p.Y*3, p.Z) + .45*noise.Gradient(82, p.X*30, p.Y*4, p.Z)
	fine := .5 + .5*math.Sin(84*p.X+.9*math.Sin(p.Y*9+p.Z))
	return rgb(.097, .039, .020).Scale((.49 + .50*p.Z) * (.84 + .30*wave + .16*fine))
}
func coiffure(body geom, layered bool) []assembly.Part {
	rng := rand.New(rand.NewSource(479))
	var parts []assembly.Part
	for _, side := range []float64{-1, 1} {
		var locks geom
		var pigment []maths.Vec
		add := func(p, uv maths.Vec) { locks.pts = append(locks.pts, p); pigment = append(pigment, uv) }
		count := 88
		guide := comb
		if layered {
			count = 62
			guide = layeredGuide
		}
		const steps = 36
		const cross = 9
		for lock := 0; lock < count; lock++ {
			u := (float64(lock) + .5) / float64(count)
			width := (.85 + .45*rng.Float64()) / float64(count)
			length := .92 + .14*rng.Float64() + .042*math.Sin(16*u+side)
			if layered {
				u += .35 / float64(count) * math.Sin(31*u+side)
				width *= 1 + .30*math.Sin(18*u+side)
				length += .05 * math.Sin(21*u+.7*side)
			}
			bend := rng.Float64() - .5
			shade := .25 + .75*rng.Float64()
			base := len(locks.pts)
			for j := 0; j <= steps; j++ {
				t := float64(j) / steps
				// Locks overlap near the scalp and separate into tapered tips.
				taper := 1 - .94*maths.Smooth(.78, 1, t)
				for k := 0; k <= cross; k++ {
					s := float64(k) / cross
					du := (2*s - 1) * width * taper
					track := u + du + .011*bend*math.Sin(2*math.Pi*t) + .007*math.Sin(6*u+5*t)*t*t
					p := guide(track, .006+t*length, side)
					normal := v(p.X/.093, (p.Y-1.551)/.121, (p.Z+.021)/.117).Unit()
					rib := (.00035 + .0014*math.Sin(math.Pi*s)) * (.26 + .74*math.Sin(math.Pi*t))
					p = p.Add(normal.Scale(rib + .001*math.Sin(35*u+side)))
					p.Y -= .016 * maths.Smooth(.78, 1, t) * math.Pow(math.Sin(math.Pi*u), 2)
					add(p, v(s, t, shade))
					if j > 0 && k > 0 {
						n := base + j*(cross+1) + k
						aa, bb, cc, dd := n-cross-2, n-cross-1, n, n-1
						if side > 0 {
							aa, cc = cc, aa
						}
						locks.faces = append(locks.faces, [3]int{aa, bb, cc}, [3]int{aa, cc, dd})
					}
				}
			}
		}
		finish := &thing.Material{Name: "swept auburn locks", Texture: paint(lockTexture), Gloss: .036}
		if layered {
			finish.Texture = paint(shaftColour)
			finish.Gloss = .075
			finish.Satin = .47
		}
		parts = append(parts, assembly.Part{Rest: locks.pts, Paint: pigment, Faces: locks.faces, Material: finish})
	}
	// The inner layer is close to the skull. Receding ends let the
	// individual lock tips and their cast shadows describe the hairline.
	var under geom
	const cols = 96
	const rows = 25
	for _, side := range []float64{-1, 1} {
		base := len(under.pts)
		for j := 0; j <= rows; j++ {
			for i := 0; i <= cols; i++ {
				p := comb(float64(i)/cols, .004+.94*float64(j)/rows, side)
				if layered {
					p = layeredGuide(float64(i)/cols, .0001+.957*float64(j)/rows, side)
				}
				n := v(p.X/.093, (p.Y-1.551)/.121, (p.Z+.021)/.117).Unit()
				under.pts = append(under.pts, p.Sub(n.Scale(.0011)))
				if i > 0 && j > 0 {
					k := base + j*(cols+1) + i
					a, b, c, d := k-cols-2, k-cols-1, k, k-1
					if side > 0 {
						a, c = c, a
					}
					under.faces = append(under.faces, [3]int{a, b, c}, [3]int{a, c, d})
				}
			}
		}
	}
	parts = append(parts, assembly.Part{Rest: under.pts, Faces: under.faces, Material: &thing.Material{Name: "hair underlayer", Colour: rgb(.062, .024, .014), Gloss: .12}})
	roots, _, _ := scalp(body)
	parts = append(parts, roots)
	if layered {
		parts = append(parts, partAndFlyaways()...)
	}
	return parts
}

// curls grow against the actual scalp surface. Small groups of neighbouring
// fibres share a curved guide but have individual roots, phase and taper.
type scalpSurface func(float64, float64) (maths.Vec, maths.Vec)

func scalp(body geom) (assembly.Part, scalpSurface, func(float64) float64) {
	skull, e := mesh.New(body.pts, body.faces, nil, false)
	must(e)
	center := v(0, 1.551, -.025)
	edge := func(a float64) float64 {
		front := math.Cos(a)
		return 1.78 - .66*front + .038*math.Sin(3*a+.9) + .012*math.Sin(15*a)
	}
	surface := func(a, t float64) (maths.Vec, maths.Vec) {
		angle := max(.0005, t) * edge(a)
		n := v(math.Sin(angle)*math.Sin(a), math.Cos(angle), math.Sin(angle)*math.Cos(a))
		// Ellipsoidal sampling keeps the cranial coverage even as the frontal
		// hairline rises above the temples.
		dir := v(n.X*.082, n.Y*.109, n.Z*.103).Unit()
		ray := maths.Ray{At: center.Add(dir.Scale(.25)), Dir: dir.Neg()}
		hit, ok := skull.Meet(ray, 0, .25)
		if !ok {
			panic("curl root missed scalp")
		}
		p := ray.Point(hit.Along())
		return p, dir
	}
	under := geom{}
	const columns = 100
	const rows = 32
	for j := 0; j <= rows; j++ {
		for i := 0; i < columns; i++ {
			a := 2 * math.Pi * float64(i) / columns
			p, n := surface(a, float64(j)/rows)
			under.pts = append(under.pts, p.Add(n.Scale(.0007)))
			if j > 0 {
				k := (j-1)*columns + i
				l := (j-1)*columns + (i+1)%columns
				under.faces = append(under.faces, [3]int{k, k + columns, l}, [3]int{l, k + columns, l + columns})
			}
		}
	}
	return assembly.Part{Rest: under.pts, Faces: under.faces, Material: &thing.Material{Name: "short auburn roots", Colour: rgb(.055, .022, .013), Gloss: .03}}, surface, edge
}
func curls(body geom) []assembly.Part {
	rng := rand.New(rand.NewSource(8163))
	roots, surface, edge := scalp(body)
	parts := []assembly.Part{roots}
	var strands geom
	var chart []maths.Vec
	const groups = 640
	const fibres = 5
	const steps = 24
	const ring = 3
	for i := 0; i < groups; i++ {
		a := math.Mod(float64(i)*2.3999632297+.050*(rng.Float64()-.5), 2*math.Pi)
		u := math.Acos(1-rng.Float64()*(1-math.Cos(edge(a)))) / edge(a)
		root, n := surface(a, u)
		tangent := v(.82-.54*n.X, -.32-.87*math.Max(0, -n.Y), -.29-.16*n.Z)
		tangent = tangent.Sub(n.Scale(tangent.Dot(n))).Unit()
		side := n.Cross(tangent).Unit()
		radius := .0022 + .0035*rng.Float64()
		rise := .002 + .008*rng.Float64()
		length := .014 + .016*rng.Float64()
		if u > .88 {
			length *= .80
		}
		phase := 2 * math.Pi * rng.Float64()
		twist := 4.5 + 4.5*rng.Float64()
		shade := .15 + .78*rng.Float64()
		for f := 0; f < fibres; f++ {
			base := len(strands.pts)
			shift := (rng.Float64() - .5) * .75
			offset := (rng.Float64() - .5) * .0033
			thickness := .00024 + .00053*rng.Float64()
			if f == 0 {
				thickness = .0012
			}
			for j := 0; j <= steps; j++ {
				t := float64(j) / steps
				ang := phase + shift + twist*t
				curl := radius * (.45 + .55*math.Sin(math.Pi*t))
				p := root.Add(tangent.Scale(length * t)).Add(side.Scale(offset + curl*(math.Sin(ang)-math.Sin(phase+shift)))).Add(n.Scale(.00065 + rise*math.Sin(math.Pi*t) + curl*(1-math.Cos(ang))))
				width := thickness * (1 - .92*maths.Smooth(.56, 1, t))
				for k := 0; k < ring; k++ {
					q := 2 * math.Pi * float64(k) / ring
					strands.pts = append(strands.pts, p.Add(side.Scale(width*math.Cos(q))).Add(n.Scale(width*math.Sin(q))))
					chart = append(chart, v(float64(k)/ring, t, shade))
					if j > 0 {
						i0 := base + (j-1)*ring + k
						i1 := base + (j-1)*ring + (k+1)%ring
						strands.faces = append(strands.faces, [3]int{i0, i1, i0 + ring}, [3]int{i1, i1 + ring, i0 + ring})
					}
				}
			}
		}
	}
	parts = append(parts, assembly.Part{Rest: strands.pts, Paint: chart, Faces: strands.faces, Material: &thing.Material{Name: "curl fibres", Texture: paint(func(p maths.Vec) maths.Light {
		return rgb(.127, .051, .023).Scale((.37 + .86*p.Z) * (.71 + .25*math.Sin(p.Y*3+.7) + .10*p.X))
	}), Gloss: .15}})
	return parts
}
