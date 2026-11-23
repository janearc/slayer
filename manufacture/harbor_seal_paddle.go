package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// A flipper's skin is a flat cushion over its five digits: two sheets,
// one each side of the digits' plane, that meet all round their rim. The
// counts are picked: enough rows and columns that the rim reads round.
const (
	sealPaddleRows    = 20
	sealPaddleColumns = 17
	// sealPaddleRootCap and sealPaddleTipCap are the shares of the
	// length over which the cushion thins to nothing at each end.
	sealPaddleRootCap = .10
	sealPaddleTipCap  = .30
)

// sealPaddleShape is a flipper's flesh round its digits, in metres but
// for the margin: how far the skin runs back from the knuckles into the
// body and on past the digits' tips, how far the web stands past the
// outer digits as a share of the gap between digits, and the cushion's
// half thickness at its root and at its tip. All picked.
type sealPaddleShape struct {
	root, tip, margin, thick, thin float64
}

var (
	// sealForePaddle is the fore flipper's flesh: its root covers the
	// wrist and the forearm's end, inside the trunk's skin.
	sealForePaddle = sealPaddleShape{root: .085, tip: .014, margin: .8,
		thick: .022, thin: .005}
	// sealHindPaddle is the hind flipper's: its root covers the tarsals
	// and the ankle.
	sealHindPaddle = sealPaddleShape{root: .090, tip: .014, margin: .5,
		thick: .022, thin: .005}
)

// sealForeDigitBones is the names of one fore digit's bones from its
// metacarpal out, digits counted from one at the thumb.
func sealForeDigitBones(prefix string, digit int) []string {
	names := []string{fmt.Sprintf("%smetacarpal%d", prefix, digit)}
	for bone := 1; bone < len(sealForeDigits[digit-1]); bone++ {
		names = append(names,
			fmt.Sprintf("%sphalanx%d-%d", prefix, digit, bone))
	}
	return names
}

// sealHindDigitBones is the names of one hind digit's bones from its
// metatarsal out, digits counted from one.
func sealHindDigitBones(prefix string, digit int) []string {
	var names []string
	for bone := range sealHindDigits[digit-1] {
		names = append(names,
			fmt.Sprintf("%sdigit%d-%d", prefix, digit, bone+1))
	}
	return names
}

// digitLine is one digit's middle line at rest through the named bones'
// ends, begun back from the first bone's head and run tip metres past
// the last one's tail.
func (made *sealBones) digitLine(names []string, back maths.Vec,
	tip float64,
) ([]maths.Vec, error) {
	var line []maths.Vec
	for _, name := range names {
		head, tail, found := made.ends(name)
		if !found {
			return nil, fmt.Errorf("harbor seal: no bone %s", name)
		}
		if line == nil {
			line = append(line, head.Add(back), head)
		}
		line = append(line, tail)
	}
	last, before := line[len(line)-1], line[len(line)-2]
	return append(line, last.Add(last.Sub(before).Unit().Scale(tip))), nil
}

// digitLines is a flipper's five digits' lines, each from the names
// bones gives for it, all begun the shape's root back along the middle
// digit's way.
func (made *sealBones) digitLines(bones func(digit int) []string,
	shape sealPaddleShape,
) ([][]maths.Vec, error) {
	const digits = len(sealForeDigits)
	middle := bones((digits + 1) / 2)[0]
	head, tail, found := made.ends(middle)
	if !found {
		return nil, fmt.Errorf("harbor seal: no bone %s", middle)
	}
	back := head.Sub(tail).Unit().Scale(shape.root)
	lines := make([][]maths.Vec, digits)
	for digit := range lines {
		line, err := made.digitLine(bones(digit+1), back, shape.tip)
		if err != nil {
			return nil, err
		}
		lines[digit] = line
	}
	return lines, nil
}

// sealAlong is the point a share of the way along a line, by length.
func sealAlong(line []maths.Vec, share float64) maths.Vec {
	whole := 0.
	for at := 1; at < len(line); at++ {
		whole += line[at].Sub(line[at-1]).Len()
	}
	left := whole * share
	for at := 1; at < len(line)-1; at++ {
		run := line[at].Sub(line[at-1]).Len()
		if left <= run {
			return line[at-1].Add(line[at].Sub(line[at-1]).Scale(left / run))
		}
		left -= run
	}
	last, before := line[len(line)-1], line[len(line)-2]
	return before.Add(last.Sub(before).Unit().Scale(left))
}

// sealPaddleMiddle is the point of the digits' sheet across digits from
// the first, counted in digits and running on past the outer ones, a
// share of the way along.
func sealPaddleMiddle(lines [][]maths.Vec, across, share float64,
) maths.Vec {
	first := min(max(int(math.Floor(across)), 0), len(lines)-2)
	low := sealAlong(lines[first], share)
	high := sealAlong(lines[first+1], share)
	return low.Add(high.Sub(low).Scale(across - float64(first)))
}

// thickness is the cushion's half thickness across digits from the
// first and a share of the way along: round across the flipper, thinning
// to its tip, and closing to nothing at the rim.
func (shape sealPaddleShape) thickness(across, last, share float64,
) float64 {
	offset := (across - last/2) / (last/2 + shape.margin)
	round := math.Sqrt(math.Max(0, 1-offset*offset))
	end := 0.
	if share < sealPaddleRootCap {
		end = 1 - share/sealPaddleRootCap
	}
	if share > 1-sealPaddleTipCap {
		end = (share - (1 - sealPaddleTipCap)) / sealPaddleTipCap
	}
	round *= math.Sqrt(math.Max(0, 1-end*end))
	return (shape.thick + (shape.thin-shape.thick)*share) * round
}

// sealPaddleSurface is a flipper's skin over its digits' lines: the
// upper sheet's points row by row from the root, then the lower sheet's.
func sealPaddleSurface(lines [][]maths.Vec, shape sealPaddleShape,
) sealSurface {
	const rows, columns = sealPaddleRows, sealPaddleColumns
	last := float64(len(lines) - 1)
	middle := lines[len(lines)/2]
	along := middle[len(middle)-1].Sub(middle[0]).Unit()
	span := sealAlong(lines[len(lines)-1], .5).Sub(sealAlong(lines[0], .5))
	normal := along.Cross(span).Unit()
	surface := sealSurface{points: make([]maths.Vec, 2*rows*columns)}
	for row := range rows {
		share := float64(row) / (rows - 1)
		for column := range columns {
			across := -shape.margin +
				(last+2*shape.margin)*float64(column)/(columns-1)
			lift := normal.Scale(shape.thickness(across, last, share))
			at := sealPaddleMiddle(lines, across, share)
			surface.points[row*columns+column] = at.Add(lift)
			surface.points[(rows+row)*columns+column] = at.Sub(lift)
		}
	}
	surface.faces = sealPaddleFaces()
	return surface
}

// sealPaddleRim is whether a grid point lies on the cushion's rim, where
// the two sheets share their points.
func sealPaddleRim(row, column int) bool {
	return row == 0 || row == sealPaddleRows-1 || column == 0 ||
		column == sealPaddleColumns-1
}

// sealPaddlePoint is the place of a grid point's point: in the upper
// sheet, or in the lower, which takes the upper's along the rim.
func sealPaddlePoint(row, column int, lower bool) int {
	at := row*sealPaddleColumns + column
	if lower && !sealPaddleRim(row, column) {
		at += sealPaddleRows * sealPaddleColumns
	}
	return at
}

// sealPaddleFaces is the cushion's triangles: each grid cell cut in two
// on the upper sheet and, turned over, on the lower. A triangle with all
// three corners on the rim is in both sheets at once and is left out.
func sealPaddleFaces() [][3]int {
	var faces [][3]int
	for row := 0; row+1 < sealPaddleRows; row++ {
		for column := 0; column+1 < sealPaddleColumns; column++ {
			cell := [4][2]int{{row, column}, {row + 1, column},
				{row + 1, column + 1}, {row, column + 1}}
			for _, half := range [2][3]int{{0, 1, 3}, {1, 2, 3}} {
				var upper, lower [3]int
				rim := 0
				for at, corner := range half {
					place := cell[corner]
					if sealPaddleRim(place[0], place[1]) {
						rim++
					}
					upper[at] = sealPaddlePoint(place[0], place[1], false)
					lower[len(half)-1-at] = sealPaddlePoint(place[0],
						place[1], true)
				}
				if rim < len(half) {
					faces = append(faces, upper, lower)
				}
			}
		}
	}
	return faces
}
