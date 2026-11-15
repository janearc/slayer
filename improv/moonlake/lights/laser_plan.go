package lights

import (
	"errors"
	"math"
)

// PlanLasers lists the splats AddLasers would draw for a width by height
// frame, without drawing them. Cues are read as AddLasers reads them. It
// refuses bad sizes, mask lengths, reach, a beam count over 256, and numbers
// that are not finite or are over 1e7. It returns nothing when Rig is 0 or
// the gates are shut. Splats well off the frame are dropped, and over 1<<20
// splats is an error.
func PlanLasers(width, height int, config LaserConfig, input LaserInput) ([]PlannedSplat, error) {
	if !dimensions(width, height) || len(input.Open) != width*height || len(input.Rim) != width || input.CloseTrees != nil && len(input.CloseTrees) != width*height || input.Camera.Focal <= 0 || config.Reach <= 0 || config.Reach > 1e7 || input.Cues.Beams < 0 || input.Cues.Beams > 256 {
		return nil, errors.New("laser: invalid dimensions, masks, reach or beam count")
	}
	values := []float64{config.Azimuth, config.EyeOverLake, config.RigDeck, config.WaterZ, config.Reach, input.Seconds, input.ShoreRow, input.Camera.Focal, input.Camera.Width, input.Camera.Height, input.Cues.Rig, input.Cues.Sweep, input.Cues.Gate, input.Cues.GateLeft, input.Cues.GateRight, input.Cues.FigurePhase, input.Moon[0], input.Moon[1], input.Moon[2]}
	for _, value := range values {
		if !finite(value) || math.Abs(value) > 1e7 {
			return nil, errors.New("laser: invalid parameter")
		}
	}
	for _, vector := range []Vec{config.FarTree, config.ShoreLeft, config.ShoreRight, input.Camera.Forward, input.Camera.Right, input.Camera.Up} {
		for _, value := range []float64{vector.X, vector.Y, vector.Z} {
			if !finite(value) || math.Abs(value) > 1e7 {
				return nil, errors.New("laser: invalid vector")
			}
		}
	}
	for _, value := range input.Cues.Colour {
		if !finite(value) || math.Abs(value) > 1e7 {
			return nil, errors.New("laser: invalid colour")
		}
	}
	cue := input.Cues
	if cue.Rig == 0 {
		return nil, nil
	}
	colour := [3]float64{0.08, 1, 0.12}
	if cue.HasColour {
		colour = cue.Colour
	}
	if cue.Rig > 0 {
		for channel := range colour {
			colour[channel] *= cue.Rig
		}
	}
	gates := [2]float64{cue.Gate, cue.Gate}
	if cue.GateLeft >= 0 {
		gates[0] = cue.GateLeft
	}
	if cue.GateRight >= 0 {
		gates[1] = cue.GateRight
	}
	if gates[0] < 0.01 && gates[1] < 0.01 {
		return nil, nil
	}
	swing := 45 * math.Sin(input.Seconds*1.7)
	tilt := -4.5 + 2.2*math.Sin(input.Seconds*1.1+0.6)
	spread := 22 + 14*math.Sin(input.Seconds*0.9)
	if cue.Sweep >= 0 {
		wave := 2 * math.Pi * cue.Sweep
		swing = 45 * math.Sin(wave)
		tilt = -4.5 + 2.2*math.Sin(2*wave+0.6)
		spread = 22 + 14*math.Sin(3*wave)
	}
	var result []PlannedSplat
	add := func(x, y, sigma, amount float64, colour [3]float64) error {
		if !finite(x) || !finite(y) || !finite(sigma) || !finite(amount) || sigma <= 0 {
			return errors.New("laser: invalid splat")
		}
		radius := math.Ceil(3 * sigma)
		if x < -radius-1 || y < -radius-1 || x > float64(width)+radius+1 || y > float64(height)+radius+1 {
			return nil
		}
		if len(result) >= 1<<20 {
			return errors.New("laser: splat limit exceeded")
		}
		result = append(result, PlannedSplat{X: x, Y: y, Sigma: sigma, Amount: amount, Colour: colour})
		return nil
	}
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	camera := input.Camera
	for _, head := range LaserHeads(config) {
		gate := gates[head.Bank]
		if gate < 0.01 {
			continue
		}
		lit := colour
		for channel := range lit {
			lit[channel] *= gate
		}
		ways, weight, split := FigureWays(cue.Figure, head.Aim+head.Mirror*swing, tilt+head.Lift, spread, cue.Beams, cue.FigurePhase)
		for beam, way := range ways {
			if split && cue.Parity >= 0 && (beam+head.Flip)%2 != cue.Parity {
				continue
			}
			each := lit
			for channel := range each {
				each[channel] *= weight
			}
			for distanceAlong := 0.0; distanceAlong < config.Reach; {
				point := head.Origin.Add(way.Scale(distanceAlong))
				if point.Z < config.WaterZ {
					if x, y, ahead := camera.Project(point); ahead {
						if err := add(x, y, 1.2, lum*60*LaserFog(point, input.Seconds, config.WaterZ), each); err != nil {
							return nil, err
						}
					}
					break
				}
				x, y, ahead := camera.Project(point)
				distance := math.Sqrt(point.Dot(point))
				step := math.Max(0.05, 0.5*distance/camera.Focal)
				distanceAlong += step
				if !finite(distanceAlong) {
					return nil, errors.New("laser: invalid march")
				}
				if !ahead || x < -4 || y < -4 || x >= camera.Width+4 || y >= camera.Height+4 {
					continue
				}
				xi, yi := int(x), int(y)
				if xi >= 0 && yi >= 0 && xi < width && yi < height && !input.Open[yi*width+xi] && yi < input.Rim[xi] && distance > 80 {
					continue
				}
				if input.CloseTrees != nil && xi >= 0 && yi >= 0 && xi < width && yi < height && input.CloseTrees[yi*width+xi] {
					continue
				}
				bright := lum * (0.12 + 24*LaserFog(point, input.Seconds, config.WaterZ)) * 2.2 * math.Min(1, float64(width)/1280)
				pixels := step / distance * camera.Focal
				if err := add(x, y, 0.5, bright*pixels*0.88, each); err != nil {
					return nil, err
				}
				if err := add(x, y, 1.6, bright*pixels*0.12, each); err != nil {
					return nil, err
				}
			}
		}
		x, y, ahead := camera.Project(head.Origin)
		if !ahead || x < 0 || y < 0 || x >= camera.Width || y >= camera.Height {
			continue
		}
		if input.CloseTrees != nil && input.CloseTrees[int(y)*width+int(x)] {
			continue
		}
		scale := camera.Width / 1280
		most := math.Max(lit[0], math.Max(lit[1], lit[2]))
		core := [3]float64{}
		for channel := range core {
			core[channel] = 0.5*lit[channel] + 0.5*most
		}
		if err := add(x, y, 0.8*scale, lum*40*scale, core); err != nil {
			return nil, err
		}
		if err := add(x, y, 5*scale, lum*25*scale, lit); err != nil {
			return nil, err
		}
	}
	return result, nil
}
