package pinnipeds

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestSealRigPreservesOneClosedSkinAcrossPoseSweep(t *testing.T) {
	rig := NewSealRig()
	rest, err := rig.Skin(SealRigPose{})
	if err != nil {
		t.Fatal(err)
	}
	faces := rest.Faces()
	if !rest.Closed() || !sealSkinConnected(rest) {
		t.Fatal("rest skin is not one closed connected mesh")
	}
	volume := sealSkinVolume(rest)
	maxDrift := 0.
	for step := -18; step <= 18; step++ {
		share := float64(step) / 18
		pose := SealRigPose{Yaw: share * math.Pi / 2,
			Pitch: -.42 * math.Abs(share), Fore: [2]float64{-.28 * share, share},
			Hind: [2]float64{.43 * share, -.38 * share}}
		posed, err := rig.Skin(pose)
		if err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		if !posed.Closed() || !slices.Equal(posed.Faces(), faces) {
			t.Fatalf("step %d changed the skin connectivity", step)
		}
		drift := math.Abs(sealSkinVolume(posed)/volume - 1)
		maxDrift = math.Max(maxDrift, drift)
	}
	if maxDrift > .045 {
		t.Fatalf("signed mesh volume drift %.3f%% exceeds 4.5%%", maxDrift*100)
	}
	t.Logf("%d shared triangles, rest volume %.6f m³, max signed mesh volume drift %.3f%%",
		len(faces), volume, 100*maxDrift)
}

func TestSealRigKeepsSpineLengthAtNinetyDegrees(t *testing.T) {
	rig := NewSealRig()
	const samples = 192
	var length float64
	pose := SealRigPose{Yaw: math.Pi / 2}
	previous := rig.frame(pose, sealShoulder).Origin
	for index := 1; index <= samples; index++ {
		station := sealShoulder + (sealHip-sealShoulder)*float64(index)/samples
		at := rig.frame(pose, station).Origin
		length += at.Sub(previous).Len()
		previous = at
	}
	if math.Abs(length-(sealShoulder-sealHip)) > .001 {
		t.Fatalf("spine arc %.6f m, rest %.6f m", length, sealShoulder-sealHip)
	}
	tangent := rig.frame(pose, sealHip).Out(maths.V(1, 0, 0)).Sub(previous).Unit()
	across := maths.V(1, 0, 0)
	if math.Abs(tangent.Dot(across)) > 1e-10 {
		t.Fatalf("tail tangent was not at a right angle to the head: %v", tangent)
	}
	t.Logf("90° tail tangent, spine arc %.6f m over %.6f m at rest", length, sealShoulder-sealHip)
}

func sealSkinVolume(skin *mesh.Mesh) float64 {
	var volume float64
	for index := range skin.FaceCount() {
		a, b, c := skin.Corners(index)
		volume += a.Dot(b.Cross(c)) / 6
	}
	return volume
}

func sealSkinConnected(skin *mesh.Mesh) bool {
	faces := skin.Faces()
	adjacent := make(map[int][]int)
	for _, face := range faces {
		for index, corner := range face {
			adjacent[corner] = append(adjacent[corner], face[(index+1)%len(face)])
		}
	}
	pending := []int{faces[0][0]}
	seen := map[int]bool{faces[0][0]: true}
	for len(pending) != 0 {
		corner := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for _, next := range adjacent[corner] {
			if !seen[next] {
				seen[next] = true
				pending = append(pending, next)
			}
		}
	}
	return len(seen) == len(adjacent)
}

func TestSealRigMovesFlexStartWithoutChangingTheRestSkin(t *testing.T) {
	neutral := NewSealRig()
	base, err := neutral.Skin(SealRigPose{})
	if err != nil {
		t.Fatal(err)
	}
	bent := SealRigPose{Yaw: math.Pi / 2, Fore: [2]float64{.58, -.24}}
	basePoints := base.Points()
	var tails []maths.Vec
	for _, start := range []float64{-.16, .30, .52} {
		rig := newSealRig(start)
		rest, err := rig.Skin(SealRigPose{})
		if err != nil {
			t.Fatal(err)
		}
		for index, point := range rest.Points() {
			if point.Sub(basePoints[index]).Len() > 1e-10 {
				t.Fatalf("flex start %g changed the rest at vertex %d", start, index)
			}
		}
		posed, err := rig.Skin(bent)
		if err != nil {
			t.Fatal(err)
		}
		if !posed.Closed() || !slices.Equal(posed.Faces(), base.Faces()) {
			t.Fatalf("flex start %g changed the skin topology", start)
		}
		tail := rig.frame(bent, sealHip).Origin
		tails = append(tails, tail)
		t.Logf("flex begins at %.2f m, 90° tail offset %.3f m, signed volume drift %+.3f%%",
			start, tail.Z, (sealSkinVolume(posed)/sealSkinVolume(base)-1)*100)
	}
	if tails[2].Z-tails[0].Z < .18 {
		t.Fatalf("flex starts did not vary the body arc appreciably: %v", tails)
	}
}
