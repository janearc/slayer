package draft

import (
	"bytes"
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

// SetThingScale checks and sets the uniform factor of a supported placement.
// Callers must also check whether its constructor deforms over time.
func (document *Document) SetThingScale(id string, scale float64) error {
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale < .01 || scale > 100 {
		return fmt.Errorf("size factor must be between 0.01 and 100")
	}
	tree, placements, err := document.poseNodes()
	if err != nil {
		return err
	}
	for _, placement := range placements {
		if placement.object.id != id {
			continue
		}
		if !placement.object.scaleEditable {
			return fmt.Errorf("scene object %q does not support a size factor", id)
		}
		if placement.object.scale == scale {
			return nil
		}
		value := member(placement.block, "scale")
		if value == nil {
			value = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float"}
			placement.block.Content = append(placement.block.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "scale"}, value)
		}
		setNumber(value, scale)
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
			return fmt.Errorf("size edit would make invalid set YAML: %w", err)
		}
		document.source = bytes.Clone(encoded.Bytes())
		return nil
	}
	return fmt.Errorf("scene object %q is not an editable placement", id)
}
