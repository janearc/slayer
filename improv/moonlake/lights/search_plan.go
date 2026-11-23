package lights

import (
	"errors"
	"math"
)

type PlannedSplat struct {
	X, Y, Sigma, Amount float64
	Colour              [3]float64
	SkyOnly             bool
}

// PlanSearchlights lists the splats addSearchlights would draw for a width by
// height frame, without drawing them. Splats for moves other than viewer are
// marked SkyOnly. It refuses bad sizes, mask lengths, a count outside 1 to
// 256, values not finite, and a camera that does not match the frame. It
// returns nothing when Level times Gate is under 0.01.
func PlanSearchlights(width, height int, config SearchConfig, input SearchInput) ([]PlannedSplat, error) {
	if !dimensions(width, height) || len(input.Open) != width*height || input.CloseTrees != nil && len(input.CloseTrees) != width*height || config.Count <= 0 || config.Count > 256 {
		return nil, errors.New("search: invalid dimensions, masks or count")
	}
	for _, value := range []float64{config.Gain, input.Level, input.Gate, input.Phase, input.MovePhase, input.Azimuth, input.ShoreRow, input.Camera.Focal, input.Camera.Width, input.Camera.Height, input.FarTree.X, input.FarTree.Y, input.FarTree.Z, input.Moon[0], input.Moon[1], input.Moon[2]} {
		if !finite(value) {
			return nil, errors.New("search: nonfinite input")
		}
	}
	for _, vector := range []Vec{input.Camera.Forward, input.Camera.Right, input.Camera.Up} {
		for _, value := range []float64{vector.X, vector.Y, vector.Z} {
			if !finite(value) || math.Abs(value) > 1e7 {
				return nil, errors.New("search: invalid camera vector")
			}
		}
	}
	if input.Camera.Focal <= 0 || input.Camera.Width != float64(width) || input.Camera.Height != float64(height) || input.MovePhase < 0 || input.MovePhase > 1e6 {
		return nil, errors.New("search: invalid camera or phase")
	}
	level := input.Level * input.Gate
	if level < 0.01 {
		return nil, nil
	}
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	behind := input.FarTree.Add(Toward(input.Azimuth, 0).Scale(200)).Add(Vec{0, 0, 10})
	wave := 2 * math.Pi * input.Phase
	colour := [3]float64{0.85, 0.9, 1}
	opening := 0.5 - 0.5*math.Cos(2*math.Pi*input.MovePhase)
	running := int(math.Floor(input.MovePhase*2*float64(config.Count))) % config.Count
	camera := input.Camera
	var result []PlannedSplat
	add := func(x, y, sigma, amount float64, skyOnly bool) error {
		if !finite(x) || !finite(y) || !finite(sigma) || !finite(amount) || sigma <= 0 {
			return errors.New("search: invalid splat")
		}
		radius := math.Ceil(3 * sigma)
		if x < -radius-1 || y < -radius-1 || x > float64(width)+radius+1 || y > float64(height)+radius+1 {
			return nil
		}
		if len(result) >= 1<<20 {
			return errors.New("search: splat limit exceeded")
		}
		result = append(result, PlannedSplat{X: x, Y: y, Sigma: sigma, Amount: amount, Colour: colour, SkyOnly: skyOnly})
		return nil
	}
	for index := 0; index < config.Count; index++ {
		out := float64(2*(index%2) - 1)
		rank := float64(index / 2)
		phase := float64(index) * 2 * math.Pi / float64(config.Count)
		origin := behind.Add(Toward(input.Azimuth+out*90, 0).Scale(140 * (rank + 1)))
		bright := level
		var direction Vec
		switch input.Move {
		case "fan":
			direction = Toward(input.Azimuth+out*(8+(rank+1)*14*opening), 70-40*opening)
		case "viewer":
			at := origin.Scale(-1).Unit()
			side := at.Cross(Vec{0, 0, 1}).Unit()
			direction = at.Add(side.Scale(out * (rank + 0.5) * 0.017)).Unit()
		case "chase":
			if index != running {
				bright *= 0.12
			}
			fallthrough
		default:
			direction = Toward(input.Azimuth+out*(20+8*rank+18*math.Sin(wave+phase)), 42+13*math.Sin(2*wave+phase))
		}
		toEye := direction.Dot(origin.Scale(-1).Unit())
		cleared := false
		for distanceAlong := 5.0; distanceAlong < 6000; {
			point := origin.Add(direction.Scale(distanceAlong))
			x, y, ahead := camera.Project(point)
			distance := math.Sqrt(point.Dot(point))
			sigma := math.Max(0.8, distanceAlong*math.Tan(2.5*math.Pi/180)/distance*camera.Focal)
			step := math.Max(1, 0.5*sigma*distance/camera.Focal)
			distanceAlong += step
			if !finite(step) || !finite(distanceAlong) {
				return nil, errors.New("search: invalid march step")
			}
			if !ahead || distance < 30 {
				continue
			}
			if input.Move == "viewer" && distance < 500 {
				break
			}
			haze := math.Min(1, math.Exp(-(point.Z-input.FarTree.Z)/900))
			sigma = math.Min(sigma, 40*camera.Width/1280)
			amount := config.Gain * bright * lum * haze * (step / distance * camera.Focal) * sigma
			if input.Move != "viewer" {
				if err := add(x, y, sigma, amount, true); err != nil {
					return nil, err
				}
				continue
			}
			xi, yi := int(x), int(y)
			if xi < 0 || yi < 0 || xi >= width || yi >= height {
				continue
			}
			if input.CloseTrees != nil && input.CloseTrees[yi*width+xi] {
				continue
			}
			if !cleared && input.Open[yi*width+xi] {
				cleared = true
				glare := math.Pow(math.Max(0, (toEye-0.995)/0.005), 3)
				scale := camera.Width / 1280
				if err := add(x, y, 6*scale, glare*lum*bright*30*scale, false); err != nil {
					return nil, err
				}
				if err := add(x, y, 1.2*scale, glare*lum*bright*20*scale, false); err != nil {
					return nil, err
				}
			}
			if err := add(x, y, sigma, amount, false); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}
