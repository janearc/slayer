// Package cacti makes the physical parts shared by columnar cacti.
package cacti

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// Stem is a pleated column with an optional quarter-circle bend, an upright
// run and a rounded cap. Dimensions are in metres. Curve, Cap and the number
// of sides supplied to Skin must be positive, and sides must be at least 3.
type Stem struct {
	Run, Radius, Bend, Depth float64
	Ribs, Curve, Cap         int
}

type station struct {
	at, along maths.Vec
	radius    float64
}

func (stem Stem) stations() []station {
	var rows []station
	if stem.Bend > 0 {
		for index := 0; index <= stem.Curve; index++ {
			angle := math.Pi * .5 * float64(index) / float64(stem.Curve)
			sine, cosine := math.Sincos(angle)
			at := maths.V(stem.Bend*sine, stem.Bend*(1-cosine), 0)
			rows = append(rows, station{at: at, along: maths.V(cosine, sine, 0), radius: stem.Radius})
		}
	} else {
		rows = append(rows, station{along: maths.YAxis, radius: stem.Radius})
	}
	shoulder := maths.V(stem.Bend, stem.Bend+stem.Run, 0)
	rows = append(rows, station{at: shoulder, along: maths.YAxis, radius: stem.Radius})
	for index := 1; index <= stem.Cap; index++ {
		angle := math.Pi * .5 * float64(index) / float64(stem.Cap)
		sine, cosine := math.Sincos(angle)
		if index == stem.Cap {
			sine, cosine = 1, 0
		}
		rows = append(rows, station{at: shoulder.Add(maths.YAxis.Scale(stem.Radius * sine)), along: maths.YAxis, radius: stem.Radius * cosine})
	}
	return rows
}

func (stem Stem) ring(row station, sides int) []maths.Vec {
	side := maths.V(row.along.Y, -row.along.X, 0)
	ring := make([]maths.Vec, sides)
	for index := range sides {
		angle := 2 * math.Pi * float64(index) / float64(sides)
		sine, cosine := math.Sincos(angle)
		rib := 1 - stem.Depth + stem.Depth*math.Cos(float64(stem.Ribs)*angle)
		radial := side.Scale(cosine).Sub(maths.ZAxis.Scale(sine))
		ring[index] = row.at.Add(radial.Scale(row.radius * rib))
	}
	return ring
}

// Skin returns the ordered faces of a closed column.
func Skin[F geometry.Triangle](stem Stem, sides int) []F {
	var skin []F
	rows := stem.stations()
	previous := stem.ring(rows[0], sides)
	for index := range sides {
		next := (index + 1) % sides
		skin = append(skin, F{rows[0].at, previous[next], previous[index]})
	}
	for _, row := range rows[1:] {
		current := stem.ring(row, sides)
		for index := range previous {
			next := (index + 1) % len(previous)
			if row.radius == 0 {
				skin = append(skin, F{previous[index], previous[next], row.at})
			} else {
				skin = geometry.Quad(skin, previous[index], previous[next], current[next], current[index])
			}
		}
		previous = current
	}
	return skin
}

// Bounds includes the extent of spine tips, including when they are not drawn.
func (stem Stem) Bounds(spineLength float64) maths.Box {
	radius := stem.Radius + spineLength
	bottom := 0.0
	if stem.Bend > 0 {
		bottom = -radius
	}
	return maths.SegmentBox(maths.V(-radius, bottom, -radius), maths.V(stem.Bend+radius, stem.Bend+stem.Run+radius, radius), 0)
}

var errUncountedSpines = errors.New("manufacture: a stem's spines along it are not a count")

// Spines appends bounded, radial spine instances to the straight part of a
// column. The bend and cap remain bare. The prototype can be shared by all
// columns of the plant; the returned placements include the column's pose.
func (stem Stem) Spines(members []shape.MeshMember, prototype shape.Shape,
	pose maths.Subspace, spacing, rootShare float64, limit int,
) ([]shape.MeshMember, error) {
	ratio := stem.Run / spacing
	if !(ratio > 0) || math.IsInf(ratio, 0) || ratio > float64(limit) {
		return nil, errUncountedSpines
	}
	count := int(ratio)
	remaining := limit - len(members)
	if remaining < 0 || stem.Ribs <= 0 || count > remaining/stem.Ribs {
		return nil, fmt.Errorf("manufacture: cactus exceeds spine budget")
	}
	for row := range count {
		share := float64(row+1) / float64(count+1)
		centre := maths.V(stem.Bend, stem.Bend+stem.Run*share, 0)
		for rib := range stem.Ribs {
			angle := (2 * math.Pi) * float64(rib) / float64(stem.Ribs)
			sine, cosine := math.Sincos(angle)
			out := maths.V(cosine, 0, -sine)
			root := centre.Add(out.Scale(stem.Radius * rootShare))
			local := maths.NewSubspace(root, maths.Upright(out), 1)
			members = append(members, shape.MeshMember{Mesh: prototype, Space: local.Within(pose)})
		}
	}
	return members, nil
}
