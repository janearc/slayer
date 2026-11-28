package workspace

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"

	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

const authoredMapNotice = "Orbit rotates the plan. Travel pans or zooms with Shift. Squares show authored positions. Hollow marks show stand centres or referenced anchors. Lines show the terrain cut and declared rectangular stand areas. Choose overlapping marks by name. Terrain elevation, object size and generated geometry are unavailable."

type authoredPlan struct {
	east, north, span  float64
	angle              float64
	readOnly, editable bool
	groups             map[string]bool
	terrain            *draft.MapRectangle
	areas              map[string]*draft.MapRectangle
}

// prepareAuthoredMap only reads scene source. A job may show its referenced
// set's plan, with the job remaining the active and unchanged document.
func prepareAuthoredMap(ctx context.Context, document *draft.Document, path string) (*stagedView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	readOnly := document.Kind() == "job"
	if readOnly {
		reference := document.ReferencedSet()
		if reference == "" {
			return nil, nil
		}
		if !filepath.IsAbs(reference) {
			reference = filepath.Join(filepath.Dir(path), reference)
		}
		info, err := os.Stat(reference)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("job scene must be a regular file")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		document, _, err = ReadDocument(reference)
		if err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source, available, err := document.AuthoredMap()
	if err != nil || !available {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return makeAuthoredMap(source.Marks, readOnly, source.Terrain), nil
}

func makeAuthoredMap(objects []draft.MapMark, readOnly bool, terrain *draft.MapRectangle) *stagedView {
	scene := &placement.DisplayScene{Width: 1, Height: 1}
	west, east, south, north := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	plan := &authoredPlan{span: 20, readOnly: readOnly, terrain: terrain,
		groups: make(map[string]bool), areas: make(map[string]*draft.MapRectangle)}
	if terrain != nil {
		west, east, south, north = terrain.West, terrain.East, terrain.South, terrain.North
	}
	for _, object := range objects {
		west, east = min(west, object.East), max(east, object.East)
		south, north = min(south, object.North), max(north, object.North)
		mark := placement.DisplayObject{ID: object.ID, Name: object.Name, At: maths.Local()}
		mark.At.Origin = maths.V(object.East, 0, -object.North)
		if object.Group {
			plan.groups[mark.ID] = true
			if object.Area != nil {
				plan.areas[mark.ID] = object.Area
			}
		}
		scene.Objects = append(scene.Objects, mark)
	}
	if len(objects) > 0 || terrain != nil {
		plan.east, plan.north = west/2+east/2, south/2+north/2
		plan.span = max(plan.span, (east-west)*1.12, (north-south)*1.12)
	}
	notice := authoredMapNotice
	if len(objects) == 0 {
		notice += " No explicit placements are available."
	}
	if readOnly {
		notice += " Open the job’s scene to edit its placements."
	}
	return &stagedView{scene: scene, notice: notice, plan: plan}
}

func (plan *authoredPlan) point(mark placement.DisplayObject) (x, y float64) {
	return plan.project(mark.At.Origin.X, -mark.At.Origin.Z)
}

func (plan *authoredPlan) project(east, north float64) (x, y float64) {
	x, y = plan.rotate(east-plan.east, plan.north-north)
	return x/plan.span + .5, y/plan.span + .5
}

func (plan *authoredPlan) rotate(east, south float64) (x, y float64) {
	sin, cos := math.Sincos(plan.angle)
	return east*cos - south*sin, east*sin + south*cos
}

func (plan *authoredPlan) worldDelta(x, y float64) (east, north float64) {
	sin, cos := math.Sincos(plan.angle)
	return (x*cos + y*sin) * plan.span, (x*sin - y*cos) * plan.span
}

func (plan *authoredPlan) retainFraming(previous *authoredPlan, marks []placement.DisplayObject, selected string) {
	if previous == nil {
		return
	}
	for _, mark := range marks {
		if selected != "" && mark.ID != selected {
			continue
		}
		x, y := previous.point(mark)
		if x > 0 && x < 1 && y > 0 && y < 1 {
			plan.east, plan.north, plan.span, plan.angle = previous.east, previous.north, previous.span, previous.angle
			return
		}
	}
}

func (work *Workspace) updateAuthoredMap() {
	live := &work.working
	size := int(fitImageBounds(work.viewer.Bounds(), 1).Width())
	if size < 2 {
		return
	}
	if size != live.width || size != live.height {
		live.width, live.height, live.dirty = size, size, true
	}
	if live.dirty {
		bounds := image.Rect(0, 0, size, size)
		pixels := image.NewRGBA(bounds)
		draw.Draw(pixels, bounds, mapColour(appearance.Panel), image.Point{}, draw.Src)
		live.plan.drawContext(pixels, work.play.selected)
		var selected placement.DisplayObject
		for _, mark := range live.scene.Objects {
			if mark.ID == work.play.selected {
				selected = mark
				continue
			}
			live.plan.drawMark(pixels, mark, appearance.Cyan, 2)
		}
		if selected.ID != "" {
			live.plan.drawMark(pixels, selected, appearance.Pink, 4)
		}
		work.viewer.setWorking(pixels, 1)
		live.dirty, live.displayed = false, true
	}
	work.caption = work.workingCaption()
	work.viewer.SetNavigable(true)
}

func mapColour(swatch widget.Color) *image.Uniform {
	r, g, b, a := swatch.RGBA8()
	return image.NewUniform(color.NRGBA{R: r, G: g, B: b, A: a})
}

func (plan *authoredPlan) drawMark(pixels *image.RGBA, mark placement.DisplayObject, swatch widget.Color, radius int) {
	x, y := plan.point(mark)
	size := float64(pixels.Bounds().Dx())
	px, py := int(x*size), int(y*size)
	rect := image.Rect(px-radius, py-radius, px+radius+1, py+radius+1)
	ink := mapColour(swatch)
	if plan.groups[mark.ID] {
		for _, edge := range []image.Rectangle{
			image.Rect(rect.Min.X-1, rect.Min.Y-1, rect.Max.X+1, rect.Min.Y),
			image.Rect(rect.Min.X-1, rect.Max.Y, rect.Max.X+1, rect.Max.Y+1),
			image.Rect(rect.Min.X-1, rect.Min.Y, rect.Min.X, rect.Max.Y),
			image.Rect(rect.Max.X, rect.Min.Y, rect.Max.X+1, rect.Max.Y),
		} {
			draw.Draw(pixels, edge, ink, image.Point{}, draw.Src)
		}
		return
	}
	draw.Draw(pixels, rect, ink, image.Point{}, draw.Src)
}

func (work *Workspace) pickAuthoredMap(x, y float64) placement.DisplayObject {
	live := &work.working
	radius := 8.0
	var selected placement.DisplayObject
	for _, mark := range live.scene.Objects {
		mx, my := live.plan.point(mark)
		dx, dy := math.Abs(mx-x)*float64(live.width), math.Abs(my-y)*float64(live.height)
		dist := math.Hypot(dx, dy)
		if live.plan.groups[mark.ID] {
			edge := max(dx, dy)
			if edge >= 6 {
				continue
			}
			dist = math.Abs(edge - 3)
		}
		if dist < radius {
			radius, selected = dist, mark
		}
	}
	return selected
}

func (work *Workspace) mapDrag(x0, y0, x1, y1 float64) (east, north float64) {
	return work.working.plan.worldDelta(x1-x0, y1-y0)
}

func (work *Workspace) jobMapMarks() []placement.DisplayObject {
	live := &work.working
	if live.plan == nil || !live.plan.readOnly || live.scene == nil {
		return nil
	}
	return live.scene.Objects
}

func (work *Workspace) refuseMapEdit() error {
	if plan := work.working.plan; plan != nil && plan.readOnly {
		return fmt.Errorf("open the job’s scene to edit its placements")
	} else if plan != nil && (!plan.editable || !work.state.document.Matches(work.working.source)) {
		return fmt.Errorf("map placement edits require a checked terrain window")
	}
	return nil
}

// chooseWorkingDisplay keeps display style independent of the explicit choice
// to build a full scene. The next changed draft opens from source again.
func (work *Workspace) chooseWorkingDisplay(mode string) error {
	switch mode {
	case "full-scene", "authored-map":
		previous := work.working.detail
		work.working.detail = ""
		build := work.stagedBuilder()
		if build == nil {
			work.working.detail = previous
			return fmt.Errorf("the geometry helper is unavailable")
		}
		work.cancelViewportGesture()
		work.working.fullScene = mode == "full-scene"
		if work.working.mode == "" || work.working.mode == "rendered" {
			work.working.mode = "edges"
		}
		work.refreshLayout = true
		work.requestStaged(build, true)
		return nil
	default:
		return work.SetWorkingMode(mode)
	}
}

func (ball *cameraBall) moveMap(zoom bool) {
	live := &ball.work.working
	if live.plan != ball.mapStart {
		return
	}
	dx := float64(ball.current.X - ball.start.X)
	dy := float64(ball.current.Y - ball.start.Y)
	frame := ball.mapFrame
	switch {
	case !ball.translation:
		frame.angle = math.Remainder(frame.angle+dx/ball.radius, 2*math.Pi)
	case zoom:
		frame.span = max(1, min(1e9, frame.span*math.Exp(dy/ball.radius)))
	default:
		east, north := frame.worldDelta(dx/float64(live.width), dy/float64(live.height))
		frame.east -= east
		frame.north -= north
	}
	*live.plan = frame
	live.dirty = true
	ball.work.invalidate()
}

func (ball *cameraBall) finishMap(keep bool) {
	if ball.mapStart == nil {
		return
	}
	live := &ball.work.working
	if live.plan == ball.mapStart && !keep {
		*live.plan = ball.mapFrame
		live.dirty = true
	}
	ball.mapStart = nil
	ball.dragging = false
	ball.work.invalidate()
}
