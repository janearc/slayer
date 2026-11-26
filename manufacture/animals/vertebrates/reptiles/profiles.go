// Package reptiles describes the distinct authored reptile studies.
package reptiles

import (
	_ "embed"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/janearc/slayer/manufacture/profilecheck"
)

//go:embed profiles.yaml
var embedded []byte

// Profile holds the authored defaults for one explicitly named scaffold.
// Its pose controls are not anatomical measurements.
type Profile struct {
	Kind, Scaffold                     string
	Phase, Yaw, Posture, Fold, Seconds float64
	Scale                              float64
	Axial                              [8][6]float64
	Sections                           [16][4]float64
	Masses                             [3][6]float64
	Body                               [8][5]float64
	Cage                               [11][4]float64
	Density                            [8][7]float64
	ContinuousHead                     [7]float64
	ContinuousFins                     [6]float64
	FinRoots, FinOrigin                [2]float64
	FinSections                        [6][4]float64
	Eye                                [4]float64
	LimbRoots                          [2]float64
	LimbPoints                         [3][3]float64
	LimbRadii                          [3]float64
	Wing                               [3][3]float64
	WingLengths                        [3]float64
	CageWing                           [5][3]float64
	CageWingLengths                    [4]float64
	CageLeg                            [3][3]float64
	CageLegLengths                     [2]float64
	CageBends                          [3][3]float64
}

type authored struct {
	Kind, Scaffold, Status, By, Date, Source string
	Phase, Yaw, Posture, Fold, Seconds       *float64
	Scale                                    *float64
	Axial                                    *[8][6]float64
	Sections                                 *[16][4]float64
	Masses                                   *[3][6]float64
	Body                                     *[8][5]float64
	Cage                                     *[11][4]float64
	Density                                  *[8][7]float64
	ContinuousHead                           *[7]float64    `yaml:"continuous-head"`
	ContinuousFins                           *[6]float64    `yaml:"continuous-fins"`
	FinRoots                                 *[2]float64    `yaml:"fin-roots"`
	FinOrigin                                *[2]float64    `yaml:"fin-origin"`
	FinSections                              *[6][4]float64 `yaml:"fin-sections"`
	Eye                                      *[4]float64
	LimbRoots                                *[2]float64    `yaml:"limb-roots"`
	LimbPoints                               *[3][3]float64 `yaml:"limb-points"`
	LimbRadii                                *[3]float64    `yaml:"limb-radii"`
	Wing                                     *[3][3]float64
	WingLengths                              *[3]float64    `yaml:"wing-lengths"`
	CageWing                                 *[5][3]float64 `yaml:"cage-wing"`
	CageWingLengths                          *[4]float64    `yaml:"cage-wing-lengths"`
	CageLeg                                  *[3][3]float64 `yaml:"cage-leg"`
	CageLegLengths                           *[2]float64    `yaml:"cage-leg-lengths"`
	CageBends                                *[3][3]float64 `yaml:"cage-bends"`
}

// Book owns validated study profiles.
type Book struct{ profiles map[string]Profile }

// Embedded returns an independent copy of the default authored book.
func Embedded() (string, []byte) { return "reptile-profiles.yaml", slices.Clone(embedded) }

// Read checks the complete book and copies its values.
func Read(file string, raw []byte) (*Book, error) {
	rows, err := profilecheck.Rows[authored](file, raw, "reptile book", "reptile profile", nil)
	if err != nil {
		return nil, err
	}
	book := &Book{profiles: make(map[string]Profile, len(rows))}
	for _, row := range rows {
		if !profilecheck.Authored(strings.TrimSpace(row.Kind), row.Status,
			strings.TrimSpace(row.By), strings.TrimSpace(row.Date), strings.TrimSpace(row.Source)) {
			return nil, fmt.Errorf("%s: invalid reptile provenance for %q", file, row.Kind)
		}
		if _, exists := book.profiles[row.Kind]; exists {
			return nil, fmt.Errorf("%s: duplicate reptile %q", file, row.Kind)
		}
		valid := func(p *float64, max float64) bool {
			return p != nil && !math.IsNaN(*p) && !math.IsInf(*p, 0) && math.Abs(*p) <= max
		}
		if !valid(row.Phase, 100000) {
			return nil, fmt.Errorf("%s: invalid phase for %q", file, row.Kind)
		}
		p := Profile{Kind: row.Kind, Scaffold: row.Scaffold, Phase: *row.Phase}
		cageRig := row.CageWing != nil || row.CageWingLengths != nil || row.CageLeg != nil || row.CageLegLengths != nil || row.CageBends != nil
		switch row.Scaffold {
		case "pliosaur":
			if row.Yaw != nil || row.Posture != nil || row.Fold != nil || row.Seconds != nil || !valid(row.Scale, 10) || *row.Scale < .01 || row.Axial != nil || row.Sections == nil || row.Masses == nil || row.Body != nil || row.Cage != nil || row.Density != nil || row.ContinuousHead == nil || row.ContinuousFins == nil || row.FinRoots == nil || row.FinOrigin == nil || row.FinSections == nil || row.Eye == nil || row.LimbRoots != nil || row.LimbPoints != nil || row.LimbRadii != nil || row.Wing != nil || row.WingLengths != nil || cageRig {
				return nil, fmt.Errorf("%s: invalid pliosaur controls", file)
			}
			p.Scale, p.Sections, p.Masses = *row.Scale, *row.Sections, *row.Masses
			p.FinRoots, p.FinOrigin, p.FinSections, p.Eye = *row.FinRoots, *row.FinOrigin, *row.FinSections, *row.Eye
			p.ContinuousHead, p.ContinuousFins = *row.ContinuousHead, *row.ContinuousFins
			if !validContinuousPliosaur(p.ContinuousHead, p.ContinuousFins) {
				return nil, fmt.Errorf("%s: invalid continuous pliosaur head or fins", file)
			}
			for _, values := range [][]float64{p.FinRoots[:], p.FinOrigin[:], p.Eye[:]} {
				if !boundedValues(values[:], 10) {
					return nil, fmt.Errorf("%s: invalid pliosaur landmarks", file)
				}
			}
			if p.FinRoots[0] <= 0 || p.FinRoots[1] >= 0 || p.FinOrigin[1] <= 0 || p.Eye[3] < .001 {
				return nil, fmt.Errorf("%s: invalid pliosaur landmarks", file)
			}
			for i, f := range p.FinSections {
				if !boundedValues(f[:], 10) {
					return nil, fmt.Errorf("%s: invalid pliosaur fin section %d", file, i)
				}
				if f[1] < .001 || f[2] < .001 || (i > 0 && f[0] <= p.FinSections[i-1][0]) || (i == 0 && f[0] != p.FinOrigin[1]) {
					return nil, fmt.Errorf("%s: invalid pliosaur fin section %d", file, i)
				}
			}
			for i, section := range p.Sections {
				if !boundedValues(section[:], 10) {
					return nil, fmt.Errorf("%s: invalid pliosaur section %d", file, i)
				}
				if section[2] < .001 || section[3] < .001 || (i > 0 && section[0] <= p.Sections[i-1][0]) {
					return nil, fmt.Errorf("%s: invalid pliosaur section %d", file, i)
				}
			}
			for i, mass := range p.Masses {
				for j, n := range mass {
					if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1.8 || (j >= 3 && n < .001) {
						return nil, fmt.Errorf("%s: invalid pliosaur mass %d", file, i)
					}
				}
			}
		case "crocodylian":
			if !valid(row.Yaw, 1) || !valid(row.Posture, 1) || row.Fold != nil || row.Seconds != nil || !valid(row.Scale, 10) || *row.Scale < .01 || row.Axial == nil || row.Sections != nil || row.Masses != nil || row.Body != nil || row.Cage != nil || row.Density != nil || row.ContinuousHead != nil || row.ContinuousFins != nil || row.FinRoots != nil || row.FinOrigin != nil || row.FinSections != nil || row.Eye != nil || row.LimbRoots == nil || row.LimbPoints == nil || row.LimbRadii == nil || row.Wing != nil || row.WingLengths != nil || cageRig {
				return nil, fmt.Errorf("%s: invalid crocodylian controls", file)
			}
			p.Yaw, p.Posture = *row.Yaw, *row.Posture
			p.Scale, p.Axial = *row.Scale, *row.Axial
			p.LimbRoots, p.LimbPoints, p.LimbRadii = *row.LimbRoots, *row.LimbPoints, *row.LimbRadii
			for _, n := range p.LimbRoots {
				if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1.8 {
					return nil, fmt.Errorf("%s: invalid crocodylian limb roots", file)
				}
			}
			if p.LimbRoots[0] <= p.LimbRoots[1] {
				return nil, fmt.Errorf("%s: invalid crocodylian limb roots", file)
			}
			for i, point := range p.LimbPoints {
				if !boundedValues(point[:], 1.5) {
					return nil, fmt.Errorf("%s: invalid crocodylian limb point %d", file, i)
				}
				if math.IsNaN(p.LimbRadii[i]) || math.IsInf(p.LimbRadii[i], 0) || p.LimbRadii[i] < .001 || p.LimbRadii[i] > 1 {
					return nil, fmt.Errorf("%s: invalid crocodylian limb radius %d", file, i)
				}
				if i > 0 {
					prev := p.LimbPoints[i-1]
					dx, dy, dz := point[0]-prev[0], point[1]-prev[1], point[2]-prev[2]
					if dx*dx+dy*dy+dz*dz < 1e-10 {
						return nil, fmt.Errorf("%s: coincident crocodylian joints", file)
					}
				}
			}
			for i, s := range p.Axial {
				if !boundedValues(s[:], 1.8) {
					return nil, fmt.Errorf("%s: invalid crocodylian station %d", file, i)
				}
				if s[1] <= s[0] || (i > 0 && s[0] != p.Axial[i-1][1]) || s[2] < .001 || s[3] < .001 || s[4] < .001 || s[5] < .001 {
					return nil, fmt.Errorf("%s: invalid crocodylian station %d", file, i)
				}
			}
		case "pterosaur":
			if !valid(row.Fold, 1) || *row.Fold < 0 || !valid(row.Seconds, 100000) || row.Yaw != nil || row.Posture != nil || row.Scale != nil || row.Axial != nil || row.Sections != nil || row.Masses != nil || row.Body == nil || row.Cage == nil || row.Density == nil || row.ContinuousHead != nil || row.ContinuousFins != nil || row.FinRoots != nil || row.FinOrigin != nil || row.FinSections != nil || row.Eye != nil || row.LimbRoots != nil || row.LimbPoints != nil || row.LimbRadii != nil || row.Wing == nil || row.WingLengths == nil || row.CageWing == nil || row.CageWingLengths == nil || row.CageLeg == nil || row.CageLegLengths == nil || row.CageBends == nil {
				return nil, fmt.Errorf("%s: invalid pterosaur controls", file)
			}
			p.Fold, p.Seconds = *row.Fold, *row.Seconds
			p.Body, p.Cage, p.Density = *row.Body, *row.Cage, *row.Density
			p.Wing, p.WingLengths = *row.Wing, *row.WingLengths
			if !validWing(p.Wing, p.WingLengths) {
				return nil, fmt.Errorf("%s: invalid pterosaur wing rig", file)
			}
			p.CageWing, p.CageWingLengths, p.CageLeg, p.CageLegLengths, p.CageBends = *row.CageWing, *row.CageWingLengths, *row.CageLeg, *row.CageLegLengths, *row.CageBends
			if !validCage(p) {
				return nil, fmt.Errorf("%s: invalid pterosaur cage rig", file)
			}
			for i, s := range p.Body {
				if !boundedValues(s[:], 10) {
					return nil, fmt.Errorf("%s: invalid pterosaur body station %d", file, i)
				}
				if s[3] < .001 || s[4] < .001 || (i > 0 && s[0] <= p.Body[i-1][0]) {
					return nil, fmt.Errorf("%s: invalid pterosaur body station %d", file, i)
				}
			}
			for i, s := range p.Cage {
				if !boundedValues(s[:], 10) {
					return nil, fmt.Errorf("%s: invalid pterosaur cage station %d", file, i)
				}
				if s[2] < .001 || s[3] < .001 || (i > 0 && s[0] <= p.Cage[i-1][0]) {
					return nil, fmt.Errorf("%s: invalid pterosaur cage station %d", file, i)
				}
			}
			for i, s := range p.Density {
				for j, n := range s {
					if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 5 || (j >= 3 && n < .001) {
						return nil, fmt.Errorf("%s: invalid pterosaur density mass %d", file, i)
					}
				}
			}
		default:
			return nil, fmt.Errorf("%s: unknown reptile scaffold %q", file, row.Scaffold)
		}
		book.profiles[row.Kind] = p
	}
	return book, nil
}

func validCage(p Profile) bool {
	var wing [3][3]float64
	copy(wing[:], p.CageWing[:3])
	if !validWing(wing, [3]float64{p.CageWingLengths[0], p.CageWingLengths[1], 1}) ||
		!validWing(p.CageLeg, [3]float64{p.CageLegLengths[0], p.CageLegLengths[1], 1}) {
		return false
	}
	for _, point := range p.CageWing[3:] {
		if !boundedValues(point[:], 10) {
			return false
		}
	}
	for _, length := range p.CageWingLengths[2:] {
		if math.IsNaN(length) || math.IsInf(length, 0) || length < .001 || length > 10 {
			return false
		}
	}
	for _, bend := range p.CageBends {
		length2 := 0.0
		for _, n := range bend {
			if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 10 {
				return false
			}
			length2 += n * n
		}
		if length2 < 1e-12 {
			return false
		}
	}
	return true
}

// validWing checks that both wrist endpoints and the path between them can
// be reached by the two arm segments; it does not claim flight dynamics.
func validWing(points [3][3]float64, lengths [3]float64) bool {
	for _, p := range points {
		if !boundedValues(p[:], 10) {
			return false
		}
	}
	for _, n := range lengths {
		if math.IsNaN(n) || math.IsInf(n, 0) || n < .001 || n > 10 {
			return false
		}
	}
	if points[0][2] <= 0 || points[1][2] <= 0 || points[2][2] <= 0 {
		return false
	}
	var start, delta [3]float64
	var start2, end2, dot, step2 float64
	for i := range 3 {
		start[i] = points[1][i] - points[0][i]
		delta[i] = points[2][i] - points[1][i]
		start2 += start[i] * start[i]
		end := points[2][i] - points[0][i]
		end2 += end * end
		dot += start[i] * delta[i]
		step2 += delta[i] * delta[i]
	}
	t := 0.0
	if step2 > 0 {
		t = math.Max(0, math.Min(1, -dot/step2))
	}
	min2 := 0.0
	for i := range 3 {
		n := start[i] + t*delta[i]
		min2 += n * n
	}
	perpStep := delta[1]*delta[1] + delta[2]*delta[2]
	perpT := 0.0
	if perpStep > 0 {
		perpT = math.Max(0, math.Min(1, -(start[1]*delta[1]+start[2]*delta[2])/perpStep))
	}
	y, z := start[1]+perpT*delta[1], start[2]+perpT*delta[2]
	return min2 > 1e-12 && y*y+z*z > 1e-12 && math.Sqrt(min2) >= math.Abs(lengths[0]-lengths[1]) && math.Sqrt(math.Max(start2, end2)) <= lengths[0]+lengths[1]
}

// boundedValues accepts only finite values within a symmetric bound.
func boundedValues(values []float64, limit float64) bool {
	for _, n := range values {
		if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > limit {
			return false
		}
	}
	return true
}

// Get returns a value copy of one checked profile.
func (book *Book) Get(name string) (Profile, bool) {
	if book == nil {
		return Profile{}, false
	}
	p, ok := book.profiles[name]
	return p, ok
}

// Species lists profile names in stable order.
func (book *Book) Species() []string {
	if book == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(book.profiles))
}

// validContinuousPliosaur bounds the authored head and fin controls to the
// existing field grid and the fixed eye and tooth attachment region.
func validContinuousPliosaur(head [7]float64, fins [6]float64) bool {
	for _, values := range [][]float64{head[:], fins[:]} {
		for _, v := range values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
	}
	return head[0] >= .55 && head[0] <= .75 && head[1] >= 1.30 && head[1] <= 1.50 &&
		head[2] >= .02 && head[2] <= .06 && head[3] >= .05 && head[3] <= .12 &&
		head[4] >= 0 && head[4] <= .04 && head[5] >= .07 && head[5] <= .14 && head[6] >= .02 && head[6] <= .06 &&
		fins[0] >= .12 && fins[0] <= .24 && fins[1] >= -.78 && fins[1] <= -.66 &&
		fins[2] >= .18 && fins[2] <= .22 && fins[3] >= .80 && fins[3] <= 1.0 &&
		fins[4] >= .12 && fins[4] <= .22 && fins[5] >= .18 && fins[5] <= .28
}
