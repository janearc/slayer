package lights

import "math"

type Vec struct{ X, Y, Z float64 }

// Add is the sum of a and b.
func (a Vec) Add(b Vec) Vec { return Vec{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

// Sub is a minus b, the vector that leads from b to a.
func (a Vec) Sub(b Vec) Vec { return Vec{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

// Scale is a with every part multiplied by s.
func (a Vec) Scale(s float64) Vec { return Vec{a.X * s, a.Y * s, a.Z * s} }

// Dot is the dot product of a and b, a single number.
func (a Vec) Dot(b Vec) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// Unit is a shortened or stretched to length 1. A zero vector gives parts
// that are not numbers.
func (a Vec) Unit() Vec { return a.Scale(1 / math.Sqrt(a.Dot(a))) }

// Cross is the cross product of a and b, a vector at right angles to both,
// by the right-hand rule.
func (a Vec) Cross(b Vec) Vec {
	return Vec{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

// Toward is the unit vector pointing at azimuth az and altitude alt, both in
// degrees. Azimuth 0 is along Y and 90 along X; altitude lifts toward Z.
func Toward(az, alt float64) Vec {
	a, h := az*math.Pi/180, alt*math.Pi/180
	return Vec{math.Sin(a) * math.Cos(h), math.Cos(a) * math.Cos(h), math.Sin(h)}
}

// Bearing is the azimuth in degrees, from -180 to 180, of the line from one
// point to the other, measured as Toward does and ignoring height.
func Bearing(from, to Vec) float64 {
	return math.Atan2(to.X-from.X, to.Y-from.Y) * 180 / math.Pi
}

type CameraSettings struct {
	Azimuth, LookDown, GaugeWidth, Focal float64
}

type Camera struct {
	Forward, Right, Up   Vec
	Focal, Width, Height float64
}

// NewCamera is a camera for a picture width by height pixels, facing settings.
// Azimuth at altitude LookDown, in degrees. Focal is in pixels: width times
// the settings Focal over GaugeWidth, so those two must share a unit.
func NewCamera(width, height int, settings CameraSettings) Camera {
	forward := Toward(settings.Azimuth, settings.LookDown)
	right := forward.Cross(Vec{0, 0, 1}).Unit()
	return Camera{Forward: forward, Right: right, Up: right.Cross(forward),
		Focal: float64(width) / 2 / (settings.GaugeWidth / 2 / settings.Focal),
		Width: float64(width), Height: float64(height)}
}

// Project gives the pixel x and y where p, an offset from the camera, lands.
// The third result is false, with zero coordinates, when p is not more than 1
// in front of the camera. The pixel may lie outside the picture.
func (c Camera) Project(p Vec) (float64, float64, bool) {
	ahead := p.Dot(c.Forward)
	if ahead <= 1 {
		return 0, 0, false
	}
	return c.Width/2 + p.Dot(c.Right)/ahead*c.Focal,
		c.Height/2 - p.Dot(c.Up)/ahead*c.Focal, true
}

// Ray is the unit vector from the camera through the middle of pixel x, y.
func (c Camera) Ray(x, y int) Vec {
	return c.Forward.Add(c.Right.Scale((float64(x) + 0.5 - c.Width/2) /
		c.Focal)).Add(c.Up.Scale(-(float64(y) + 0.5 - c.Height/2) /
		c.Focal)).Unit()
}

// ConeChord finds where a ray from the origin along d is inside an upright
// cone with its tip at apex, whose radius is slope times the height below the
// tip, cut off below height low. It returns the distances along d where the
// ray enters and leaves, and false when it is never inside for a positive
// length. Distances are counted from the origin and start at zero.
func ConeChord(d, apex Vec, slope, low float64) (float64, float64, bool) {
	k2 := slope * slope
	a := d.X*d.X + d.Y*d.Y - k2*d.Z*d.Z
	b := -2*(d.X*apex.X+d.Y*apex.Y) + 2*k2*apex.Z*d.Z
	c := apex.X*apex.X + apex.Y*apex.Y - k2*apex.Z*apex.Z
	var stretches [][2]float64
	if math.Abs(a) < 1e-12 {
		if b == 0 {
			return 0, 0, false
		}
		t := -c / b
		if b > 0 {
			stretches = append(stretches, [2]float64{math.Inf(-1), t})
		} else {
			stretches = append(stretches, [2]float64{t, math.Inf(1)})
		}
	} else {
		disc := b*b - 4*a*c
		if disc < 0 {
			if a < 0 {
				stretches = append(stretches, [2]float64{math.Inf(-1), math.Inf(1)})
			}
		} else {
			root := math.Sqrt(disc)
			t1, t2 := (-b-root)/(2*a), (-b+root)/(2*a)
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if a > 0 {
				stretches = append(stretches, [2]float64{t1, t2})
			} else {
				stretches = append(stretches,
					[2]float64{math.Inf(-1), t1}, [2]float64{t2, math.Inf(1)})
			}
		}
	}
	lowT, highT := 0.0, math.Inf(1)
	switch {
	case d.Z > 0:
		lowT, highT = math.Max(0, low/d.Z), apex.Z/d.Z
	case d.Z < 0:
		lowT, highT = math.Max(0, apex.Z/d.Z), low/d.Z
	default:
		if low > 0 || apex.Z < 0 {
			return 0, 0, false
		}
	}
	for _, s := range stretches {
		in, out := math.Max(s[0], lowT), math.Min(s[1], highT)
		if out > in {
			return in, out, true
		}
	}
	return 0, 0, false
}
