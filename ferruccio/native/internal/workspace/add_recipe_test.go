package workspace

import (
	"testing"

	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
	"os"
	"path/filepath"
)

const addRecipeFixture = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Recipe addition
  setting: space
  seed: 19
  light: none
  recipes:
    low: {kind: tower, irregularity: 0, pins: {plan: rectangle, skin: concrete, storeys: 6, width: {value: 20, unit: m}, taper: false, twist: false, setbacks: false, spire: false}}
  set:
    things:
      first: {recipe: low, seed: 3, complexity: quick, east: 30, north: 0, height: {value: 0, unit: m}}
  film: {width: 640, height: 400, wide: 40, rays: 1, exposure: 1}
  cameras:
    view:
      location: {at: [0, -60, 20]}
      orientation: {look_at: [30, 0, 10]}
`

func TestAddExistingRecipeIsUndoableWithoutRendering(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	work.play.disclosure = "add"
	if err := work.addChosen("second", addChoice{field: "recipe", name: "low"}, "8", [3]string{"35", "-2", "4"}); err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || objects[1].Position() != [3]float64{35, -2, 4} || work.play.selected != objects[1].ID() {
		t.Fatalf("added objects = %+v, %v", objects, err)
	}
	if work.pending.document != nil || work.Running() {
		t.Fatal("adding an object requested an offline render")
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != addRecipeFixture || work.pending.document != nil {
		t.Fatal("undo did not restore source without rendering")
	}
}

func TestAddRecipePanelAcceptsPointerSubmission(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	work.rebuild()
	drawLibraryWindow(t, work)
	button := libraryWidget(work.root, "Add…")
	if button == nil {
		t.Fatal("missing add control")
	}
	bounds := button.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	if bounds.Min.X < 0 || bounds.Max.X > 1280 || bounds.Min.Y < 0 || bounds.Max.Y > 800 {
		t.Fatalf("add control outside window: %v", bounds)
	}
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	drawLibraryWindow(t, work)
	field, ok := libraryWidget(work.root, "New object name").(*textfield.Widget)
	if !ok {
		t.Fatal("missing name input")
	}
	field.SetText("second")
	apply := libraryWidget(work.root, "Add object")
	if apply == nil {
		t.Fatal("missing submit control")
	}
	bounds = apply.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	if bounds.Min.X < 0 || bounds.Max.X > 1280 || bounds.Min.Y < 0 || bounds.Max.Y > 800 {
		t.Fatalf("submit outside window: %v", bounds)
	}
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || objects[1].Name() != "second" {
		t.Fatalf("pointer submission objects = %+v, %v", objects, err)
	}
}

func TestAddExistingRecipeRejectsBadInput(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	for _, test := range []struct {
		seed string
		at   [3]string
	}{
		{"wrong", [3]string{"0", "0", "0"}},
		{"1", [3]string{"0", "NaN", "0"}},
	} {
		if err := work.addChosen("second", addChoice{field: "recipe", name: "low"}, test.seed, test.at); err == nil || document.Text() != addRecipeFixture {
			t.Fatalf("bad input changed source: %v", err)
		}
	}
}

func TestAddChoicesComeFromRecipesAndConstructorRegistry(t *testing.T) {
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	choices, err := addChoices(document, "city")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, choice := range choices {
		if choice.field == "kind" && choice.name == "city-grid-10000" {
			found = true
		}
	}
	if !found {
		t.Fatal("real city constructor absent from search")
	}
	choices, err = addChoices(document, "low")
	if err != nil || len(choices) == 0 || choices[0].field != "recipe" || choices[0].name != "low" {
		t.Fatalf("authored recipe search = %+v, %v", choices, err)
	}
}

func TestAddChoicesIncludeCatalogueFishAndPlants(t *testing.T) {
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ query, field string }{{"pacific-bluefin-tuna", "animal"}, {"spruce", "plant"}} {
		choices, err := addChoices(document, test.query)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, choice := range choices {
			if choice.field == test.field && choice.name == test.query && choice.profile != nil && choice.about != "" {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing catalogue entry %q", test.query)
		}
	}
}

func TestAddConstructorChecksRegistryAndIsUndoable(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	work.play.disclosure = "add"
	if err := work.addChosen("house", addChoice{field: "kind", name: "city-rowhouse"}, "7", [3]string{"2", "3", "4"}); err != nil {
		t.Fatal(err)
	}
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || objects[1].Kind() != "city-rowhouse" {
		t.Fatalf("constructor placement = %+v, %v", objects, err)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != addRecipeFixture || work.pending.document != nil {
		t.Fatal("constructor undo failed or queued a render")
	}
	if err := work.addChosen("house", addChoice{field: "kind", name: "not-a-constructor"}, "7", [3]string{"2", "3", "4"}); err == nil || document.Text() != addRecipeFixture {
		t.Fatal("unknown constructor was accepted")
	}
}

func TestSearchFindsAndAddsConstructorThroughControls(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	work.toggleDisclosure("add")
	drawLibraryWindow(t, work)
	search, ok := libraryWidget(work.root, "Search objects").(*textfield.Widget)
	if !ok {
		t.Fatal("missing object search")
	}
	search.SetText("city-rowhouse")
	work.ui.Window().Context().RequestFocus(search)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	drawLibraryWindow(t, work)
	choice, ok := libraryWidget(work.root, "Choose object to add").(*dropdown.Widget)
	if !ok || choice.SelectedValue() != "kind: city-rowhouse" {
		t.Fatalf("searched choice = %v", choice)
	}
	name, ok := libraryWidget(work.root, "New object name").(*textfield.Widget)
	if !ok {
		t.Fatal("missing name input")
	}
	name.SetText("house")
	apply := libraryWidget(work.root, "Add object")
	if apply == nil {
		t.Fatal("missing add button")
	}
	bounds := apply.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	objects, err := document.Objects()
	if err != nil || len(objects) != 2 || objects[1].Kind() != "city-rowhouse" {
		t.Fatalf("search and add = %+v, %v", objects, err)
	}
}

func TestAddProfileValidatesBookBeforeChangingDraft(t *testing.T) {
	work := libraryEventWorkspace(t)
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	work.play.disclosure = "add"
	choices, err := addChoices(document, "pacific-bluefin-tuna")
	if err != nil {
		t.Fatal(err)
	}
	var fish addChoice
	for _, choice := range choices {
		if choice.field == "animal" && choice.name == "pacific-bluefin-tuna" {
			fish = choice
			break
		}
	}
	if fish.profile == nil {
		t.Fatal("fish choice missing")
	}
	_, raw, err := manufacture.EmbeddedAnimalBook("fish")
	if err != nil {
		t.Fatal(err)
	}
	book := filepath.Join(t.TempDir(), "fish.yaml")
	if err := os.WriteFile(book, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := work.addProfile("tuna", fish, book, "4", [3]string{"1", "2", "3"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := document.Value("scene", "recipes", "tuna-recipe", "animal", "file"); resolvedBook(work.state.path, got) != book {
		t.Fatalf("saved book = %q", got)
	}
	work.undoCamera(work.ui.Window().Context(), false)
	if document.Text() != addRecipeFixture || work.pending.document != nil {
		t.Fatal("profile undo failed or queued a render")
	}
	if err := work.addProfile("tuna", fish, filepath.Join(t.TempDir(), "missing.yaml"), "4", [3]string{"1", "2", "3"}); err == nil || document.Text() != addRecipeFixture {
		t.Fatal("missing book changed the document")
	}
	_, plants, err := manufacture.EmbeddedPlantBook("fern")
	if err != nil {
		t.Fatal(err)
	}
	wrong := filepath.Join(t.TempDir(), "fern.yaml")
	if err := os.WriteFile(wrong, plants, 0600); err != nil {
		t.Fatal(err)
	}
	if err := work.addProfile("tuna", fish, wrong, "4", [3]string{"1", "2", "3"}); err == nil || document.Text() != addRecipeFixture {
		t.Fatal("mismatched book changed the document")
	}
}

func TestAddProfileStoresRelativeBookForSavedScene(t *testing.T) {
	work := libraryEventWorkspace(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.yaml")
	document, err := draft.Open(path, []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.path, work.state.document = path, document
	work.editor.SetText(addRecipeFixture)
	choices, err := addChoices(document, "pacific-bluefin-tuna")
	if err != nil {
		t.Fatal(err)
	}
	var fish addChoice
	for _, choice := range choices {
		if choice.field == "animal" && choice.name == "pacific-bluefin-tuna" {
			fish = choice
			break
		}
	}
	if fish.profile == nil {
		t.Fatal("fish choice missing")
	}
	_, raw, err := manufacture.EmbeddedAnimalBook("fish")
	if err != nil {
		t.Fatal(err)
	}
	book := filepath.Join(dir, "fish.yaml")
	if err := os.WriteFile(book, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := work.addProfile("tuna", fish, book, "4", [3]string{"1", "2", "3"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := document.Value("scene", "recipes", "tuna-recipe", "animal", "file"); got != "fish.yaml" {
		t.Fatalf("profile path = %q", got)
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Read(path, saved); err != nil {
		t.Fatalf("saved scene could not load its profile: %v", err)
	}
	newPath := filepath.Join(t.TempDir(), "copy.yaml")
	if err := document.RebaseProfileBooks(path, newPath); err != nil {
		t.Fatal(err)
	}
	if err := document.Save(newPath); err != nil {
		t.Fatal(err)
	}
	saved, err = os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Read(newPath, saved); err != nil {
		t.Fatalf("moved scene could not load its profile: %v", err)
	}
}

func TestAddProfileUsesBuiltInBookWhenNoneChosen(t *testing.T) {
	work := libraryEventWorkspace(t)
	path := filepath.Join(t.TempDir(), "scene.yaml")
	document, err := draft.Open(path, []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.path, work.state.document = path, document
	work.editor.SetText(addRecipeFixture)
	choices, err := addChoices(document, "pacific-bluefin-tuna")
	if err != nil {
		t.Fatal(err)
	}
	var fish addChoice
	for _, choice := range choices {
		if choice.field == "animal" && choice.name == "pacific-bluefin-tuna" {
			fish = choice
			break
		}
	}
	if fish.profile == nil {
		t.Fatal("fish choice missing")
	}
	if err := work.addProfile("tuna", fish, "", "4", [3]string{"1", "2", "3"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := document.Value("scene", "recipes", "tuna-recipe", "animal", "file"); got != set.BuiltinProfileBook {
		t.Fatalf("profile book = %q", got)
	}
	if err := document.Save(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Read(path, saved); err != nil {
		t.Fatalf("saved scene did not load: %v", err)
	}
	if err := document.RebaseProfileBooks(path, filepath.Join(t.TempDir(), "copy.yaml")); err != nil {
		t.Fatal(err)
	}
	if got, _ := document.Value("scene", "recipes", "tuna-recipe", "animal", "file"); got != set.BuiltinProfileBook {
		t.Fatalf("save as changed built-in book to %q", got)
	}
	choices, err = addChoices(document, "fern")
	if err != nil {
		t.Fatal(err)
	}
	var fern addChoice
	for _, choice := range choices {
		if choice.field == "plant" && choice.name == "fern" {
			fern = choice
			break
		}
	}
	if fern.profile == nil {
		t.Fatal("fern choice missing")
	}
	if err := work.addProfile("fern", fern, "", "5", [3]string{"2", "3", "0"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := document.Value("scene", "recipes", "fern-recipe", "plant", "file"); got != set.BuiltinProfileBook {
		t.Fatalf("plant book = %q", got)
	}
	if _, err := set.Read(path, []byte(document.Text())); err != nil {
		t.Fatalf("scene with built-in fish and plant did not load: %v", err)
	}
}

func TestSearchAndAddFishThroughControls(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	document, err := draft.Open("test.yaml", []byte(addRecipeFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(addRecipeFixture)
	work.toggleDisclosure("add")
	drawLibraryWindow(t, work)
	search, ok := libraryWidget(work.root, "Search objects").(*textfield.Widget)
	if !ok {
		t.Fatal("missing object search")
	}
	search.SetText("pacific-bluefin-tuna")
	work.ui.Window().Context().RequestFocus(search)
	work.ui.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	drawLibraryWindow(t, work)
	choice, ok := libraryWidget(work.root, "Choose object to add").(*dropdown.Widget)
	if !ok || choice.SelectedValue() != "animal: pacific-bluefin-tuna" {
		t.Fatalf("searched fish = %v", choice)
	}
	_, raw, err := manufacture.EmbeddedAnimalBook("fish")
	if err != nil {
		t.Fatal(err)
	}
	bookPath := filepath.Join(t.TempDir(), "fish.yaml")
	if err := os.WriteFile(bookPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	book, ok := libraryWidget(work.root, "Profile book path").(*textfield.Widget)
	if !ok {
		t.Fatal("missing external profile field")
	}
	book.SetText(bookPath)
	name, ok := libraryWidget(work.root, "New object name").(*textfield.Widget)
	if !ok {
		t.Fatal("missing name input")
	}
	name.SetText("tuna")
	apply := libraryWidget(work.root, "Add object")
	if apply == nil {
		t.Fatal("missing add button")
	}
	bounds := apply.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	if got, _ := document.Value("scene", "recipes", "tuna-recipe", "animal", "file"); resolvedBook(work.state.path, got) != bookPath {
		t.Fatalf("fish book after pointer submission = %q", got)
	}
}

func resolvedBook(scene, book string) string {
	if filepath.IsAbs(book) {
		return book
	}
	return filepath.Join(filepath.Dir(scene), book)
}

func TestAddGroundPanelSubmitsMapCoordinates(t *testing.T) {
	work := libraryEventWorkspace(t)
	work.ui.Window().HandleResize(1280, 800)
	path := filepath.Join(t.TempDir(), "ground.yaml")
	document, err := draft.Open(path, []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document, work.state.path = document, path
	work.editor.SetText(groundEditFixture)
	work.play.selected = ""
	work.working.parsed, work.working.source, work.working.path = &read, groundEditFixture, path
	work.toggleDisclosure("add")
	work.play.addFilter = "yurt"
	work.rebuild()
	drawLibraryWindow(t, work)
	field, ok := libraryWidget(work.root, "New object name").(*textfield.Widget)
	if !ok {
		t.Fatal("missing name input")
	}
	field.SetText("ground-yurt")
	if libraryWidget(work.root, "Above ground (m)") == nil {
		t.Fatal("height field did not describe ground-relative height")
	}
	apply := libraryWidget(work.root, "Add object")
	if apply == nil {
		t.Fatal("missing add button")
	}
	bounds := apply.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
	if bounds.Max.X > 1280 || bounds.Max.Y > 800 {
		t.Fatalf("add button outside window: %v", bounds)
	}
	clickLibraryPoint(work, geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2))
	objects, err := document.Objects()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, object := range objects {
		if object.Name() == "ground-yurt" {
			heading, said := object.Heading()
			if !object.Grounded() || !said || heading != 0 || object.Position() != [3]float64{1076, 8949, 0} {
				t.Fatalf("added %+v", object)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("terrain object not added by panel")
	}
}
