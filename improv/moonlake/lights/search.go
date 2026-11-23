package lights

import "math"

type SearchConfig struct {
	Gain  float64
	Count int
}

type SearchInput struct {
	Open, CloseTrees              []bool
	Camera                        Camera
	Moon                          [3]float64
	FarTree                       Vec
	Azimuth, ShoreRow             float64
	Level, Gate, Phase, MovePhase float64
	Move                          string
}

// addSearchlights draws config.Count searchlight beams from behind the far
// tree into sky. It does nothing when Level times Gate is under 0.01. Move
// picks the pattern: fan, viewer, chase (one beam bright, the rest at 12
// percent) or, by default, a sweep. Only viewer also draws into front and
// respects CloseTrees; the others paint open sky only.
func addSearchlights(sky, front Image, config SearchConfig, input SearchInput) {
	level := input.Level * input.Gate
	if level < 0.01 {
		return
	}
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	behind := input.FarTree.Add(Toward(input.Azimuth, 0).Scale(200)).Add(Vec{0, 0, 10})
	w := 2 * math.Pi * input.Phase
	colour := [3]float64{0.85, 0.9, 1}
	opening := 0.5 - 0.5*math.Cos(2*math.Pi*input.MovePhase)
	running := int(math.Floor(input.MovePhase*2*float64(config.Count))) % config.Count
	cam := input.Camera
	for k := range config.Count {
		out := float64(2*(k%2) - 1)
		rank := float64(k / 2)
		phase := float64(k) * 2 * math.Pi / float64(config.Count)
		origin := behind.Add(Toward(input.Azimuth+out*90, 0).Scale(140 * (rank + 1)))
		bright := level
		var dir Vec
		switch input.Move {
		case "fan":
			dir = Toward(input.Azimuth+out*(8+(rank+1)*14*opening), 70-40*opening)
		case "viewer":
			at := origin.Scale(-1).Unit()
			side := at.Cross(Vec{0, 0, 1}).Unit()
			dir = at.Add(side.Scale(out * (rank + 0.5) * 0.017)).Unit()
		case "chase":
			if k != running {
				bright *= 0.12
			}
			fallthrough
		default:
			dir = Toward(input.Azimuth+out*(20+8*rank+18*math.Sin(w+phase)),
				42+13*math.Sin(2*w+phase))
		}
		toEye := dir.Dot(origin.Scale(-1).Unit())
		cleared := false
		const spread = 2.5 * math.Pi / 180
		for s := 5.0; s < 6000; {
			p := origin.Add(dir.Scale(s))
			x, y, ahead := cam.Project(p)
			dist := math.Sqrt(p.Dot(p))
			sigma := math.Max(0.8, s*math.Tan(spread)/dist*cam.Focal)
			step := math.Max(1, 0.5*sigma*dist/cam.Focal)
			s += step
			if !ahead || dist < 30 {
				continue
			}
			if input.Move == "viewer" && dist < 500 {
				break
			}
			haze := math.Min(1, math.Exp(-(p.Z-input.FarTree.Z)/900))
			sigma = math.Min(sigma, 40*cam.Width/1280)
			pixels := step / dist * cam.Focal
			amount := config.Gain * bright * lum * haze * pixels * sigma
			if input.Move != "viewer" {
				splatSky(sky, input.Open, x, y, sigma, amount, colour)
				continue
			}
			xi, yi := int(x), int(y)
			if xi < 0 || yi < 0 || xi >= sky.Width || yi >= sky.Height {
				continue
			}
			if input.CloseTrees != nil && input.CloseTrees[yi*sky.Width+xi] {
				continue
			}
			if !cleared && input.Open[yi*sky.Width+xi] {
				cleared = true
				glare := math.Pow(math.Max(0, (toEye-0.995)/0.005), 3)
				k := cam.Width / 1280
				Splat(sky, front, x, y, 6*k, glare*lum*bright*30*k, colour, input.ShoreRow)
				Splat(sky, front, x, y, 1.2*k, glare*lum*bright*20*k, colour, input.ShoreRow)
			}
			Splat(sky, front, x, y, sigma, amount, colour, input.ShoreRow)
		}
	}
}

// splatSky adds a blurred round spot of colour at x, y, with spread sigma and
// total amount, to open sky pixels only. It reaches 2.5 sigma, and a spot well
// off the picture is skipped.
func splatSky(sky Image, open []bool, x, y, sigma, amount float64, colour [3]float64) {
	r := int(math.Ceil(2.5 * sigma))
	if x < -float64(r) || y < -float64(r) || x > float64(sky.Width+r) || y > float64(sky.Height+r) {
		return
	}
	norm := amount / (2 * math.Pi * sigma * sigma)
	for yy := max(0, int(y)-r); yy <= min(sky.Height-1, int(y)+r); yy++ {
		for xx := max(0, int(x)-r); xx <= min(sky.Width-1, int(x)+r); xx++ {
			if !open[yy*sky.Width+xx] {
				continue
			}
			dx, dy := float64(xx)+0.5-x, float64(yy)+0.5-y
			wgt := norm * math.Exp(-(dx*dx+dy*dy)/(2*sigma*sigma))
			i := sky.At(xx, yy)
			for c := range 3 {
				sky.Light[i+c] += wgt * colour[c]
			}
		}
	}
}

// Splat adds a blurred round spot of colour at x, y, with spread sigma and
// total amount. It reaches 3 sigma. Pixels from row shoreRow down go to front
// and those above to sky. Pixels off the picture are skipped.
func Splat(sky, front Image, x, y, sigma, amount float64, colour [3]float64, shoreRow float64) {
	r := int(math.Ceil(3 * sigma))
	norm := amount / (2 * math.Pi * sigma * sigma)
	for yy := int(y) - r; yy <= int(y)+r; yy++ {
		for xx := int(x) - r; xx <= int(x)+r; xx++ {
			if xx < 0 || yy < 0 || xx >= sky.Width || yy >= sky.Height {
				continue
			}
			dx, dy := float64(xx)+0.5-x, float64(yy)+0.5-y
			w := norm * math.Exp(-(dx*dx+dy*dy)/(2*sigma*sigma))
			target := sky
			if float64(yy) >= shoreRow {
				target = front
			}
			i := target.At(xx, yy)
			for c := range 3 {
				target.Light[i+c] += w * colour[c]
			}
		}
	}
}
