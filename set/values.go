package set

import (
	"math"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// block is a step that reads a block of keys, none of them for later.
func (reading reader) block(steps map[string]step) step {
	return func(_, value *yaml.Node) error {
		return reading.keys(value, steps, nil)
	}
}

// each reads every key of a block the same way, whatever its name: a
// camera's name, an asset's kind.
func (reading reader) each(value *yaml.Node,
	read func(name, block *yaml.Node) error) error {
	if value.Kind != yaml.MappingNode {
		return reading.refuse(value, "this takes keys under it")
	}
	for index := 0; index+1 < len(value.Content); index += 2 {
		if err := read(value.Content[index],
			value.Content[index+1]); err != nil {
			return err
		}
	}
	return nil
}

// text is a step that reads a word or words into target.
func (reading reader) text(target *string) step {
	return func(name, value *yaml.Node) error {
		if value.Kind != yaml.ScalarNode || value.Value == "" {
			return reading.refuse(value, "%s takes a word", name.Value)
		}
		*target = value.Value
		return nil
	}
}

// only is a step that takes one of a few words, and nothing else.
func (reading reader) only(words ...string) step {
	return func(name, value *yaml.Node) error {
		if value.Kind != yaml.ScalarNode ||
			!slices.Contains(words, value.Value) {
			return reading.refuse(value, "%s takes %v, not %q",
				name.Value, words, value.Value)
		}
		return nil
	}
}

// word is a step that takes one of a few words into target.
func (reading reader) word(target *string, words ...string) step {
	check := reading.only(words...)
	return func(name, value *yaml.Node) error {
		if err := check(name, value); err != nil {
			return err
		}
		*target = value.Value
		return nil
	}
}

// number is a step that reads a finite number, from least to most, into
// target.
func (reading reader) number(target *float64, least, most float64) step {
	return func(name, value *yaml.Node) error {
		var amount float64
		if value.Decode(&amount) != nil || math.IsNaN(amount) ||
			math.IsInf(amount, 0) || amount < least || amount > most {
			return reading.refuse(value, "%s takes a number from %v to "+
				"%v, not %q", name.Value, least, most, value.Value)
		}
		*target = amount
		return nil
	}
}
