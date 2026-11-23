package named

import (
	_ "embed"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/research"
	"gopkg.in/yaml.v3"
)

// What zappy says over the real magnetar. The sentences are data, in
// magnetar-script.yaml, drafts to be rewritten; each figure in one names
// the value behind it, and Check holds the figure to that value at the
// digits it is written with.

//go:embed magnetar-script.yaml
var scriptRaw []byte

// ErrScript is a sentence whose figures do not stand on a value.
const ErrScript refusal = "named: magnetar script"

// Line is one sentence zappy says: its words, the shot it goes under,
// whether the short film says it, and the name of the value behind each
// figure in it.
type Line struct {
	Say     string            `yaml:"say"`
	Shot    string            `yaml:"shot"`
	Film    bool              `yaml:"film"`
	Figures map[string]string `yaml:"figures"`
}

// the shots a line may go under
var shots = []string{"zappy", "beside", "field", "shells", "world",
	"flare"}

// figure is a number as a sentence writes it: digits, commas between
// thousands, and a fraction.
var figure = regexp.MustCompile(`\d[\d,]*(\.\d+)?`)

// numberWords are words that would hide a figure from the check; a
// sentence writes its figures in digits. One is left out, since zappy
// says "the real one".
var numberWords = []string{"two", "three", "four", "five", "six",
	"seven", "eight", "nine", "ten", "eleven", "twelve", "twenty",
	"thirty", "forty", "fifty", "hundred", "thousand", "million",
	"billion", "trillion", "half", "third", "quarter", "fifth", "twice",
	"double", "dozen"}

// MagnetarScript is the sentences in the order zappy says them, each
// checked. The file is part of the package, so a file that does not read
// or check is a fault in the package, and stops it.
func MagnetarScript() []Line {
	var file struct {
		Lines []Line `yaml:"lines"`
	}
	if err := yaml.Unmarshal(scriptRaw, &file); err != nil {
		panic(fmt.Errorf("%w: magnetar-script.yaml: %w", ErrScript, err))
	}
	for _, line := range file.Lines {
		if err := line.Check(); err != nil {
			panic(err)
		}
	}
	return file.Lines
}

// Check is nothing when the line goes under a shot there is, writes no
// figure in words, and every figure it writes is listed and is its
// value at the digits written; otherwise what is wrong, by name.
func (line Line) Check() error {
	if !slices.Contains(shots, line.Shot) {
		return fmt.Errorf("%w: %q goes under shot %q; the shots are %s",
			ErrScript, line.Say, line.Shot, strings.Join(shots, ", "))
	}
	if word := wordedFigure(line.Say); word != "" {
		return fmt.Errorf("%w: %q says %q; write the figure in digits "+
			"and name its value", ErrScript, line.Say, word)
	}
	written := figure.FindAllString(line.Say, -1)
	for _, each := range written {
		if err := line.stands(each); err != nil {
			return err
		}
	}
	for each := range line.Figures {
		if !slices.Contains(written, each) {
			return fmt.Errorf("%w: %q lists %q, which it does not say",
				ErrScript, line.Say, each)
		}
	}
	return nil
}

// stands is nothing when the figure written is listed and is its value
// rounded to the digits written.
func (line Line) stands(written string) error {
	name, listed := line.Figures[written]
	if !listed {
		return fmt.Errorf("%w: %q says %s with no value behind it; "+
			"name a registered or computed value", ErrScript, line.Say,
			written)
	}
	value, err := valueNamed(name)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrScript, line.Say, err)
	}
	said, digits := readFigure(written)
	if !rounded(value, digits, said) {
		return fmt.Errorf("%w: %q says %s but %s is %g", ErrScript,
			line.Say, written, name, value)
	}
	return nil
}

// wordedFigure is the first number word in a sentence, or none.
func wordedFigure(say string) string {
	for _, word := range strings.FieldsFunc(strings.ToLower(say),
		func(r rune) bool { return r < 'a' || r > 'z' }) {
		if slices.Contains(numberWords, word) {
			return word
		}
	}
	return ""
}

// readFigure is a written figure's value and how many digits of it are
// significant: every digit past the leading zeros of a fraction, and of
// a whole number every digit but its trailing zeros.
func readFigure(written string) (float64, int) {
	plain := strings.ReplaceAll(written, ",", "")
	value, _ := strconv.ParseFloat(plain, 64)
	digits := strings.TrimLeft(strings.ReplaceAll(plain, ".", ""), "0")
	if !strings.Contains(plain, ".") {
		digits = strings.TrimRight(digits, "0")
	}
	return value, max(len(digits), single)
}

// rounded is whether a value rounded to digits significant digits is
// the figure said, to within the rounding of the arithmetic.
func rounded(value float64, digits int, said float64) bool {
	if value == 0 {
		return said == 0
	}
	place := math.Pow(10, math.Ceil(math.Log10(math.Abs(value)))-
		float64(digits))
	return math.Abs(math.Round(value/place)*place-said) <=
		roundingSlack*math.Abs(said)
}

// roundingSlack is how far apart, as a share, two figures may be and
// still be the same figure: the arithmetic's own rounding, picked by the
// author of this scene, 2026-11-12, far above float64's and far below
// any figure's last digit.
const roundingSlack = 1e-9

// valueNamed is a computed value by its name, or a registered one; a
// name neither holds is refused.
func valueNamed(name string) (float64, error) {
	if value, found := computed()[name]; found {
		return value, nil
	}
	read, err := research.Read()
	if err != nil {
		return 0, err
	}
	number, err := read.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%s is neither computed here nor "+
			"registered in the research data", name)
	}
	return number.Value, nil
}

// computed are the values the script names that the scene works out
// from registered ones, each by its name, with the model in the
// function that makes it.
func computed() map[string]float64 {
	body := realStar()
	values := map[string]float64{
		"magnetar-diameter-km":    diameter * body.radius,
		"zappy-radius-km":         zappyKm(),
		"zappy-times-larger":      zappyKm() / body.radius,
		"zappy-beam-spin-seconds": lantern.Zappy().Beams.Spin.Seconds(),
		"sgr1806-field-gauss":     body.field,
		"one-light-year":          single,
		"flare-speed":             flareSpeed,
		"sgr1806-distance-light-years": research.Value(
			"sgr1806-distance-kpc") / lightYearKpc,
		"flare-sun-years-low": sunYears(spikeEnergy(research.Value(
			"sgr1806-spike-energy-palmer-1e46-erg-at-15kpc"))),
		"flare-sun-years-high": sunYears(spikeEnergy(research.Value(
			"sgr1806-spike-energy-1e46-erg-at-15kpc"))),
		"flare-sunlight-minutes-low": sunlightMinutes(research.Value(
			"sgr1806-spike-fluence-palmer-erg-cm2")),
		"flare-sunlight-minutes-high": sunlightMinutes(research.Value(
			"sgr1806-spike-fluence-erg-cm2")),
	}
	for _, each := range dangers() {
		out, _ := body.reach(each.field)
		values["reach-"+each.name+"-km"] = out
	}
	values["reach-lab-earth-widths"] = values["reach-lab-km"] / earthKm()
	values["world-at-km"] = worldAt
	values["world-field-earth-times"] = body.fieldAt(worldAt,
		math.Pi/diameter) / research.Value("earth-field-gauss")
	values["world-flare-sunlight-years"] = sunlightAt(research.Value(
		"sgr1806-spike-fluence-palmer-erg-cm2"), worldAt) / julianYearS
	values["light-cylinder-km"] = body.lightCylinder()
	values["light-cylinder-moon-distances"] = body.lightCylinder() /
		moonKm()
	return values
}
