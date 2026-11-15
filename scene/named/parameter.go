package named

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

// Kind is what kind of value a parameter takes, in words.
type Kind string

// the kinds of value a parameter takes
const (
	// Whole is a whole number: an integer, or a number with no fraction.
	Whole Kind = "whole number"
	// Number is any number.
	Number Kind = "number"
	// Switch is true or false.
	Switch Kind = "switch"
	// Duration is a span of time, written as Go writes one: 24h, 1.5s.
	Duration Kind = "duration"
	// Choice is one of a list of names.
	Choice Kind = "one of a list"
)

// the refusals a scene makes of the values it is given
var (
	// ErrNoParameter is a key the scene has no parameter for.
	ErrNoParameter = errors.New("named: no such parameter")
	// ErrKind is a value of the wrong kind for its parameter.
	ErrKind = errors.New("named: a value of the wrong kind")
	// ErrBounds is a number outside its parameter's bounds.
	ErrBounds = errors.New("named: a value out of bounds")
	// ErrChoice is a name not among its parameter's choices.
	ErrChoice = errors.New("named: not one of the choices")
)

// Parameter is one thing a scene takes: its name, kind, default, bounds
// where it has them, its choices when it is one of a list, and one line
// of help.
type Parameter struct {
	name     string
	kind     Kind
	fallback any
	bounds   Bounds
	choices  []string
	help     string
}

// Name is the parameter's key in a job's with block.
func (one Parameter) Name() string { return one.name }

// Kind is what kind of value it takes.
func (one Parameter) Kind() Kind { return one.kind }

// Default is its value when none is given: an int, a float64, a bool, a
// time.Duration or a string, by its kind.
func (one Parameter) Default() any { return one.fallback }

// Bounds are the least and most a number may be; a parameter that is not
// a number, or has none, is unbounded.
func (one Parameter) Bounds() Bounds { return one.bounds }

// Choices are the names it may be, when it is one of a list.
func (one Parameter) Choices() []string { return slices.Clone(one.choices) }

// Help is one line saying what it does.
func (one Parameter) Help() string { return one.help }

// own is the parameter with a list of choices of its own.
func (one Parameter) own() Parameter {
	one.choices = slices.Clone(one.choices)
	return one
}

// whole is a whole-number parameter.
func whole(name string, fallback int, bounds Bounds,
	help string) Parameter {
	return Parameter{name: name, kind: Whole, fallback: fallback,
		bounds: bounds, help: help}
}

// number is a number parameter.
func number(name string, fallback float64, bounds Bounds,
	help string) Parameter {
	return Parameter{name: name, kind: Number, fallback: fallback,
		bounds: bounds, help: help}
}

// toggle is a switch parameter.
func toggle(name string, fallback bool, help string) Parameter {
	return Parameter{name: name, kind: Switch, fallback: fallback,
		help: help}
}

// span is a duration parameter.
func span(name string, fallback time.Duration, help string) Parameter {
	return Parameter{name: name, kind: Duration, fallback: fallback,
		help: help}
}

// oneOf is a parameter that is one of a list of names.
func oneOf(name, fallback string, choices []string,
	help string) Parameter {
	return Parameter{name: name, kind: Choice, fallback: fallback,
		choices: choices, help: help}
}

// values are a scene's parameters once checked, every one present, each
// held as its kind's Go type.
type values map[string]any

// whole is the whole number of that name.
func (held values) whole(name string) int { return held[name].(int) }

// number is the number of that name.
func (held values) number(name string) float64 {
	return held[name].(float64)
}

// toggle is the switch of that name.
func (held values) toggle(name string) bool { return held[name].(bool) }

// span is the duration of that name.
func (held values) span(name string) time.Duration {
	return held[name].(time.Duration)
}

// choice is the name chosen for the parameter of that name.
func (held values) choice(name string) string {
	return held[name].(string)
}

// read is the scene's values, as readValues reads them.
func (scene Scene) read(with map[string]any) (values, error) {
	return readValues(scene.name, scene.parameters, with)
}

// readValues is a scene's values: each given one checked against its
// parameter, and the rest at their defaults. The first key or value that
// is wrong is refused, in the order of the keys' names.
func readValues(name string, parameters []Parameter,
	with map[string]any) (values, error) {
	held := values{}
	for _, one := range parameters {
		held[one.name] = one.fallback
	}
	keys := make([]string, 0, len(with))
	for key := range with {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		one, found := parameterOf(parameters, key)
		if !found {
			return nil, fmt.Errorf("%w: %s has no %q; its parameters "+
				"are %s", ErrNoParameter, name, key,
				strings.Join(keysOf(parameters), ", "))
		}
		value, err := one.take(with[key])
		if err != nil {
			return nil, fmt.Errorf("%s's %w", name, err)
		}
		held[key] = value
	}
	return held, nil
}

// parameterOf is the parameter of that name, if there is one.
func parameterOf(parameters []Parameter, name string) (Parameter, bool) {
	for _, one := range parameters {
		if one.name == name {
			return one, true
		}
	}
	return Parameter{}, false
}

// keysOf are the names of the parameters, in their order.
func keysOf(parameters []Parameter) []string {
	all := make([]string, len(parameters))
	for i, one := range parameters {
		all[i] = one.name
	}
	return all
}

// owned is a copy of the parameters, each with a list of choices of its
// own.
func owned(parameters []Parameter) []Parameter {
	held := make([]Parameter, len(parameters))
	for i, one := range parameters {
		held[i] = one.own()
	}
	return held
}

// take is a given value as the parameter holds it, or why it cannot be.
func (one Parameter) take(given any) (any, error) {
	value, fits := readers[one.kind](given)
	if !fits {
		return nil, fmt.Errorf("%s is a %s, not %v (%T): %w", one.name,
			one.kind, given, given, ErrKind)
	}
	return value, one.holds(value)
}

// holds is why a value the parameter's kind took is outside its bounds or
// its choices, or nothing.
func (one Parameter) holds(value any) error {
	if one.kind == Choice && !slices.Contains(one.choices,
		value.(string)) {
		return fmt.Errorf("%s is one of %s, not %q: %w", one.name,
			strings.Join(one.choices, ", "), value, ErrChoice)
	}
	measure, numeric := asFloat(value)
	if numeric && !one.bounds.Holds(measure) {
		return fmt.Errorf("%s is %s, not %v: %w", one.name, one.bounds,
			value, ErrBounds)
	}
	return nil
}

// readers read a given value as each kind holds it, or say it is not of
// that kind.
var readers = map[Kind]func(any) (any, bool){
	Whole:    readWhole,
	Number:   readNumber,
	Switch:   readSwitch,
	Duration: readDuration,
	Choice:   readChoice,
}

// readWhole is a whole number as an int: any integer that fits, or a
// float64 with no fraction, as a JSON decoder gives every number.
func readWhole(given any) (any, bool) {
	switch value := given.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case uint64:
		return int(value), value <= math.MaxInt
	case float64:
		return int(value), value == math.Trunc(value) &&
			math.Abs(value) <= maxExact
	}
	return nil, false
}

// maxExact is the largest whole number a float64 holds exactly, two to the
// 53rd: defined by the float's 53-bit significand.
const maxExact = 1 << 53

// readNumber is a number as a float64, from any integer or float.
func readNumber(given any) (any, bool) {
	switch value := given.(type) {
	case float64:
		return value, !math.IsNaN(value)
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint64:
		return float64(value), true
	}
	return nil, false
}

// readSwitch is a switch, from a bool only.
func readSwitch(given any) (any, bool) {
	value, fits := given.(bool)
	return value, fits
}

// readDuration is a duration from Go's duration text or a time.Duration.
// A bare number is refused: it says no unit.
func readDuration(given any) (any, bool) {
	if value, fits := given.(time.Duration); fits {
		return value, true
	}
	text, fits := given.(string)
	if !fits {
		return nil, false
	}
	value, err := time.ParseDuration(text)
	return value, err == nil
}

// readChoice is a name, from a string.
func readChoice(given any) (any, bool) {
	value, fits := given.(string)
	return value, fits
}

// asFloat is a whole number or a number as a float64, to hold to bounds;
// a value of another kind is not one.
func asFloat(value any) (float64, bool) {
	switch held := value.(type) {
	case int:
		return float64(held), true
	case float64:
		return held, true
	}
	return 0, false
}
