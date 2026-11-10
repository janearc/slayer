package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// These contact witnesses inspect the emitted closed pieces, not just a
// helper's chosen attachment or whether each disconnected mesh is watertight.
func TestOWLBearingMembersMeetTheirPlinthAndPrimaryFrame(test *testing.T) {
	pose := (owlObservatory{}).frame()
	for _, sign := range []float64{-1, 1} {
		assembly := &instrumentAssembly{fine: cut{around: leastAround}}
		owlBearing(assembly, sign, pose)
		var rods []part
		for _, piece := range assembly.object.parts {
			if _, rod := piece.shape.(tube); rod {
				rods = append(rods, piece)
			}
		}
		if assembly.trouble != nil || len(rods) != 3 {
			test.Fatal("bearing lost its axle or two legs", assembly.trouble)
		}
		// The independently reviewed primary-plane joints are at x=+-54,
		// y=45, z=0; the plinth top is platform 1.5 plus base 2 = 3.5 m.
		axle := rods[0]
		length := axle.shape.(tube).length.InMetres()
		end := axle.space.Out(maths.YAxis.Scale(length))
		if end.Sub(maths.V(sign*54, 45, 0)).Len() > 1e-8 {
			test.Fatal("axle ends short of the primary-plane attachment", end)
		}
		for _, leg := range rods[1:] {
			if !(leg.space.Origin.Y < 3.5) {
				test.Fatal("leg foot floats above its plinth", leg.space.Origin)
			}
		}
	}
}

func TestOWLCorrectorSupportsBeginInsideActualFrameJoints(test *testing.T) {
	pose := (owlObservatory{}).frame()
	frame := &instrumentAssembly{fine: cut{around: leastAround}}
	(owlObservatory{}).structure(frame, pose)
	mirrors := &instrumentAssembly{fine: cut{around: leastAround},
		object: good{dressed: looks{}}}
	(owlObservatory{}).corrector(mirrors, pose)
	var supports int
	for _, piece := range mirrors.object.parts {
		if _, rod := piece.shape.(tube); !rod {
			continue
		}
		supports++
		attached := false
		for _, station := range frame.object.parts {
			if _, joint := station.shape.(box); !joint {
				continue
			}
			bounds := partBounds([]part{station})
			point := piece.space.Origin
			attached = attached || (point.X >= bounds.Min.X && point.X <= bounds.Max.X &&
				point.Y >= bounds.Min.Y && point.Y <= bounds.Max.Y &&
				point.Z >= bounds.Min.Z && point.Z <= bounds.Max.Z)
		}
		if !attached {
			test.Fatal("corrector support starts in empty space", piece.space.Origin)
		}
	}
	if supports != 4 {
		test.Fatal("corrector lost a support", supports)
	}
}
