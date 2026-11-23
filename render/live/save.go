package live

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// edits are what `s` writes back into the job file; an empty one is left
// as the file has it.
type edits struct {
	moment   time.Duration
	said     bool // whether the moment is written
	camera   string
	exposure float64 // nought writes none
}

// How a number is written: as few digits as read back the same, of a
// float64. Both are defined by strconv.
const (
	shortest    = -1
	float64Bits = 64
)

// indent is how far the job file is indented when written back. Picked
// 2026-11-12: the example jobs in doc/examples/jobs/ indent so.
const indent = 2

// withEdits is the job file's YAML with the edits made in it, its
// comments and the rest of it kept. A file that is not one mapping is
// refused.
func withEdits(raw []byte, change edits) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	if len(document.Content) != one ||
		document.Content[none].Kind != yaml.MappingNode {
		return nil, errors.New("the job file is not one mapping of " +
			"keys; write it back by hand")
	}
	top := document.Content[none]
	if change.said {
		momentSeconds := change.moment.Seconds()
		set(top, "at", quantity(momentSeconds, "s"))
	}
	if change.camera != "" {
		set(top, "camera", cameraNamed(top, change.camera))
	}
	if change.exposure > none {
		film := valueOf(top, "film")
		if film == nil || film.Kind != yaml.MappingNode {
			film = flow()
			set(top, "film", film)
		}
		set(film, "exposure", scalar(change.exposure))
	}
	var written bytes.Buffer
	encoder := yaml.NewEncoder(&written)
	encoder.SetIndent(indent)
	if err := encoder.Encode(&document); err != nil {
		return nil, err
	}
	return written.Bytes(), encoder.Close()
}

// valueOf is a mapping's value for a key, or nil.
func valueOf(mapping *yaml.Node, key string) *yaml.Node {
	for i := none; i+one < len(mapping.Content); i += pair {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+one]
		}
	}
	return nil
}

// set gives a mapping's key a value, in its place if it has one and at
// the end if it has none.
func set(mapping *yaml.Node, key string, value *yaml.Node) {
	for i := none; i+one < len(mapping.Content); i += pair {
		if mapping.Content[i].Value == key {
			value.HeadComment = mapping.Content[i+one].HeadComment
			value.LineComment = mapping.Content[i+one].LineComment
			mapping.Content[i+one] = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
}

// cameraNamed is a camera by name, keeping the zoom the job's camera
// says.
func cameraNamed(top *yaml.Node, name string) *yaml.Node {
	lens := flow()
	set(lens, "name", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str",
		Value: name})
	if was := valueOf(top, "camera"); was != nil {
		if zoom := valueOf(was, "zoom"); zoom != nil {
			set(lens, "zoom", zoom)
		}
	}
	return lens
}

// quantity is a quantity as a job writes one: {value: N, unit: U}.
func quantity(value float64, unit string) *yaml.Node {
	made := flow()
	set(made, "value", scalar(value))
	set(made, "unit", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str",
		Value: unit})
	return made
}

// flow is an empty mapping written on one line.
func flow() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map",
		Style: yaml.FlowStyle}
}

// scalar is a number as YAML writes it, as short as it reads back.
func scalar(value float64) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode,
		Value: strconv.FormatFloat(value, 'g', shortest, float64Bits)}
}

// save writes the moment, camera and exposure the keys chose into the
// job file, beside it and renamed into place with its mode kept, then
// reads it again as the job shown. The editor has the file open, so the
// status row says to reload it.
func (watching *Watcher) save() Change {
	change := watching.edits()
	if change == (edits{}) {
		watching.foot.note("nothing to write: the keys have not moved " +
			"the moment, the camera or the exposure")
		return Unchanged
	}
	if err := writeEdits(watching.given.Path, change); err != nil {
		watching.foot.note(err.Error())
		return Unchanged
	}
	watching.view.camera, watching.view.said = "", false
	watching.view.thirds = none
	watching.words = "saved; reload the job in the editor"
	return watching.reloaded()
}

// edits is what the keys chose that the file can be told: a still's
// moment, a camera by name, and the exposure scaled by the thirds of a
// stop, which needs the exposure the scene was staged at.
func (watching *Watcher) edits() edits {
	change := edits{camera: watching.view.camera}
	if watching.view.said && watching.still() {
		change.moment, change.said = watching.view.moment, true
	}
	if watching.view.thirds == none {
		return change
	}
	if watching.facts.exposure <= none {
		watching.foot.note("the scene says no exposure to scale; " +
			"write film.exposure in the job by hand")
		return change
	}
	change.exposure = watching.facts.exposure * gainOf(watching.view.thirds)
	return change
}

// writeEdits makes the edits in the file at path.
func writeEdits(path string, change edits) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	written, err := withEdits(raw, change)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return replaced(path, written, info.Mode().Perm())
}

// reloaded reads the job file just written as the job shown, so its own
// change is not seen again as one, and is what that costs against the
// job shown before it.
func (watching *Watcher) reloaded() Change {
	read, err := watching.given.Read(watching.given.Path)
	if err != nil {
		watching.foot.troubled(err.Error())
		return Unchanged
	}
	read.Key()
	was := watching.wanted
	watching.base = read
	watching.stamps = stampsOf(watching.followed())
	watching.relaid()
	return Compare(was, watching.wanted)
}
