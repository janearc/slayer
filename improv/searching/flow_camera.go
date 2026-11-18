package searching

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"math"
)

// FlowTravel pulls away from an authored landmark before holding the wide
// reveal. Radius changes logarithmically so the nearby view does not vanish
// in the first frame. Directions and aim interpolate smoothly.
type FlowTravel struct {
	To       [3]float64 `yaml:"to"`
	LookFrom [3]float64 `yaml:"look-from"`
	LookTo   [3]float64 `yaml:"look-to"`
	Until    string     `yaml:"until"`
}

// checkFlowTravel excludes coincident eyes and opposing directions whose
// interpolation would cross the aim. The old fixed view needs no travel.
func (edit *Edit) checkFlowTravel() error {
	p := edit.Plan.FlowStudy
	if p.Travel == nil {
		return nil
	}
	t := p.Travel
	until, err := edit.Cue(t.Until)
	a, b := vector(p.Eye).Sub(vector(t.LookFrom)), vector(t.To).Sub(vector(t.LookTo))
	if err != nil || until <= edit.FlowStart || until > edit.FlowFold ||
		!pointOK(t.To) || !pointOK(t.LookFrom) || !pointOK(t.LookTo) ||
		a.Len() < 0.1 || b.Len() < 0.1 || a.Unit().Dot(b.Unit()) < -0.95 {
		return fmt.Errorf("invalid flow travel: keep a clear view and finish by the fold")
	}
	return nil
}

// flowCamera samples an absolute song time, independent of previous frames.
func (edit *Edit) flowCamera(at float64) (maths.Vec, maths.Vec) {
	p := edit.Plan.FlowStudy
	if p.Travel == nil {
		return vector(p.Eye), maths.Vec{}
	}
	t := p.Travel
	until, _ := edit.Cue(t.Until)
	f := smooth((at - edit.FlowStart) / (until - edit.FlowStart))
	aim := vector(t.LookFrom).Scale(1 - f).Add(vector(t.LookTo).Scale(f))
	a, b := vector(p.Eye).Sub(vector(t.LookFrom)), vector(t.To).Sub(vector(t.LookTo))
	radius := math.Exp(math.Log(a.Len())*(1-f) + math.Log(b.Len())*f)
	direction := a.Unit().Scale(1 - f).Add(b.Unit().Scale(f)).Unit()
	return aim.Add(direction.Scale(radius)), aim
}
