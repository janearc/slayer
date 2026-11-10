package island

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/frame"
)

// normalAt estimates the source surface's normal from four neighbouring
// source heights in the fixed world. The stencil does not change with the
// camera or leaf depth. It is a shading approximation, not new geometry or
// proof of source resolution, relief error or a shadow-ray offset bound.
func (book *vertexBook) normalAt(key lattice) (maths.Vec, error) {
	if normal, found := book.normals[key]; found {
		return normal, nil
	}
	if len(book.normals) >= book.terrain.detail.Points {
		return maths.Vec{}, fmt.Errorf("island: normal stencil budget exhausted")
	}
	lon, lat := book.terrain.location(float64(key.column), float64(key.row))
	detail := book.terrain.detail
	coordinates := [4][2]float64{{lon - detail.NormalLongitude, lat}, {lon + detail.NormalLongitude, lat}, {lon, lat + detail.NormalLatitude}, {lon, lat - detail.NormalLatitude}}
	var points [4]maths.Vec
	for index, coordinate := range coordinates {
		book.normalQueries++
		height, has, err := book.terrain.from.Height(coordinate[0], coordinate[1])
		if err != nil {
			return maths.Vec{}, err
		}
		if !has || !finite(height) || height < detail.MinimumHeight || height > detail.MaximumHeight {
			return maths.Vec{}, fmt.Errorf("island: normal stencil needs finite source heights within the declared envelope")
		}
		points[index], err = book.terrain.frame.Position(frame.Location{Longitude: measure.Degrees(coordinate[0]), Latitude: measure.Degrees(coordinate[1]), Height: measure.Metres(height)})
		if err != nil {
			return maths.Vec{}, err
		}
	}
	// The source row direction is south, so south cross east faces outward.
	across := points[1].Sub(points[0])
	down := points[3].Sub(points[2])
	normal := down.Cross(across)
	length := normal.Len()
	if !finite(length) || length == 0 {
		return maths.Vec{}, fmt.Errorf("island: degenerate source normal stencil")
	}
	normal = normal.Scale(1 / length)
	book.normals[key] = normal
	return normal, nil
}
