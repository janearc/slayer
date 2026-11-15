package named_test

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/named"
)

// the traced scenes the registry holds, in its order, and every scene,
// the traced first and then those drawn as maps
var (
	tracedNames = []string{"swarm", "zappy", "orbital", "ring", "moonrise",
		"magnetar"}
	everyName = append(slices.Clone(tracedNames), "weather")
)

// The registry lists each scene once, in its order.
func TestTheRegistryListsEachSceneOnce(t *testing.T) {
	var listed []string
	for _, scene := range named.All() {
		listed = append(listed, scene.Name())
	}
	if !slices.Equal(listed, everyName) {
		t.Errorf("the registry lists %v, want %v", listed, everyName)
	}
}

// Every scene listed is found by its name.
func TestEverySceneIsFoundByName(t *testing.T) {
	for _, name := range tracedNames {
		scene, err := named.Get(name)
		if err != nil || scene.Name() != name {
			t.Errorf("%s: found %v, %v", name, scene, err)
		}
	}
}

// anti-tests: a name no scene has, a near miss among them, is refused,
// and the refusal lists the names there are.
func TestANameNoSceneHasIsRefused(t *testing.T) {
	for _, wrong := range []string{"Swarm", "swarms", "", "lantern"} {
		scene, err := named.Get(wrong)
		if !errors.Is(err, named.ErrNoScene) || scene != nil {
			t.Errorf("%q: found %v, %v; want ErrNoScene", wrong, scene,
				err)
			continue
		}
		for _, name := range everyName {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("%q's refusal does not list %s: %v", wrong,
					name, err)
			}
		}
	}
}

// A copy from All is a caller's own: changing what it lists changes
// nothing in the registry.
func TestAllHandsOutCopies(t *testing.T) {
	first := named.All()
	first[0] = named.Scene{}
	if named.All()[0].Name() != everyName[0] {
		t.Error("changing All's list changed the registry")
	}
}

// Each scene says what it is, what its unit is, and that the cpu draws
// it; its defaults are a picture of some size.
func TestEverySceneSaysWhatItIs(t *testing.T) {
	for _, scene := range named.Traced() {
		made := scene.Defaults()
		if scene.About() == "" || scene.Unit() == "" ||
			!scene.Draws(named.CPU) || made.Width() < once ||
			made.Height() < once || made.Rays() < once ||
			made.Wide() <= none {
			t.Errorf("%s says too little: %q in %q, %v, %+v",
				scene.Name(), scene.About(), scene.Unit(),
				scene.Renderers(), made)
		}
	}
}

// anti-tests: a renderer no scene names draws none of them, and changing
// the list of renderers a scene hands out changes nothing in it.
func TestARendererNotNamedDrawsNothing(t *testing.T) {
	for _, scene := range named.Traced() {
		if scene.Draws(named.Renderer("gpu-native")) {
			t.Errorf("%s says gpu-native draws it", scene.Name())
		}
		listed := scene.Renderers()
		listed[0] = "gpu-native"
		if scene.Draws("gpu-native") {
			t.Errorf("%s's renderers changed from outside",
				scene.Name())
		}
	}
}

// Every parameter of every scene reads its own default, given as the
// kind it is, and the scene checks clean with nothing given.
func TestEveryParameterReadsItsDefault(t *testing.T) {
	for _, scene := range named.All() {
		if err := scene.Check(nil); err != nil {
			t.Errorf("%s with nothing given: %v", scene.Name(), err)
		}
		for _, one := range scene.Parameters() {
			with := map[string]any{one.Name(): one.Default()}
			if err := scene.Check(with); err != nil {
				t.Errorf("%s's %s at its default: %v", scene.Name(),
					one.Name(), err)
			}
			if one.Help() == "" || one.Kind() == "" {
				t.Errorf("%s's %s has no help or kind",
					scene.Name(), one.Name())
			}
		}
	}
}

// anti-tests: every parameter given a value of the wrong kind refuses it
// by name.
func TestEveryParameterRefusesTheWrongKind(t *testing.T) {
	for _, scene := range named.All() {
		for _, one := range scene.Parameters() {
			wrong := wrongKind[one.Kind()]
			err := scene.Check(map[string]any{one.Name(): wrong})
			if !errors.Is(err, named.ErrKind) ||
				!strings.Contains(err.Error(), one.Name()) {
				t.Errorf("%s's %s took %v (%T): %v", scene.Name(),
					one.Name(), wrong, wrong, err)
			}
		}
	}
}

// wrongKind is a value of another kind for each kind: text where a
// number goes, a number where a switch or a name goes, a bare number
// with no unit where a duration goes.
var wrongKind = map[named.Kind]any{
	named.Whole:    "sixty",
	named.Number:   "more",
	named.Switch:   once,
	named.Duration: float64(once),
	named.Choice:   true,
}

// anti-tests: a key the scene has not, a near miss of one it has, is
// refused, and the refusal lists the keys it has.
func TestAKeyTheSceneHasNotIsRefused(t *testing.T) {
	for _, scene := range named.All() {
		near := scene.Parameters()[0].Name() + "s"
		err := scene.Check(map[string]any{near: once})
		if !errors.Is(err, named.ErrNoParameter) {
			t.Errorf("%s took %q: %v", scene.Name(), near, err)
			continue
		}
		for _, one := range scene.Parameters() {
			if !strings.Contains(err.Error(), one.Name()) {
				t.Errorf("%s's refusal does not list %s: %v",
					scene.Name(), one.Name(), err)
			}
		}
	}
}

// anti-tests: a build asked for a key the scene has not is refused before
// anything is built.
func TestABuildWithAWrongKeyIsRefused(t *testing.T) {
	for _, scene := range named.Traced() {
		built, err := scene.Build(map[string]any{"wide": once},
			io.Discard)
		if !errors.Is(err, named.ErrNoParameter) || built != nil {
			t.Errorf("%s built with -wide: %v, %v", scene.Name(), built,
				err)
		}
	}
}

// A parameter's choices are a caller's own copy.
func TestChoicesAreACopy(t *testing.T) {
	for _, scene := range named.All() {
		for _, one := range scene.Parameters() {
			choices := one.Choices()
			if len(choices) == 0 {
				continue
			}
			choices[0] = "changed"
			if slices.Contains(one.Choices(), "changed") {
				t.Errorf("%s's %s choices changed from outside",
					scene.Name(), one.Name())
			}
		}
	}
}

// containsAll is whether the text holds every one of the words.
func containsAll(text string, words ...string) bool {
	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}
	return true
}
