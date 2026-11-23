package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// v1Label and v2Label are the lines a set file says its schema on.
const (
	v1Label = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v1.json`
	v2Label = `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json`
)

// smallScene is the least a set says, under a $schema line of its own.
const smallScene = `scene:
  name: small
  light: ambient
  film: {width: 64, height: 36, rays: 1, exposure: 1}
  cameras:
    only:
      title: the only
      orientation: {look_at: [0, 0, 2]}
      location: {at: [0, -5, 2]}
`

// The later focus readout must preserve the authored target under either
// schema, independently of the historical v1 digest projection.
func TestCameraFocusPreservesAuthoredTargetAcrossSchemas(t *testing.T) {
	for _, schema := range []string{v1Label, v2Label} {
		made, err := set.Read("focus.yaml", []byte(schema+"\n"+smallScene))
		if err != nil {
			t.Fatal(err)
		}
		shot, err := made.Shot("only", nil)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := shot.Focus(), maths.V(0, 2, 0); got != want {
			t.Errorf("%s focus = %v, want %v", schema, got, want)
		}
	}
}

// The four scenes written against v2 are read whole.
func TestV2ScenesAreRead(t *testing.T) {
	for _, name := range []string{"doc/corona.yaml",
		"doc/orbital-and-lantern.yaml", "set/testdata/webb-looking-out.yaml",
		"set/testdata/keyhole-to-ground.yaml"} {
		if _, err := set.Read(name, draft(t, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// A set file that says the version 2 schema is checked against it, and
// one that says version 1 against version 1, so a key only version 2 has
// is refused by version 1's check, by the schema it names.
func TestTheSchemaACheckIsAgainstIsTheOneTheFileNames(t *testing.T) {
	good := v2Label + "\n" + smallScene
	if _, err := set.Read("good.yaml", []byte(good)); err != nil {
		t.Errorf("a small version 2 set is refused: %v", err)
	}
	for why, each := range map[string]struct{ text, want string }{
		"an unknown key under version 2": {
			v2Label + "\n" + smallScene + "  mood: grey\n",
			"ecosystem-v2.json#"},
		"a version 2 key under version 1": {
			v1Label + "\n" + smallScene + "  setting: space\n",
			"ecosystem-v1.json#"},
		"a schema nobody has": {
			`"$schema": https://example.test/other.json` + "\n" + smallScene,
			"ecosystem-v1.json#"},
	} {
		_, err := set.Read("bad.yaml", []byte(each.text))
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: read says %v, want a refusal by %s", why, err,
				each.want)
		}
	}
}

// A version 1 file that says the version 2 schema reads exactly as it did
// under version 1: the same set, field for field, the same shots, so
// version 2 is a superset in what the reader makes of it, not only in the
// schema's keys.
func TestVersionOneRelabelledVersionTwoReadsTheSame(t *testing.T) {
	relabelled := 0
	for path, source := range baselineSources(t) {
		text := string(source)
		if !strings.Contains(text, v1Label) {
			continue
		}
		was := readDump(path, source)
		if strings.HasPrefix(was, "refused") {
			continue
		}
		retitled := strings.Replace(text, v1Label, v2Label, 1)
		now := readDump(path, []byte(retitled))
		if now != was {
			t.Errorf("%s: relabelled version 2, it reads differently", path)
		}
		relabelled++
	}
	if relabelled == 0 {
		t.Error("no version 1 file says its schema, so nothing was compared")
	}
}
