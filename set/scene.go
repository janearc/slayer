package set

import (
	"math"

	yaml "go.yaml.in/yaml/v3"
)

// kinds is every kind of asset a set file may ask for.
var kinds = []string{"house"}

// scene reads the scene: its name, its light, where and when it is,
// its cameras, and the set.
func (reading reader) scene(_, value *yaml.Node) error {
	made := reading.set
	return reading.keys(value, map[string]step{
		"name":    reading.text(&made.name),
		"light":   reading.word(&made.light, "ambient", "moon", "sun"),
		"terrain": reading.terrain,
		"film":    reading.film,
		"moon":    reading.moon,
		"sun":     reading.sun,
		"planet":  reading.planet,
		"hud":     reading.hud,
		"world":   reading.world,
		"location": reading.block(map[string]step{
			"lat": reading.number(&made.latitude, -90, 90),
			"lon": reading.number(&made.longitude, -180, 180)}),
		"timeofday": reading.clock,
		"cameras":   reading.cameras,
		"set": reading.block(map[string]step{
			"bounded_box": reading.block(map[string]step{
				"size_x": reading.number(&made.bounds[0], 0, math.Inf(1)),
				"size_y": reading.number(&made.bounds[1], 0, math.Inf(1)),
				"size_z": reading.number(&made.bounds[2], 0,
					math.Inf(1))}),
			"assets": reading.assets,
			"things": reading.things,
			"stands": reading.stands,
			"texts":  reading.texts}),
	}, nil)
}

// cameras reads each camera, by its name. A camera stands off an asset
// and points to one, or stands at a place and looks at a place, never
// some of each.
func (reading reader) cameras(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		shot := camera{name: name.Value, eye: "human"}
		var at, lookAt [3]float64
		err := reading.keys(block, map[string]step{
			"title": reading.text(&shot.title),
			"eye":   reading.only("human"),
			"orientation": reading.block(map[string]step{
				"point_to": reading.text(&shot.pointTo),
				"alpha":    reading.number(&shot.alpha, -90, 90),
				"look_at": reading.three(&lookAt, -farthest,
					farthest)}),
			"location": reading.block(map[string]step{
				"loc_relative": reading.block(map[string]step{
					"object": reading.text(&shot.from),
					"standoff_x": reading.number(&shot.standoff[0],
						math.Inf(-1), math.Inf(1)),
					"standoff_y": reading.number(&shot.standoff[1],
						math.Inf(-1), math.Inf(1)),
					"standoff_z": reading.number(&shot.standoff[2],
						math.Inf(-1), math.Inf(1))}),
				"at": reading.three(&at, -farthest, farthest)}),
		}, nil)
		if err == nil {
			err = reading.placed(block, &shot, at, lookAt)
		}
		reading.set.cameras = append(reading.set.cameras, shot)
		return err
	})
}

// placed takes a camera's place and the place it looks at, when it says
// them, and refuses one that says only one of them, or says either
// beside an asset it stands off or points to.
func (reading reader) placed(block *yaml.Node, shot *camera, at,
	lookAt [3]float64) error {
	orientation := valueNode(block, "orientation")
	location := valueNode(block, "location")
	stands, looks := says(location, "at"), says(orientation, "look_at")
	offAsset := says(location, "loc_relative") ||
		says(orientation, "point_to") || says(orientation, "alpha")
	switch {
	case !stands && !looks:
		return nil
	case stands != looks || offAsset:
		return reading.refuse(block, "camera %s stands at a place and "+
			"looks at one, both, or stands off an asset and points to "+
			"one, not some of each", shot.name)
	}
	shot.at, shot.lookAt = &at, &lookAt
	return nil
}

// assets reads each asset, by its kind. Only a house's name is read
// yet; how a house is made is kept as unrendered.
func (reading reader) assets(_, value *yaml.Node) error {
	return reading.each(value, func(kind, block *yaml.Node) error {
		if err := reading.only(kinds...)(kind, kind); err != nil {
			return err
		}
		thing := asset{kind: kind.Value}
		err := reading.keys(block, map[string]step{
			"name": reading.text(&thing.name),
		}, []string{"autogen", "autogen_parameters"})
		reading.set.assets = append(reading.set.assets, thing)
		return err
	})
}

// clock reads a time of day, hhmm on a local clock, as minutes after
// midnight.
func (reading reader) clock(name, value *yaml.Node) error {
	var hhmm int
	if value.Decode(&hhmm) != nil || hhmm < 0 || hhmm/100 > 23 ||
		hhmm%100 > 59 {
		return reading.refuse(value, "%s takes a time as hhmm, 0000 to "+
			"2359, not %q", name.Value, value.Value)
	}
	reading.set.clock = hhmm/100*60 + hhmm%100
	return nil
}
