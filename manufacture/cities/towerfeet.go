package cities

import "github.com/janearc/slayer/maths"

// footSquare is how nearly square two walls of a plan must meet, as the
// largest dot of their outward directions, for the plan to be a rectangle.
// A choice.
const footSquare = 1e-6

// FootFace is a wall of a tower's lowest segment that stands on the ground
// and stands up straight: a place for a row of shops. Mid is the middle of
// its foot, Out the horizontal unit vector away from the tower there, Along
// the horizontal unit vector along it to the right of a viewer looking at
// it from outside, and Length how long it is.
//
// Clear is the stretches of it,
// as distances from its middle along it, that no other part of the plan
// covers; none given means none of it is clear, and one stretch from
// minus half its length to half is all of it. All of it is in the tower's
// frame, in metres.
type FootFace struct {
	Mid, Out, Along maths.Vec
	Length          float64
	Clear           [][2]float64
}

// open says no other part of the plan covers any of the wall: its one clear
// stretch is all of it.
func (face FootFace) open() bool {
	return len(face.Clear) == 1 && face.Clear[0][0] <= -face.Length/2 &&
		face.Clear[0][1] >= face.Length/2
}

// FootFaces are the walls of a tower that front open ground, for shops: the
// upright walls of the lowest segment of a tower of one shaft. A tower of
// more shafts has none, as it stands on a base. The tower is not drawn to
// say them, and nothing of it changes.
func (tower *Tower) FootFaces() ([]FootFace, error) {
	if len(tower.Shafts) != 1 {
		return nil, nil
	}
	drawing, err := newDrawer(signCulture, 0)
	if err != nil {
		return nil, err
	}
	shaft := tower.Shafts[0]
	bottom, top, err := shaft.layer(0)
	if err != nil {
		return nil, err
	}
	height := float64(shaft.Segments[0].Storeys) * tower.storeyHeight()
	var faces []FootFace
	for _, frame := range frames(bottom, top, 0, height) {
		if !drawing.upright(frame) {
			continue
		}
		out := level(frame.out)
		faces = append(faces, FootFace{Mid: frame.origin, Out: out,
			Along: wallEdge{out: out}.along(), Length: frame.length,
			Clear: frame.spans})
	}
	return faces, drawing.trouble
}

// Footed opens a storey at the foot of the tower for shops to fill, of the
// height given, if the tower is a single plain rectangle in plan with four
// upright walls nothing covers, and says whether it did. The tower then
// stands on that storey: it is that much taller, and its walls, its windows
// and the places for its signs start above it. Nothing of the tower is
// drawn in the open storey; the caller fills it.
func (tower *Tower) Footed(height float64) (bool, error) {
	faces, err := tower.FootFaces()
	if err != nil || len(faces) != 4 || !(height > 0) {
		return false, err
	}
	for index, face := range faces {
		next := faces[(index+1)%len(faces)]
		if !face.open() || face.Out.Dot(next.Out) > footSquare ||
			face.Out.Dot(next.Out) < -footSquare {
			return false, nil
		}
	}
	tower.foot = height
	return true, nil
}

// Foot is the height of the open storey at the foot of the tower, or
// nought for none.
func (tower *Tower) Foot() float64 { return tower.foot }
