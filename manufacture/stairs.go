package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// paintFinish is the name of the paint finish in thing's book of
// finishes: the one finish a yard look wears, colour and all.
const paintFinish = "painted"

// stairFlight is one flight of stairs as a design books it: so many
// steps of a rise, a going and a width, carried by stringers, with a
// rail held up by posts and the space beneath it filled by bars or a
// panel. Every family names its material and its finish; the yard
// resolves the material through its palette and checks the finish
// against the look that palette wears.
//
// A flight is laid out as the
// same boxes and rods a hand-placed design names, in the same order,
// so its tessellation, its conversion and its geometry records are the
// yard's own.
type stairFlight struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Steps int `yaml:"steps"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Rise float64 `yaml:"rise"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Going float64 `yaml:"going"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Width float64 `yaml:"width"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Treads stairTreads `yaml:"treads"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Stringers *stairSlope `yaml:"stringers"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Posts *stairUprights `yaml:"posts"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Rail *stairSlope `yaml:"rail"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Infill *stairInfill `yaml:"infill"`
}

// stairTreads is the flight's walking surface: one box per step, as
// deep and as thick as booked, the lowest one's near corner placed by
// its first spot.
type stairTreads struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Material string `yaml:"material"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Finish string `yaml:"finish"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Depth float64 `yaml:"depth"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Thickness float64 `yaml:"thickness"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	First stairSpot `yaml:"first"`
}

// stairSlope is one sloping member down each side of a flight, a
// stringer under the treads or a handrail above them: the ends of one
// side's rod, its radius, and how far from the centreline each side
// sits.
type stairSlope struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Material string `yaml:"material"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Finish string `yaml:"finish"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Radius float64 `yaml:"radius"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Offset float64 `yaml:"offset"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Low stairSpot `yaml:"low"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	High stairSpot `yaml:"high"`
}

// stairUprights is a row of vertical rods down one side of a flight,
// the posts that carry a rail or the bars that fill beneath it: the
// first rod's two ends, and how far each next rod steps along the
// flight and up it.
type stairUprights struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Material string `yaml:"material"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Finish string `yaml:"finish"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Count int `yaml:"count"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Radius float64 `yaml:"radius"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Offset float64 `yaml:"offset"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	First stairBar `yaml:"first"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	YStep float64 `yaml:"y-step"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	ZStep float64 `yaml:"z-step"`
}

// stairInfill is what fills the space under a flight's rail: a row of
// bars, or a panel. One or the other, never both and never neither.
type stairInfill struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Bars *stairUprights `yaml:"bars"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Panel *stairPanel `yaml:"panel"`
}

// stairPanel is a closed balustrade slab on each side of a flight: how
// thick it is, how far from the centreline it sits, and the heights of
// its edges and the depths of its front and back.
type stairPanel struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Material string `yaml:"material"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Finish string `yaml:"finish"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Thickness float64 `yaml:"thickness"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Offset float64 `yaml:"offset"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Bottom float64 `yaml:"bottom"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Top float64 `yaml:"top"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Front float64 `yaml:"front"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Back float64 `yaml:"back"`
}

// stairSpot is a place in a flight's own frame: a height and a depth,
// metres from where the flight's foot stands.
type stairSpot struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Y float64 `yaml:"y"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Z float64 `yaml:"z"`
}

// stairBar is the first of a row of uprights: the depth it stands at
// and the heights of its two ends.
type stairBar struct {
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Z float64 `yaml:"z"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Bottom float64 `yaml:"bottom"`
	//ninja:vow gate Puzzlemouth decodes this private YAML field
	Top float64 `yaml:"top"`
}

// wear is what one family of a flight names: the family, the material
// it is made of, and the finish it was given.
type wear struct {
	family   string
	material string
	finish   string
}

// wears lists what every family of the flight names, in the order its
// parts are laid out, for a facility to check against its palette
// before anything is built.
func (flight stairFlight) wears() []wear {
	worn := []wear{{"treads", flight.Treads.Material,
		flight.Treads.Finish}}
	if flight.Stringers != nil {
		worn = append(worn, wear{"stringers",
			flight.Stringers.Material, flight.Stringers.Finish})
	}
	if flight.Posts != nil {
		worn = append(worn, wear{"posts",
			flight.Posts.Material, flight.Posts.Finish})
	}
	if flight.Rail != nil {
		worn = append(worn, wear{"rail",
			flight.Rail.Material, flight.Rail.Finish})
	}
	if flight.Infill != nil {
		if flight.Infill.Bars != nil {
			worn = append(worn, wear{"infill bars",
				flight.Infill.Bars.Material,
				flight.Infill.Bars.Finish})
		}
		if flight.Infill.Panel != nil {
			worn = append(worn, wear{"infill panel",
				flight.Infill.Panel.Material,
				flight.Infill.Panel.Finish})
		}
	}
	return worn
}

// parts lays the flight out as the parts a hand-placed design would
// name: every tread, then each side's stringer, rail, posts and
// infill, the left side before the right. Nonsense parameters are
// refused instead of built.
func (flight stairFlight) parts() ([]yardPart, error) {
	if flight.Steps < 1 {
		return nil, fmt.Errorf("manufacture: stairs of %d steps",
			flight.Steps)
	}
	if flight.Rise <= 0 || flight.Going <= 0 || flight.Width <= 0 {
		return nil, fmt.Errorf(
			"manufacture: stairs rise %v, going %v, width %v",
			flight.Rise, flight.Going, flight.Width)
	}
	if flight.Treads.Depth <= 0 || flight.Treads.Thickness <= 0 {
		return nil, fmt.Errorf(
			"manufacture: stairs treads %v deep, %v thick",
			flight.Treads.Depth, flight.Treads.Thickness)
	}
	if flight.Infill != nil {
		bars, panel := flight.Infill.Bars, flight.Infill.Panel
		if bars == nil && panel == nil {
			return nil, fmt.Errorf(
				"manufacture: stairs infill is neither bars nor a panel")
		}
		if bars != nil && panel != nil {
			return nil, fmt.Errorf(
				"manufacture: stairs infill is bars and a panel together")
		}
	}
	pieces := make([]yardPart, 0, flight.Steps+4)
	treads := flight.Treads
	for step := range flight.Steps {
		at := maths.V(-flight.Width/2,
			treads.First.Y+float64(step)*flight.Rise,
			treads.First.Z-float64(step)*flight.Going)
		pieces = append(pieces, yardPart{Material: treads.Material,
			Box: &yardBox{At: at,
				Size: maths.V(flight.Width, treads.Thickness,
					treads.Depth)}})
	}
	for _, side := range [2]float64{-1, 1} {
		if flight.Stringers != nil {
			built, err := flight.Stringers.lay(side, "stringers")
			if err != nil {
				return nil, err
			}
			pieces = append(pieces, built...)
		}
		if flight.Rail != nil {
			built, err := flight.Rail.lay(side, "rail")
			if err != nil {
				return nil, err
			}
			pieces = append(pieces, built...)
		}
		if flight.Posts != nil {
			built, err := flight.Posts.lay(side, "posts")
			if err != nil {
				return nil, err
			}
			pieces = append(pieces, built...)
		}
		if flight.Infill != nil && flight.Infill.Bars != nil {
			built, err := flight.Infill.Bars.lay(side, "infill bars")
			if err != nil {
				return nil, err
			}
			pieces = append(pieces, built...)
		}
		if flight.Infill != nil && flight.Infill.Panel != nil {
			built, err := flight.Infill.Panel.lay(side)
			if err != nil {
				return nil, err
			}
			pieces = append(pieces, built...)
		}
	}
	return pieces, nil
}

// lay lays one side's sloping member, a rod from its low end to its
// high end.
func (slope *stairSlope) lay(side float64, name string) ([]yardPart, error) {
	if slope.Radius <= 0 || slope.Offset <= 0 {
		return nil, fmt.Errorf("manufacture: stairs %s radius %v, offset %v",
			name, slope.Radius, slope.Offset)
	}
	from := maths.V(side*slope.Offset, slope.Low.Y, slope.Low.Z)
	to := maths.V(side*slope.Offset, slope.High.Y, slope.High.Z)
	return []yardPart{{Material: slope.Material,
		Rod: &yardRod{From: from, To: to, Radius: slope.Radius}}}, nil
}

// lay lays one side's row of uprights, the first rod and each next one
// stepped along the flight and up it.
func (uprights *stairUprights) lay(side float64,
	name string,
) ([]yardPart, error) {
	if uprights.Count < 1 || uprights.Radius <= 0 || uprights.Offset <= 0 {
		return nil, fmt.Errorf("manufacture: stairs %s of %d, radius %v, "+
			"offset %v", name, uprights.Count, uprights.Radius,
			uprights.Offset)
	}
	pieces := make([]yardPart, 0, uprights.Count)
	for index := range uprights.Count {
		first := uprights.First
		bottom := first.Bottom + float64(index)*uprights.YStep
		top := first.Top + float64(index)*uprights.YStep
		depth := first.Z + float64(index)*uprights.ZStep
		from := maths.V(side*uprights.Offset, bottom, depth)
		to := maths.V(side*uprights.Offset, top, depth)
		pieces = append(pieces, yardPart{Material: uprights.Material,
			Rod: &yardRod{From: from, To: to, Radius: uprights.Radius}})
	}
	return pieces, nil
}

// lay lays one side's balustrade panel, a box from its bottom back
// corner.
func (panel *stairPanel) lay(side float64) ([]yardPart, error) {
	if panel.Thickness <= 0 || panel.Offset <= 0 ||
		panel.Top <= panel.Bottom || panel.Front <= panel.Back {
		return nil, fmt.Errorf(
			"manufacture: stairs panel %v thick, %v to %v high, "+
				"%v to %v deep", panel.Thickness, panel.Bottom, panel.Top,
			panel.Front, panel.Back)
	}
	size := maths.V(panel.Thickness, panel.Top-panel.Bottom,
		panel.Front-panel.Back)
	at := maths.V(side*panel.Offset-panel.Thickness/2, panel.Bottom,
		panel.Back)
	return []yardPart{{Material: panel.Material,
		Box: &yardBox{At: at, Size: size}}}, nil
}
