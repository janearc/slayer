package session

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// worldSet is a set drawn from its world half alone, whose film says
// the given line, which may be left out.
func worldSet(t *testing.T, filmLine string) set.Set {
	t.Helper()
	text := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: a world alone
  setting: space
  light: none
  FILM
  cameras:
    view:
      title: view
      location: {at: [0, -5, 0]}
      orientation: {look_at: [0, 0, 0]}
  world:
    unit: 1
    binaries:
      pair:
        generate:
          primary: {mass: 2.45, age: 1.2e7}
          companion: {mass: 0.62, temperature: 25000, radius: 0.0122}
          separation: 9.2
          eccentricity: 0.16
        middle: [0, 0, 0]
        time: 0
`
	text = strings.Replace(text, "FILM", filmLine, 1)
	read, err := set.Read("world-alone.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// A set drawn from its world half alone is developed at the exposure its
// film says, as a set with things is, and a job that says one overrides
// it. The anti-test: a film that says none, and a job that says none,
// leave the picture developed as hillside does, with no print of its own.
func TestAWorldAlonesFilmExposureDevelopsThePicture(t *testing.T) {
	film := "film: {width: 8, height: 8, rays: 1, exposure: 0.25}"
	plain := asked{set: "world-alone.yaml"}.sized(8, 8, 1)
	drawn := stagedWorld(worldSet(t, film), plain)
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if drawn.given.print == nil || drawn.given.print.Exposure != 0.25 {
		t.Errorf("the film's exposure of 0.25 gave a print of %v",
			drawn.given.print)
	}
	job := plain
	job.exposure = 2
	job.said.exposure = true
	drawn = stagedWorld(worldSet(t, film), job)
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if drawn.given.print == nil || drawn.given.print.Exposure != 2 {
		t.Errorf("the job's exposure of 2 gave a print of %v",
			drawn.given.print)
	}
	bare := "film: {width: 8, height: 8, rays: 1}"
	drawn = stagedWorld(worldSet(t, bare), plain)
	if drawn.err != nil {
		t.Fatal(drawn.err)
	}
	if drawn.given.print != nil {
		t.Errorf("a film with no exposure gave a print of %v",
			drawn.given.print)
	}
}
