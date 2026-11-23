package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The roles of a yurt's parts. What each is made of is in data.
const (
	// roleCover is the felt over the roof.
	roleCover partRole = "cover"
	// roleFrame is the lattice wall.
	roleFrame partRole = "frame"
	// roleDoor is the door in the wall.
	roleDoor partRole = "door"
	// roleCrown is the ring the roof poles meet in.
	roleCrown partRole = "crown"
)

// halfTurn is how many degrees are half a turn, which is how a heading
// in data becomes an angle.
const halfTurn = 180

// door is a door standing proud of a wall: a box, centred on its place
// across and standing on it, half its thickness out of the wall.
type door struct{ Width, Height, Thickness float64 }

// Rendered is the door's box, with its place in its own space and the
// look of the door role: a pane, of the door role.
func (door door) Rendered(level Detail) (Rendered, error) {
	return pane{Width: door.Width, Height: door.Height,
		Thickness: door.Thickness, Role: roleDoor}.Rendered(level)
}

// yurtSizes are the named dimensions of a yurt, in metres, but the
// heading, in degrees.
type yurtSizes struct {
	wallRadius, wallHeight, roofTop, crownRadius      float64
	crownLap, crownSeat, crownRise                    float64
	doorWidth, doorHeight, doorThickness, doorHeading float64
}

// readSizes is the yurt's dimensions from its data, or why one is not
// there.
func readSizes() (yurtSizes, error) {
	var sizes yurtSizes
	for name, into := range map[string]*float64{
		"wall-radius": &sizes.wallRadius, "wall-height": &sizes.wallHeight,
		"roof-top": &sizes.roofTop, "crown-radius": &sizes.crownRadius,
		"crown-lap": &sizes.crownLap, "crown-seat": &sizes.crownSeat,
		"crown-rise": &sizes.crownRise, "door-width": &sizes.doorWidth,
		"door-height": &sizes.doorHeight, "door-thickness": &sizes.doorThickness,
		"door-heading": &sizes.doorHeading,
	} {
		value, err := choice(name)
		if err != nil {
			return yurtSizes{}, err
		}
		*into = value
	}
	return sizes, nil
}

// masses are the yurt's three closed volumes: the lattice wall, the roof
// from the wall's circle up to the crown's, and the crown ring, which
// sinks into the roof's top and stands a little above it.
func (sizes yurtSizes) masses() []mass {
	wall := outlineCircle{Radius: sizes.wallRadius}
	roofEnd := outlineCircle{Radius: sizes.crownRadius - sizes.crownLap}
	ring := outlineCircle{Radius: sizes.crownRadius}
	return []mass{
		{Name: "wall", Role: roleFrame, Place: maths.Local(),
			Form: prism{Bottom: wall, Top: wall, Height: sizes.wallHeight}},
		{Name: "roof", Role: roleCover, Place: lifted(sizes.wallHeight),
			Form: prism{Bottom: wall, Top: roofEnd,
				Height: sizes.roofTop - sizes.wallHeight}},
		{Name: "crown", Role: roleCrown,
			Place: lifted(sizes.roofTop - sizes.crownSeat),
			Form: prism{Bottom: ring, Top: ring,
				Height: sizes.crownSeat + sizes.crownRise}},
	}
}

// lifted is the place a height above the yurt's own origin.
func lifted(height float64) maths.Subspace {
	return maths.NewSubspace(maths.V(0, height, 0), maths.Identity, 0)
}

// doorPlace is where the door stands: on the wall at its heading, turned
// to face out from the yurt's middle.
func (sizes yurtSizes) doorPlace() maths.Subspace {
	turn := maths.Rotation(maths.YAxis, sizes.doorHeading*math.Pi/halfTurn)
	return maths.NewSubspace(turn.Out(maths.V(0, 0, sizes.wallRadius)),
		turn, 0)
}

// yurt is a typical Mongolian ger as a design: a lattice wall, a felt roof
// rising to a crown ring, and a door on the wall. Every dimension and
// every colour is data, and picked.
func yurt() (Design, error) {
	sizes, err := readSizes()
	if err != nil {
		return Design{}, err
	}
	door := door{Width: sizes.doorWidth, Height: sizes.doorHeight,
		Thickness: sizes.doorThickness}
	return Design{Name: "yurt", Massing: sizes.masses(),
		Elements: []placedElement{{Element: door, Place: sizes.doorPlace()}}}, nil
}
