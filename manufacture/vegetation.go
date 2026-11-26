package manufacture

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/plants/seasonal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// seasonalPlant carries a picked architecture and an explicit foliage season.
type seasonalPlant struct {
	profile plantProfile
	season  string
}

// SeasonNames names the authored foliage states, not simulated phenology.
func SeasonNames() []string { return []string{"summer", "autumn", "winter"} }

// WithSeason changes a plant's foliage independently of its species/scaffold.
// Other factory kinds refuse this control instead of silently ignoring it.
func WithSeason(row Kind, season string) (Kind, error) {
	plant, found := row.how.(seasonalPlant)
	if !found {
		return Kind{}, fmt.Errorf("manufacture: %s has no foliage season", row.name)
	}
	valid := false
	for _, name := range SeasonNames() {
		valid = valid || season == name
	}
	if !valid {
		return Kind{}, fmt.Errorf("manufacture: unknown foliage season %q", season)
	}
	plant.season = season
	row.how = plant
	return row, nil
}

// made supplies the ordinary quick specimen.
func (plant seasonalPlant) made(from seed) (good, error) {
	return plant.madeAt(from, "quick")
}

// madeAt uses one seeded scaffold and fixed physical leaf cells at every
// detail.
func (plant seasonalPlant) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	plan, err := plant.profile.Plan(from)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	bark, err := plantBark(plan, detail)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	level := detail
	if level == "hull" || (plant.profile.Deciduous && plant.season == "winter") {
		level = "clumps"
	}
	cells, err := plant.profile.Attachments(plan, level)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	bounds := plan.Bounds()
	for _, cell := range cells {
		bounds = enclosing(bounds, transformed(cell.Bounds, cell.Pose))
	}
	made := good{parts: bark, dressed: plant.looks(), extent: &bounds}
	return plant.foliage(made, cells, detail)
}

// plantDryLeaf follows the two ordinary bark/foliage material slots.
const plantDryLeaf slot = leaf + spaceOne

// looks keeps the autumn palette distinct from winter leaf removal.
func (plant seasonalPlant) looks() looks {
	profile := plant.profile
	green := profile.Green
	if profile.Deciduous && plant.season == "autumn" {
		green = profile.Autumn
	}
	return looks{body: plain("bark", profile.Bark.R, profile.Bark.G, profile.Bark.B),
		leaf:         plain("broad foliage", green.R, green.G, green.B),
		plantDryLeaf: plain("old leaf skirt", profile.Dry.R, profile.Dry.G, profile.Dry.B)}
}

// foliage keeps shared twigs even when deciduous leaves are absent.
func (plant seasonalPlant) foliage(made good, cells []seasonal.Attachment,
	detail string,
) (good, error) {
	leafless := plant.profile.Deciduous && plant.season == "winter"
	if detail == "hull" && !leafless {
		var crown *maths.Box
		for _, cell := range cells {
			box := transformed(cell.Bounds, cell.Pose)
			if crown == nil {
				crown = &box
			} else {
				joined := enclosing(*crown, box)
				crown = &joined
			}
		}
		if crown != nil {
			made.parts = append(made.parts, part{shape: crownShape{*crown},
				space: maths.Local(), of: leaf})
		}
		return made, nil
	}
	var bark surface
	for _, piece := range made.parts {
		bark = append(bark, piece.skin()...)
	}
	wood, err := bark.meshed()
	if err != nil {
		//ninja:vow feint value API; caller refuses the incomplete plant
		return good{}, err
	}
	groups := map[slot][]shape.MeshMember{body: {{Mesh: wood, Space: maths.Local()}}}
	for _, cell := range cells {
		groups[body] = append(groups[body], shape.MeshMember{Mesh: cell.Wood, Space: cell.Pose})
		if leafless {
			continue
		}
		groups[leaf] = append(groups[leaf], shape.MeshMember{Mesh: cell.Green, Space: cell.Pose})
		if cell.Dry != nil {
			groups[plantDryLeaf] = append(groups[plantDryLeaf],
				shape.MeshMember{Mesh: cell.Dry, Space: cell.Pose})
		}
	}
	made.parts = nil
	return made.withMeshGroups(groups)
}

func plantBark(plan seasonal.Plan, detail string) ([]part, error) {
	var parts []part
	fine := cut{around: int(plantValue("stem-sides-" + detail))}
	for _, axis := range plan.Woody() {
		run := axis.End.Sub(axis.Root)
		length := measure.Metres(run.Len())
		tube, err := newTube(length, measure.Metres(axis.Radius), measure.Metres(axis.Tip))
		if err != nil {
			return nil, err
		}
		pose, err := spaceAlong(axis.Root, run)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part{shape: tube, fine: fine, space: pose, of: body})
	}
	return parts, nil
}
