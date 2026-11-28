package workspace

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/manufacture"
)

// canScaleThing checks source placement and the current constructor's motion.
func (work *Workspace) canScaleThing() bool {
	object, ok := work.selectedObject()
	if !ok || !object.ScaleEditable() {
		return false
	}
	if work.play.scaleDocument == work.state.document && work.play.scaleID == object.ID() && work.play.scalePath == work.state.path && work.state.document.Matches(work.play.scaleSource) {
		return true
	}
	parsed := work.authoringSet()
	if parsed == nil {
		return false
	}
	for _, thing := range parsed.Things() {
		if "scene.set.things/"+thing.Name() != object.ID() {
			continue
		}
		if thing.Alight() {
			return false
		}
		var row manufacture.Kind
		var err error
		if thing.Recipe() == "" {
			row, err = manufacture.KindNamed(thing.Kind())
		} else if recipe, found := parsed.Recipe(thing.Recipe()); found {
			var made *manufacture.Kind
			made, err = recipe.Construction()
			if err == nil && made != nil {
				row = *made
			} else if err == nil {
				return false
			}
		} else {
			return false
		}
		eligible := err == nil && !manufacture.Deforms(row) && !(object.Grounded() && (manufacture.StandsOnTerrain(row) || thing.Kind() == "fire" || strings.HasPrefix(thing.Name(), "rig ")))
		if eligible {
			work.play.scaleDocument, work.play.scaleID, work.play.scaleSource, work.play.scalePath = work.state.document, object.ID(), work.state.document.Text(), work.state.path
		}
		return eligible
	}
	return false
}

// thingScalePanel offers the renderer's uniform geometric size factor.
func (work *Workspace) thingScalePanel() widget.Widget {
	object, ok := work.selectedObject()
	if !ok || !work.canScaleThing() {
		return work.text("This placement cannot use a uniform size factor.")
	}
	if work.play.sizeObject != object.ID() {
		factor, _ := object.SizeScale()
		work.play.sizeObject, work.play.sizeText = object.ID(), strconv.FormatFloat(factor, 'g', -1, 64)
	}
	field := textfield.New(textfield.InitialValue(work.play.sizeText), textfield.A11yLabel("Uniform size factor"), textfield.PainterOpt(appearance.TextFieldPainter{Size: work.size}), textfield.OnChange(func(text string) { work.play.sizeText = text }), textfield.OnSubmit(func(text string) { work.problem(work.submitThingScale(text)) }))
	field.SetID("Uniform size factor")
	apply := work.action("Set size", true, func() bool { return false }, func() { work.problem(work.submitThingScale(field.Text())) })
	smaller := work.action("− 10%", false, func() bool { return false }, func() { work.problem(work.stepThingScale(1 / 1.1)) })
	larger := work.action("+ 10%", false, func() bool { return false }, func() { work.problem(work.stepThingScale(1.1)) })
	return primitives.VBox(work.text("Uniform size factor"), field,
		work.paragraph("Geometry scales; lamp power stays unchanged.", appearance.Ink), primitives.HBox(smaller, larger).Gap(12), apply).Gap(12)
}

// submitThingScale turns typed text into a checked size edit.
func (work *Workspace) submitThingScale(text string) error {
	factor, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return fmt.Errorf("enter a uniform size factor")
	}
	if err := work.changeThingScale(factor); err != nil {
		return err
	}
	work.play.sizeObject = ""
	work.refreshLayout = true
	return nil
}

// changeThingScale makes one undoable source edit without starting a render.
func (work *Workspace) changeThingScale(factor float64) error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	if !work.canScaleThing() {
		return fmt.Errorf("this placement cannot use a uniform size factor")
	}
	object, _ := work.selectedObject()
	previous, _ := object.SizeScale()
	base, current := work.poseBase()
	before := work.state.document.Text()
	work.rememberProvisional()
	if err := work.state.document.SetThingScale(work.play.selected, factor); err != nil {
		return err
	}
	if work.state.document.Text() == before {
		return nil
	}
	work.rememberWorking(work.working.scene)
	if current {
		work.previewScale(base, object, factor/previous)
	}
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.play.scaleDocument, work.play.scaleID, work.play.scaleSource, work.play.scalePath = work.state.document, work.play.selected, work.state.document.Text(), work.state.path
	work.needsCheck = false
	work.working.dirty = true
	work.caption = "Size changed · working view updating"
	work.status = "Size changed · ⌘Z to undo"
	work.diagnostic = ""
	work.invalidate()
	return nil
}

// stepThingScale adjusts the current factor without waiting for restaging.
func (work *Workspace) stepThingScale(multiplier float64) error {
	object, ok := work.selectedObject()
	if !ok {
		return fmt.Errorf("choose an authored object")
	}
	factor, _ := object.SizeScale()
	if err := work.changeThingScale(factor * multiplier); err != nil {
		return err
	}
	work.play.sizeObject = ""
	work.refreshLayout = true
	return nil
}
