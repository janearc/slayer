package draft

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// GroupArea reads an explicit terrain stand region whose habit consumes it.
// Coordinates are metres from the terrain's southwest corner.
func (document *Document) GroupArea(id string) ([4]float64, bool) {
	var values [4]float64
	tree, err := document.Tree()
	if err != nil {
		return values, false
	}
	area := groupAreaNode(tree, id)
	if area == nil {
		return values, false
	}
	for i, key := range []string{"west", "south", "east", "north"} {
		node := member(area, key)
		if node == nil {
			return values, false
		}
		values[i], err = strconv.ParseFloat(node.Value, 64)
		if err != nil || math.IsNaN(values[i]) || math.IsInf(values[i], 0) {
			return values, false
		}
	}
	return values, true
}

// groupAreaNode finds a terrain region used by a supported stand habit.
func groupAreaNode(tree *yaml.Node, id string) *yaml.Node {
	if len(tree.Content) != 1 || !strings.HasPrefix(id, "scene.set.stands/") {
		return nil
	}
	scene := member(tree.Content[0], "scene")
	setting := member(scene, "setting")
	if setting != nil && setting.Value != "ground" {
		return nil
	}
	stands := member(member(scene, "set"), "stands")
	stand := member(stands, strings.TrimPrefix(id, "scene.set.stands/"))
	habit := member(stand, "habit")
	if habit == nil || (habit.Value != "grid" && habit.Value != "lattice" && habit.Value != "scatter") {
		return nil
	}
	area := member(stand, "area")
	if area == nil || area.Kind != yaml.MappingNode {
		return nil
	}
	return area
}

// SetGroupArea atomically changes the stand's authored region.
func (document *Document) SetGroupArea(id string, values [4]float64) error {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < -1e7 || value > 1e7 {
			return fmt.Errorf("region coordinates must be finite and between -10000000 and 10000000 metres")
		}
	}
	if values[2] <= values[0] || values[3] <= values[1] {
		return fmt.Errorf("east must exceed west, and north must exceed south")
	}
	if current, ok := document.GroupArea(id); ok && current == values {
		return nil
	}
	tree, err := document.Tree()
	if err != nil {
		return err
	}
	area := groupAreaNode(tree, id)
	if area == nil {
		return fmt.Errorf("this group has no editable terrain region")
	}
	for i, key := range []string{"west", "south", "east", "north"} {
		node := member(area, key)
		if node == nil {
			return fmt.Errorf("region is missing %s", key)
		}
		setNumber(node, values[i])
	}
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(tree); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return fmt.Errorf("region edit would make invalid set YAML: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}
