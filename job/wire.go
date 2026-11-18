package job

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

// document is a job as it is written, decoded only after the schema has
// checked it. A key left unsaid is its zero, or nil; the schema refuses
// the zero of every key where zero would mean something.
type document struct {
	Schema   string    `yaml:"$schema" json:"$schema"`
	Scene    sceneWire `yaml:"scene" json:"scene"`
	Film     *filmWire `yaml:"film" json:"film,omitempty"`
	Renderer string    `yaml:"renderer" json:"renderer,omitempty"`
	Quality  string    `yaml:"quality" json:"quality,omitempty"`

	// Vibes, Fastlight and Shortcuts are the job's flags and the parts
	// they turn on, each on, off or unsaid; see flags.go.
	Vibes     string            `yaml:"vibes" json:"vibes,omitempty"`
	Fastlight string            `yaml:"fastlight" json:"fastlight,omitempty"`
	Shortcuts map[string]string `yaml:"shortcuts" json:"shortcuts,omitempty"`

	Camera   *cameraWire `yaml:"camera" json:"camera,omitempty"`
	At       *quantity   `yaml:"at" json:"at,omitempty"`
	FPS      int         `yaml:"fps" json:"fps,omitempty"`
	Duration *quantity   `yaml:"duration" json:"duration,omitempty"`
	Start    *quantity   `yaml:"start" json:"start,omitempty"`

	// Stretch is the scene-duration: the scene time a film spans.
	Stretch *quantity `yaml:"scene-duration" json:"scene-duration,omitempty"`

	Shutter *shutterWire `yaml:"shutter" json:"shutter,omitempty"`
	Print   *printWire   `yaml:"print" json:"print,omitempty"`
	Stock   *stockWire   `yaml:"stock" json:"stock,omitempty"`
	Titles  *titlesWire  `yaml:"titles" json:"titles,omitempty"`
	Sound   *soundWire   `yaml:"sound" json:"sound,omitempty"`
	Outputs *outputsWire `yaml:"outputs" json:"outputs,omitempty"`
	Limits  *limitsWire  `yaml:"limits" json:"limits,omitempty"`
}

// quantity is a number and its unit, as {value: 20, unit: s}.
type quantity struct {
	Value float64 `yaml:"value" json:"value"`
	Unit  string  `yaml:"unit" json:"unit"`
}

// duration is a quantity of seconds as a duration, converted as a
// sequence plan converts it.
func (amount quantity) duration() time.Duration {
	return time.Duration(amount.Value * float64(time.Second))
}

// secondsOf is a duration as a quantity of seconds.
func secondsOf(span time.Duration) *quantity {
	return &quantity{Value: span.Seconds(), Unit: seconds}
}

type sceneWire struct {
	Set    string         `yaml:"set" json:"set,omitempty"`
	Smoke  int            `yaml:"smoke" json:"smoke,omitempty"`
	Named  string         `yaml:"named" json:"named,omitempty"`
	With   map[string]any `yaml:"with" json:"with,omitempty"`
	Play   string         `yaml:"play" json:"play,omitempty"`
	Frames string         `yaml:"frames" json:"frames,omitempty"`
}

type filmWire struct {
	Preset   string  `yaml:"preset" json:"preset,omitempty"`
	Width    int     `yaml:"width" json:"width,omitempty"`
	Height   int     `yaml:"height" json:"height,omitempty"`
	Rays     int     `yaml:"rays" json:"rays,omitempty"`
	Wide     float64 `yaml:"wide" json:"wide,omitempty"`
	Tall     float64 `yaml:"tall" json:"tall,omitempty"`
	Exposure float64 `yaml:"exposure" json:"exposure,omitempty"`
	Aperture float64 `yaml:"aperture" json:"aperture,omitempty"`
	Focus    float64 `yaml:"focus" json:"focus,omitempty"`
	Mode     string  `yaml:"mode" json:"mode,omitempty"`
}

// point is a place in scene metres: x east, y up, z south.
type point struct {
	X float64 `yaml:"x" json:"x"` //ninja:vow name x, y, z at every call site
	Y float64 `yaml:"y" json:"y"` //ninja:vow name x, y, z at every call site
	Z float64 `yaml:"z" json:"z"` //ninja:vow name x, y, z at every call site
}

// vec is the point as the maths package holds one.
func (place point) vec() maths.Vec { return maths.V(place.X, place.Y, place.Z) }

type cameraWire struct {
	Name   string     `yaml:"name" json:"name,omitempty"`
	Unit   string     `yaml:"unit" json:"unit,omitempty"`
	LookAt *point     `yaml:"look-at" json:"look-at,omitempty"`
	Up     *point     `yaml:"up" json:"up,omitempty"`
	Moves  []moveWire `yaml:"moves" json:"moves,omitempty"`
	Zoom   *zoomWire  `yaml:"zoom" json:"zoom,omitempty"`
}

type moveWire struct {
	From     point     `yaml:"from" json:"from"`
	To       point     `yaml:"to" json:"to"`
	Speed    *quantity `yaml:"speed" json:"speed,omitempty"`
	Duration *quantity `yaml:"duration" json:"duration,omitempty"`
	Ease     string    `yaml:"ease" json:"ease,omitempty"`
	Look     *lookWire `yaml:"look" json:"look,omitempty"`
}

type lookWire struct {
	From point `yaml:"from" json:"from"`
	To   point `yaml:"to" json:"to"`
}

type zoomWire struct {
	From     quantity `yaml:"from" json:"from"`
	To       quantity `yaml:"to" json:"to"`
	Duration quantity `yaml:"duration" json:"duration"`
	Ease     string   `yaml:"ease" json:"ease,omitempty"`
}

type shutterWire struct {
	Angle    quantity `yaml:"angle" json:"angle"`
	Glimpses int      `yaml:"glimpses" json:"glimpses"`
}

type printWire struct {
	Knee  float64  `yaml:"knee" json:"knee"`
	Top   float64  `yaml:"top" json:"top"`
	Bloom *float64 `yaml:"bloom" json:"bloom,omitempty"`
	Stops *float64 `yaml:"stops" json:"stops,omitempty"`
}

type stockWire struct {
	Name  string   `yaml:"name" json:"name"`
	Seed  *int64   `yaml:"seed" json:"seed,omitempty"`
	Rated *float64 `yaml:"rated" json:"rated,omitempty"`
}

type titlesWire struct {
	Theme    string    `yaml:"theme" json:"theme,omitempty"`
	Rows     int       `yaml:"rows" json:"rows,omitempty"`
	Elements titleList `yaml:"elements" json:"elements,omitempty"`
}

// titleList is a titles block's elements in the order written, which is
// the order they are drawn in, so it is never held as a map.
type titleList []namedTitle

type namedTitle struct {
	name  string
	title titleWire
}

type titleWire struct {
	Kind   string    `yaml:"kind" json:"kind"`
	Words  string    `yaml:"words" json:"words,omitempty"`
	Count  *ramp     `yaml:"count" json:"count,omitempty"`
	Format string    `yaml:"format" json:"format,omitempty"`
	Value  *ramp     `yaml:"value" json:"value,omitempty"`
	Left   *float64  `yaml:"left" json:"left,omitempty"`
	Top    *float64  `yaml:"top" json:"top,omitempty"`
	Right  *float64  `yaml:"right" json:"right,omitempty"`
	Bottom *float64  `yaml:"bottom" json:"bottom,omitempty"`
	Across *float64  `yaml:"across" json:"across,omitempty"`
	Down   *float64  `yaml:"down" json:"down,omitempty"`
	Align  string    `yaml:"align" json:"align,omitempty"`
	Ink    string    `yaml:"ink" json:"ink,omitempty"`
	Ground string    `yaml:"ground" json:"ground,omitempty"`
	Fill   string    `yaml:"fill" json:"fill,omitempty"`
	Empty  string    `yaml:"empty" json:"empty,omitempty"`
	From   *quantity `yaml:"from" json:"from,omitempty"`
	Until  *quantity `yaml:"until" json:"until,omitempty"`
}

// UnmarshalYAML reads the elements in the order the mapping writes them.
//
//ninja:vow waste yaml.v3 resolves this exported method by name
func (list *titleList) UnmarshalYAML(node *yaml.Node) error {
	for i := 0; i+1 < len(node.Content); i += halves {
		var title titleWire
		if err := node.Content[i+1].Decode(&title); err != nil {
			return err
		}
		*list = append(*list, namedTitle{node.Content[i].Value, title})
	}
	return nil
}

// MarshalJSON writes the elements as one object, in the order written.
// It cannot fail: an element is strings and finite numbers.
//
//ninja:vow waste encoding/json resolves this exported method by name
func (list titleList) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, each := range list {
		if i > nothing {
			out.WriteByte(',')
		}
		name, _ := json.Marshal(each.name)
		title, _ := json.Marshal(each.title)
		out.Write(name)
		out.WriteByte(':')
		out.Write(title)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

// ramp is a number that may change across the film: from, until start
// seconds, then evenly to to over so many seconds. A number written
// alone is from and to both, and is written back alone.
type ramp struct {
	From, To, Start, Over float64
	alone                 bool
}

type rampWire struct {
	From  float64 `yaml:"from" json:"from"`
	To    float64 `yaml:"to" json:"to"`
	Start float64 `yaml:"start" json:"start,omitempty"`
	Over  float64 `yaml:"over" json:"over,omitempty"`
}

// UnmarshalYAML reads a number alone or a block of from, to, start and
// over.
//
//ninja:vow waste yaml.v3 resolves this exported method by name
func (value *ramp) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		var number float64
		err := node.Decode(&number)
		*value = ramp{From: number, To: number, alone: true}
		return err
	}
	var block rampWire
	err := node.Decode(&block)
	*value = ramp{From: block.From, To: block.To, Start: block.Start,
		Over: block.Over}
	return err
}

// MarshalJSON writes the ramp as it was read: alone, or as a block.
//
//ninja:vow waste encoding/json resolves this exported method by name
func (value ramp) MarshalJSON() ([]byte, error) {
	if value.alone {
		return json.Marshal(value.From)
	}
	return json.Marshal(rampWire{From: value.From, To: value.To,
		Start: value.Start, Over: value.Over})
}

type soundWire struct {
	Tune    string    `yaml:"tune" json:"tune,omitempty"`
	Voicing string    `yaml:"voicing" json:"voicing,omitempty"`
	File    string    `yaml:"file" json:"file,omitempty"`
	From    *quantity `yaml:"from" json:"from,omitempty"`
	Score   string    `yaml:"score" json:"score,omitempty"`
}

type outputsWire struct {
	Picture string `yaml:"picture" json:"picture,omitempty"`
	Raw     string `yaml:"raw" json:"raw,omitempty"`
	Frames  string `yaml:"frames" json:"frames,omitempty"`
	Movie   string `yaml:"movie" json:"movie,omitempty"`
	Note    string `yaml:"note" json:"note,omitempty"`
}

type limitsWire struct {
	Memory *quantity `yaml:"memory" json:"memory,omitempty"`
	Time   *quantity `yaml:"time" json:"time,omitempty"`
}
