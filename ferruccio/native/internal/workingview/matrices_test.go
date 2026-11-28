package workingview

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

func TestWorkingMatricesSubtractLargeEyeOriginBeforeNarrowing(t *testing.T) {
	eye := maths.NewSubspace(maths.V(3, 2, -1), maths.About(maths.YAxis, .2), 2)
	pose := maths.NewSubspace(maths.V(3.125, 2.75, -31.5), maths.About(maths.ZAxis, .3), 1.5)
	shift := maths.V(-318490735, 115921147, 5e8)
	travelledEye, travelledPose := eye, pose
	travelledEye.Origin = eye.Origin.Add(shift)
	travelledPose.Origin = pose.Origin.Add(shift)
	local := &placement.DisplayScene{Eye: eye, Width: 320, Height: 200}
	distant := &placement.DisplayScene{Eye: travelledEye, Width: 320, Height: 200}
	localCamera, distantCamera := cameraMatrix(local, 320, 200), cameraMatrix(distant, 320, 200)
	localInstance := instanceMatrix(pose, eye.Origin)
	distantInstance := instanceMatrix(travelledPose, travelledEye.Origin)
	if localCamera != distantCamera || localInstance != distantInstance {
		t.Fatal("translating scene and camera together changed GPU transforms")
	}
	if distantInstance[12] != .125 || distantInstance[13] != .75 || distantInstance[14] != -30.5 {
		t.Fatalf("instance translation lost local detail: %v", distantInstance[12:15])
	}
	// The transformed model axes still light in world space.
	worldX := pose.Orientation.Out(maths.XAxis.Scale(pose.Scale))
	if distantInstance[0] != float32(worldX.X) || distantInstance[1] != float32(worldX.Y) || distantInstance[2] != float32(worldX.Z) {
		t.Fatal("camera-relative positioning altered world-space lighting axes")
	}
	// Clip-space w is the positive depth in the eye's native frame.
	clip := multiply(distantCamera, distantInstance)
	wantDepth := -travelledEye.In(travelledPose.Origin).Z
	if math.Abs(float64(clip[15])-wantDepth) > 1e-5 {
		t.Fatalf("projected depth = %g, want %g", clip[15], wantDepth)
	}
}
