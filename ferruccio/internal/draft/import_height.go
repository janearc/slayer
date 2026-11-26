package draft

import (
	"bytes"
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

// SetImportHeight sets one imported model's physical target height in metres.
// It checks the complete edited set before changing the draft.
func (document *Document) SetImportHeight(id string, metres float64) error {
	if math.IsNaN(metres) || math.IsInf(metres, 0) || metres <= 0 || metres > 1e6 {
		return fmt.Errorf("import height must be positive and at most 1000000 m")
	}
	tree, placements, err := document.poseNodes()
	if err != nil {
		return err
	}
	for _, placement := range placements {
		if placement.object.id != id {
			continue
		}
		if placement.object.kind != "import" {
			return fmt.Errorf("scene object %q is not an import", id)
		}
		value, old, err := poseQuantity(placement.block, "height", "m")
		if err != nil {
			return err
		}
		if value != nil && old == metres {
			return nil
		}
		if value == nil {
			value = quantityNode(placement.block, "height", "m")
		}
		setNumber(value, metres)
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
			return fmt.Errorf("import height would make invalid set YAML: %w", err)
		}
		document.source = bytes.Clone(encoded.Bytes())
		return nil
	}
	return fmt.Errorf("scene object %q is not an editable import", id)
}
