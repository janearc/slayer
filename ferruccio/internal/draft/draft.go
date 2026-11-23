// Package draft edits one Slayer job without discarding its source comments.
package draft

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/yamlfile"
	"gopkg.in/yaml.v3"
)

// Options supplies a document and its local, pinned schema.
type Options struct {
	Name             string
	Source, Schema   []byte
	Kind, SchemaName string
}

// Change identifies one scalar in the existing document.
type Change struct {
	Path       []string
	Value, Tag string
}

// Document owns the editable source, including invalid work in progress.
type Document struct {
	name             string
	source, saved    []byte
	schema           *yamlfile.Schema
	kind, schemaName string
}

// New checks construction inputs; invalid YAML remains editable.
func New(options Options) (*Document, error) {
	if strings.TrimSpace(options.Name) == "" {
		return nil, fmt.Errorf("a document needs a name")
	}
	if options.Kind == "legacy-set" {
		return &Document{name: options.Name, source: bytes.Clone(options.Source), saved: bytes.Clone(options.Source), kind: options.Kind}, nil
	}
	if options.Kind == "" {
		options.Kind = "job"
	}
	if options.SchemaName == "" {
		options.SchemaName = "https://github.com/janearc/slayer/schemas/job-v1.json"
	}
	schema, err := yamlfile.NewSchema(options.SchemaName, options.Schema)
	if err != nil {
		return nil, err
	}
	return &Document{name: options.Name, source: bytes.Clone(options.Source), saved: bytes.Clone(options.Source), schema: schema, kind: options.Kind, schemaName: options.SchemaName}, nil
}

// Text returns the current source, including an invalid edit.
func (document *Document) Text() string { return string(document.source) }

// Name is the document's displayed file name.
func (document *Document) Name() string { return document.name }

// Replace records an editor change without silently validating it away.
func (document *Document) Replace(source string) { document.source = []byte(source) }

// Check validates structure through Puzzlemouth and the pinned schema.
func (document *Document) Check() error {
	_, err := document.Tree()
	return err
}

// Set applies a scalar edit only when the complete candidate is valid.
func (document *Document) Set(change Change) error {
	if len(change.Path) == 0 {
		return fmt.Errorf("an edit needs a document path")
	}
	node, err := document.Tree()
	if err != nil {
		return err
	}
	target, err := scalar(node.Content[0], change.Path)
	if err != nil {
		return err
	}
	tag := change.Tag
	if tag == "" {
		tag = "!!str"
	}
	if tag != "!!str" && tag != "!!int" && tag != "!!float" {
		return fmt.Errorf("unsupported scalar type")
	}
	target.Kind, target.Tag, target.Value = yaml.ScalarNode, tag, change.Value
	target.Content = nil
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return err
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}

// Value reads a scalar for an inspector control.
func (document *Document) Value(path ...string) (string, error) {
	node, err := document.Tree()
	if err != nil {
		return "", err
	}
	current := node.Content[0]
	for _, key := range path {
		found := member(current, key)
		if found == nil {
			return "", fmt.Errorf("field is not specified")
		}
		current = found
	}
	if current.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("field is not a scalar")
	}
	return current.Value, nil
}

// Dirty reports whether the source differs from its saved state.
func (document *Document) Dirty() bool { return !bytes.Equal(document.source, document.saved) }

// Save writes the draft atomically, including invalid work the user may repair.
func (document *Document) Save(path string) error {
	if err := atomicWrite(path, document.source); err != nil {
		return err
	}
	document.saved = bytes.Clone(document.source)
	document.name = filepath.Base(path)
	return nil
}

// CanNavigate reports whether the document has camera controls the editor can preserve.
func (document *Document) CanNavigate() bool {
	if document.kind == "catalogue" {
		return true
	}
	if document.kind != "set" && document.kind != "legacy-set" {
		return false
	}
	_, err := document.simpleSetCamera()
	return err == nil
}

const (
	// minimumWide is the narrowest field of view, in degrees, that zoom works in.
	minimumWide = 5
	// maximumWide is the widest field of view, in degrees, that zoom works in.
	maximumWide = 120
	// wideGrains is how many parts of a degree a zoomed angle is written
	// to: a millionth, far finer than a picture shows.
	wideGrains = 1e6
)

// zoomed is the field of view after a zoom, held to the range zoom works in
// and to a millionth of a degree, so the angle written is the one meant: 50
// by 1.1 is written 55, where the multiplication gives 55.00000000000001.
func zoomed(wide, factor float64) float64 {
	held := math.Max(minimumWide, math.Min(maximumWide, wide*factor))
	return math.Round(held*wideGrains) / wideGrains
}

// writtenWide is a catalogue's wide angle as written, read without its
// schema, so an angle the schema refuses can be named in the zoom's own
// words. it is false where none is written or what is written is no number.
func (document *Document) writtenWide() (float64, bool) {
	var plain map[string]any
	if err := yamlfile.Read(document.name, document.source, &plain); err != nil {
		return 0, false
	}
	switch wide := plain["wide"].(type) {
	case float64:
		return wide, true
	case int:
		return float64(wide), true
	}
	return 0, false
}

// checkZoomRange refuses to zoom an authored field of view outside the range
// zoom works in, so that value is left as written and never rewritten.
func checkZoomRange(wide float64, name string) error {
	if wide < minimumWide || wide > maximumWide {
		return fmt.Errorf("%s is %g degrees; zoom works only from %d to %d degrees, so it is left as written", name, wide, minimumWide, maximumWide)
	}
	return nil
}

// Navigate applies a supported camera orbit and zoom as one valid YAML edit.
func (document *Document) Navigate(yawDelta, pitchDelta, zoomFactor float64) error {
	if math.IsNaN(yawDelta) || math.IsInf(yawDelta, 0) ||
		math.IsNaN(pitchDelta) || math.IsInf(pitchDelta, 0) ||
		math.IsNaN(zoomFactor) || math.IsInf(zoomFactor, 0) || zoomFactor < 0 {
		return fmt.Errorf("navigation deltas must be finite and zoom factor cannot be negative")
	}
	if document.kind == "catalogue" {
		return document.navigateCatalogue(yawDelta, pitchDelta, zoomFactor)
	}
	if document.kind == "set" || document.kind == "legacy-set" {
		return document.navigateSet(yawDelta, pitchDelta, zoomFactor)
	}
	return fmt.Errorf("camera navigation is not available for this document")
}

// navigateCatalogue edits the catalogue's named orbit fields and wide angle.
// the catalogue's schema holds wide to the range zoom works in, so a
// catalogue that reads has an angle zoom can change; one that does not read
// for its angle is refused in the zoom's words, not the schema's.
func (document *Document) navigateCatalogue(yawDelta, pitchDelta, zoomFactor float64) error {
	tree, err := document.Tree()
	if err != nil {
		if wide, written := document.writtenWide(); written && zoomFactor > 0 {
			if refusal := checkZoomRange(wide, "wide"); refusal != nil {
				return refusal
			}
		}
		return err
	}
	root := tree.Content[0]
	orbit := member(root, "orbit")
	readNumber := func(node *yaml.Node, name string, fallback float64) (float64, error) {
		if node == nil {
			return fallback, nil
		}
		if node.Kind != yaml.ScalarNode {
			return 0, fmt.Errorf("%s must be a number", name)
		}
		value, err := strconv.ParseFloat(node.Value, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, fmt.Errorf("%s must be a finite number", name)
		}
		return value, nil
	}
	yawNode, pitchNode := member(orbit, "yaw"), member(orbit, "pitch")
	yaw, err := readNumber(yawNode, "orbit.yaw", 0)
	if err != nil {
		return err
	}
	pitch, err := readNumber(pitchNode, "orbit.pitch", 0)
	if err != nil {
		return err
	}
	wideNode := member(root, "wide")
	wide, err := readNumber(wideNode, "wide", 50)
	if err != nil {
		return err
	}
	changed := false
	angleChanged := yawDelta != 0 || pitchDelta != 0
	if angleChanged {
		newYaw := math.Mod(math.Mod(yaw, 360)+math.Mod(yawDelta, 360), 360)
		if newYaw > 180 {
			newYaw -= 360
		} else if newYaw < -180 {
			newYaw += 360
		}
		newPitch := math.Max(-80, math.Min(80, pitch+pitchDelta))
		if newYaw != yaw || newPitch != pitch || orbit == nil {
			if orbit == nil {
				orbit = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
				root.Content = append(root.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "orbit"}, orbit)
				yawNode = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float"}
				pitchNode = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float"}
				orbit.Content = []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "yaw"}, yawNode,
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "pitch"}, pitchNode,
				}
			}
			setNumber(yawNode, newYaw)
			setNumber(pitchNode, newPitch)
			changed = true
		}
	}
	if zoomFactor > 0 {
		newWide := zoomed(wide, zoomFactor)
		if newWide != wide || wideNode == nil {
			if wideNode == nil {
				wideNode = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float"}
				root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "wide"}, wideNode)
			}
			setNumber(wideNode, newWide)
			changed = true
		}
	}
	if !changed {
		return nil
	}
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
		return fmt.Errorf("navigation edit would make invalid catalogue settings: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}

// setCameraNodes names the first set camera's simple world-space pose.
type setCameraNodes struct {
	root, at, lookAt, wide *yaml.Node
}

// simpleSetCamera finds a camera described by a fixed point and target,
// refusing rigs and paths whose semantics these controls cannot preserve.
func (document *Document) simpleSetCamera() (*setCameraNodes, error) {
	tree, err := document.Tree()
	if err != nil {
		return nil, err
	}
	root := tree.Content[0]
	scene := member(root, "scene")
	cameras := member(scene, "cameras")
	if cameras == nil || cameras.Kind != yaml.MappingNode || len(cameras.Content) < 2 {
		return nil, fmt.Errorf("scene.cameras needs a camera")
	}
	camera := cameras.Content[1]
	if camera.Kind != yaml.MappingNode || member(camera, "moves") != nil {
		return nil, fmt.Errorf("the first camera must be fixed, without a moves path")
	}
	if eye := member(camera, "eye"); eye != nil &&
		(eye.Kind != yaml.ScalarNode || eye.Value != "human") {
		return nil, fmt.Errorf("the first camera must use the human eye")
	}
	location, orientation := member(camera, "location"), member(camera, "orientation")
	if location == nil || orientation == nil ||
		member(location, "relative") != nil || member(location, "loc_relative") != nil ||
		member(orientation, "point_to") != nil || member(orientation, "alpha") != nil {
		return nil, fmt.Errorf("the first camera needs a fixed at and look_at pose")
	}
	at, lookAt := member(location, "at"), member(orientation, "look_at")
	if _, err := readVector(at, "location.at"); err != nil {
		return nil, err
	}
	if _, err := readVector(lookAt, "orientation.look_at"); err != nil {
		return nil, err
	}
	film := member(scene, "film")
	for _, rigKey := range []string{"frame", "lens", "shutter", "glimpses"} {
		if member(film, rigKey) != nil {
			return nil, fmt.Errorf("scene.film.%s rig cannot be controlled", rigKey)
		}
	}
	wide := member(camera, "wide")
	if wide == nil {
		wide = member(film, "wide")
	}
	return &setCameraNodes{root: root, at: at, lookAt: lookAt, wide: wide}, nil
}

// navigateSet rotates the first fixed set camera in east, north, up space.
func (document *Document) navigateSet(yawDelta, pitchDelta, zoomFactor float64) error {
	nodes, err := document.simpleSetCamera()
	if err != nil {
		return err
	}
	at, err := readVector(nodes.at, "location.at")
	if err != nil {
		return err
	}
	target, err := readVector(nodes.lookAt, "orientation.look_at")
	if err != nil {
		return err
	}
	var wide float64
	if zoomFactor > 0 {
		wide, err = readNumber(nodes.wide, "scene.film.wide")
		if err != nil {
			return err
		}
		if err := checkZoomRange(wide, "scene.film.wide"); err != nil {
			return err
		}
	}
	if yawDelta != 0 || pitchDelta != 0 {
		at, err = rotateCamera(at, target, yawDelta, pitchDelta)
		if err != nil {
			return err
		}
		writeVector(nodes.at, at)
	}
	if zoomFactor > 0 {
		setNumber(nodes.wide, zoomed(wide, zoomFactor))
	}
	if yawDelta == 0 && pitchDelta == 0 && zoomFactor == 0 {
		return nil
	}
	var encoded bytes.Buffer
	encoder := yaml.NewEncoder(&encoded)
	encoder.SetIndent(2)
	if err := encoder.Encode(&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{nodes.root}}); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if _, err := document.read(encoded.Bytes()); err != nil {
		return fmt.Errorf("navigation edit would make invalid set YAML: %w", err)
	}
	document.source = bytes.Clone(encoded.Bytes())
	return nil
}

// readVector accepts exactly three finite numeric YAML scalars.
func readVector(node *yaml.Node, name string) ([3]float64, error) {
	var vector [3]float64
	if node == nil || node.Kind != yaml.SequenceNode || len(node.Content) != len(vector) {
		return vector, fmt.Errorf("%s must be [east, north, up]", name)
	}
	for index, value := range node.Content {
		number, err := readNumber(value, name)
		if err != nil {
			return vector, err
		}
		vector[index] = number
	}
	return vector, nil
}

// readNumber reads a finite YAML integer or floating-point scalar.
func readNumber(node *yaml.Node, name string) (float64, error) {
	if node == nil || node.Kind != yaml.ScalarNode ||
		(node.Tag != "!!int" && node.Tag != "!!float") {
		return 0, fmt.Errorf("%s must be a number", name)
	}
	value, err := strconv.ParseFloat(node.Value, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%s must be a finite number", name)
	}
	return value, nil
}

// PreviewCamera applies Navigate's canonical orbit to a transient ENU pose.
// Coordinates are east, north, up in metres; yaw and pitch are degrees.
// It changes no document text or undo history.
func PreviewCamera(at, target [3]float64, yaw, pitch float64) ([3]float64, error) {
	for _, value := range []float64{at[0], at[1], at[2], target[0], target[1], target[2], yaw, pitch} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return at, fmt.Errorf("camera preview coordinates and angles must be finite")
		}
	}
	return rotateCamera(at, target, yaw, pitch)
}

// rotateCamera applies yaw around up and pitch above the east-north plane.
func rotateCamera(at, target [3]float64, yawDelta, pitchDelta float64) ([3]float64, error) {
	dx, dy, dz := at[0]-target[0], at[1]-target[1], at[2]-target[2]
	radius := math.Hypot(math.Hypot(dx, dy), dz)
	if radius == 0 || math.IsInf(radius, 0) {
		return at, fmt.Errorf("camera position must be a finite distance from its look_at target")
	}
	yaw := math.Atan2(dy, dx) + math.Mod(yawDelta, 360)*math.Pi/180
	pitchDelta = math.Max(-160, math.Min(160, pitchDelta))
	pitch := math.Atan2(dz, math.Hypot(dx, dy)) + pitchDelta*math.Pi/180
	pitch = math.Max(-80*math.Pi/180, math.Min(80*math.Pi/180, pitch))
	horizontal := radius * math.Cos(pitch)
	next := [3]float64{
		target[0] + horizontal*math.Cos(yaw),
		target[1] + horizontal*math.Sin(yaw),
		target[2] + radius*math.Sin(pitch),
	}
	for _, value := range next {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return at, fmt.Errorf("camera navigation produced a non-finite position")
		}
	}
	return next, nil
}

// writeVector updates numeric coordinates without replacing their YAML nodes.
func writeVector(node *yaml.Node, vector [3]float64) {
	for index, value := range vector {
		setNumber(node.Content[index], value)
	}
}

// setNumber updates a YAML scalar while retaining its comments and position.
func setNumber(node *yaml.Node, value float64) {
	node.Kind = yaml.ScalarNode
	node.Tag = "!!float"
	node.Value = strconv.FormatFloat(value, 'f', -1, 64)
	node.Content = nil
}

// member reads an explicit mapping key from the schema-checked syntax tree.
func member(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// scalar prepares an optional path without mutating the stored document.
func scalar(node *yaml.Node, path []string) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("the parent is not a mapping")
	}
	child := member(node, path[0])
	if child == nil {
		child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: path[0]}, child)
	}
	if len(path) == 1 {
		return child, nil
	}
	return scalar(child, path[1:])
}

// atomicWrite changes the target only after the new contents are closed safely.
func atomicWrite(path string, source []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".atelier-save-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(source); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
