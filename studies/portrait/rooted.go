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

// Coordinates run from crown to an editable follicle boundary, in radians.
// The frontal limit is higher than the temporal and occipital limits.
type follicleRegion struct{ front, temple, nape float64 }

func (r follicleRegion) edge(a float64) float64 {
	front := math.Cos(a)
	if front >= 0 {
		return r.temple + (r.front-r.temple)*front*front
	}
	return r.temple + (r.nape-r.temple)*front*front
}

func rootedGroom(body geom, region follicleRegion, crownFall float64) []assembly.Part {
	skull, e := mesh.New(body.pts, body.faces, nil, false)
	must(e)
	centre := v(0, 1.551, -.025)
	surface := func(a, polar float64) (maths.Vec, maths.Vec) {
		d := v(.082*math.Sin(a)*math.Sin(polar), .109*math.Cos(polar), .103*math.Cos(a)*math.Sin(polar)).Unit()
		ray := maths.Ray{At: centre.Add(d.Scale(.22)), Dir: d.Neg()}
		hit, ok := skull.Meet(ray, 0, .22)
		if !ok {
			panic("follicle outside cranial surface")
		}
		n := skull.NormalAt(hit)
		if n.Dot(d) < 0 {
			n = n.Neg()
		}
		return ray.Point(hit.Along()), n
	}
	var hair geom
	var chart []maths.Vec
	rng := rand.New(rand.NewSource(4343))
	const steps = 16
	const ring = 4
	// Neighbours share a guide and retain individual roots, lengths and tones.
	for clump := 0; clump < 1050; clump++ {
		a := 2*math.Pi*rng.Float64() - math.Pi
		edge := region.edge(a)
		u := math.Acos(1-rng.Float64()*(1-math.Cos(edge))) / edge
		side := 1.
		if a < -.32+.09*math.Sin(4*u)+.11*(rng.Float64()-.5) {
			side = -1
		}
		flow := side * (.32 + 1.04*math.Pow(math.Max(0, math.Cos(a)), 2)) * (.8 + .24*rng.Float64())
		crown := (1 - maths.Smooth(.62, .91, u)) * maths.Smooth(.12, .70, math.Cos(a-.44))
		for fibre := 0; fibre < 5; fibre++ {
			ra := a + .038*(rng.Float64()-.5)
			rootPolar := math.Max(.001, math.Min(.997, u+.026*(rng.Float64()-.5))) * region.edge(ra)
			endA := ra + flow*(1-.78*u)
			extra := crownFall * crown
			if extra > 0 {
				endA += side * .34 * crown * maths.Smooth(0, .05, crownFall)
			}
			fall := .012 + .045*(1-math.Cos(endA))*.5
			lift := .0035 + .005*rng.Float64() + extra*.21
			end := .83 + .17*rng.Float64()
			thickness := .00033 + .00036*rng.Float64()
			shade := .20 + .74*rng.Float64()
			var points, normals [steps + 1]maths.Vec
			for j := 0; j <= steps; j++ {
				t := float64(j) / steps * end
				startFall := .84 - .19*maths.Smooth(0, .05, extra)
				track := math.Min(1, t/startFall)
				angle := ra + (endA-ra)*track
				polar := rootPolar + (region.edge(endA)-rootPolar)*track
				p, n := surface(angle, polar)
				// Lift smoothly out of the real follicle; tips graze the temporal scalp.
				p = p.Add(n.Scale(lift*math.Sin(math.Pi*.78*track) + .00008))
				if t > startFall {
					free := (t - startFall) / (1 - startFall)
					p = p.Add(v(side*.004, -fall-extra, -.006+extra*.18).Scale(free))
					p = p.Add(v(side*.15, 0, .10).Scale(extra * math.Sin(math.Pi*free)))
				}
				points[j], normals[j] = p, n
			}
			base := len(hair.pts)
			for j, p := range points {
				t := float64(j) / steps
				tangent := points[min(j+1, steps)].Sub(points[max(j-1, 0)]).Unit()
				across := tangent.Cross(normals[j]).Unit()
				up := across.Cross(tangent).Unit()
				width := thickness * (.22 + .78*maths.Smooth(0, .10, t)) * (1 - .95*maths.Smooth(.48, 1, t))
				for k := 0; k < ring; k++ {
					phase := 2 * math.Pi * float64(k) / ring
					hair.pts = append(hair.pts, p.Add(across.Scale(width*math.Cos(phase))).Add(up.Scale(.72*width*math.Sin(phase))))
					chart = append(chart, v(float64(k)/ring, t, shade))
					if j > 0 {
						i := base + (j-1)*ring + k
						h := base + (j-1)*ring + (k+1)%ring
						hair.faces = append(hair.faces, [3]int{i, h, i + ring}, [3]int{h, h + ring, i + ring})
					}
				}
			}
		}
	}
	return []assembly.Part{{Rest: hair.pts, Paint: chart, Faces: hair.faces, Material: &thing.Material{Name: "scalp-grown chestnut fibres", Texture: paint(func(p maths.Vec) maths.Light { return rgb(.115, .055, .033).Scale((.43 + .70*p.Z) * (.80 + .20*p.Y)) }), Gloss: .035, Satin: .62}}}
}

// Short follicles form a pigmented transition inside the coverage boundary.
// The narrow irregular part retains some of the original scalp colour.
func (r follicleRegion) roots(p maths.Vec, skin maths.Light) maths.Light {
	d := v(p.X/.082, (p.Y-1.551)/.109, (p.Z+.025)/.103)
	if d.Y < -.36 || d.Len() > 1.35 {
		return skin
	}
	d = d.Unit()
	a := math.Atan2(d.X, d.Z)
	polar := math.Acos(math.Max(-1, math.Min(1, d.Y)))
	grain := noise.Gradient(901, p.X*620, p.Y*620, p.Z*620)
	edge := r.edge(a) + .025*math.Sin(21*a) + .015*grain
	cover := maths.Smooth(-.035, .135, edge-polar)
	part := gauss((a+.32-.09*math.Sin(polar*4))/.075) * maths.Smooth(.10, .40, polar)
	c := rgb(.074, .036, .022).Scale(.82 + .42*grain)
	return mix(skin, c, cover*(1-.22*part))
}
