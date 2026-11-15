// Package explain composes a light show's explainer frame: the finished
// film, whole, over a panel that shows the beat, the rig's lamps, the
// chant's words and a plain sentence saying what is happening. The
// README beside this file is the design; Parts is what a show supplies.
package explain

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"unicode/utf8"

	yaml "gopkg.in/yaml.v3"
)

// ErrExplain is wrapped by every error the package returns.
var ErrExplain = errors.New("explain")

// Placer places a moment written in a show's own terms, "bar.beat" or
// seconds, on its clock, in seconds; a place it lacks is refused by name.
type Placer interface {
	Moment(at string, atSec *float64) (float64, error)
}

// place is a moment as the script writes it: bar.beat, or seconds.
type place struct {
	At    string   `yaml:"at"`
	AtSec *float64 `yaml:"at_sec"`
}

// span is a short as the script writes it: its start and end.
type span struct {
	From  place `yaml:"from"`
	Until place `yaml:"until"`
}

// said is one sentence as the script writes it.
type said struct {
	place `yaml:",inline"`
	Says  string `yaml:"says"`
}

// written is a script as it is on disk.
type written struct {
	Shorts map[string]span `yaml:"shorts"`
	Says   []said          `yaml:"says"`
	Chant  []struct {
		FromSec  float64 `yaml:"from_sec"`
		UntilSec float64 `yaml:"until_sec"`
		Words    string  `yaml:"words"`
	} `yaml:"chant"`
}

// Sentence is one line of the script: what it says and from when, in
// seconds.
type Sentence struct {
	from  float64
	words string
}

// Words is what the sentence says.
func (of Sentence) Words() string { return of.words }

// From is when the sentence comes up, in seconds.
func (of Sentence) From() float64 { return of.from }

// Script is an explainer's script placed on a show's clock: its shorts
// by name, its sentences in order, and the chant's lines.
type Script struct {
	shorts    map[string][2]float64
	sentences []Sentence
	chant     []line
}

// ReadScript reads a script file, placing it with a show's placer.
func ReadScript(path string, places Placer) (*Script, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrExplain, err)
	}
	return ParseScript(raw, places)
}

// ParseScript reads a script from its yaml. A sentence or a short at a
// place the show lacks is refused by name, and so are sentences out of
// order, a short that is empty and a chant line that ends before it
// starts.
func ParseScript(raw []byte, places Placer) (*Script, error) {
	if places == nil {
		return nil, fmt.Errorf("%w: a script needs a placer", ErrExplain)
	}
	var on written
	if err := yaml.Unmarshal(raw, &on); err != nil {
		return nil, fmt.Errorf("%w: the script: %w", ErrExplain, err)
	}
	made := &Script{shorts: map[string][2]float64{}}
	if err := made.readShorts(on, places); err != nil {
		return nil, err
	}
	if err := made.readSentences(on, places); err != nil {
		return nil, err
	}
	for _, each := range on.Chant {
		if each.UntilSec <= each.FromSec {
			return nil, fmt.Errorf("%w: the chant line %q, %v s to %v s, "+
				"is empty", ErrExplain, each.Words, each.FromSec,
				each.UntilSec)
		}
		made.chant = append(made.chant, line{words: each.Words,
			from: each.FromSec, until: each.UntilSec})
	}
	return made, nil
}

// readShorts places each short's start and end.
func (made *Script) readShorts(on written, places Placer) error {
	for name, each := range on.Shorts {
		from, err := places.Moment(each.From.At, each.From.AtSec)
		if err != nil {
			return fmt.Errorf("%w: the short %s's start: %w", ErrExplain,
				name, err)
		}
		until, err := places.Moment(each.Until.At, each.Until.AtSec)
		if err != nil {
			return fmt.Errorf("%w: the short %s's end: %w", ErrExplain,
				name, err)
		}
		if until <= from {
			return fmt.Errorf("%w: the short %s, %v s to %v s, is empty",
				ErrExplain, name, from, until)
		}
		made.shorts[name] = [2]float64{from, until}
	}
	return nil
}

// readSentences places each sentence, in order.
func (made *Script) readSentences(on written, places Placer) error {
	for _, each := range on.Says {
		at, err := places.Moment(each.At, each.AtSec)
		if err != nil {
			return fmt.Errorf("%w: the sentence %q: %w", ErrExplain,
				each.Says, err)
		}
		if n := len(made.sentences); n > 0 && at <= made.sentences[n-1].from {
			return fmt.Errorf("%w: the sentence %q at %v s comes before "+
				"the one above it; keep the script in order", ErrExplain,
				each.Says, at)
		}
		made.sentences = append(made.sentences, Sentence{from: at,
			words: each.Says})
	}
	return nil
}

// Short is a short's start and end, in seconds, and whether the script
// has one of that name.
func (of *Script) Short(name string) (from, until float64, ok bool) {
	both, ok := of.shorts[name]
	return both[0], both[1], ok
}

// Shorts is the names of the script's shorts, sorted.
func (of *Script) Shorts() []string {
	names := make([]string, 0, len(of.shorts))
	for name := range of.shorts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Says is the sentence standing at a moment: the script's latest at or
// before it. Before the first there is none.
func (of *Script) Says(seconds float64) (Sentence, bool) {
	index := sort.Search(len(of.sentences), func(i int) bool {
		return of.sentences[i].from > seconds
	}) - 1
	if index < 0 {
		return Sentence{}, false
	}
	return of.sentences[index], true
}

// TooFast is every sentence that comes up inside a short too long to
// read in the time it stands there, until the next or the short's end,
// at readingRate, as errors naming it; none when all can be read. A
// short the script lacks is refused.
func (of *Script) TooFast(short string) ([]error, error) {
	from, until, ok := of.Short(short)
	if !ok {
		return nil, fmt.Errorf("%w: no short %q; the script has %v",
			ErrExplain, short, of.Shorts())
	}
	var slow []error
	for i, each := range of.sentences {
		if each.from < from || each.from >= until {
			continue
		}
		ends := until
		if i+1 < len(of.sentences) {
			ends = min(ends, of.sentences[i+1].from)
		}
		needs := float64(utf8.RuneCountInString(each.words)) / readingRate
		if needs > ends-each.from {
			slow = append(slow, fmt.Errorf("%w: in %s, %q needs %.1f s to "+
				"read and stands %.1f s; shorten it or give it more bars",
				ErrExplain, short, each.words, needs, ends-each.from))
		}
	}
	return slow, nil
}
