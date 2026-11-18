package draft

import (
	"bytes"
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

// MoveCamera translates the first fixed set camera and its target by the same
// east, north and up displacement in metres. It preserves lens and scene fields.
// Unsupported cameras and invalid candidates leave the draft unchanged.
func (document *Document) MoveCamera(east, north, up float64) error {
	if document.kind != "set" {
		return fmt.Errorf("camera translation requires a schema-checked set")
	}
	delta := [3]float64{east, north, up}
	for _, value := range delta {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("camera translation must be finite")
		}
	}
	nodes, err := document.simpleSetCamera()
	if err != nil {
		return err
	}
	at, err := readVector(nodes.at, "location.at")
	if err != nil {
		return err
	}
	target, err := readVector(nodes.lookAt, "orientation.look_at")
	if err != nil {
		return err
	}
	for axis, value := range delta {
		at[axis] += value
		target[axis] += value
		if math.IsNaN(at[axis]) || math.IsInf(at[axis], 0) || math.IsNaN(target[axis]) || math.IsInf(target[axis], 0) {
			return fmt.Errorf("camera translation would make a nonfinite pose")
		}
	}
	if at == target {
		return fmt.Errorf("camera translation would collapse the eye and target")
	}
	if east == 0 && north == 0 && up == 0 {
		return nil
	}
	writeVector(nodes.at, at)
	writeVector(nodes.lookAt, target)
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{nodes.root}}); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return fmt.Errorf("camera translation would make invalid set YAML: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}
