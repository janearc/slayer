//go:build draw && darwin

package gpu

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"

	scenes "github.com/janearc/slayer/scene/named"
)

// zappy's shape for these checks: few rings of few panels, so a frame
// encodes and draws quickly while every channel still has a shell to
// move; and a small tall picture. Picked 2026-11-12 for these checks.
const (
	zappyRings, zappyPanels      = 2, 6
	zappyAcross, zappyDown       = 27, 48
	zappyFrames                  = 6
	zappyStep                    = time.Second / 24
	zappyRestingFrames           = 2
	zappyChannelShare, zappyTurn = 0.37, 0.61803398875
)

// smallZappy is zappy built small, with its channels.
func smallZappy(t *testing.T) *scenes.Built {
	t.Helper()
	scene, err := scenes.Get("zappy")
	if err != nil {
		t.Fatal(err)
	}
	built, err := scene.Build(map[string]any{"orbits": zappyRings,
		"panels": zappyPanels}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(built.Channels()) == 0 {
		t.Fatal("zappy has no channels to move")
	}
	return built
}

// zappyAt is zappy at a frame: its clock moved on, and from the frames
// after the first few every channel moved somewhere in its range, a
// different share of it each frame, so shells are made again, the star
// breathes and the beams turn; the last frames rest again.
func zappyAt(t *testing.T, built *scenes.Built, index int) (*thing.World,
	time.Duration) {
	t.Helper()
	at := time.Duration(index) * zappyStep
	built.Set(at)
	values := map[string]float64{}
	moving := index >= zappyRestingFrames &&
		index < zappyFrames-zappyRestingFrames
	for number, channel := range built.Channels() {
		if !moving {
			continue
		}
		_, share := math.Modf(float64(index)*zappyTurn +
			float64(number)*zappyChannelShare)
		values[channel.Name] = channel.Least +
			(channel.Most-channel.Least)*share
	}
	if err := built.Play(values); err != nil {
		t.Fatal(err)
	}
	return built.World(), at
}

// TestKeptTreesGiveTheInputsAFullEncodeGives encodes each frame of a
// zappy film twice, with a keeper carried from frame to frame and with
// none: every input is the same to the byte, and the keeper did serve
// trees, so the comparison is of kept trees and not of grown ones.
func TestKeptTreesGiveTheInputsAFullEncodeGives(t *testing.T) {
	built := smallZappy(t)
	keeper := newKeptTrees()
	for index := range zappyFrames {
		world, at := zappyAt(t, built, index)
		want, err := encodeGrown(world, nil, built.Eye(), nil, at,
			zappyAcross, zappyDown)
		if err != nil {
			t.Fatal(err)
		}
		got, err := encodeKept(world, nil, built.Eye(), nil, at,
			zappyAcross, zappyDown, keeper)
		if err != nil {
			t.Fatal(err)
		}
		keeper.turn()
		for place, input := range want.inputs() {
			if string(got.inputs()[place]) != string(input) {
				t.Fatalf("frame %d: input %d differs from a full encode",
					index, place)
			}
		}
	}
	if keeper.served == 0 {
		t.Fatal("the keeper served no tree, so nothing kept was compared")
	}
}

// TestAWrongKeptTreeIsSeen is the anti-test: one shell's kept tree put
// under the other shell's mesh makes inputs that differ from a full
// encode, so the comparison above would catch a tree kept wrongly.
func TestAWrongKeptTreeIsSeen(t *testing.T) {
	built := smallZappy(t)
	keeper := newKeptTrees()
	world, at := zappyAt(t, built, 0)
	if _, err := encodeKept(world, nil, built.Eye(), nil, at,
		zappyAcross, zappyDown, keeper); err != nil {
		t.Fatal(err)
	}
	keeper.turn()
	var meshes []*mesh.Mesh
	for skin := range keeper.kept {
		meshes = append(meshes, skin)
	}
	if len(meshes) < 2 {
		t.Fatalf("zappy kept %d trees; two are needed to swap", len(meshes))
	}
	keeper.kept[meshes[0]] = keeper.kept[meshes[1]]
	want, err := encodeGrown(world, nil, built.Eye(), nil, at, zappyAcross,
		zappyDown)
	if err != nil {
		t.Fatal(err)
	}
	got, err := encodeKept(world, nil, built.Eye(), nil, at, zappyAcross,
		zappyDown, keeper)
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for place, input := range want.inputs() {
		same = same && string(got.inputs()[place]) == string(input)
	}
	if same {
		t.Fatal("a tree kept under the wrong mesh went unseen")
	}
}

// TestAKeeperHoldsOnlyTheLastFrame runs a film whose shells are made
// again every frame: the keeper never holds more trees than one frame
// draws, however long the film.
func TestAKeeperHoldsOnlyTheLastFrame(t *testing.T) {
	built := smallZappy(t)
	keeper := newKeptTrees()
	most := 0
	for index := range zappyFrames {
		world, at := zappyAt(t, built, index)
		encoded, err := encodeKept(world, nil, built.Eye(), nil, at,
			zappyAcross, zappyDown, keeper)
		if err != nil {
			t.Fatal(err)
		}
		drawn := len(encoded.meshes)
		keeper.turn()
		most = max(most, drawn)
		if keeper.held() != drawn {
			t.Fatalf("frame %d drew %d meshes and %d trees are held",
				index, drawn, keeper.held())
		}
	}
	if most == 0 {
		t.Fatal("no frame drew a mesh")
	}
}

// TestFramesDrawZappyAsThePerCallPath draws the zappy film through a
// drawer, its keeper serving the shells' trees, and each frame is the
// per-call picture, bit for bit.
func TestFramesDrawZappyAsThePerCallPath(t *testing.T) {
	built := smallZappy(t)
	frames := newFrames(softwareTraversal)
	defer frames.Close()
	lens := film.Pinhole{Wide: shotLens.Wide}
	for index := range zappyFrames {
		world, at := zappyAt(t, built, index)
		got, _, err := frames.Draw(world, uint64(index+1), built.Eye(),
			lens, at, zappyAcross, zappyDown, 1)
		mustDevice(t, err)
		want, _, err := SampledMeasured(world, built.Eye(), lens, at,
			zappyAcross, zappyDown, 1)
		mustDevice(t, err)
		sameBits(t, "a zappy frame", want, got)
	}
	if frames.trees.served == 0 {
		t.Fatal("the drawer's keeper served no tree")
	}
}
