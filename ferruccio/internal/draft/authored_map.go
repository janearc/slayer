package draft

import (
	"math"

	"gopkg.in/yaml.v3"
)

// MapMark is a position from source. For a stand it is the centre of a
// declared area or a referenced anchor, without implying member positions.
type MapMark struct {
	ID, Name    string
	East, North float64
	Group       bool
	Area        *MapRectangle
}

// MapRectangle is an authored extent in the map’s east and north metres.
type MapRectangle struct {
	West, South, East, North float64
}

// MapSource holds the marks and any measurable terrain cut.
type MapSource struct {
	Marks   []MapMark
	Terrain *MapRectangle
}

// AuthoredMap returns the explicit positions available without loading surveyed
// terrain. A sourced ground window makes the plan available even if it has no
// positions that can be drawn.
func (document *Document) AuthoredMap() (MapSource, bool, error) {
	if document.kind != "set" || document.schemaName != "https://github.com/janearc/slayer/schemas/ecosystem-v2.json" {
		return MapSource{}, false, nil
	}
	tree, err := document.Tree()
	if err != nil {
		return MapSource{}, false, err
	}
	scene := member(tree.Content[0], "scene")
	setting := member(scene, "setting")
	ground := setting == nil || setting.Value == "ground"
	terrain := member(scene, "terrain")
	if !ground || member(terrain, "from") == nil {
		return MapSource{}, false, nil
	}
	_, placements, err := document.posesIn(tree)
	if err != nil {
		return MapSource{}, true, err
	}
	marks := make([]MapMark, 0, len(placements))
	anchors := make(map[string]MapMark, len(placements))
	for _, node := range placements {
		object := node.object
		at := object.Position()
		mark := MapMark{ID: object.ID(), Name: object.Name(), East: at[0], North: at[1]}
		marks = append(marks, mark)
		if object.Grounded() {
			anchors[mark.Name] = mark
		}
	}
	stands := member(member(scene, "set"), "stands")
	origin := mapTerrainOrigin(terrain)
	return MapSource{
		Marks:   appendStandMarks(marks, stands, anchors, origin),
		Terrain: mapTerrainRectangle(terrain, origin),
	}, true, nil
}

// mapTerrainOrigin follows the stage's cut and laid-area coordinates.
// A cut includes the first row plus the whole number of steps down.
func mapTerrainOrigin(terrain *yaml.Node) *MapMark {
	down, downOK := terrainMapNumber(member(terrain, "down"), 1, 1e5)
	step, stepOK := terrainMapNumber(member(terrain, "step"), .01, 1e4)
	if !downOK || !stepOK {
		return nil
	}
	lat, lon := member(terrain, "lat"), member(terrain, "lon")
	west, north := member(terrain, "west"), member(terrain, "north")
	origin := &MapMark{}
	if lat != nil || lon != nil {
		_, latOK := terrainMapNumber(lat, -90, 90)
		_, lonOK := terrainMapNumber(lon, -180, 180)
		if !latOK || !lonOK || west != nil || north != nil {
			return nil
		}
		origin.North = down
	} else {
		for i, node := range []*yaml.Node{west, north} {
			if node == nil {
				continue
			}
			value, ok := terrainMapNumber(node, 0, math.MaxFloat64)
			if !ok {
				return nil
			}
			if i == 0 {
				origin.East = value
			} else {
				origin.North = value
			}
		}
	}
	origin.North -= float64(int(down/step)) * step
	return origin
}

// mapTerrainRectangle follows the first and last samples of the staged cut.
func mapTerrainRectangle(terrain *yaml.Node, origin *MapMark) *MapRectangle {
	if origin == nil {
		return nil
	}
	across, acrossOK := terrainMapNumber(member(terrain, "across"), 1, 1e5)
	step, stepOK := terrainMapNumber(member(terrain, "step"), .01, 1e4)
	down, downOK := terrainMapNumber(member(terrain, "down"), 1, 1e5)
	if !acrossOK || !stepOK || !downOK {
		return nil
	}
	return mapRectangle(origin.East, origin.North,
		origin.East+math.Trunc(across/step)*step, origin.North+math.Trunc(down/step)*step)
}

// terrainMapNumber uses the same YAML numeric decoding as the set reader.
func terrainMapNumber(node *yaml.Node, least, most float64) (float64, bool) {
	amount := math.NaN()
	if node == nil || node.Decode(&amount) != nil {
		return 0, false
	}
	return amount, amount >= least && amount <= most
}

func mapRectangle(west, south, east, north float64) *MapRectangle {
	if !(west < east && south < north) || math.IsInf(west, 0) ||
		math.IsInf(south, 0) || math.IsInf(east, 0) || math.IsInf(north, 0) {
		return nil
	}
	return &MapRectangle{West: west, South: south, East: east, North: north}
}

func appendStandMarks(marks []MapMark, stands *yaml.Node, anchors map[string]MapMark, origin *MapMark) []MapMark {
	if stands == nil || stands.Kind != yaml.MappingNode {
		return marks
	}
	for i := 0; i+1 < len(stands.Content); i += 2 {
		name, block := stands.Content[i].Value, stands.Content[i+1]
		mark := MapMark{ID: "scene.set.stands/" + name, Name: name, Group: true}
		if rectangle := member(block, "area"); rectangle != nil {
			area, ok := readGroupArea(rectangle)
			if origin == nil || !ok || area[0] >= area[2] || area[1] >= area[3] {
				continue
			}
			mark.East = origin.East + (area[0]/2 + area[2]/2)
			mark.North = origin.North + (area[1]/2 + area[3]/2)
			if math.IsInf(mark.East, 0) || math.IsInf(mark.North, 0) {
				continue
			}
			mark.Area = mapRectangle(origin.East+area[0], origin.North+area[1], origin.East+area[2], origin.North+area[3])
		} else if anchor, ok := anchors[scenePath(member(member(block, "within"), "of"))]; ok {
			mark.East, mark.North = anchor.East, anchor.North
		} else {
			continue
		}
		marks = append(marks, mark)
	}
	return marks
}
