package hair

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

type fibres struct {
	points, paint, normals []maths.Vec
	faces                  [][3]int
	optical                *mesh.FibreField
}

func (group growth) at(points, normals []maths.Vec, faces [][3]int, skin *mesh.Mesh) (thing.Thing, error) {
	var cloth fibres
	if group.material.Fibre != nil {
		cloth.optical = new(mesh.FibreField)
	}
	var samples [64]maths.Vec
	for _, root := range group.roots {
		face := faces[root.face]
		basis := edges(points, face)
		posed := locate(points, normals, root.face, face, root.weights, basis)
		direction := basis.flow(root.flow)
		direction = direction.Sub(posed.Normal.Scale(direction.Dot(posed.Normal)))
		if direction.Len() < leastVector || !finite(direction.Len()) {
			direction = posed.Normal.Cross(basis.area().Cross(basis.first)).Unit()
		}
		along := direction.Unit()
		frame := shaftFrame{along, posed.Normal, posed.Normal.Cross(along).Unit()}
		adjust := root.strand.pinOffsets(posed.Point, frame, points, normals)
		var path []maths.Vec
		if rings := root.strand.Course.rings; len(rings) > 0 {
			path = samples[:len(rings)]
			for i, ring := range rings {
				path[i] = ring.posed(points, normals)
			}
		}
		cloth.add(root, posed.Point, frame, group.segments, group.sides, skin, group.skinGap, adjust, path)
		if cloth.optical != nil {
			cloth.optical.FaceEnds = append(cloth.optical.FaceEnds, len(cloth.faces))
		}
	}
	made, err := cloth.mesh()
	if err != nil {
		return thing.Thing{}, fmt.Errorf("hair %q: %w", group.material.Name, err)
	}
	return thing.Thing{Shape: made, Material: &group.material, Motion: thing.Still(maths.Local())}, nil
}

func (cloth fibres) mesh() (*mesh.Mesh, error) {
	if cloth.optical != nil {
		return mesh.NewWithFibres(cloth.points, cloth.paint, cloth.faces, cloth.normals, *cloth.optical)
	}
	return mesh.NewWithRestPoints(cloth.points, cloth.paint, cloth.faces, cloth.normals, true)
}

// heading gives the unit tangent along an authored shaft in its posed
// start frame. Sampling at step midpoints keeps the polygonal path length
// independent of how much the strand turns.
func (strand Strand) heading(t float64, along, side, normal maths.Vec) maths.Vec {
	angle := strand.Sweep*t + strand.Wave*math.Sin(2*math.Pi*t)
	lift := strand.RootLift + (strand.TipLift-strand.RootLift)*maths.Smooth(0, 1, t)
	return along.Scale(math.Cos(angle) * math.Cos(lift)).
		Add(side.Scale(math.Sin(angle) * math.Cos(lift))).
		Add(normal.Scale(math.Sin(lift)))
}

// add grows a capped, tapered shaft from a vertex on the skin. Intermediate
// cross sections follow a transported frame, avoiding a flip when the shaft
// approaches the skin normal.
func (cloth *fibres) add(root follicle, origin maths.Vec, frame shaftFrame,
	steps, sides int, skin *mesh.Mesh, gap float64, adjust guideOffsets, path []maths.Vec,
) {
	start := len(cloth.points)
	site := root.paint
	strand := root.strand
	centre := origin
	lastTangent := strand.heading(0, frame.along, frame.side, frame.up)
	radial := frame.side
	guided := strand.Guide.active()
	followsCourse := len(path) != 0
	tracksSkin := gap > 0 || strand.SkinFollow > 0
	guideEnd := strand.GuideEnd
	if guideEnd == 0 {
		guideEnd = 1
	}
	if guided {
		lead := frame.out(strand.Guide.Lead)
		lastTangent = lead.Unit()
		if adjust.lead != (maths.Vec{}) {
			pinned := lead.Add(adjust.lead.Scale(1 / strand.Length)).Unit()
			if pinned != (maths.Vec{}) {
				lastTangent = pinned
				if axis := frame.up.Cross(pinned).Unit(); axis != (maths.Vec{}) {
					radial = axis
				}
			}
		}
	}
	if followsCourse {
		direction := path[0].Sub(origin).Unit()
		if direction != (maths.Vec{}) {
			lastTangent = direction
			if axis := frame.up.Cross(direction).Unit(); axis != (maths.Vec{}) {
				radial = axis
			}
		}
	}
	cloth.vertex(centre, site, lastTangent.Neg(), centre, lastTangent)
	for step := 1; step <= steps; step++ {
		t := float64(step) / float64(steps)
		middle := (float64(step) - .5) / float64(steps)
		tangent := strand.heading(middle, frame.along, frame.side, frame.up)
		next := centre.Add(tangent.Scale(strand.Length / float64(steps)))
		progress := t * guideEnd
		if followsCourse {
			next = path[step-1]
		} else if guided {
			next = origin.Add(frame.out(strand.Guide.point(progress)).Scale(strand.Length))
			if adjust != (guideOffsets{}) {
				next = next.Add(adjust.at(progress))
			}
		}
		if tracksSkin {
			lift := next.Sub(origin).Dot(frame.up)
			next = fitSkin(skin, next, max(gap, strand.Radius), lift, strand.SkinFollow*strand.Guide.retaining(progress))
		}
		if guided || followsCourse || tracksSkin {
			if travel := next.Sub(centre).Unit(); travel != (maths.Vec{}) {
				tangent = travel
			}
		}
		centre = next
		if step == steps {
			tip := cloth.vertex(centre, site, tangent, centre, tangent)
			last := tip - sides
			for k := range sides {
				next := (k + 1) % sides
				cloth.faces = append(cloth.faces, [3]int{last + k, last + next, tip})
			}
			break
		}
		radial = transport(radial, lastTangent, tangent)
		lastTangent = tangent
		binormal := tangent.Cross(radial).Unit()
		radius := strand.Radius * strand.Taper.scale(t)
		row := len(cloth.points)
		for k := range sides {
			angle := 2 * math.Pi * float64(k) / float64(sides)
			n := radial.Scale(math.Cos(angle)).Add(binormal.Scale(math.Sin(angle)))
			cloth.vertex(centre.Add(n.Scale(radius)), site, n, centre, tangent)
			next := (k + 1) % sides
			if step == 1 {
				cloth.faces = append(cloth.faces, [3]int{start, row + next, row + k})
				continue
			}
			last := row - sides
			cloth.faces = append(cloth.faces, [3]int{last + k, last + next, row + k},
				[3]int{last + next, row + next, row + k})
		}
	}
}

// fitSkin blends the lift of a ring centre against the nearest oriented
// face while leaving the follicle on its original skin triangle.
func fitSkin(skin *mesh.Mesh, point maths.Vec, gap, lift, follow float64) maths.Vec {
	meet, ok := skin.SurfaceNear(point)
	if !ok {
		return point
	}
	above := point.Sub(meet.Point).Dot(meet.Normal)
	wanted := max(gap, above+follow*(lift-above))
	return point.Add(meet.Normal.Scale(wanted - above))
}

// transport uses the shortest turn from one shaft tangent to the next.
func transport(axis, previous, tangent maths.Vec) maths.Vec {
	turn := previous.Cross(tangent)
	amount := turn.Len()
	if amount < leastVector {
		return axis
	}
	rotation := maths.Rotation(turn.Scale(1/amount), math.Atan2(amount, previous.Dot(tangent)))
	return rotation.Out(axis)
}

func (cloth *fibres) vertex(point, rest, normal, centre, tangent maths.Vec) int {
	index := len(cloth.points)
	cloth.points = append(cloth.points, point)
	cloth.paint = append(cloth.paint, rest)
	cloth.normals = append(cloth.normals, normal)
	if cloth.optical != nil {
		cloth.optical.Centres = append(cloth.optical.Centres, centre)
		cloth.optical.Tangents = append(cloth.optical.Tangents, tangent)
	}
	return index
}
