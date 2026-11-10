package manufacture

// radioDish keeps a thin axial shell behind the paraboloid instead of
// borrowing the optical mirror's flat, solid backing. Thickness is scenery.
type radioDish struct{ radius, depth, thickness float64 }

// skin closes both curved faces at the rim and their exact axial poles.
func (dish radioDish) skin(fine cut) surface {
	levels := int(summitValue("radio-radial-levels"))
	var points [][coordinatePair]float64
	for index := range levels + spaceOne {
		point := dish.point(index, levels)
		point[spaceOne] -= dish.thickness
		points = append(points, point)
	}
	for index := levels; index >= spaceZero; index-- {
		points = append(points, dish.point(index, levels))
	}
	return (turnedProfile{points: points}).skin(fine)
}

// point follows z=r*r/(4*f), with depth already derived from focal ratio.
func (dish radioDish) point(index, levels int) [coordinatePair]float64 {
	share := float64(index) / float64(levels)
	return [coordinatePair]float64{dish.radius * share, dish.depth * share * share}
}
