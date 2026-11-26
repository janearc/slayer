package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestAlgaProfileStagesOnLandAndInSpace(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/algae/kelp/kelp.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "kelp.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, setting, placement string
	}{
		{"space", "  setting: space\n  light: none\n  world: {unit: 1}\n", "east: 0, north: 0, height: {value: 0, unit: m}"},
		{"land", "", "position: {east: {value: 0, unit: m}, north: {value: 0, unit: m}, up: {value: 0, unit: m}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: kelp
` + tc.setting + `  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {alga: {family: kelp, file: kelp.yaml, species: macrocystis-pyrifera, condition: moderate}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, ` + tc.placement + `}
`
			read, err := set.Read(filepath.Join(dir, tc.name+".yaml"), []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			stage := testStaging()
			stage.recipes = read
			if tc.name == "space" {
				err = stage.afloat(read)
			} else {
				stage.given = mustParse(t)
				err = stage.stand(read)
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(stage.world.Things) == 0 {
				t.Fatal("alga staged no surfaces")
			}
		})
	}
}
