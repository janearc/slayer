package theropods

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/manufacture/profilecheck"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed allosaurus-profiles.yaml
var allosaurusProfilesRaw []byte

type theropodProfile struct {
	Kind, Status, By, Date, Source, Scaffold string
	Scale                                    float64
	Hide, Keratin, Eye                       [3]float64
	Stations                                 [][5]float64
}

// TheropodProfiles stores profiles for explicitly supported theropod scaffolds.
// The only scaffold currently implemented is the Allosaurus walking study.
type TheropodProfiles struct{ profiles map[string]theropodProfile }

// ReadTheropodProfiles snapshots and validates an authored profile book.
func ReadTheropodProfiles(file string, raw []byte) (*TheropodProfiles, error) {
	rows, err := profilecheck.Rows[theropodProfile](file, raw, "theropod book", "theropod profile", nil)
	if err != nil {
		return nil, err
	}
	book := &TheropodProfiles{profiles: make(map[string]theropodProfile, len(rows))}
	for _, profile := range rows {
		if !profilecheck.Authored(profile.Kind, profile.Status, profile.By, profile.Date, profile.Source) {
			return nil, fmt.Errorf("%s: invalid provenance for %q", file, profile.Kind)
		}
		if profile.Scaffold != "allosaurus-study" {
			return nil, fmt.Errorf("%s: unknown theropod scaffold %q", file, profile.Scaffold)
		}
		if !positive(profile.Scale) || profile.Scale > 10 {
			return nil, fmt.Errorf("%s: invalid theropod scale", file)
		}
		// The station coordinates are limited to ten metres. These small
		// numerical margins keep adjacent rings and their radii representable
		// at that scale; they are not measurements of the animal.
		const minimumStationFigure = 1e-6
		if profile.Scale < minimumStationFigure {
			return nil, fmt.Errorf("%s: theropod scale is too small for stable geometry", file)
		}
		for _, colour := range [][3]float64{profile.Hide, profile.Keratin, profile.Eye} {
			for _, c := range colour {
				if math.IsNaN(c) || math.IsInf(c, 0) || c < 0 || c > 1 {
					return nil, fmt.Errorf("%s: invalid theropod colour", file)
				}
			}
		}
		if len(profile.Stations) != 17 {
			return nil, fmt.Errorf("%s: allosaurus study needs 17 stations", file)
		}
		for i, s := range profile.Stations {
			for _, v := range s {
				if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 10 {
					return nil, fmt.Errorf("%s: invalid station", file)
				}
			}
			if s[2] < minimumStationFigure || s[2] > 2 || s[3] < minimumStationFigure || s[3] > 2 || s[4] < .5 || s[4] > 2 || i > 0 && s[0]-profile.Stations[i-1][0] < minimumStationFigure {
				return nil, fmt.Errorf("%s: invalid station %d", file, i)
			}
		}
		// Catmull-Rom can overshoot even when both station radii are
		// positive. Check each cubic's extrema, including between the mesh's
		// samples, so its cross-sections cannot invert or become singular.
		for i := 0; i+1 < len(profile.Stations); i++ {
			for _, dimension := range []int{2, 3, 4} {
				before := profile.Stations[max(i-1, 0)][dimension]
				low := profile.Stations[i][dimension]
				high := profile.Stations[i+1][dimension]
				after := profile.Stations[min(i+2, len(profile.Stations)-1)][dimension]
				minimum, maximum := minimumStationFigure, 2.0
				if dimension == 4 {
					minimum = .5
				}
				if !allosaurusInterpolantWithin(before, low, high, after, minimum, maximum) {
					return nil, fmt.Errorf("%s: invalid interpolated station %d dimension %d", file, i, dimension)
				}
			}
		}
		// Numerically positive radii can still collapse after rounding or
		// produce zero-area triangles. Check the actual rest mesh.
		if _, err := allosaurusBodyWithStations(0, profile.Stations); err != nil {
			return nil, fmt.Errorf("%s: invalid theropod rest mesh: %w", file, err)
		}
		if _, exists := book.profiles[profile.Kind]; exists {
			return nil, fmt.Errorf("%s: duplicate theropod %q", file, profile.Kind)
		}
		book.profiles[profile.Kind] = profile
	}
	return book, nil
}

// allosaurusInterpolantWithin checks endpoints and every interior extremum
// of the same Catmull-Rom cubic used by the body mesh.
func allosaurusInterpolantWithin(before, low, high, after, minimum, maximum float64) bool {
	m0, m1 := (high-before)/2, (after-low)/2
	a := 3 * (2*low - 2*high + m0 + m1)
	b := 2 * (-3*low + 3*high - 2*m0 - m1)
	c := m0
	within := func(t float64) bool {
		if t < 0 || t > 1 {
			return true
		}
		value := geometry.CatmullRom(before, low, high, after, t)
		return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
	}
	if !within(0) || !within(1) {
		return false
	}
	if a == 0 {
		return b == 0 || within(-c/b)
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return true
	}
	root := math.Sqrt(disc)
	q := -.5 * (b + math.Copysign(root, b))
	if q == 0 {
		return within(-b / (2 * a))
	}
	return within(q/a) && within(c/q)
}

// Recipe retains a checked profile at a particular phase of its walk.
type Recipe struct {
	profile theropodProfile
	seconds float64
}

// NewTheropod selects a checked study profile.
func (book *TheropodProfiles) NewTheropod(species string, seconds float64) (*Recipe, error) {
	if book == nil {
		return nil, fmt.Errorf("no theropod book")
	}
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || math.Abs(seconds) > 100000 {
		return nil, fmt.Errorf("invalid theropod time")
	}
	profile, ok := book.profiles[species]
	if !ok {
		return nil, fmt.Errorf("no theropod profile %q", species)
	}
	return &Recipe{profile: profile, seconds: seconds}, nil
}

// Build constructs the checked study and its bounding box.
func (row *Recipe) Build(at maths.Subspace) ([]thing.Thing, maths.Box, error) {
	parts, err := allosaurusAt(row.profile, row.seconds, at)
	if err != nil {
		return nil, maths.Box{}, err
	}
	box := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)), Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, part := range parts {
		box = geometry.Enclosing(box, geometry.Transformed(part.Shape.Bounds(), part.Motion.At(0)))
	}
	return parts, box, nil
}

// Embedded returns an independent copy of the default book.
func Embedded() []byte { return append([]byte(nil), allosaurusProfilesRaw...) }

func positive(number float64) bool { return number > 0 && !math.IsInf(number, 0) }

var defaultAllosaurusProfile = func() theropodProfile {
	book, err := ReadTheropodProfiles("allosaurus-profiles.yaml", allosaurusProfilesRaw)
	if err != nil {
		panic(err)
	}
	profile, ok := book.profiles["allosaurus"]
	if !ok {
		panic("missing Allosaurus profile")
	}
	return profile
}()

// DefaultStudy constructs the embedded Allosaurus walking study.
func DefaultStudy(seconds float64, at maths.Subspace) ([]thing.Thing, error) {
	return allosaurusAt(defaultAllosaurusProfile, seconds, at)
}
