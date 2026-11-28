package fibre

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Direction is a sampled direction and its integrated path weight. The
// sample already follows its path's normalized angular density.
type Direction struct {
	Light  Light
	Weight maths.Light
}

// Sample draws one direction from each path's conditional distribution
// in R, TT, TRT, Higher order. The three shares are independent uniform
// samples strictly between zero and one. Multiply each returned weight
// by external radiance and visibility in that direction.
func (event Event) Sample(shares [3]float64) ([4]Direction, error) {
	var directions [4]Direction
	if event.model == nil {
		return directions, fmt.Errorf("fibre: event must be made with At")
	}
	for _, share := range shares {
		if !(share > 0 && share < 1) {
			return directions, fmt.Errorf("fibre: sample shares must lie between zero and one")
		}
	}
	radial, turn, azimuth := shares[0], shares[1], shares[2]
	for p, lobe := range event.lobes {
		if lobe.energy == (maths.Light{}) {
			continue
		}
		kappa := event.model.kappa[p]
		// The longitudinal distribution is the marginal of a spherical
		// von Mises-Fisher distribution. Invert its polar cosine CDF,
		// rotate from its lobe axis, and retain its longitudinal sine.
		polar := polarCosine(radial, kappa)
		elevation := safeAsin(-lobe.sine*polar +
			lobe.cosine*complement(polar)*math.Cos(2*math.Pi*turn))
		phi := (2*azimuth - 1) * math.Pi
		if p < 3 {
			scale := event.model.logistic
			// Work from both tails so extreme valid shares stay finite.
			edge := 1 / (1 + math.Exp(math.Pi/scale))
			lower := edge + azimuth*(1-2*edge)
			upper := edge + (1-azimuth)*(1-2*edge)
			phi = lobe.azimuth + scale*(math.Log(lower)-math.Log(upper))
		}
		directions[p] = Direction{Light{elevation, phi}, lobe.energy}
	}
	return directions, nil
}

func polarCosine(radial, kappa float64) float64 {
	if kappa < .5 {
		return 1 + math.Log1p((1-radial)*math.Expm1(-2*kappa))/kappa
	}
	return max(-1, 1+math.Log(radial+(1-radial)*math.Exp(-2*kappa))/kappa)
}

func safeAsin(sine float64) float64 { return math.Asin(max(-1, min(1, sine))) }
