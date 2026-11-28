package set

import (
	"time"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

const (
	poseMostKeys    = 256
	poseMostProps   = 16
	poseMostSeconds = 86400
	poseMostOffset  = 100000
	poseMostDegrees = 360
	poseLeastScale  = 0.0001
	poseMostScale   = 100
)

func (reading reader) poseRecipe(name, block *yaml.Node) error {
	var kind string
	var spec manufacture.BioformClipSpec
	if err := reading.keys(block, map[string]step{
		"kind":      reading.word(&kind, manufacture.KindNames()...),
		"pose-clip": reading.poseClip(&spec),
	}, nil); err != nil {
		return err
	}
	if err := reading.all(block, "a pose recipe", "kind", "pose-clip"); err != nil {
		return err
	}
	row, err := manufacture.KindNamed(kind)
	if err != nil {
		return reading.refuse(block, "%v", err)
	}
	row, err = manufacture.WithBioformClip(row, spec)
	if err != nil {
		return reading.refuse(block, "%v", err)
	}
	reading.set.recipes[name.Value] = Recipe{kind: kind, profileKind: &row, poseClip: true}
	return nil
}

func (reading reader) poseClip(spec *manufacture.BioformClipSpec) step {
	return func(_, block *yaml.Node) error {
		var timing string
		if err := reading.keys(block, map[string]step{
			"timing": reading.word(&timing, "linear", "smooth"),
			"keys": func(_, value *yaml.Node) error {
				if value.Kind != yaml.SequenceNode || len(value.Content) == 0 || len(value.Content) > poseMostKeys {
					return reading.refuse(value, "pose keys take a list of 1 to %d entries", poseMostKeys)
				}
				for _, node := range value.Content {
					key, err := reading.poseKey(node)
					if err != nil {
						return err
					}
					spec.Keys = append(spec.Keys, key)
				}
				return nil
			},
			"attachments": func(_, value *yaml.Node) error {
				return reading.poseProps(value, spec)
			},
		}, nil); err != nil {
			return err
		}
		spec.Smooth = timing == "smooth"
		return reading.all(block, "a pose clip", "keys")
	}
}

var poseAxes = [...]maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis}

func (reading reader) poseKey(block *yaml.Node) (skeleton.PoseKey, error) {
	var spec skeleton.PoseKey
	var seconds float64
	var offset, root [3]float64
	if err := reading.keys(block, map[string]step{
		"at":          reading.quantity(&seconds, "s", 0, poseMostSeconds),
		"root-offset": reading.three(&offset, -poseMostOffset, poseMostOffset),
		"root-turns":  reading.three(&root, -poseMostDegrees, poseMostDegrees),
		"turns": func(_, value *yaml.Node) error {
			return reading.each(value, func(name, block *yaml.Node) error {
				if name.ShortTag() != "!!str" || name.Value == "" {
					return reading.refuse(name, "a turn needs a named bone")
				}
				var degrees [3]float64
				if err := reading.three(&degrees, -poseMostDegrees, poseMostDegrees)(name, block); err != nil {
					return err
				}
				spec.Pose = turnBone(spec.Pose, name.Value, degrees)
				return nil
			})
		},
	}, nil); err != nil {
		return spec, err
	}
	spec.At = time.Duration(seconds * float64(time.Second))
	translation := maths.V(offset[0], offset[1], offset[2])
	spec.Pose = spec.Pose.Carried(skeleton.NewFrame(poseOrientation(root), translation))
	return spec, reading.all(block, "a pose key", "at")
}

func turnBone(pose skeleton.Pose, name string, degrees [3]float64) skeleton.Pose {
	if degrees == [3]float64{} {
		return pose.With(name, skeleton.Turn{})
	}
	for i, angle := range degrees {
		if angle != 0 {
			pose = pose.Compose(name, skeleton.NewTurn(poseAxes[i], angle*maths.Degree))
		}
	}
	return pose
}

func poseOrientation(degrees [3]float64) maths.Orientation {
	var rotation maths.Orientation
	for i, angle := range degrees {
		if angle != 0 {
			rotation = maths.Rotation(poseAxes[i], angle*maths.Degree).Then(rotation)
		}
	}
	return rotation
}

func (reading reader) poseProps(value *yaml.Node, spec *manufacture.BioformClipSpec) error {
	if len(value.Content)/2 > poseMostProps {
		return reading.refuse(value, "a clip takes at most %d attachments", poseMostProps)
	}
	return reading.each(value, func(name, block *yaml.Node) error {
		if name.ShortTag() != "!!str" || name.Value == "" {
			return reading.refuse(name, "an attachment needs a name")
		}
		prop := manufacture.BoneProp{Name: name.Value}
		var offset, turns [3]float64
		if err := reading.keys(block, map[string]step{
			"bone":   reading.text(&prop.Bone),
			"kind":   reading.word(&prop.Kind, manufacture.KindNames()...),
			"offset": reading.three(&offset, -poseMostOffset, poseMostOffset),
			"turns":  reading.three(&turns, -poseMostDegrees, poseMostDegrees),
			"scale":  reading.number(&prop.Placement.Scale, poseLeastScale, poseMostScale),
		}, nil); err != nil {
			return err
		}
		if err := reading.all(block, "an attachment", "bone", "kind"); err != nil {
			return err
		}
		prop.Placement.Origin = maths.V(offset[0], offset[1], offset[2])
		prop.Placement.Orientation = poseOrientation(turns)
		spec.Props = append(spec.Props, prop)
		return nil
	})
}
