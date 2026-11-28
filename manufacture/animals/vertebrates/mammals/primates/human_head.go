package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

const headBase = 1.455

// Head stations run from the neck through the jaw and brow into the vault.
// The lower occipital surface turns into the nape, while the upper frontal
// surface carries more of the curvature toward the vertex.
var humanHead = [...]section{
	{headBase, .051, .044, .054},
	{1.467, .049, .048, .055},
	{1.477, .048, .052, .057},
	{1.487, .052, .063, .060},
	{1.497, .057, .074, .066},
	{1.509, .061, .082, .073},
	{1.523, .066, .082, .082},
	{1.541, .071, .079, .091},
	{1.559, .073, .077, .097},
	{1.578, .074, .077, .100},
	{1.598, .074, .079, .101},
	{1.620, .073, .079, .095},
	{1.646, .066, .072, .077},
	{1.669, .055, .059, .052},
	{1.686, .041, .040, .037},
	{1.694, .030, .028, .025},
	{1.697, .022, .020, .016},
	{humanHeight, 0, .002, -.002},
}

func headStations() []float64 {
	const rows = 145
	step := (humanHeight - headBase) / rows
	stations := make([]float64, 0, rows+48)
	for row := range rows {
		y := headBase + float64(row)*step
		stations = append(stations, y)
		// Resolve the lip join and nasal floor more finely than the scalp.
		if y >= 1.502 && y < 1.542 {
			stations = append(stations, y+step/3, y+2*step/3)
		}
	}
	return stations
}

// The facial rise leaves the crown and neck base at their original heights.
// It is constant across both globes and tapers behind the ear toward the
// occiput to keep the back of the vault from climbing with the brow.
func headPosition(chart maths.Vec) maths.Vec {
	face := maths.Smooth(1.476, 1.535, chart.Y)
	forehead := 1 - maths.Smooth(1.576, humanHeight, chart.Y)
	sagittal := mix(.48, 1, maths.Smooth(-.065, -.022, chart.Z))
	chart.Y += .025 * face * forehead * sagittal
	return chart
}

func (model *humanMesh) headVertex(chart maths.Vec, weights jointWeights) int {
	return model.add(humanVertex{point: headPosition(chart), chart: chart, weights: weights})
}

func (anatomy humanAnatomy) head(model *humanMesh, neck []int) {
	stations := headStations()
	sockets := faceEyeSockets(stations)
	nares := nasalSockets(stations)
	var ears [2]socket
	if anatomy.spec.auricles != nil {
		ears = auricleSockets(stations)
	}
	grid := make([][]int, len(stations))
	grid[0] = neck
	for offset, y := range stations[1:] {
		row := offset + 1
		rim := make([]int, humanAround)
		for i := range rim {
			angle := 2 * math.Pi * float64(i) / humanAround
			point := anatomy.headPoint(angle, y)
			weights := blendWeight("neck", "head", maths.Smooth(1.463, 1.515, y))
			rim[i] = model.headVertex(point, weights)
		}
		grid[row] = rim
		lower := grid[row-1]
		for i, first := range lower {
			eye := sockets[0].covers(i, row-1) || sockets[1].covers(i, row-1)
			naris := nares[0].covers(i, row-1) || nares[1].covers(i, row-1)
			ear := anatomy.spec.auricles != nil && (ears[0].covers(i, row-1) || ears[1].covers(i, row-1))
			if !eye && !naris && !ear {
				next := (i + 1) % len(lower)
				model.quad(first, lower[next], rim[next], rim[i])
			}
		}
	}
	vertex := humanHead[len(humanHead)-1]
	crown := model.headVertex(maths.V(0, vertex.y, vertex.front), fixedWeight("head"))
	model.cap(grid[len(grid)-1], crown)
	for i, side := range humanSides {
		model.lashMargins[i] = side.orbit().lids(model, sockets[i].headRim(grid))
		side.naris(model, nares[i].headRim(grid))
	}
	if form := anatomy.spec.auricles; form != nil {
		model.earFaces[0] = len(model.faces)
		for i, side := range humanSides {
			side.auricle(*form).grow(model, ears[i].headRim(grid))
		}
		model.earFaces[1] = len(model.faces)
	}
}

func (opening socket) headRim(grid [][]int) []int {
	rim := opening.rim()
	for i, index := range rim {
		rim[i] = grid[index/humanAround][index%humanAround]
	}
	return rim
}

func headSection(y float64) section {
	index := 0
	for index < len(humanHead)-2 && y > humanHead[index+1].y {
		index++
	}
	start, end := humanHead[index], humanHead[index+1]
	// At the crown the radius goes to zero like the square root of the
	// height, so the last ring approaches a dome instead of a point.
	if index == len(humanHead)-2 {
		share := math.Sqrt((humanHeight - y) / (humanHeight - start.y))
		return section{y, share * start.w,
			mix(end.front, start.front, share), mix(end.back, start.back, share)}
	}
	before := humanHead[max(0, index-1)]
	after := humanHead[min(len(humanHead)-1, index+2)]
	length := end.y - start.y
	share := (y - start.y) / length
	value := func(first, left, right, last float64) float64 {
		from := length * (right - first) / (end.y - before.y)
		to := length * (last - left) / (after.y - start.y)
		if index == len(humanHead)-3 {
			// Match the tangent of the square-root crown segment.
			to = length * (last - right) / (2 * (after.y - end.y))
		}
		return hermite(left, right, from, to, share)
	}
	return section{y, value(before.w, start.w, end.w, after.w),
		value(before.front, start.front, end.front, after.front),
		value(before.back, start.back, end.back, after.back)}
}

func (anatomy humanAnatomy) headPoint(angle, y float64) maths.Vec {
	ring := headSection(y)
	sine, cosine := math.Sincos(angle)
	point := maths.V(ring.w*sine, y, faceDepth(ring, ring.w*math.Abs(sine), cosine))
	if y < 1.497 {
		neck, _ := anatomy.torso.wall(angle, y)
		point = mixPoint(neck, point, maths.Smooth(headBase, 1.497, y))
	}
	point = mandibularShape(point, sine, cosine)
	if anatomy.spec.auricles == nil {
		point.X += sine * pinnaRelief(math.Abs(math.Remainder(angle, 2*math.Pi)), y)
	}
	return point
}

// The lids sample this same facial wall as the surrounding head.
func faceDepth(ring section, x, cosine float64) float64 {
	depth := .5 * ((ring.front+ring.back)*cosine + ring.front - ring.back)
	y := ring.y
	front := maths.Smooth(.16, .86, cosine)
	front *= maths.Smooth(1.470, 1.492, y) * (1 - maths.Smooth(1.607, 1.636, y))
	if front == 0 {
		return depth
	}
	malar := .008 * bell((x-.044)/.024) * bell((y-1.544+.18*(x-.044))/.021)
	buccal := .0032 * bell((x-.032)/.025) * bell((y-1.523)/.023)
	temple := -.0035 * bell((x-.057)/.016) * bell((y-1.578)/.020)
	depth += front * (malar + buccal + temple + orbitalRelief(x, y) + mouthRelief(x, y) + noseRelief(x, y))
	return orbitalDepth(depth, x, y)
}

func pinnaRelief(angle, y float64) float64 {
	tilt := 1.644 - 2.8*(y-1.550)
	arc := (angle - tilt) / .177
	height := (y - 1.551) / .033
	distance := math.Hypot(arc, height)
	helix := .0078 * bell((distance-.82)/.18) * maths.Smooth(-.80, -.49, height)
	concha := -.0054 * bell((arc+.14)/.39) * bell((height+.14)/.39)
	antihelix := .0038 * bell((arc-(.36-.39*height*height))/.16) * bell((height-.08)/.62)
	crura := .0026 * bell((arc+.02+.39*height)/.14) * bell((height-.30)/.28)
	tragus := .0035 * bell((arc+.52)/.17) * bell((height+.16)/.21)
	antitragus := .0025 * bell((arc-.04)/.19) * bell((height+.46)/.16)
	lobe := .0072 * bell((arc-.12)/.53) * bell((height+.78)/.25)
	return .011*bell(distance/.93) + helix + concha + antihelix + crura + tragus + antitragus + lobe
}
