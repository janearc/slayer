package manufacture

// surveyMirror is a backed annular reflective skin. Its shallow figure is
// authored scenery; source diameters do not supply an optical prescription.
type surveyMirror struct {
	outer, inner, depth, backing float64
}

// skin keeps a real central opening, including the inner wall and underside.
func (mirror surveyMirror) skin(fine cut) surface {
	points := [][coordinatePair]float64{
		{mirror.inner, -mirror.backing},
		{mirror.outer, -mirror.backing},
		{mirror.outer, mirror.depth}}
	minimum := int(groundValue("survey-radial-minimum"))
	interval := int(groundValue("survey-radial-cuts-per-step"))
	steps := max(minimum, fine.around/interval)
	squaredRadius := mirror.outer * mirror.outer
	for index := range steps {
		step := steps - spaceOne - index
		share := float64(step) / float64(steps)
		radius := mirror.inner + (mirror.outer-mirror.inner)*share
		height := mirror.depth * radius * radius / squaredRadius
		points = append(points, [coordinatePair]float64{radius, height})
	}
	return (turnedProfile{points: points}).skin(fine)
}
