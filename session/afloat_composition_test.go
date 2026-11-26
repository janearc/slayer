package session

import (
	"testing"

	"github.com/janearc/slayer/set"
)

func TestSpaceSetStagesACompositionRecipe(t *testing.T) {
	source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: post
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far:
      location: {at: [0, -20, 2]}
      orientation: {look_at: [0, 0, 2]}
  recipes:
    post:
      composition:
        inline:
          version: 1
          name: post
          parts:
            - {id: foot, type: leg, at: [0, 0, 0], size: [2, 3, 2], ground: true}
  set:
    things:
      post: {recipe: post, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	read, err := set.Read("post.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	stage := testStaging()
	stage.recipes = read
	if err := stage.afloat(read); err != nil {
		t.Fatal(err)
	}
	if stage.made.things != 1 || len(stage.world.Things) == 0 {
		t.Fatalf("staged %d things with %d surfaces", stage.made.things, len(stage.world.Things))
	}
}
