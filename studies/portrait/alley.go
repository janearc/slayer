// A standing figure and a wet brick service passage. Measurements are metres.
// The rig stays in rest space so future clothing can use the same skin binding.
package main

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

type figureRig struct {
	Carry   func(maths.Vec) maths.Vec
	Anchors map[string]func(maths.Vec) maths.Vec
}

func standing(body geom) figureRig { return standingPose(body, false) }
func standingPose(body geom, relaxed bool) figureRig {
	return articulated(body, relaxed, stance{})
}
func articulated(body geom, relaxed bool, s stance) figureRig {
	rig := skeleton.New()
	add := func(name, parent string, a, b maths.Vec) int {
		must(rig.Add(name, parent, a, b, .025, .025))
		i, _ := rig.Find(name)
		return i
	}
	pelvis := add("pelvis", "", v(0, .88, -.02), v(0, 1.03, -.02))
	spine := add("spine", "pelvis", v(0, .96, -.02), v(0, 1.36, -.02))
	head := add("head", "spine", v(0, 1.42, -.025), v(0, 1.65, -.025))
	pose := skeleton.Pose{}.With("head", skeleton.NewTurn(maths.YAxis, .04)).With("spine", skeleton.NewTurn(maths.ZAxis, -.075))
	type limb struct{ arm, fore, hand, thigh, shin, foot int }
	sides := map[int]limb{}
	for _, side := range []int{-1, 1} {
		sign := float64(side)
		label := fmt.Sprint(side)
		lateral := func(x, y, z float64) maths.Vec { return v(sign*x, y, z) }
		armName, foreName, handName := "upper arm"+label, "forearm"+label, "hand"+label
		thighName, shinName, footName := "thigh"+label, "shin"+label, "foot"+label
		arm := add(armName, "spine", lateral(.165, 1.325, -.04), lateral(.427, 1.29, -.052))
		fore := add(foreName, armName, lateral(.427, 1.29, -.052), lateral(.646, 1.29, -.05))
		hand := add(handName, foreName, lateral(.646, 1.29, -.05), lateral(.81, 1.29, -.07))
		thigh := add(thighName, "pelvis", lateral(.088, .86, -.02), lateral(.086, .465, -.02))
		shin := add(shinName, thighName, lateral(.086, .465, -.02), lateral(.097, .075, -.048))
		foot := add(footName, shinName, lateral(.097, .075, -.048), lateral(.11, .03, .11))
		sides[side] = limb{arm, fore, hand, thigh, shin, foot}
		drop, bend, hip, knee, footTurn := 1.30, .24, .035, .025, -.06
		if side < 0 {
			drop, bend, hip, knee, footTurn = 1.30, .38, -.26, .20, .06
		}
		roll := -.42
		if relaxed {
			if side > 0 {
				drop, bend, hip, knee, footTurn, roll = 1.22, .61, .025, .018, -.043, -.035
			} else {
				drop, bend, hip, knee, footTurn, roll = 1.28, .55, -.34, .39, .05, .045
			}
		}
		pose = pose.With(armName, skeleton.NewTurn(maths.ZAxis, -sign*drop)).
			With(foreName, skeleton.NewTurn(maths.YAxis, -sign*bend)).
			With(handName, skeleton.NewTurn(maths.XAxis, roll)).
			With(thighName, skeleton.NewTurn(maths.XAxis, hip)).
			With(shinName, skeleton.NewTurn(maths.XAxis, knee)).
			With(footName, skeleton.NewTurn(maths.XAxis, footTurn))
		if s != (stance{}) {
			pose = s.articulate(pose, side)
		}
	}
	pose = pose.Compose("head", skeleton.NewTurn(maths.YAxis, s.head)).Compose("spine", skeleton.NewTurn(maths.ZAxis, s.lean)).Compose("spine", skeleton.NewTurn(maths.YAxis, s.twist)).Compose("spine", skeleton.NewTurn(maths.XAxis, s.pitch))
	frames := rig.Frames(pose)
	place := func(p maths.Vec) maths.Vec {
		side := 1
		if p.X < 0 {
			side = -1
		}
		l := sides[side]
		x := math.Abs(p.X)
		arm := maths.Smooth(.114, .212, x) * maths.Smooth(1.205, 1.275, p.Y)
		if x > .22 && p.Y > 1.18 {
			arm = 1
		}
		neck := maths.Smooth(1.377, 1.447, p.Y)
		leg := (1 - maths.Smooth(.775, .94, p.Y))
		fore := maths.Smooth(.388, .461, x)
		hand := maths.Smooth(.616, .671, x)
		shin := 1 - maths.Smooth(.42, .5, p.Y)
		foot := 1 - maths.Smooth(.055, .11, p.Y)
		trunk := 1 - leg - arm
		back := maths.Smooth(.89, 1.09, p.Y)
		w := []skeleton.Influence{}
		for _, joint := range []struct {
			i int
			w float64
		}{{pelvis, trunk * (1 - back)}, {spine, trunk * back * (1 - neck)}, {head, trunk * back * neck}, {l.arm, arm * (1 - fore)}, {l.fore, arm * fore * (1 - hand)}, {l.hand, arm * fore * hand}, {l.thigh, leg * (1 - shin)}, {l.shin, leg * shin * (1 - foot)}, {l.foot, leg * shin * foot}} {
			w = append(w, skeleton.NewInfluence(joint.i, joint.w))
		}
		rest := p
		if x > .688 && p.Y > 1.18 {
			// Relax the fingers about the palm crease. Curl increases toward the tips.
			angle := .65 * maths.Smooth(.687, .81, x)
			pivot := v(float64(side)*.688, 1.279, p.Z)
			if relaxed {
				// Bend the finger centreline by arc length from its palm crease.
				// A point on the back of a finger retains its local thickness.
				digit := maths.Smooth(.006, .102, -p.Z)
				curvature := 2.1 + 1.9*digit
				d := x - .688
				theta := curvature * d
				thickness := p.Y - pivot.Y
				rest.X = float64(side) * (.688 + math.Sin(theta)/curvature + thickness*math.Sin(theta))
				rest.Y = pivot.Y - (1-math.Cos(theta))/curvature + thickness*math.Cos(theta)
				rest.Z = p.Z + (-.058-p.Z)*.18*maths.Smooth(.685, .79, x)
			} else {
				rest = skeleton.TurnAbout(pivot, maths.ZAxis, -float64(side)*angle).Apply(p)
			}
		}
		q := skeleton.Carry(rest, frames, w)
		// The free leg sets down ahead and to the side; the chest balances the hip.
		if side < 0 {
			q.X -= .10 * (1 - maths.Smooth(.04, .86, p.Y))
		}
		q.X += .043 * maths.Smooth(.10, .87, p.Y)
		if relaxed {
			// Shift weight over the planted hip while keeping the planted sole fixed.
			height := maths.Smooth(.09, .79, p.Y)
			q.X += .051*height - .016*maths.Smooth(1.1, 1.44, p.Y)
			hipTilt := gauss((p.Y - .83) / .24)
			q.Y += .087 * p.X * hipTilt
			if side < 0 {
				q.X += .022 * gauss((p.Y-.46)/.17)
			}
			// Lower the lateral clavicle, deltoid and attached arm continuously.
			girdle := maths.Smooth(.04, .175, x) * maths.Smooth(1.16, 1.32, p.Y)
			q.Y -= (.026 + .006*float64(side)) * girdle
			q.Z += (.006 - .008*float64(side)) * girdle
		}
		return q
	}
	floor := map[int]float64{-1: math.Inf(1), 1: math.Inf(1)}
	for _, p := range body.pts {
		if p.Y < .12 {
			side := 1
			if p.X < 0 {
				side = -1
			}
			floor[side] = math.Min(floor[side], place(p).Y)
		}
	}
	fmt.Printf("sole offsets before grounding (m): left %.5f, right %.5f\n", floor[-1], floor[1])
	turn := maths.Rotation(maths.YAxis, -.30)
	carry := func(p maths.Vec) maths.Vec {
		side := 1
		if p.X < 0 {
			side = -1
		}
		q := place(p)
		upper := maths.Smooth(.12, .82, p.Y)
		q.Y -= floor[side] * (1 - upper)
		if s != (stance{}) {
			q.Y -= .5 * (floor[-1] + floor[1]) * upper
			q.X += float64(side)*s.footSpread*(1-upper) + s.shift*upper
		}
		return turn.Out(q)
	}
	anchors := make(map[string]func(maths.Vec) maths.Vec)
	for name, joint := range map[string]int{"head": head, "left-wrist": sides[1].hand, "right-wrist": sides[-1].hand, "pelvis": pelvis} {
		origin := rig.Bone(joint).Head()
		target := carry(origin)
		anchors[name] = func(p maths.Vec) maths.Vec { return target.Add(turn.Out(frames[joint].Direction(p.Sub(origin)))) }
	}
	return figureRig{Carry: carry, Anchors: anchors}

}

func stone(p maths.Vec, scale float64) float64 {
	return noise.Gradient(57, p.X*scale, p.Y*scale, p.Z*scale)
}
func sootBrick(p maths.Vec) maths.Light {
	// Running bond on each wall. Streaks are long in gravity's direction.
	u := p.Z
	if p.Z < -5.90 {
		u = p.X
	}
	row := math.Floor(p.Y / .083)
	u += .117*math.Mod(row, 2) + .0018*stone(p, 48)
	course := p.Y / .083
	brick := u / .234
	distance := math.Min(.083*math.Abs(course-math.Round(course)), .234*math.Abs(brick-math.Round(brick)))
	shade := .81 + .28*noise.Gradient(41, math.Floor(brick)*1.41, row*.87, .39) + .22*stone(p, 79) + .11*stone(p, 438)
	damp := .62 + .28*maths.Smooth(.05, 1.23, p.Y) + .13*noise.Gradient(89, u*10, p.Y*1.4, 0)
	fired := rgb(.175, .082, .061).Scale(shade * damp)
	mortar := rgb(.14, .132, .116).Scale(.72 + .25*stone(p, 160))
	streak := noise.Gradient(91, u*9, p.Y*.8, 0) + .37*noise.Gradient(92, u*37, p.Y*1.4, 0)
	return mix(mortar.Scale(.67), fired, maths.Smooth(.0022, .0053, distance)).Scale(.93 - .27*maths.Smooth(-.4, .6, streak))
}
func pavement(p maths.Vec) maths.Light {
	grain := .08*stone(p, 460) + .21*stone(p, 113) + .25*stone(p, 23) + .18*stone(p, 3)
	u := p.X + .43*math.Mod(math.Floor(p.Z/.62), 2)
	gap := math.Min(math.Abs(u/.84-math.Round(u/.84))*.84, math.Abs(p.Z/.62-math.Round(p.Z/.62))*.62)
	return rgb(.112, .121, .124).Scale((.74 + grain) * (.34 + .66*maths.Smooth(.001, .009, gap)))
}

// pipeFrame spans the cross-section of a conduit in any direction.
func pipeFrame(direction maths.Vec) (maths.Vec, maths.Vec) {
	axis := direction.Unit()
	u := axis.Cross(maths.ZAxis)
	if u.Len() < .01 {
		u = axis.Cross(maths.YAxis)
	}
	u = u.Unit()
	return u, axis.Cross(u)
}
func dryFooting(x, z float64) bool { return math.Abs(x) < .3 && math.Abs(z) < .36 }
func serviceAlley(w *thing.World) (maths.Vec, maths.Vec, float64) {
	material := func(name string, c maths.Light, gloss float64) *thing.Material {
		return &thing.Material{Name: name, Colour: c, Gloss: gloss}
	}
	masonry := &thing.Material{Name: "soot-darkened brick and lime", Texture: paint(sootBrick)}
	floor := &thing.Material{Name: "damp flagstones", Texture: paint(pavement), Gloss: .26}
	iron := material("oxide-black iron", rgb(.057, .067, .069), .34)
	flaking := &thing.Material{Name: "weathered bottle-green door", Texture: paint(func(p maths.Vec) maths.Light {
		return mix(rgb(.056, .098, .094), rgb(.18, .113, .066), maths.Smooth(.07, .65, stone(p, 35)+.48*stone(p, 120)))
	}), Gloss: .18}
	concrete := material("worn limestone", rgb(.18, .178, .15), .12)
	glass := &thing.Material{Name: "frosted amber glass", Glow: rgb(4.2, 2.25, .89)}
	blueGlass := &thing.Material{Name: "lit workshop window", Shine: paint(func(p maths.Vec) maths.Light {
		return rgb(.11, .24, .28).Scale(.63 + .33*stone(p, 7) + .17*stone(p, 70) + .15*noise.Gradient(9, p.X*23, p.Y*.8, 0))
	}), Gloss: .09}
	groups := map[*thing.Material]*geom{}
	var order []*thing.Material
	group := func(m *thing.Material) *geom {
		if groups[m] == nil {
			groups[m] = &geom{}
			order = append(order, m)
		}
		return groups[m]
	}
	box := func(m *thing.Material, x, y, z, dx, dy, dz float64) {
		g := group(m)
		k := len(g.pts)
		for _, a := range [][3]float64{{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1}, {-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1}} {
			g.pts = append(g.pts, v(x+a[0]*dx/2, y+a[1]*dy/2, z+a[2]*dz/2))
		}
		for _, q := range [][4]int{{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 4, 7, 3}, {1, 2, 6, 5}, {3, 7, 6, 2}, {0, 1, 5, 4}} {
			g.faces = append(g.faces, [3]int{k + q[0], k + q[1], k + q[2]}, [3]int{k + q[0], k + q[2], k + q[3]})
		}
	}
	pipe := func(m *thing.Material, a, b maths.Vec, r float64) {
		g := group(m)
		k := len(g.pts)
		u, vv := pipeFrame(b.Sub(a))
		for _, p := range []maths.Vec{a, b} {
			for j := 0; j < 12; j++ {
				t := float64(j) * math.Pi / 6
				g.pts = append(g.pts, p.Add(u.Scale(r*math.Cos(t))).Add(vv.Scale(r*math.Sin(t))))
			}
		}
		for j := 0; j < 12; j++ {
			i := k + j
			n := k + (j+1)%12
			g.faces = append(g.faces, [3]int{i, n, i + 12}, [3]int{n, n + 12, i + 12})
		}
	}
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: floor})
	box(masonry, -1.15, 2.4, -1.65, .25, 4.8, 10)
	box(masonry, 1.39, 2.4, -1.65, .25, 4.8, 10)
	box(masonry, .1, 2.4, -6.1, 3.4, 4.8, .27)
	// The raised wall plinth, salt-worn rails and doorway establish scale.
	for _, x := range []float64{-1.018, 1.255} {
		box(concrete, x, .078, -1.6, .08, .155, 9.0)
	}
	for _, x := range []float64{-1.00, 1.22} {
		for _, z := range []float64{-.91, -3.93} {
			s := math.Copysign(1, x)
			box(iron, x, .99, z, .08, 1.98, .91)
			box(flaking, x-.045*s, .93, z, .04, 1.84, .79)
			for _, y := range []float64{.45, 1.21} {
				box(iron, x-.069*s, y, z, .014, .50, .64)
				box(flaking, x-.080*s, y, z, .012, .45, .59)
			}
			box(concrete, x-.048*s, .054, z, .20, .105, .96)
			box(iron, x-.096*s, .92, z+.31, .057, .037, .112)
		}
	}
	for _, z := range []float64{-1.82, -3.19, .85} {
		pipe(iron, v(-.961, .08, z), v(-.961, 4, z), .035)
		for _, y := range []float64{.42, 1.36, 2.05} {
			pipe(iron, v(-.961, y, z), v(-.961, y+.041, z), .044)
		}
	}
	for _, x := range []float64{-.62, .09, .81} {
		box(iron, x, 1.62, -5.93, .63, 1.05, .09)
		box(blueGlass, x, 1.62, -5.865, .54, .94, .035)
		box(iron, x, 1.62, -5.832, .029, .96, .019)
		for _, y := range []float64{1.45, 1.78} {
			box(iron, x, y, -5.831, .57, .032, .021)
		}
	}
	// A grated gully runs to the downpipe at the doorway. Utility boxes
	// and conduit break up the long wall at an ordinary person's scale.
	box(iron, -.70, .002, .51, .23, .004, .42)
	for j := 0; j < 12; j++ {
		box(concrete, -.70, .006, .316+.034*float64(j), .212, .008, .011)
	}
	box(iron, -.986, 1.36, -.22, .072, .26, .18)
	box(flaking, -.94, 1.36, -.22, .027, .221, .157)
	pipe(iron, v(-.99, 1.485, -.22), v(-.99, 2.65, -.22), .009)
	for _, z := range []float64{-3.55, -4.05} {
		cx := .94
		pipe(iron, v(cx, .014, z), v(cx, .65, z), .196)
		for _, y := range []float64{.031, .25, .55, .65} {
			pipe(iron, v(cx, y-.009, z), v(cx, y+.009, z), .205)
		}
		box(iron, cx, .66, z, .365, .018, .30)
	}
	// Two caged sconces pair visible luminous glass with inverse-square sources.
	lamp := func(x, y, z float64) {
		s := math.Copysign(1, x)
		box(iron, x+.035*s, y, z, .076, .23, .13)
		cx := x - .085*s
		pipe(iron, v(x+.03*s, y, z), v(cx, y, z), .038)
		pipe(glass, v(cx, y-.095, z), v(cx, y+.096, z), .060)
		for _, yy := range []float64{-.110, -.038, .038, .110} {
			r := .065
			dy := .010
			if math.Abs(yy) > .08 {
				r = .079
				dy = .024
			}
			pipe(iron, v(cx, y+yy-dy/2, z), v(cx, y+yy+dy/2, z), r)
		}
		for j := 0; j < 6; j++ {
			angle := float64(j) * math.Pi / 3
			ox, oz := .065*math.Cos(angle), .065*math.Sin(angle)
			pipe(iron, v(cx+ox, y-.115, z+oz), v(cx+ox, y+.115, z+oz), .003)
		}

	}
	lamp(-.91, 1.76, .12)
	lamp(1.13, 1.85, -3.08)
	spot := func(p, target maths.Vec, strength maths.Light) {
		w.Lamp(film.Spot{At: p, Aim: target.Sub(p).Unit(), Strength: strength, Cone: 1.62, Soft: .35})
	}
	for _, offset := range []float64{-.065, 0, .065} {
		spot(v(-.737, 1.79+offset, .17+offset), v(.05, .98, .16), rgb(.74, .49, .33))
	}
	spot(v(-.49, 2.07, .91), v(0, 1.12, 0), rgb(1.50, 1.00, .67))
	spot(v(.98, 1.86, -3.08), v(.2, .45, -2.7), rgb(1.52, .78, .38))
	// Skylight catches the hair and shoulders from the mouth of the alley.
	spot(v(.91, 2.26, -1.06), v(0, 1.05, 0), rgb(2.8, 4.75, 6.4))
	spot(v(.22, 1.80, 2.22), v(0, 1.12, 0), rgb(2.0, 2.28, 2.65))
	w.Lamp(film.Flood{Strength: rgb(.058, .085, .12)})
	for i := 0; i < 5; i++ {
		z := -1.2 - float64(i)*.88
		for j := 0; j < 14; j++ {
			wire := func(u float64) maths.Vec {
				return v(-1.03+2.35*u, 2.75+.18*math.Cos(float64(i))-.46*math.Sin(math.Pi*u), z+.19*u)
			}
			pipe(iron, wire(float64(j)/14), wire(float64(j+1)/14), .009)
		}
	}
	// Distinct wet patches reflect the lamps over rough stone.
	wet := material("standing rainwater", rgb(.035, .05, .056), .93)
	wet.Index = 1.333
	rng := rand.New(rand.NewSource(726))
	for n := 0; n < 46; n++ {
		x, z := (rng.Float64()-.5)*2.25, rng.Float64()*6.6-5.0
		// Keep the contact patch dry; divert its runoff into the curb channel.
		if dryFooting(x, z) || n%3 != 0 {
			x = math.Copysign(.76+.20*rng.Float64(), x)
		}
		a, b := .055+.21*rng.Float64(), .12+.41*rng.Float64()
		g := group(wet)
		k := len(g.pts)
		g.pts = append(g.pts, v(x, .0012, z))
		for j := 0; j < 28; j++ {
			t := float64(j) * 2 * math.Pi / 28
			mod := .72 + .18*math.Sin(3*t+.3*float64(n)) + .1*rng.Float64()
			g.pts = append(g.pts, v(x+a*mod*math.Cos(t), .0012, z+b*mod*math.Sin(t)))
		}
		for j := 0; j < 28; j++ {
			g.faces = append(g.faces, [3]int{k, k + 1 + (j+1)%28, k + 1 + j})
		}
	}
	for _, m := range order {
		g := groups[m]
		surface, e := mesh.New(g.pts, g.faces, nil, false)
		must(e)
		w.Add(thing.Thing{Shape: surface, Material: m})
	}
	w.Sky = sky.Gradient{Horizon: rgb(.011, .022, .036), Zenith: rgb(.084, .13, .18), Ground: rgb(.006, .011, .017)}
	return v(.52, 1.02, 2.75), v(-.13, .89, -.14), .630
}
