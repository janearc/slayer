package cheeptune

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// kinds are the channels the Game Boy has, and how many of each.
var kinds = []string{"pulse", "wave", "noise"}

var kindLimit = map[string]int{"pulse": 2, "wave": 1, "noise": 1}

// duties are the only pulse widths the hardware has.
var duties = []float64{0.125, 0.25, 0.5, 0.75}

// Voicing is what plays a score: each role bound to a channel on one machine,
// and what that channel sounds like.
type Voicing struct {
	Name     string    `json:"name"`
	Machine  string    `json:"machine"`
	Channels []channel `json:"channels"`
	Drop     []string  `json:"drop"`

	// the studio's mix; the Game Boy has none of these.
	Master *float64 `json:"master"`
	Reverb *reverb  `json:"reverb"`
	Delay  *repeats `json:"delay"`
}

// channel is one of the machine's voices. It plays a role or echoes one.
// Which fields it may carry depends on its kind, and a field on the wrong
// kind is refused rather than ignored.
type channel struct {
	Kind   string   `json:"kind"`
	Role   string   `json:"role"`
	Echo   string   `json:"echo"`
	Delay  string   `json:"delay"`
	Gain   *float64 `json:"gain"`
	Volume *float64 `json:"volume"`
	Detune float64  `json:"detune"`
	Pan    string   `json:"pan"`

	Duty     *float64  `json:"duty"`
	Envelope *envelope `json:"envelope"`
	Vibrato  *vibrato  `json:"vibrato"`
	Arp      arp       `json:"arp"`
	Table    string    `json:"table"`

	Hits map[string]hit `json:"hits"`

	// the studio's fields; studio.go says what each is.
	Place  *float64        `json:"place"`
	Send   *send           `json:"send"`
	Voices int             `json:"voices"`
	Glide  *float64        `json:"glide"`
	FM     *fmSound        `json:"fm"`
	Analog *analog         `json:"analog"`
	Pluck  *pluck          `json:"pluck"`
	Modal  *modal          `json:"modal"`
	Drums  map[string]drum `json:"drums"`

	delay frac
	table [32]uint8
}

// envelope is attack, decay and release in seconds, and sustain from nought
// to one. It is the one envelope shape; a general curve is not offered.
type envelope struct {
	Attack  float64 `json:"attack"`
	Decay   float64 `json:"decay"`
	Sustain float64 `json:"sustain"`
	Release float64 `json:"release"`
}

// vibrato is rate in hertz, depth in cents, and how long a note holds still
// before it starts to move.
type vibrato struct {
	Rate  float64 `json:"rate"`
	Depth float64 `json:"depth"`
	Delay float64 `json:"delay"`
}

// hit is a sound on the noise channel, named by the score. Shift and divisor
// are the two numbers the hardware takes; short selects the seven-bit
// register, which is tonal where the fifteen-bit one is hiss.
type hit struct {
	Shift   int      `json:"shift"`
	Divisor int      `json:"divisor"`
	Short   bool     `json:"short"`
	Volume  *float64 `json:"volume"`
	Decay   float64  `json:"decay"`
}

// arp is how fast a chord cycles through its notes. A number is driver ticks
// a step, the chip idiom where a chord shimmers. A string is beats a step,
// n or n/d, where a chord is heard as broken.
type arp struct {
	ticks int
	beats frac
	set   bool
}

// UnmarshalJSON reads an arp as ticks (a number) or beats (a string).
func (a *arp) UnmarshalJSON(b []byte) error {
	var n float64
	if json.Unmarshal(b, &n) == nil {
		if n != float64(int(n)) || n < 1 || n > 60 {
			return fmt.Errorf(
				"arp %v: ticks are a whole number, 1 to 60", n)
		}
		a.ticks, a.set = int(n), true
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("arp is ticks (3) or beats (\"1/8\")")
	}
	f, err := parseFrac(s)
	if err != nil {
		return fmt.Errorf("arp: %w", err)
	}
	a.beats, a.set = f, true
	return nil
}

// MarshalJSON writes an arp back the way it was read.
func (a arp) MarshalJSON() ([]byte, error) {
	switch {
	case !a.set:
		return []byte("null"), nil
	case a.ticks > 0:
		return []byte(strconv.Itoa(a.ticks)), nil
	}
	return json.Marshal(a.beats.String())
}

// String is the arp as check prints it.
func (a arp) String() string {
	switch {
	case !a.set:
		return "3 ticks (default)"
	case a.ticks > 0:
		return fmt.Sprintf("%d ticks", a.ticks)
	}
	return a.beats.String() + " beats"
}

// tables are the built-in wave tables, by name.
var tables = map[string][32]uint8{
	"triangle": func() (t [32]uint8) {
		for i := range t {
			if i < 16 {
				t[i] = uint8(i)
			} else {
				t[i] = uint8(31 - i)
			}
		}
		return
	}(),
	"saw": func() (t [32]uint8) {
		for i := range t {
			t[i] = uint8(i / 2)
		}
		return
	}(),
	"square": func() (t [32]uint8) {
		for i := 16; i < 32; i++ {
			t[i] = 15
		}
		return
	}(),
}

// ReadVoicing reads and checks a voicing on its own, from a path or a
// built-in named with "@". Whether it fits a score is Bind's question.
func ReadVoicing(path string) (*Voicing, error) {
	b, err := readDoc(path)
	if err != nil {
		return nil, err
	}
	return ParseVoicing(b)
}

// ParseVoicing is ReadVoicing on bytes already in hand.
func ParseVoicing(b []byte) (*Voicing, error) {
	var v Voicing
	if err := decodeStrict(b, &v); err != nil {
		return nil, fmt.Errorf("voicing: %w", err)
	}
	if err := v.check(); err != nil {
		return nil, err
	}
	return &v, nil
}

// check is every rule a voicing keeps on its own, all problems joined.
func (v *Voicing) check() error {
	var errs []error
	fail := func(f string, a ...any) {
		errs = append(errs, fmt.Errorf(f, a...))
	}
	if v.Name == "" {
		fail("voicing has no name")
	}
	if !known(v.Machine, machineNameList) {
		fail("unknown machine %q; machines are %s",
			v.Machine, strings.Join(machineNameList, ", "))
	}
	if len(v.Channels) == 0 {
		fail("voicing has no channels")
	}
	plays := map[string]int{}
	if m, ok := machines[v.Machine]; ok && len(v.Channels) > 0 {
		var merrs []error
		merrs, plays = m.check(v)
		errs = append(errs, merrs...)
	}
	for _, d := range v.Drop {
		if !known(d, roles) {
			fail("drop: unknown role %q", d)
		}
		if plays[d] > 0 {
			fail("drop: %s is dropped and also played", d)
		}
	}
	return errors.Join(errs...)
}

// does is what a channel does, on any machine: exactly one of play and
// echo, and an echo's delay and gain.
func (c *channel) does(at string) []error {
	var errs []error
	fail := func(f string, a ...any) {
		errs = append(errs, fmt.Errorf(at+": "+f, a...))
	}
	// what a channel does: exactly one of play and echo.
	switch {
	case c.Role != "" && c.Echo != "":
		fail("plays %s and echoes %s; a channel does one",
			c.Role, c.Echo)
	case c.Role == "" && c.Echo == "":
		fail("neither plays a role nor echoes one")
	case c.Role != "" && !known(c.Role, roles):
		fail("unknown role %q; roles are %s", c.Role,
			strings.Join(roles, ", "))
	case c.Echo != "" && !known(c.Echo, roles):
		fail("echoes unknown role %q", c.Echo)
	}
	if c.Echo != "" {
		if c.Delay == "" {
			fail("an echo needs a delay in beats")
		} else if d, err := parseFrac(c.Delay); err != nil {
			fail("delay: %v", err)
		} else {
			c.delay = d
		}
		if c.Gain == nil {
			fail("an echo needs a gain")
		}
	} else {
		if c.Delay != "" {
			fail("delay is only for an echo")
		}
		if c.Gain != nil {
			fail("gain is only for an echo; " +
				"volume is for any channel")
		}
	}
	unit := func(name string, p *float64) {
		if p != nil && (*p < 0 || *p > 1) {
			fail("%s %v is outside 0 to 1", name, *p)
		}
	}
	unit("gain", c.Gain)
	unit("volume", c.Volume)
	return errs
}

// readTable is a named table or thirty-two hex digits, one a step.
func (c *channel) readTable(at string) []error {
	if c.Table == "" {
		return []error{fmt.Errorf("%s: a wave needs a table: %s, "+
			"or 32 hex digits", at, tableNames())}
	}
	if t, ok := tables[c.Table]; ok {
		c.table = t
		return nil
	}
	if len(c.Table) != 32 {
		return []error{fmt.Errorf(
			"%s: table %q is not %s or 32 hex digits",
			at, c.Table, tableNames())}
	}
	for i := range 32 {
		d, err := strconv.ParseUint(c.Table[i:i+1], 16, 8)
		if err != nil {
			return []error{fmt.Errorf(
				"%s: table step %d %q is not hex",
				at, i+1, c.Table[i:i+1])}
		}
		c.table[i] = uint8(d)
	}
	return nil
}

// tableNames lists the built-in tables, for an error.
func tableNames() string {
	return "triangle, saw, square"
}

// knownFloat is whether x is exactly one of set.
func knownFloat(x float64, set []float64) bool {
	return slices.Contains(set, x)
}

// plural is the s a count past one takes.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
