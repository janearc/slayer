package manufacture

import (
	"fmt"

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

// vegetationRows exposes the architecture names in the ordinary factory.
func vegetationRows() map[string]maker {
	rows := map[string]maker{}
	for name, profile := range plantProfiles {
		rows[name] = seasonalPlant{profile: profile, season: "summer"}
	}
	return rows
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
	plan, err := plant.profile.plan(from)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	bark, err := plan.bark(detail)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	level := detail
	if level == "hull" || (plant.profile.Deciduous && plant.season == "winter") {
		level = "clumps"
	}
	cells, err := plant.attachments(plan, level)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	bounds := plan.bounds()
	for _, cell := range cells {
		bounds = enclosing(bounds, transformed(cell.prototype.bounds, cell.pose))
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
func (plant seasonalPlant) foliage(made good, cells []plantAttachment,
	detail string,
) (good, error) {
	leafless := plant.profile.Deciduous && plant.season == "winter"
	if detail == "hull" && !leafless {
		var crown *maths.Box
		for _, cell := range cells {
			box := transformed(cell.prototype.bounds, cell.pose)
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
		groups[body] = append(groups[body], shape.MeshMember{Mesh: cell.prototype.wood, Space: cell.pose})
		if leafless {
			continue
		}
		groups[leaf] = append(groups[leaf], shape.MeshMember{Mesh: cell.prototype.green, Space: cell.pose})
		if cell.prototype.dry != nil {
			groups[plantDryLeaf] = append(groups[plantDryLeaf],
				shape.MeshMember{Mesh: cell.prototype.dry, Space: cell.pose})
		}
	}
	made.parts = nil
	made.groups = map[slot]*shape.MeshGroup{}
	for role, members := range groups {
		group, err := shape.NewMeshGroup(members)
		if err != nil {
			// Its caller refuses the incomplete plant.
			//ninja:vow feint value API requires result
			return good{}, err
		}
		made.groups[role] = group
	}
	return made, nil
}
