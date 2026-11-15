package lights

import (
	"math"
	"testing"
)

func TestBeamGeometryCameraInstances(t *testing.T) {
	a := NewCamera(256, 192, CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150})
	before := a
	b := NewCamera(256, 192, CameraSettings{Azimuth: 90, LookDown: 10, GaugeWidth: 70.41, Focal: 50})
	if a != before || a == b {
		t.Fatal("camera instances share settings")
	}
	x, y, ahead := a.Project(a.Forward.Scale(1000))
	if !ahead || math.Abs(x-128) > 1e-10 || math.Abs(y-96) > 1e-10 {
		t.Fatalf("axis projection: %v %v %v", x, y, ahead)
	}
	if _, _, ahead := a.Project(a.Forward.Scale(-1000)); ahead {
		t.Fatal("rear point projected")
	}
	ray := a.Ray(40, 60)
	x, y, ahead = a.Project(ray.Scale(1000))
	if !ahead || math.Abs(x-40.5) > 1e-10 || math.Abs(y-60.5) > 1e-10 || math.Abs(ray.Dot(ray)-1) > 1e-12 {
		t.Fatalf("pixel ray: %v %v %v", x, y, ray)
	}
}

func TestBeamGeometryConeChord(t *testing.T) {
	for _, test := range []struct {
		name                string
		d, apex             Vec
		slope, low, in, out float64
		crosses             bool
	}{
		{"horizontal", Vec{1, 0, 0}, Vec{10, 0, 10}, 0.5, -10, 5, 15, true},
		{"miss", Vec{0, 1, 0}, Vec{10, 0, 10}, 0.5, -10, 0, 0, false},
		{"above", Vec{1, 0, 0}, Vec{10, 0, -1}, 0.5, -10, 0, 0, false},
		{"below", Vec{1, 0, 0}, Vec{10, 0, 10}, 0.5, 1, 0, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			in, out, crosses := ConeChord(test.d, test.apex, test.slope, test.low)
			if in != test.in || out != test.out || crosses != test.crosses {
				t.Fatalf("chord=%v %v %v", in, out, crosses)
			}
		})
	}
}
