package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Pycnofibres are the hair-like filaments that covered the bodies of
// pterosaurs; the sources this repository cites for the animal (Kellner
// et al. 2010, PMC2842671, on Jeholopterus) report them on soft-tissue
// specimens, and the fibres of Hatz itself are unknown. Here each is a
// real thin blade of geometry, rooted in the skin of the trunk and neck,
// combed back along the body and standing a little off it. They are not
// paint. The wings carry none: their covering is the membrane's own
// stiffening fibres, drawn on the membrane.

// The numbers the fibres use.
const (
	// fibresPerCell is how many fibres are rooted in each cell of the
	// torso's mesh. Authored.
	fibresPerCell = 3
	// fibreShortest and fibreLongest bound a fibre's length, in metres.
	// Authored: a few centimetres, as on the small pterosaurs reported.
	fibreShortest = .028
	fibreLongest  = .058
	// fibreWidth is a fibre's width at its root, in metres.
	fibreWidth = .0035
	// fibreSink is how far a root sits inside the skin, in metres.
	fibreSink = .002
	// fibreComb and fibreLift are the shares of a fibre's direction that
	// run back along the body and out from it; fibreScatter is the share
	// that is random.
	fibreComb    = .75
	fibreLift    = .50
	fibreScatter = .25
	// fibreBelly is the density of fibres on the underside as a share of
	// the density on top. Authored.
	fibreBelly = .35
	// fibreTailEnd is how many rings at the tail are left bare.
	fibreTailEnd = 12
)

// fibreHash is a repeatable value from 0 to 1 for a fibre's cell and
// which of its fibres, and what the value is for.
func fibreHash(ring, side, which, purpose int) float64 {
	return pterosaurHash(ring*fibresPerCell+which, side, purpose)
}

// fibreDensity is the chance, from 0 to 1, that a cell has its fibres:
// full on the back and flanks, thin on the belly.
func fibreDensity(side, sides int) float64 {
	angle := 2 * math.Pi * float64(side) / float64(sides)
	if math.Sin(angle) < -.5 {
		return fibreBelly
	}
	return 1
}

// pycnofibres are blades rooted in the skin of a lofted body, as points
// and triangles. Ring 0 of the surface is taken to be the tail end.
func pycnofibres(surface tubeSurface) ([]maths.Vec, [][3]int) {
	faces := surface.faces
	sign := 1.0
	if solidVolume(surface.points, faces) < 0 {
		sign = -1
	}
	normals := faceNormals(surface.points, faces)
	var points []maths.Vec
	var blades [][3]int
	for ring := fibreTailEnd; ring < surface.rings-1; ring++ {
		for side := 0; side < surface.sides; side++ {
			density := fibreDensity(side, surface.sides)
			if fibreHash(ring, side, 0, 0) > density {
				continue
			}
			for which := 0; which < fibresPerCell; which++ {
				cell := fibreCell{surface, normals, sign, ring,
					side, which}
				root, tip, across := cell.blade()
				first := len(points)
				points = append(points, root.Sub(across),
					root.Add(across), tip)
				blades = append(blades,
					[3]int{first, first + 1, first + 2})
			}
		}
	}
	return points, blades
}

// fibreCell is one fibre's place: a cell of the surface and which of the
// cell's fibres it is.
type fibreCell struct {
	surface tubeSurface
	normals []maths.Vec
	sign    float64
	ring    int
	side    int
	which   int
}

// hash is the cell's repeatable value from 0 to 1 for a purpose.
func (cell fibreCell) hash(purpose int) float64 {
	return fibreHash(cell.ring, cell.side, cell.which, purpose)
}

// blade is the fibre's root centre, its tip, and its half width vector.
func (cell fibreCell) blade() (maths.Vec, maths.Vec, maths.Vec) {
	surface, ring, side := cell.surface, cell.ring, cell.side
	here := surface.at(ring, side)
	along := surface.at(ring+1, side).Sub(here)
	around := surface.at(ring, side+1).Sub(here)
	root := here.Add(along.Scale(cell.hash(1))).
		Add(around.Scale(cell.hash(2)))
	index := ring*surface.sides + side%surface.sides
	normal := cell.normals[index].Scale(cell.sign).Unit()
	scatter := maths.V(cell.hash(3)-.5, cell.hash(4)-.5, cell.hash(5)-.5)
	direction := along.Unit().Neg().Scale(fibreComb).
		Add(normal.Scale(fibreLift)).
		Add(scatter.Scale(fibreScatter)).Unit()
	length := fibreShortest + (fibreLongest-fibreShortest)*cell.hash(6)
	root = root.Sub(normal.Scale(fibreSink))
	return root, root.Add(direction.Scale(length)),
		around.Unit().Scale(fibreWidth / 2)
}
