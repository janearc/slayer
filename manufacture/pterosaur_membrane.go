package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The membrane is a surface stretched between pinned boundaries. Along
// one boundary (the leading edge: bones, or the body) the skin is fixed
// to the skeleton. The other boundary is another set of pinned points
// or the free trailing edge. Between a point of the first and its
// partner on the second runs a chord of skin of fixed length, the length
// it has in the glide pose plus a little slack. Where the pose brings the
// two ends closer than that length, the chord does not shrink and does
// not tear: it pleats out of the plane, with the arc length held, so the
// skin's area survives a fold. Where the pose pulls the ends further
// apart than the length, the chord goes straight and the stretch is a
// number the caller can read. See doc/bioforms/hatz-next/02-plan.md.

// The numbers the membrane uses.
const (
	// slackDenseSteps is how finely one chord is drawn to measure its
	// length.
	slackDenseSteps = 240
	// slackBisections is how many halvings find a pleat's depth.
	slackBisections = 40
	// slackStraight is the relative slack below which a chord is drawn
	// straight.
	slackStraight = 1e-4
	// membraneDegenerate is the length of cross product below which a
	// chord is taken to lie along the boundary.
	membraneDegenerate = 1e-6
)

// pleatLine is count+1 points from from to to, with a pleat of depth
// amplitude and waves waves on the given direction. The pleat is zero at
// both ends.
func pleatLine(from, to, normal maths.Vec, amplitude, waves float64,
	count int) []maths.Vec {
	line := make([]maths.Vec, count+1)
	for i := range line {
		share := float64(i) / float64(count)
		depth := amplitude * math.Sin(math.Pi*share) *
			math.Sin(2*math.Pi*waves*share)
		onLine := from.Add(to.Sub(from).Scale(share))
		line[i] = onLine.Add(normal.Scale(depth))
	}
	return line
}

// polylineLength is the length of a path through the points.
func polylineLength(points []maths.Vec) float64 {
	total := 0.0
	for i := 1; i < len(points); i++ {
		total += points[i].Sub(points[i-1]).Len()
	}
	return total
}

// slackPolyline is a dense path from from to to of the given length,
// pleated along normal when the length exceeds the distance, and straight
// when it does not. The pleat count grows with the slack, so a deep fold
// makes many shallow pleats and not one tall one.
func slackPolyline(from, to, normal maths.Vec, length float64) []maths.Vec {
	distance := to.Sub(from).Len()
	slack := length/distance - 1
	if slack <= slackStraight {
		return pleatLine(from, to, normal, 0, 0, slackDenseSteps)
	}
	waves := pterosaurPleatBase + pterosaurPleatGain*
		clamp(slack/pterosaurPleatSaturation, 0, 1)
	low, high := 0.0, length
	for i := 0; i < slackBisections; i++ {
		middle := (low + high) / 2
		path := pleatLine(from, to, normal, middle, waves,
			slackDenseSteps)
		if polylineLength(path) < length {
			low = middle
		} else {
			high = middle
		}
	}
	return pleatLine(from, to, normal, (low+high)/2, waves, slackDenseSteps)
}

// atFractions is the points of a path at the given shares of its length,
// each from 0 to 1 in order.
func atFractions(path []maths.Vec, fractions []float64) []maths.Vec {
	cumulative := make([]float64, len(path))
	for i := 1; i < len(path); i++ {
		cumulative[i] = cumulative[i-1] + path[i].Sub(path[i-1]).Len()
	}
	total := cumulative[len(path)-1]
	points := make([]maths.Vec, len(fractions))
	next := 1
	for i, share := range fractions {
		goal := share * total
		for next < len(path)-1 && cumulative[next] < goal {
			next++
		}
		span := cumulative[next] - cumulative[next-1]
		local := 0.0
		if span > 0 {
			local = clamp((goal-cumulative[next-1])/span, 0, 1)
		}
		step := path[next].Sub(path[next-1])
		points[i] = path[next-1].Add(step.Scale(local))
	}
	return points
}

// evenFractions is steps+1 shares evenly spaced from 0 to 1.
func evenFractions(steps int) []float64 {
	shares := make([]float64, steps+1)
	for i := range shares {
		shares[i] = float64(i) / float64(steps)
	}
	return shares
}

// planeNormal is the unit direction at right angles to both a and b,
// turned to the same side as away; away itself if a and b are parallel.
func planeNormal(a, b, away maths.Vec) maths.Vec {
	normal := a.Cross(b)
	if normal.Len() < membraneDegenerate {
		return away.Unit()
	}
	if normal.Dot(away) < 0 {
		normal = normal.Neg()
	}
	return normal.Unit()
}

// membraneSpec is one panel of skin: its two boundaries sampled along its
// span, the rest length of each chord between them, and the side the
// pleats fall to.
type membraneSpec struct {
	lead, far []maths.Vec
	rest      []float64
	away      maths.Vec
	steps     int
	// joints are the arc lengths along the leading edge of the bones'
	// joints where the skin is creased.
	joints []float64
}

// The numbers of the wrinkles at the joints.
const (
	// wrinkleDepth is the crease's depth in metres. Authored.
	wrinkleDepth = .02
	// wrinkleReach is how far along the edge a joint's creases reach,
	// and wrinkleWave the spacing of the creases, both in metres.
	wrinkleReach = .3
	wrinkleWave  = .22
	// wrinkleTaper is how much the creases fade toward the far edge.
	wrinkleTaper = .6
)

// wrinkle is the depth of the skin's creases, in metres, at an arc length
// along the leading edge and a share of the way across the chord: nothing
// on the bone, deepest a little way out, fading with the distance from
// each joint.
func (spec membraneSpec) wrinkle(arc, share float64) float64 {
	envelope := math.Sqrt(share) * (1 - wrinkleTaper*share)
	total := 0.0
	for _, joint := range spec.joints {
		offset := arc - joint
		reach := offset / wrinkleReach
		total += math.Exp(-reach*reach) *
			math.Cos(2*math.Pi*offset/wrinkleWave)
	}
	return wrinkleDepth * envelope * total
}

// membraneSurface is a panel posed: rows of chord samples, the same
// points in the flat rest layout, and the triangles.
type membraneSurface struct {
	rows, columns int
	points, rest  []maths.Vec
	faces         [][3]int
}

// chordRows poses every chord of the panel.
func (spec membraneSpec) chordRows() [][]maths.Vec {
	rows := make([][]maths.Vec, len(spec.lead))
	last := len(spec.lead) - 1
	arc := spec.leadArc()
	for i := range rows {
		before, after := int(math.Max(0, float64(i-1))),
			int(math.Min(float64(last), float64(i+1)))
		tangent := spec.lead[after].Sub(spec.lead[before])
		chord := spec.far[i].Sub(spec.lead[i])
		normal := planeNormal(chord, tangent, spec.away)
		path := slackPolyline(spec.lead[i], spec.far[i], normal,
			spec.rest[i])
		rows[i] = atFractions(path, evenFractions(spec.steps))
		for j := range rows[i] {
			share := float64(j) / float64(spec.steps)
			depth := spec.wrinkle(arc[i], share)
			rows[i][j] = rows[i][j].Add(normal.Scale(depth))
		}
	}
	return rows
}

// leadArc is the arc length along the leading boundary to each sample.
func (spec membraneSpec) leadArc() []float64 {
	arc := make([]float64, len(spec.lead))
	for i := 1; i < len(arc); i++ {
		arc[i] = arc[i-1] + spec.lead[i].Sub(spec.lead[i-1]).Len()
	}
	return arc
}

// surface poses the panel: the chords, pushed out of the given balls so
// skin never passes through the body, and the flat layout of the same
// points: the arc along the leading edge across, the chord length up.
func (spec membraneSpec) surface(balls []pterosaurBall) membraneSurface {
	rows := spec.chordRows()
	columns := spec.steps + 1
	made := membraneSurface{rows: len(rows), columns: columns}
	arc := spec.leadArc()
	for i, row := range rows {
		for j, point := range row {
			made.points = append(made.points,
				pushOutside(point, balls))
			share := float64(j) / float64(spec.steps)
			made.rest = append(made.rest,
				maths.V(arc[i], share*spec.rest[i], 0))
		}
	}
	made.faces = gridFaces(len(rows), columns)
	if gridFacing(made.points, made.faces).Dot(spec.away) < 0 {
		made.faces = flippedFaces(made.faces)
	}
	return made
}

// gridFaces is the triangles of a grid of rows by columns vertices,
// numbered row by row.
func gridFaces(rows, columns int) [][3]int {
	var faces [][3]int
	for i := 0; i < rows-1; i++ {
		for j := 0; j < columns-1; j++ {
			a, b := i*columns+j, (i+1)*columns+j
			c, d := (i+1)*columns+j+1, i*columns+j+1
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	return faces
}

// gridFacing is the sum of the faces' area weighted normals: which way
// the sheet as a whole faces.
func gridFacing(points []maths.Vec, faces [][3]int) maths.Vec {
	total := maths.V(0, 0, 0)
	for _, face := range faces {
		a, b, c := points[face[0]], points[face[1]], points[face[2]]
		total = total.Add(b.Sub(a).Cross(c.Sub(a)))
	}
	return total
}

// pterosaurBall is a sphere the skin must stay outside: a stand-in for
// the body's volume where the membrane meets it.
type pterosaurBall struct {
	centre maths.Vec
	radius float64
}

// pterosaurBallMargin is how far outside a ball the skin is held, in
// metres, so it rests on the body and not in it.
const pterosaurBallMargin = .004

// pushOutside is the point moved out of any ball it is inside, to the
// ball's surface plus the margin.
func pushOutside(point maths.Vec, balls []pterosaurBall) maths.Vec {
	for _, ball := range balls {
		offset := point.Sub(ball.centre)
		reach := ball.radius + pterosaurBallMargin
		if distance := offset.Len(); distance < reach && distance > 0 {
			point = ball.centre.Add(offset.Scale(reach / distance))
		}
	}
	return point
}

// fibreRidges is the stiffening fibres laid on a membrane: a ridge of the
// given width and height along every every-th chord, built on the sheet's
// own points so it folds and flaps with it. The second result is the
// triangles.
func (made membraneSurface) fibreRidges(every int, width,
	height float64) ([]maths.Vec, [][3]int) {
	normals := faceNormals(made.points, made.faces)
	var points []maths.Vec
	var faces [][3]int
	for i := every; i < made.rows-1; i += every {
		start := len(points)
		for j := 1; j < made.columns-1; j++ {
			here := i*made.columns + j
			centre := made.points[here]
			across := made.points[here+made.columns].Sub(
				made.points[here-made.columns])
			along := across.Unit().Scale(width / 2)
			lift := normals[here].Unit().Scale(height)
			points = append(points, centre.Sub(along),
				centre.Add(lift), centre.Add(along))
		}
		faces = append(faces, ridgeFaces(start, made.columns-2)...)
	}
	return points, faces
}

// ridgeFaces is the triangles of a ridge of count sections of three points
// each, starting at point first: two strips, one up each side.
func ridgeFaces(first, count int) [][3]int {
	var faces [][3]int
	for j := 0; j < count-1; j++ {
		a := first + 3*j
		b := a + 3
		faces = append(faces,
			[3]int{a, b, a + 1}, [3]int{a + 1, b, b + 1},
			[3]int{a + 1, b + 1, a + 2},
			[3]int{a + 2, b + 1, b + 2})
	}
	return faces
}
