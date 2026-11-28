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
	_, film := document.physicalFilm()
	return physicalFrame(film)
}

// physicalFilm reads the checked set tree and its authored film.
func (document *Document) physicalFilm() (tree, film *yaml.Node) {
	if document.kind != "set" && document.kind != "legacy-set" {
		return nil, nil
	}
	tree, err := document.Tree()
	if err != nil {
		return nil, nil
	}
	return tree, member(member(tree.Content[0], "scene"), "film")
}

// physicalFrame reads a frame only when its supporting rig is present.
func physicalFrame(film *yaml.Node) ([2]float64, bool) {
	var size [2]float64
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
		dimension, err := strconv.ParseFloat(value.Value, 64)
		if err != nil || math.IsNaN(dimension) || math.IsInf(dimension, 0) || dimension < 1 || dimension > 1000 {
			return size, false
		}
		size[i] = dimension
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
	tree, film := document.physicalFilm()
	current, ok := physicalFrame(film)
	if !ok {
		return fmt.Errorf("the set does not have a complete physical film rig")
	}
	if current == size {
		return nil
	}
	frame := member(film, "frame")
	setNumber(member(frame, "width"), size[0])
	setNumber(member(frame, "height"), size[1])
	return document.replacePhysicalFilm(tree, "frame")
}

// replacePhysicalFilm validates an encoded rig edit before replacing source.
func (document *Document) replacePhysicalFilm(tree *yaml.Node, section string) error {
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
		return fmt.Errorf("%s edit would make invalid set YAML: %w", section, err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}
