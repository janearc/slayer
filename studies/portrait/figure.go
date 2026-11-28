package main

import (
	"fmt"
	"github.com/janearc/slayer/mesh"

	"github.com/janearc/slayer/bioforms/assembly"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// figureSpec is independent of camera, lighting and street geometry. Relative
// breadth is measured about the centre line in each named anatomical band.
type figureSpec struct {
	CrownFall               float64
	SoftNormals             bool
	Height, Shoulders, Hips float64
	Skin                    paint
	Hair                    bool
	Swept                   bool
	Curls                   bool
	Rooted                  *follicleRegion
	Layered                 bool
	Shape                   proportionProfile
	Gaze                    *gazeSolution
	Decorations             []assembly.Part
	// JointParts contain rest-space geometry explicitly attached to named rigid anchors.
	JointParts []jointPart
	Anchors    map[string]func(maths.Vec) maths.Vec
}

type jointPart struct {
	joint   string
	surface assembly.Part
}

func (f figureSpec) jointPoint(name string, p maths.Vec) (maths.Vec, error) {
	anchor, ok := f.Anchors[name]
	if !ok {
		return maths.Vec{}, fmt.Errorf("figure has no joint %q", name)
	}
	q := f.fit(p, anchor(p))
	if f.Gaze != nil && name == "head" {
		q = f.Gaze.head(q, 1)
	}
	return q, nil
}
func standardFigure() figureSpec {
	return figureSpec{Height: 1, Shoulders: 1, Hips: 1, Skin: skin, Hair: true}
}
func (f figureSpec) fit(p, q maths.Vec) maths.Vec {
	shoulder := maths.Smooth(1.08, 1.30, p.Y) * (1 - maths.Smooth(1.34, 1.44, p.Y))
	hip := maths.Smooth(.56, .81, p.Y) * (1 - maths.Smooth(.91, 1.08, p.Y))
	q.X *= 1 + shoulder*(f.Shoulders*(1+f.Shape.shoulder)-1) + hip*(f.Hips*(1+f.Shape.hip)-1)
	q.Y *= f.Height
	q = f.Shape.fit(p, q)
	return q
}
func (f figureSpec) build(body, eyes geom, carry func(maths.Vec) maths.Vec) ([]thing.Thing, error) {
	parts := []assembly.Part{}
	add := func(g geom, m *thing.Material) {
		parts = append(parts, assembly.Part{Rest: g.pts, Faces: g.faces, Material: m})
	}
	add(body, &thing.Material{Name: "skin", Texture: f.Skin, Gloss: .09})
	add(eyes, &thing.Material{Name: "eyes", Texture: paint(eye), Gloss: .85, Index: 1.376})
	add(lashes(body, eyes), &thing.Material{Name: "lash line", Colour: rgb(.034, .018, .012), Gloss: .10})
	if f.Hair && f.Rooted != nil {
		parts = append(parts, rootedGroom(body, *f.Rooted, f.CrownFall)...)
	} else if f.Hair && f.Curls {
		parts = append(parts, curls(body)...)
	} else if f.Hair && f.Swept {
		parts = append(parts, coiffure(body, f.Layered)...)
	} else if f.Hair {
		add(bob(), &thing.Material{Name: "auburn bob", Texture: paint(hairPaint), Gloss: .25})
		add(locks(), &thing.Material{Name: "combed locks", Texture: paint(hairPaint), Gloss: .32})
	}
	headPartsEnd := len(parts)
	parts = append(parts, f.Decorations...)
	for i := range parts {
		move := parts[i].Carry
		if move == nil {
			move = carry
		}
		index := i
		parts[i].Carry = func(p maths.Vec) maths.Vec {
			q := f.fit(p, move(p))
			if f.Gaze == nil {
				return q
			}
			if index == 1 {
				side := 0
				if p.X > 0 {
					side = 1
				}
				return f.Gaze.eye(q, side)
			}
			weight := maths.Smooth(1.378, 1.448, p.Y)
			if index >= 2 && index < headPartsEnd {
				weight = 1
			}
			return f.Gaze.head(q, weight)
		}
	}
	for _, attachment := range f.JointParts {
		if _, err := f.jointPoint(attachment.joint, maths.Vec{}); err != nil {
			return nil, err
		}
		part := attachment.surface
		part.Carry = func(p maths.Vec) maths.Vec { q, _ := f.jointPoint(attachment.joint, p); return q }
		parts = append(parts, part)
	}
	objects, err := assembly.Build(parts, carry)
	if err == nil && f.SoftNormals {
		points := objects[0].Shape.(*mesh.Mesh).Points()
		normals := skinNormals(points, body.pts, body.faces)
		objects[0].Shape, err = mesh.NewWithRestPoints(points, body.pts, body.faces, normals, true)
	}
	return objects, err
}
func figureSkin(p maths.Vec, c maths.Light) maths.Light {
	x := math.Abs(p.X)
	front := maths.Smooth(.048, .104, p.Z)
	// Mottling changes gently across the trunk and limbs. The rest-space masks
	// stay on the surface under the full-body pose and proportion controls.
	c = c.Scale(1 + .037*stone(p, 12))
	warmth := gauss((p.Y-.47)/.065)*gauss((x-.088)/.07) + gauss((x-.435)/.05)*gauss((p.Y-1.29)/.046)
	c = mix(c, rgb(.49, .224, .166), .24*warmth)
	disc := math.Hypot((x-.076)/.018, (p.Y-1.197)/.016)
	c = mix(c, rgb(.40, .195, .145), .52*front*(1-maths.Smooth(.55, 1.16, disc)))
	navel := gauss(p.X/.007) * gauss((p.Y-.971)/.008) * front
	return c.Scale(1 - .16*navel)
}
