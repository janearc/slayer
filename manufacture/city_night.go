package manufacture

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

const (
	cityWarmWindow slot = 9
	cityCoolWindow slot = 10
)

// nightCityLooks keeps wall mass dark while discrete occupied panes emit.
func nightCityLooks() looks {
	palette := cityLooks()
	palette[cityWarmWindow] = &thing.Material{Name: "city occupied warm windows", Colour: maths.Grey(.025), Glow: maths.RGB(.85, .42, .14), Gloss: .15}
	palette[cityCoolWindow] = &thing.Material{Name: "city occupied cool windows", Colour: maths.Grey(.025), Glow: maths.RGB(.24, .43, .65), Gloss: .25}
	return palette
}

// districtBuildingParts retains original day architecture, replacing only
// night glazing with actual separated panes and seeded unoccupied windows.
func districtBuildingParts(building CityBuilding, detail string, night bool) []part {
	design := cityBuildingDesign(building.Kind, sow(building.Seed))
	parts := design.parts(detail)
	if !night || detail == "hull" {
		return parts
	}
	kept := make([]part, 0, len(parts))
	for _, piece := range parts {
		if piece.of != cityGlass {
			kept = append(kept, piece)
		}
	}
	skins := map[slot]surface{}
	own := sow(building.Seed)
	for floor := range design.floors {
		y := float64(floor)*design.height/float64(design.floors) + 1
		width, depth := design.width, design.depth
		if design.tower {
			if y+districtValue("window-height") >= design.height*.93 {
				width *= .52
				depth *= .52
			} else if y+districtValue("window-height") >= design.height*.18 {
				width *= .76
				depth *= .76
			}
		}
		for wall := range 4 {
			long := width
			if wall >= 2 {
				long = depth
			}
			count := max(1, int(long/districtValue("window-pitch")))
			for bay := range count {
				centre := (float64(bay)+.5)/float64(count)*long - long/2
				role := cityPaneRole(own, wall, floor, bay)
				var a, b, c, d maths.Vec
				if wall < 2 {
					z := depth/2 + .02
					if wall == 1 {
						z = -z
					}
					a, b, c, d = maths.V(centre-districtValue("window-width")/2, y, z), maths.V(centre+districtValue("window-width")/2, y, z), maths.V(centre+districtValue("window-width")/2, y+districtValue("window-height"), z), maths.V(centre-districtValue("window-width")/2, y+districtValue("window-height"), z)
				} else {
					x := width/2 + .02
					if wall == 3 {
						x = -x
					}
					a, b, c, d = maths.V(x, y, centre-districtValue("window-width")/2), maths.V(x, y, centre+districtValue("window-width")/2), maths.V(x, y+districtValue("window-height"), centre+districtValue("window-width")/2), maths.V(x, y+districtValue("window-height"), centre-districtValue("window-width")/2)
				}
				pair := surface{{a, b, c}, {a, c, d}}
				if wall == 1 || wall == 2 {
					for i := range pair {
						pair[i][1], pair[i][2] = pair[i][2], pair[i][1]
					}
				}
				skins[role] = append(skins[role], pair...)
			}
		}
	}
	for _, role := range []slot{cityGlass, cityWarmWindow, cityCoolWindow} {
		if len(skins[role]) > 0 {
			kept = append(kept, part{shape: cityPaneSurface{skins[role]}, space: maths.Local(), of: role})
		}
	}
	return kept
}

// cityPaneRole is one window's seeded night material: unlit glass, or a
// warm or cool occupied pane, drawn per building, wall, floor and bay.
func cityPaneRole(own seed, wall, floor, bay int) slot {
	role := cityGlass
	occupied := own.draw(120+wall, floor, bay)
	occupancy := districtValue("minimum-occupancy-share") + districtValue("occupancy-range-share")*own.draw(119, 0, 0)
	if occupied < occupancy*districtValue("warm-light-share") {
		role = cityWarmWindow
	} else if occupied < occupancy {
		role = cityCoolWindow
	}
	return role
}

type cityPaneSurface struct{ triangles surface }

// skin retains mathematical panes as ordinary outward open facade surfaces.
func (panes cityPaneSurface) skin(cut) surface { return panes.triangles }

// nightLampPlaces supplies at most six real lighting anchors: four ramp
// street connections and two park path junctions, rather than per-window lamps.
func (city ClusteredCityLayout) nightLampPlaces() []maths.Vec {
	var places []maths.Vec
	for _, ramp := range city.ramps {
		at := ramp.Points[3]
		at.X += 4
		at.Y = 6
		places = append(places, at)
		if len(places) == 4 {
			break
		}
	}
	for _, park := range city.parks {
		at := park.At
		at.X += 4
		at.Y = 6
		places = append(places, at)
		if len(places) == 6 {
			break
		}
	}
	return places
}

// nightPoles makes visible fixtures at the actual sparse lighting anchors.
func (city ClusteredCityLayout) nightPoles(batch *cityBatch) {
	a := instrumentAssembly{fine: cut{around: 6}}
	for _, at := range city.nightLampPlaces() {
		foot := at.Sub(maths.V(1.2, at.Y, 0))
		a.rod(foot, foot.Add(maths.V(0, at.Y, 0)), .12, cityPavement)
		a.rod(foot.Add(maths.V(0, at.Y, 0)), at, .08, cityPavement)
		districtBox(&a, .45, .3, .45, at, cityWarmWindow)
	}
	batch.add(a.object.parts, maths.Local())
}

// nightLights describes physical downward fixtures through the existing lamp
// API. Surface emission itself is not claimed to illuminate nearby roads.
func (city ClusteredCityLayout) nightLights(made good) (good, error) {
	colour, err := newThermal(measure.Kelvin(districtValue("lamp-temperature")))
	if err != nil {
		return good{}, err
	}
	cone, err := newCone(measure.Degrees(districtValue("lamp-half-angle")), measure.Degrees(districtValue("lamp-soft-angle")))
	if err != nil {
		return good{}, err
	}
	lamp, err := newLight(colour, measure.Watts(districtValue("lamp-power")), cone)
	if err != nil {
		return good{}, err
	}
	for i, at := range city.nightLampPlaces() {
		made = made.litAs(fmt.Sprintf("city-area-lamp-%d", i+1), lamp, at, maths.Up.Neg(), measure.Metres(.5))
	}
	return made, nil
}
