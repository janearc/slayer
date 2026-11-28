package workingview

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

type matrix [16]float32

// frameMatrix maps native local points into their parent frame.
func frameMatrix(frame maths.Subspace, inverse bool) matrix {
	origin := frame.Origin
	frame.Origin = maths.Vec{}
	transform := frame.Out
	if inverse {
		transform = frame.In
		origin = transform(origin.Neg())
	}
	axes := [...]maths.Vec{transform(maths.XAxis), transform(maths.YAxis), transform(maths.ZAxis), origin}
	var result matrix
	for column, axis := range axes {
		result[column*4] = float32(axis.X)
		result[column*4+1] = float32(axis.Y)
		result[column*4+2] = float32(axis.Z)
	}
	result[15] = 1
	return result
}

// instanceMatrix subtracts the eye in float64 before uploading the placement.
// Its axes remain in the world frame for working-view lighting.
func instanceMatrix(place maths.Subspace, eye maths.Vec) matrix {
	place.Origin = place.Origin.Sub(eye)
	return frameMatrix(place, false)
}

// cameraMatrix preserves the native pinhole's horizontal field and pixel shape.
func cameraMatrix(scene *placement.DisplayScene, width, height int) matrix {
	wide := scene.Lens.Wide
	if !(wide > 0 && wide < math.Pi) {
		wide = 60 * maths.Degree
	}
	pixel := scene.Lens.Pixel
	if !(pixel > 0) {
		pixel = 1
	}
	horizontal := float32(1 / math.Tan(wide/2))
	aspect := float32(float64(width) / float64(height))
	if scene.Width > 0 && scene.Height > 0 {
		aspect = float32(float64(scene.Width) * pixel / float64(scene.Height))
	}
	// Local working-view clipping in metres; WebGPU depth is zero to one.
	const near, far = float32(placement.DisplayNear), float32(placement.DisplayFar)
	projection := matrix{0: horizontal, 5: horizontal * aspect, 10: far / (near - far), 11: -1, 14: far * near / (near - far)}
	// Instance origins are already relative to the eye. Apply only the
	// eye's inverse orientation and scale here.
	eye := scene.Eye
	eye.Origin = maths.Vec{}
	return multiply(projection, frameMatrix(eye, true))
}

// multiply returns left times right for two 4 by 4 matrices.
func multiply(left, right matrix) matrix {
	var result matrix
	for column := 0; column < 4; column++ {
		for row := 0; row < 4; row++ {
			for inner := 0; inner < 4; inner++ {
				result[column*4+row] += left[inner*4+row] * right[column*4+inner]
			}
		}
	}
	return result
}
