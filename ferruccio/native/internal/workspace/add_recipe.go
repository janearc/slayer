package workspace

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"github.com/janearc/slayer/ferruccio/native/internal/appearance"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
	"gopkg.in/yaml.v3"
)

type addChoice struct {
	field, name, group, about string
	profile                   *yaml.Node
}

// addChoices filters authored recipes, the registry and checked profile menu.
func addChoices(document *draft.Document, query string) ([]addChoice, error) {
	recipes, err := document.RecipeNames()
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	var found []addChoice
	for _, group := range []struct {
		field string
		names []string
	}{{"recipe", recipes}, {"kind", manufacture.KindNames()}} {
		for _, name := range group.names {
			if strings.Contains(strings.ToLower(name), query) {
				found = append(found, addChoice{field: group.field, name: name})
			}
		}
	}
	profiles, err := preview.CatalogueProfiles()
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		if strings.Contains(strings.ToLower(profile.Name+" "+profile.Group+" "+profile.About), query) {
			found = append(found, addChoice{field: profile.Kind, name: profile.Name, group: profile.Group, about: profile.About, profile: profile.Spec})
		}
	}
	return found, nil
}

// addRecipePanel offers authored recipes and searchable constructor kinds.
func (work *Workspace) addRecipePanel() widget.Widget {
	choices, err := addChoices(work.state.document, work.play.addFilter)
	if err != nil {
		return work.text(err.Error())
	}
	search := textfield.New(textfield.InitialValue(work.play.addQuery), textfield.Placeholder("Search recipes and constructors"), textfield.A11yLabel("Search objects"), textfield.PainterOpt(appearance.TextFieldPainter{Size: work.size}), textfield.OnChange(func(text string) { work.play.addQuery = text }), textfield.OnSubmit(func(text string) {
		work.play.addQuery, work.play.addFilter, work.play.addRecipeName = text, text, ""
		work.rebuild()
	}))
	search.SetID("Search objects")
	find := func() {
		work.play.addQuery, work.play.addFilter, work.play.addRecipeName = search.Text(), search.Text(), ""
		work.rebuild()
	}
	searchRow := primitives.HBox(primitives.Expanded(search), work.action("Find", false, func() bool { return false }, find)).Gap(8)
	if len(choices) == 0 {
		return primitives.VBox(searchRow, work.text("No matching recipes or constructors.")).Gap(8)
	}
	labels := make([]string, len(choices))
	selected := 0
	for i, option := range choices {
		labels[i] = option.field + ": " + option.name
		if option.field+":"+option.name == work.play.addRecipeName {
			selected = i
		}
	}
	work.play.addRecipeName = choices[selected].field + ":" + choices[selected].name
	choice := dropdown.New(dropdown.Items(labels...), dropdown.Selected(selected), dropdown.PainterOpt(appearance.DropdownPainter{Theme: work.theme, Size: work.size}), dropdown.OnChange(func(index int, _ string) {
		if index >= 0 && index < len(choices) {
			work.play.addRecipeName = choices[index].field + ":" + choices[index].name
			work.rebuild()
		}
	}))
	choice.SetID("Choose object to add")
	field := func(label string, value *string) *textfield.Widget {
		input := textfield.New(textfield.InitialValue(*value), textfield.A11yLabel(label), textfield.PainterOpt(appearance.TextFieldPainter{Size: work.size}), textfield.OnChange(func(text string) { *value = text }))
		input.SetID(label)
		return input
	}
	name := field("New object name", &work.play.addName)
	seed := field("Seed", &work.play.addSeed)
	east := field("East (m)", &work.play.addAt[0])
	north := field("North (m)", &work.play.addAt[1])
	heightLabel := "Up (m)"
	ground := work.state.document.GroundSetting()
	if ground {
		heightLabel = "Above ground (m)"
	}
	up := field(heightLabel, &work.play.addAt[2])
	labeled := func(label string, input widget.Widget) widget.Widget {
		return primitives.VBox(work.text(label), input).Gap(4)
	}
	description := "Add to this space scene."
	if ground {
		description = "Add to terrain: east and north use the terrain map; height is above ground."
	}
	items := []widget.Widget{work.paragraph(description, appearance.Ink), searchRow, choice}
	var book *textfield.Widget
	if choices[selected].profile != nil {
		book = field("Profile book path", &work.play.addBook)
		browse := work.action("Choose…", false, func() bool { return false }, func() {
			paths, err := work.host.ShowOpenFileDialog(gogpu.FileDialogOptions{Title: "Choose the profile book"})
			if err != nil {
				work.problem(err)
				return
			}
			if len(paths) > 0 {
				work.play.addBook = paths[0]
				work.rebuild()
			}
		})
		items = append(items, work.paragraph(choices[selected].about, appearance.Ink), work.text("Built-in book by default · choose a custom book if needed"), primitives.HBox(primitives.Expanded(book), browse).Gap(8))
	}
	items = append(items,
		primitives.HBox(primitives.Expanded(labeled("New object name", name)), primitives.Expanded(labeled("Seed", seed))).Gap(8),
		primitives.HBox(primitives.Expanded(labeled("East (m)", east)), primitives.Expanded(labeled("North (m)", north)), primitives.Expanded(labeled(heightLabel, up))).Gap(8),
		work.action("Add object", true, func() bool { return false }, func() {
			index := choice.SelectedIndex()
			if index >= 0 && index < len(choices) {
				at := [3]string{east.Text(), north.Text(), up.Text()}
				if choices[index].profile != nil && book != nil {
					work.problem(work.addProfile(name.Text(), choices[index], book.Text(), seed.Text(), at))
				} else {
					work.problem(work.addChosen(name.Text(), choices[index], seed.Text(), at))
				}
			}
		}))
	return primitives.VBox(items...).Gap(8)
}

// addChosen checks values, applies one edit and records one history entry.
func (work *Workspace) addChosen(name string, choice addChoice, seedText string, atText [3]string) error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	seed, at, err := addCoordinates(seedText, atText)
	if err != nil {
		return err
	}
	if err := work.checkTerrainAddition(at); err != nil {
		return err
	}
	var changeErr error
	switch choice.field {
	case "recipe":
		changeErr = work.state.document.AddRecipeThing(name, choice.name, seed, at)
	case "kind":
		if _, err := manufacture.KindNamed(choice.name); err != nil {
			return err
		}
		changeErr = work.state.document.AddKindThing(name, choice.name, seed, at)
	default:
		return fmt.Errorf("choose a recipe or constructor")
	}
	if changeErr != nil {
		return changeErr
	}
	work.finishedAdding(name)
	return nil
}

// addCoordinates checks the typed seed and metre coordinates.
func addCoordinates(seedText string, atText [3]string) (uint64, [3]float64, error) {
	seed, err := strconv.ParseUint(strings.TrimSpace(seedText), 10, 64)
	if err != nil {
		return 0, [3]float64{}, fmt.Errorf("enter a whole-number seed")
	}
	var at [3]float64
	for axis, text := range atText {
		at[axis], err = strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return 0, [3]float64{}, fmt.Errorf("enter each position in metres")
		}
	}
	return seed, at, nil
}

// addProfile validates a chosen book through the real set reader before the
// two-node source edit, without loading unrelated scene assets on the UI thread.
func (work *Workspace) addProfile(name string, choice addChoice, book, seedText string, atText [3]string) error {
	if err := work.fullRenderRefusal("changing the scene"); err != nil {
		return err
	}
	seed, at, err := addCoordinates(seedText, atText)
	if err != nil {
		return err
	}
	if err := work.checkTerrainAddition(at); err != nil {
		return err
	}
	if strings.TrimSpace(book) == "" {
		book = set.BuiltinProfileBook
	}
	const probe = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Profile check
  setting: space
  seed: 1
  light: none
  film: {width: 8, height: 8, wide: 40, rays: 1, exposure: 1}
  cameras:
    view: {location: {at: [0, -5, 2]}, orientation: {look_at: [0, 0, 1]}}
`
	check, err := draft.Open(work.state.path, []byte(probe))
	if err != nil {
		return err
	}
	if err := check.AddProfileThing("check", "check-recipe", choice.field, choice.profile, book, seed, at); err != nil {
		return err
	}
	if _, err := set.Read(work.state.path, []byte(check.Text())); err != nil {
		return fmt.Errorf("profile book does not match this choice: %w", err)
	}
	candidate, err := draft.Open(work.state.path, []byte(work.state.document.Text()))
	if err != nil {
		return err
	}
	bookReference := book
	if filepath.IsAbs(work.state.path) && filepath.IsAbs(book) {
		if relative, err := filepath.Rel(filepath.Dir(work.state.path), book); err == nil {
			bookReference = relative
		}
	}
	if err := candidate.AddProfileThing(name, name+"-recipe", choice.field, choice.profile, bookReference, seed, at); err != nil {
		return err
	}
	work.state.document.Replace(candidate.Text())
	work.finishedAdding(name)
	return nil
}

// finishedAdding records an already-checked source edit without a render.
func (work *Workspace) finishedAdding(name string) {
	work.stopPreview()
	work.editor.ApplyText(work.state.document.Text())
	work.play.selected = "scene.set.things/" + name
	work.needsCheck = false
	work.working.dirty = true
	work.caption = "Object added · working view updating"
	work.status = "Object added · ⌘Z to undo"
	work.diagnostic = ""
	work.refreshLayout = true
	work.invalidate()
}

// checkTerrainAddition uses the staged set's actual window before editing it.
func (work *Workspace) checkTerrainAddition(at [3]float64) error {
	if !work.state.document.GroundSetting() {
		return nil
	}
	return work.checkTerrainPosition(at)
}

// checkTerrainPosition uses the parsed terrain for a known ground placement.
func (work *Workspace) checkTerrainPosition(at [3]float64) error {
	parsed := work.authoringSet()
	if parsed == nil {
		return fmt.Errorf("wait for the current terrain to finish loading before editing its positions")
	}
	ground, ok := parsed.Ground()
	if !ok {
		return fmt.Errorf("the set has no terrain")
	}
	west, north, across, down, step := ground.Window()
	if step <= 0 || at[0] < 0 || at[1] < 0 || at[2] < 0 || math.IsNaN(at[0]) || math.IsNaN(at[1]) || math.IsNaN(at[2]) || math.IsInf(at[0], 0) || math.IsInf(at[1], 0) || math.IsInf(at[2], 0) {
		return fmt.Errorf("enter a position on the terrain and a nonnegative height above it")
	}
	column, row := math.Round((at[0]-west)/step), math.Round((north-at[1])/step)
	if column < 0 || row < 0 || column >= math.Floor(across/step)+1 || row >= math.Floor(down/step)+1 {
		return fmt.Errorf("the position is outside the terrain window")
	}
	return nil
}

// defaultTerrainAddition starts near the selected object or the map centre.
func (work *Workspace) defaultTerrainAddition() {
	if !work.state.document.GroundSetting() {
		return
	}
	if object, ok := work.selectedObject(); ok && object.Grounded() {
		at := object.Position()
		work.play.addAt = [3]string{strconv.FormatFloat(at[0], 'f', -1, 64), strconv.FormatFloat(at[1], 'f', -1, 64), "0"}
		return
	}
	parsed := work.authoringSet()
	if parsed == nil {
		return
	}
	ground, ok := parsed.Ground()
	if !ok {
		return
	}
	west, north, across, down, _ := ground.Window()
	work.play.addAt = [3]string{strconv.FormatFloat(west+across/2, 'f', -1, 64), strconv.FormatFloat(north-down/2, 'f', -1, 64), "0"}
}
