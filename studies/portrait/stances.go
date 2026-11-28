package main

import (
	"fmt"
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// Angles are radians added to the relaxed pose in the local joint frame.
type gesture struct{ raise, forward, elbow, hip, knee float64 }
type stance struct {
	left, right                                 gesture
	lean, twist, head, pitch, footSpread, shift float64
}

var repertoire = map[string]stance{
	"rest":       {},
	"wave":       {left: gesture{raise: 1.95, forward: .12, elbow: -.32}, lean: .05, head: .16, footSpread: -.06, shift: -.055},
	"offer":      {right: gesture{raise: .32, forward: .38, elbow: .85}, twist: -.36, head: -.16, pitch: .15, footSpread: .035, shift: -.09},
	"balance":    {left: gesture{raise: 1.10, forward: .15}, right: gesture{raise: .85, forward: -.12, hip: -.15, knee: .18}, lean: -.12, footSpread: .175, shift: -.06},
	"reach-high": {right: gesture{raise: 2.24, forward: .14, elbow: -.43}, left: gesture{raise: .08}, lean: .15, head: -.22, footSpread: -.08},
	"gather":     {left: gesture{raise: .12, forward: .42, elbow: .95, hip: -.72, knee: 1.24}, right: gesture{raise: .20, forward: .48, elbow: .90, hip: -.54, knee: .94}, twist: .12, pitch: .31, footSpread: .11},
	"open":       {left: gesture{raise: .75, forward: .17, elbow: .38}, right: gesture{raise: .62, forward: .14, elbow: .25}, twist: -.09, head: .20, footSpread: .075, shift: -.075},
	"stride":     {left: gesture{raise: .14, forward: .46, hip: .44, knee: .09}, right: gesture{forward: -.45, elbow: -.23, hip: -.56, knee: .29}, twist: .29, head: -.15, footSpread: -.08, shift: -.04},
}

func (s stance) articulate(p skeleton.Pose, side int) skeleton.Pose {
	g := s.right
	if side > 0 {
		g = s.left
	}
	id := fmt.Sprint(side)
	sign := float64(side)
	return p.Compose("upper arm"+id, skeleton.NewTurn(maths.ZAxis, sign*g.raise)).Compose("upper arm"+id, skeleton.NewTurn(maths.YAxis, -sign*g.forward)).Compose("forearm"+id, skeleton.NewTurn(maths.YAxis, -sign*g.elbow)).Compose("thigh"+id, skeleton.NewTurn(maths.XAxis, g.hip)).Compose("shin"+id, skeleton.NewTurn(maths.XAxis, g.knee))
}
