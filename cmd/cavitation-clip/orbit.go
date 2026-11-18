package main

import (
	"fmt"
	"math"
	"strconv"

	yaml "gopkg.in/yaml.v3"
)

// cameraOrbit is an authored path in map-space metres, not native XYZ.
// Azimuth zero is north; positive azimuth turns toward east. Elevation is
// positive up. The subject translation moves both eye and aim rigidly.
type cameraOrbit struct {
	Centre    [3]float64 `yaml:"centre"`
	Radius    float64    `yaml:"radius-m"`
	Azimuth   float64    `yaml:"azimuth-deg"`
	Turns     float64    `yaml:"turns"`
	Elevation float64    `yaml:"elevation-deg"`
	Amplitude float64    `yaml:"elevation-amplitude-deg"`
	Cycles    float64    `yaml:"elevation-cycles"`
	Phase     float64    `yaml:"elevation-phase-deg"`
	Surface   *float64   `yaml:"surface-up-m"`
}

// finiteBound reports whether v is a finite number within bound either side
// of zero.
func finiteBound(v, bound float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Abs(v) <= bound
}

// validateOrbit checks the timing, ray, view and camera orbit fields of the
// plan, and that the orbit stays below the authored surface.
func validateOrbit(p plan) error {
	if !finiteBound(p.StartSeconds, 86400) || p.StartSeconds < 0 {
		return fmt.Errorf("start-seconds must be finite from0 through86400")
	}
	if p.Rays != nil && (*p.Rays < 1 || *p.Rays > 64) {
		return fmt.Errorf("rays must be from1 through64")
	}
	if p.VerticalView != nil && (!finiteBound(*p.VerticalView, 179) || *p.VerticalView < 1) {
		return fmt.Errorf("vertical-view-deg must be finite from1 through179")
	}
	o := p.Orbit
	if o == nil {
		return nil
	}
	if p.Track != 1 {
		return fmt.Errorf("camera-orbit requires camera-tracking-fraction: 1")
	}
	for _, v := range o.Centre {
		if !finiteBound(v, 1e7) {
			return fmt.Errorf("orbit centre must be finite and bounded1e7 metres")
		}
	}
	if !finiteBound(o.Radius, 1e5) || o.Radius < .01 ||
		!finiteBound(o.Azimuth, 36000) || !finiteBound(o.Phase, 36000) ||
		!finiteBound(o.Turns, 10) || !finiteBound(o.Cycles, 10) ||
		!finiteBound(o.Elevation, 89) || !finiteBound(o.Amplitude, 89) ||
		math.Abs(o.Elevation)+math.Abs(o.Amplitude) > 89 {
		return fmt.Errorf("orbit radius/angles must be finite and bounded; elevation envelope must stay within89 degrees")
	}
	if o.Surface != nil {
		if !finiteBound(*o.Surface, 1e7) {
			return fmt.Errorf("surface-up-m must be finite and bounded1e7 metres")
		}
		// Conservative over the complete continuous path, including the
		// unsampled endpoint: both linear translation and sinusoid are bounded.
		upper := o.Centre[2] + math.Max(0, p.Velocity[2]*float64(p.Seconds))
		eyeUpper := upper + o.Radius*math.Sin((o.Elevation+math.Abs(o.Amplitude))*math.Pi/180)
		if math.Max(upper, eyeUpper) >= *o.Surface {
			return fmt.Errorf("orbit eye/aim envelope reaches authored surface-up-m")
		}
	}
	return nil
}

// points returns the eye and aim positions of the orbit at the given time,
// with delta added to the centre.
func (o cameraOrbit) points(seconds, duration float64, delta [3]float64) (eye, aim [3]float64) {
	u := seconds / duration
	theta := o.Azimuth*math.Pi/180 + 2*math.Pi*o.Turns*u
	phi := (o.Elevation + o.Amplitude*math.Sin(2*math.Pi*o.Cycles*u+o.Phase*math.Pi/180)) * math.Pi / 180
	for i := range 3 {
		aim[i] = o.Centre[i] + delta[i]
	}
	eye = [3]float64{aim[0] + o.Radius*math.Sin(theta)*math.Cos(phi), aim[1] + o.Radius*math.Cos(theta)*math.Cos(phi), aim[2] + o.Radius*math.Sin(phi)}
	return
}

// replaceTriple overwrites the three numbers of a YAML sequence with value.
func replaceTriple(node *yaml.Node, value [3]float64) error {
	if err := shiftTriple(node, [3]float64{}); err != nil {
		return err
	}
	for i := range 3 {
		node.Content[i].Tag = "!!float"
		node.Content[i].Value = strconv.FormatFloat(value[i], 'g', 17, 64)
	}
	return nil
}
