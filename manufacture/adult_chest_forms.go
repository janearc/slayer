package manufacture

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

type nativeChestStudy struct {
	spec primates.HumanSpec
}

func chestFormRows() map[string]maker {
	reference := primates.AdultHuman()
	broad := reference
	broad.Torso.Left.Width, broad.Torso.Right.Width = .098, .093
	broad.Torso.Left.Projection, broad.Torso.Right.Projection = .035, .032
	broad.Torso.Left.AreolaRadius, broad.Torso.Right.AreolaRadius = .015, .014
	broad.Breasts.Left = primates.BreastSideForm{
		RootOutward: -.009, UpperFullness: .42, LowerFullness: -.31,
		LowerPoleLength: -.37, AreolaUp: .008,
	}
	broad.Breasts.Right = primates.BreastSideForm{
		RootOutward: -.007, UpperFullness: .34, LowerFullness: -.26,
		LowerPoleLength: -.30, AreolaUp: .005,
	}

	long := reference
	long.Torso.Left.Width, long.Torso.Right.Width = .082, .085
	long.Torso.Left.Projection, long.Torso.Right.Projection = .070, .069
	long.Torso.Left.PapillaProjection, long.Torso.Right.PapillaProjection = .0035, .0033
	long.Torso.Compliance, long.Torso.AgeYears = .61, 50
	long.Breasts.Left = primates.BreastSideForm{
		RootOutward: .006, UpperFullness: -.45, LowerFullness: .37,
		LowerPoleLength: .49, AreolaOutward: .003, AreolaUp: -.012,
		PapillaOutwardTilt: .10, PapillaUpTilt: -.08,
	}
	long.Breasts.Right = primates.BreastSideForm{
		RootOutward: .005, UpperFullness: -.38, LowerFullness: .31,
		LowerPoleLength: .42, AreolaOutward: .002, AreolaUp: -.010,
		PapillaOutwardTilt: .08, PapillaUpTilt: -.07,
	}

	asymmetric := reference
	asymmetric.Torso.Left.Projection, asymmetric.Torso.Right.Projection = .066, 0
	asymmetric.Torso.Left.Width = .088
	asymmetric.Breasts.Left = primates.BreastSideForm{
		RootOutward: .008, UpperFullness: .23, LowerFullness: .18,
		AreolaOutward: .008, PapillaOutwardTilt: .22,
	}
	asymmetric.Breasts.Right.WithoutAreola = true

	return map[string]maker{
		"adult-chest-study-broad":      nativeChestStudy{broad},
		"adult-chest-study-long":       nativeChestStudy{long},
		"adult-chest-study-asymmetric": nativeChestStudy{asymmetric},
	}
}

func (row nativeChestStudy) pose(from seed) (*primates.HumanPose, error) {
	spec := row.spec.WithSkinMottleSeed(from.number).WithSkinDiffusion(primates.AdultSkinDiffusion())
	human, err := primates.NewHuman(spec)
	if err != nil {
		return nil, err
	}
	return human.Pose(skeleton.Pose{})
}

func (row nativeChestStudy) extent(from seed) (maths.Box, error) {
	pose, err := row.pose(from)
	if err != nil {
		return maths.Box{}, err
	}
	points := pose.Points()
	return geometry.GrowBox(maths.Box{Min: points[0], Max: points[0]}, points[1:]), nil
}

func (row nativeChestStudy) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	pose, err := row.pose(from)
	if err != nil {
		return good{}, err
	}
	skin, err := pose.Surface(nil)
	if err != nil {
		return good{}, err
	}
	return good{native: []thing.Thing{skin}}, nil
}
