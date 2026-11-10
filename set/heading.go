package set

import yaml "gopkg.in/yaml.v3"

// Heading is an optional clockwise rotation from the model's north pose.
// Unspecified placements retain their existing seeded turn.
func (made Thing) Heading() (float64, bool) { return said(made.heading) }

type headingReader struct {
	reading reader
	target  **float64
}

// heading checks an explicit zero independently from an omitted heading.
func (reading reader) heading(target **float64) step {
	input := headingReader{reading: reading, target: target}
	return input.read
}

// read retains the angle only after its quantity has been checked.
func (input headingReader) read(name, block *yaml.Node) error {
	value := float64(laserZero)
	if err := input.reading.quantity(&value, "deg", laserZero,
		laserTurn)(name, block); err != nil {
		return err
	}
	*input.target = &value
	return nil
}
