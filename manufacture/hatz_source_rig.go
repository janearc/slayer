package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Source: hatz_source_next.go (4266c3c7d), with bone lengths from
// 3a07cb07d. The motions and lengths are authored.
var hatzSourceBones = [4]float64{.85, 1.02, 1.11, 1.39}

func hatzSourceJoints(phase, side float64) []maths.Vec {
	root := maths.V(.15, .14, side*.21)
	recovery := math.Max(0, math.Sin(phase))
	lengths := hatzSourceBones[:]
	baseSweep := []float64{-.05, -.15, -.20, -.16}
	fold := []float64{.13, .45, .60, .50}
	lift := []float64{.48, .53, .62, .67}
	lag := []float64{0, .12, .23, .34}
	joints := []maths.Vec{root}
	for i, length := range lengths {
		sweep := baseSweep[i] - recovery*fold[i]
		elevation := lift[i] * math.Sin(phase-lag[i])
		direction := maths.V(math.Sin(sweep), math.Sin(elevation), side*math.Cos(sweep)*math.Cos(elevation)).Unit()
		joints = append(joints, joints[len(joints)-1].Add(direction.Scale(length)))
	}
	return joints
}
