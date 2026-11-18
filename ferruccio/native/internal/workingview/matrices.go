package workingview

import (
	"github.com/janearc/slayer/hillside/placement"
	"github.com/janearc/slayer/maths"
	"math"
)

type matrix [16]float32

// frameMatrix maps native local points into their parent frame.
func frameMatrix(frame maths.Subspace, inverse bool) matrix {
	transform := frame.Out
	if inverse {
		transform = frame.In
	}
	origin := transform(maths.V(0, 0, 0))
	axes := []maths.Vec{transform(maths.XAxis).Sub(origin), transform(maths.YAxis).Sub(origin), transform(maths.ZAxis).Sub(origin), origin}
	var result matrix
	for column, axis := range axes {
		result[column*4] = float32(axis.X)
		result[column*4+1] = float32(axis.Y)
		result[column*4+2] = float32(axis.Z)
	}
	result[15] = 1
	return result
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
	const near, far = float32(.05), float32(100000)
	projection := matrix{0: horizontal, 5: horizontal * aspect, 10: far / (near - far), 11: -1, 14: far * near / (near - far)}
	return multiply(projection, frameMatrix(scene.Eye, true))
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
