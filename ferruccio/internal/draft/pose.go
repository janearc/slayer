package draft

import (
	"bytes"
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

// Object is an immutable snapshot of one supported scene placement.
type Object struct {
	id, name, kind, recipe  string
	position                [3]float64
	heading                 float64
	hasHeading              bool
	importHeight            float64
	hasImportHeight         bool
	scale                   float64
	hasScale, scaleEditable bool
	grounded                bool
}

// ID identifies the authored collection and key, independent of pose.
func (object Object) ID() string { return object.id }

// Name is the authored placement key.
func (object Object) Name() string { return object.name }

// Kind is the native manufacture kind, recipe, or import.
func (object Object) Kind() string { return object.kind }

// Recipe is the authored recipe name, or empty for other objects.
func (object Object) Recipe() string { return object.recipe }

// Position is east, north and up in metres. For a grounded thing, up is its
// authored height above the terrain, not its staged world elevation.
func (object Object) Position() [3]float64 { return object.position }

// Grounded reports a placement anchored to terrain.
func (object Object) Grounded() bool { return object.grounded }

// Heading is clockwise degrees from north and whether explicitly authored.
func (object Object) Heading() (float64, bool) { return object.heading, object.hasHeading }

// Unit is the explicit position unit presented by pose controls.
func (object Object) Unit() string { return "m" }

// ImportHeight is the explicitly authored target height in metres, if any.
func (object Object) ImportHeight() (float64, bool) {
	return object.importHeight, object.hasImportHeight
}

// SizeScale is the uniform size factor and whether it was explicitly written.
func (object Object) SizeScale() (float64, bool) { return object.scale, object.hasScale }

// ScaleEditable reports whether the source placement supports a scale field.
// Its actual constructor must also be checked for deforming geometry.
func (object Object) ScaleEditable() bool { return object.scaleEditable }

// PoseChange is a relative movement and turn of one authored object.
// LengthUnit must be m and AngleUnit deg, including for a zero gesture.
type PoseChange struct {
	ID                       string
	East, North, Up, Heading float64
	LengthUnit, AngleUnit    string
}

// SetGeneratedHeading replaces an omitted ground heading with an explicit one.
func (document *Document) SetGeneratedHeading(id string, degrees float64) error {
	if math.IsNaN(degrees) || math.IsInf(degrees, 0) {
		return fmt.Errorf("heading must be finite")
	}
	tree, placements, err := document.poseNodes()
	if err != nil {
		return err
	}
	for _, placement := range placements {
		if placement.object.id != id {
			continue
		}
		if !placement.object.grounded || placement.object.hasHeading {
			return fmt.Errorf("object does not have a generated ground heading")
		}
		degrees = math.Mod(degrees, 360)
		if degrees < 0 {
			degrees += 360
		}
		setNumber(quantityNode(placement.block, "heading", "deg"), degrees)
		return document.savePoseTree(tree)
	}
	return fmt.Errorf("scene object %q is not an editable placement", id)
}

// Objects reads supported placements without opening geometry or changing text.
func (document *Document) Objects() ([]Object, error) {
	_, placements, err := document.poseNodes()
	if err != nil {
		return nil, err
	}
	objects := make([]Object, 0, len(placements))
	for _, placement := range placements {
		objects = append(objects, placement.object)
	}
	return objects, nil
}

// ChangePose applies one checked relative movement and turn atomically.
func (document *Document) ChangePose(change PoseChange) error {
	if change.LengthUnit != "m" || change.AngleUnit != "deg" {
		return fmt.Errorf("pose deltas require m and deg units")
	}
	for _, value := range []float64{change.East, change.North, change.Up, change.Heading} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("pose deltas must be finite")
		}
	}
	tree, placements, err := document.poseNodes()
	if err != nil {
		return err
	}
	for _, placement := range placements {
		if placement.object.id != change.ID {
			continue
		}
		if change.East == 0 && change.North == 0 && change.Up == 0 && change.Heading == 0 {
			return nil
		}
		deltas := [3]float64{change.East, change.North, change.Up}
		for index, delta := range deltas {
			if delta == 0 {
				continue
			}
			target := placement.coordinates[index]
			if target == nil {
				target = quantityNode(placement.block, "height", "m")
			}
			next := placement.object.position[index] + delta
			if math.IsNaN(next) || math.IsInf(next, 0) {
				return fmt.Errorf("pose produced a non-finite position")
			}
			setNumber(target, next)
		}
		if change.Heading != 0 {
			target := placement.heading
			if target == nil {
				target = quantityNode(placement.block, "heading", "deg")
			}
			next := placement.object.heading + change.Heading
			if math.IsNaN(next) || math.IsInf(next, 0) {
				return fmt.Errorf("pose produced a non-finite heading")
			}
			setNumber(target, next)
		}
		return document.savePoseTree(tree)
	}
	return fmt.Errorf("scene object %q is not an editable placement", change.ID)
}

func (document *Document) savePoseTree(tree *yaml.Node) error {
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
		return fmt.Errorf("pose edit would make invalid set YAML: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}

// poseNodes binds immutable descriptions to nodes in a fresh checked tree.
type poseNodes struct {
	object      Object
	block       *yaml.Node
	coordinates [3]*yaml.Node
	heading     *yaml.Node
}

// poseNodes only offers source placements whose native metre semantics are known.
func (document *Document) poseNodes() (*yaml.Node, []poseNodes, error) {
	if document.kind != "set" {
		return nil, nil, fmt.Errorf("object pose controls require a schema-checked set")
	}
	tree, err := document.Tree()
	if err != nil {
		return nil, nil, err
	}
	scene := member(tree.Content[0], "scene")
	var placements []poseNodes
	setting := member(scene, "setting")
	grounded := (setting == nil || setting.Value == "ground") && document.schemaName == "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"
	if grounded || setting != nil && setting.Value == "space" {
		things := member(member(scene, "set"), "things")
		if things != nil && things.Kind == yaml.MappingNode {
			for index := 0; index+1 < len(things.Content); index += 2 {
				name, block := things.Content[index].Value, things.Content[index+1]
				kind := member(block, "kind")
				recipe := member(block, "recipe")
				if (kind == nil && recipe == nil) || member(block, "east") == nil || member(block, "north") == nil ||
					member(block, "place") != nil || member(block, "orbit") != nil || member(block, "around") != nil ||
					member(block, "attitude") != nil {
					continue
				}
				object := Object{id: "scene.set.things/" + name, name: name, grounded: grounded}
				object.scale = 1
				object.scaleEditable = document.schemaName == "https://github.com/janearc/slayer/schemas/ecosystem-v2.json" && member(block, "part-spin") == nil
				if value := member(block, "scale"); value != nil {
					object.scale, err = readNumber(value, "object scale")
					if err != nil {
						return nil, nil, err
					}
					object.hasScale = true
				}
				if recipe != nil {
					object.kind, object.recipe = "recipe", recipe.Value
				} else {
					object.kind = kind.Value
				}
				placement := poseNodes{object: object, block: block}
				placement.coordinates[0], placement.coordinates[1] = member(block, "east"), member(block, "north")
				for axis := 0; axis < 2; axis++ {
					placement.object.position[axis], err = readNumber(placement.coordinates[axis], "object position")
					if err != nil {
						return nil, nil, err
					}
				}
				placement.coordinates[2], placement.object.position[2], err = poseQuantity(block, "height", "m")
				if err != nil {
					return nil, nil, err
				}
				if err := placement.readHeading(); err != nil {
					return nil, nil, err
				}
				placements = append(placements, placement)
			}
		}
	}
	world := member(scene, "world")
	if unit := member(world, "unit"); unit != nil {
		metres, err := readNumber(unit, "world.unit")
		if err != nil {
			return nil, nil, err
		}
		if metres != 1 {
			return tree, placements, nil
		}
	}
	imports := member(world, "imports")
	if imports != nil && imports.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(imports.Content); index += 2 {
			name, block := imports.Content[index].Value, imports.Content[index+1]
			at := member(block, "at")
			position, err := readVector(at, "import.at")
			if err != nil {
				return nil, nil, err
			}
			placement := poseNodes{object: Object{id: "scene.world.imports/" + name, name: name, kind: "import", position: position}, block: block}
			copy(placement.coordinates[:], at.Content)
			height, metres, err := poseQuantity(block, "height", "m")
			if err != nil {
				return nil, nil, err
			}
			placement.object.importHeight, placement.object.hasImportHeight = metres, height != nil
			if err := placement.readHeading(); err != nil {
				return nil, nil, err
			}
			placements = append(placements, placement)
		}
	}
	return tree, placements, nil
}

// readHeading retains the distinction between explicit zero and an omitted turn.
func (placement *poseNodes) readHeading() error {
	node, value, err := poseQuantity(placement.block, "heading", "deg")
	if err != nil {
		return err
	}
	placement.heading, placement.object.heading = node, value
	placement.object.hasHeading = node != nil
	return nil
}

// poseQuantity reads the exact units supported by the native placement contract.
func poseQuantity(block *yaml.Node, key, unit string) (*yaml.Node, float64, error) {
	quantity := member(block, key)
	if quantity == nil {
		return nil, 0, nil
	}
	written := member(quantity, "unit")
	if written == nil || written.Kind != yaml.ScalarNode || written.Value != unit {
		return nil, 0, fmt.Errorf("%s requires %s", key, unit)
	}
	value := member(quantity, "value")
	number, err := readNumber(value, key)
	return value, number, err
}

// quantityNode adds an explicitly unit-bearing quantity to a checked mapping.
func quantityNode(block *yaml.Node, key, unit string) *yaml.Node {
	value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "0"}
	quantity := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value"}, value,
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "unit"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: unit},
	}}
	block.Content = append(block.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, quantity)
	return value
}
