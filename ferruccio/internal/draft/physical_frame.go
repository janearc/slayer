package draft

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"gopkg.in/yaml.v3"
)

// PhysicalFrame reads the dimensions of an explicitly authored set rig.
func (document *Document) PhysicalFrame() ([2]float64, bool) {
	var size [2]float64
	if document.kind != "set" && document.kind != "legacy-set" {
		return size, false
	}
	tree, err := document.Tree()
	if err != nil {
		return size, false
	}
	film := member(member(tree.Content[0], "scene"), "film")
	frame := member(film, "frame")
	if frame == nil || member(film, "lens") == nil || member(film, "shutter") == nil || member(film, "glimpses") == nil {
		return size, false
	}
	lens := member(film, "lens")
	if member(lens, "focal") == nil || member(lens, "stop") == nil || member(lens, "focus") == nil {
		return size, false
	}
	for i, key := range []string{"width", "height"} {
		value := member(frame, key)
		if value == nil {
			return size, false
		}
		size[i], err = strconv.ParseFloat(value.Value, 64)
		if err != nil || math.IsNaN(size[i]) || math.IsInf(size[i], 0) || size[i] < 1 || size[i] > 1000 {
			return size, false
		}
	}
	return size, true
}

// SetPhysicalFrame changes a complete rig's frame without changing its lens.
func (document *Document) SetPhysicalFrame(size [2]float64) error {
	for _, n := range size {
		if math.IsNaN(n) || math.IsInf(n, 0) || n < 1 || n > 1000 {
			return fmt.Errorf("physical frame dimensions must be between 1 and 1000 mm")
		}
	}
	current, ok := document.PhysicalFrame()
	if !ok {
		return fmt.Errorf("the set does not have a complete physical film rig")
	}
	if current == size {
		return nil
	}
	tree, err := document.Tree()
	if err != nil {
		return err
	}
	frame := member(member(member(tree.Content[0], "scene"), "film"), "frame")
	setNumber(member(frame, "width"), size[0])
	setNumber(member(frame, "height"), size[1])
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
		return fmt.Errorf("frame edit would make invalid set YAML: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}
