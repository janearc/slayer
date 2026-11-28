package main

import (
	"errors"
	"fmt"
	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"strconv"
	"strings"
	"time"
)

type proportionProfile struct{ leg, torso, waist, thigh, depth, shoulder, hip float64 }

func bodyProfile(name string) (proportionProfile, error) {
	switch name {
	case "reference":
		return proportionProfile{}, nil
	case "compact":
		return proportionProfile{leg: -.095, torso: -.014, waist: .116, thigh: .113, depth: .094, shoulder: -.03, hip: .085}, nil
	case "broad-shouldered":
		return proportionProfile{leg: -.02, torso: .07, waist: .01, thigh: .07, depth: .06, shoulder: .16, hip: -.04}, nil
	case "long-legged":
		return proportionProfile{leg: .115, torso: .02, waist: -.076, thigh: -.07, depth: -.055, shoulder: .095, hip: -.035}, nil
	default:
		return proportionProfile{}, fmt.Errorf("unknown body profile %q", name)
	}
}
func (s proportionProfile) fit(p, q maths.Vec) maths.Vec {
	// Apply length above the feet and below the neck. Head and palms retain size.
	q.Y += s.leg*min(max(p.Y, 0), .86) + s.torso*min(max(p.Y-.86, 0), .50)
	waist := gauss((p.Y - 1.035) / .125)
	thigh := gauss((p.Y-.68)/.15) * (1 - maths.Smooth(.23, .31, math.Abs(p.X)))
	q.X += s.waist*p.X*waist + s.thigh*(p.X-math.Copysign(.094, p.X))*thigh
	trunk := maths.Smooth(.62, .81, p.Y) * (1 - maths.Smooth(1.23, 1.35, p.Y))
	q.Z += s.depth * (p.Z + .023) * (trunk + .75*thigh)
	return q
}
func vector(s string) (maths.Vec, error) {
	fields := strings.Split(s, ",")
	if len(fields) != 3 {
		return maths.Vec{}, errors.New("use a scene point in metres as x,y,z")
	}
	var a [3]float64
	for i, k := range fields {
		n, e := strconv.ParseFloat(strings.TrimSpace(k), 64)
		if e != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return maths.Vec{}, errors.New("need finite scene coordinates")
		}
		a[i] = n
	}
	return v(a[0], a[1], a[2]), nil
}
func transpose(r maths.Orientation) maths.Orientation {
	return maths.Orientation{X: r.In(maths.XAxis), Y: r.In(maths.YAxis), Z: r.In(maths.ZAxis)}
}
func directionAngles(d maths.Vec) (float64, float64) {
	return math.Atan2(d.X, d.Z), math.Atan2(d.Y, math.Hypot(d.X, d.Z))
}
func yawPitch(yaw, pitch float64) maths.Orientation {
	return maths.Rotation(maths.YAxis, yaw).Then(maths.Rotation(maths.XAxis, -pitch))
}

type gazeSolution struct {
	neck       maths.Vec
	basis      maths.Orientation
	yaw, pitch float64
	eyes       [2]maths.Vec
	turn       [2]maths.Orientation
}

func headRestEye(i int) maths.Vec { return v(float64(2*i-1)*.030962, 1.552503, .05863) }
func solveGaze(target maths.Vec, place func(maths.Vec) maths.Vec) (*gazeSolution, error) {
	mid := v(0, 1.55250, .05863)
	base := place(mid)
	z := place(mid.Add(v(0, 0, .002))).Sub(base).Unit()
	x0 := place(mid.Add(v(.002, 0, 0))).Sub(base).Unit()
	y := z.Cross(x0).Unit()
	g := &gazeSolution{neck: place(v(0, 1.425, -.025)), basis: maths.Orientation{X: y.Cross(z), Y: y, Z: z}}
	d := target.Sub(base)
	if d.Len() < .025 {
		return nil, errors.New("look target is too close to the eyes")
	}
	a, b := directionAngles(g.basis.In(d))
	g.yaw = max(-.90, min(.90, .72*a))
	g.pitch = max(-.48, min(.48, .65*b))
	headBasis := g.basis.Then(yawPitch(g.yaw, g.pitch))
	for i := 0; i < 2; i++ {
		g.eyes[i] = g.head(place(headRestEye(i)), 1)
		a, b = directionAngles(headBasis.In(target.Sub(g.eyes[i])))
		local := yawPitch(max(-.49, min(.49, a)), max(-.36, min(.36, b)))
		// The painted iris apex sits slightly above the mesh's geometric pole.
		neutral := yawPitch(float64(1-2*i)*.0162, .077)
		g.turn[i] = headBasis.Then(local).Then(transpose(neutral)).Then(transpose(headBasis))
	}
	return g, nil
}
func (g *gazeSolution) head(p maths.Vec, w float64) maths.Vec {
	r := g.basis.Then(yawPitch(g.yaw*w, g.pitch*w))
	return g.neck.Add(r.Out(g.basis.In(p.Sub(g.neck))))
}
func (g *gazeSolution) eye(p maths.Vec, i int) maths.Vec {
	q := g.head(p, 1)
	return g.eyes[i].Add(g.turn[i].Out(q.Sub(g.eyes[i])))
}

type figureCamera struct {
	name     string
	from, at maths.Vec
	wide     float64
}

func (c figureCamera) eye() maths.Subspace {
	move, e := camera.Timed(c.from, c.from, time.Second, "linear")
	must(e)
	path, e := camera.New([]camera.Move{move}, c.at, maths.YAxis)
	must(e)
	eye, e := path.At(0)
	must(e)
	return *eye
}

type placementResult struct {
	Camera figureCamera
	Parts  []thing.Thing
}

func placeFigure(w *thing.World, name string, parts []thing.Thing, aspect float64, actor maths.Subspace) placementResult {
	// A fixed frontal three-quarter viewing direction with bounds-derived distance.
	lo, hi := v(math.Inf(1), math.Inf(1), math.Inf(1)), v(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	var corners []maths.Vec
	for i := range parts {
		part := parts[i]
		space := part.At(0).Within(actor)
		part.Motion = thing.Still(space)
		parts[i] = part
		b := part.Shape.Bounds()
		for _, x := range []float64{b.Min.X, b.Max.X} {
			for _, y := range []float64{b.Min.Y, b.Max.Y} {
				for _, z := range []float64{b.Min.Z, b.Max.Z} {
					c := space.Out(v(x, y, z))
					corners = append(corners, c)
					lo = v(min(lo.X, c.X), min(lo.Y, c.Y), min(lo.Z, c.Z))
					hi = v(max(hi.X, c.X), max(hi.Y, c.Y), max(hi.Z, c.Z))
				}
			}
		}
		w.Add(part)
	}
	at := lo.Add(hi).Scale(.5)

	wide := .60
	// Project all AABB corners in camera coordinates to account for depth and aspect.
	back := actor.Orientation.Out(v(.16, .045, 1).Unit())
	right := maths.YAxis.Cross(back).Unit()
	up := back.Cross(right)
	distance := 0.
	for _, c := range corners {
		p := c.Sub(at)
		distance = max(distance, p.Dot(back)+1.13*max(math.Abs(p.Dot(right)), aspect*math.Abs(p.Dot(up)))/math.Tan(wide/2))
	}
	return placementResult{Camera: figureCamera{name: name + "-camera", from: at.Add(back.Scale(distance)), at: at, wide: wide}, Parts: parts}
}
func (p placementResult) view(explicit *figureCamera) figureCamera {
	if explicit != nil {
		return *explicit
	}
	return p.Camera
}
