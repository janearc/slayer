package look

import (
	"fmt"
	"math"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Divider starts every look in a looks file, so a file of looks is a stream
// of YAML documents a person can also read raw.
const Divider = "---"

// percent is the factor from a share of the picture to its percentage, and
// from a coordinate to hundredths.
const percent = 100

// Render is the look as one self-contained block of text: it starts with the
// divider, its comments say what each number means, and every string is
// quoted so that it reads back as YAML. The note a person might type is not
// part of a look yet and is said to be absent.
func Render(described Look) string {
	var text strings.Builder
	line := func(format string, arguments ...any) { fmt.Fprintf(&text, format+"\n", arguments...) }
	line("%s", Divider)
	line("look: %s", described.When.UTC().Format(time.RFC3339))
	line("draft_sha256: %s", described.DraftSHA256)
	line("file: %s", quote(described.File))
	line("display: %s", quote(described.Display))
	line("composition: %s", quote(described.Composition))
	line("units: %s", quote("metres; every point is [east, north, up] in the set's own coordinates"))
	renderCamera(line, described.Camera)
	renderSelected(line, described.Selected)
	renderPointer(line, described.Pointer)
	renderSeen(line, described)
	line("note: %s", quote("not recorded: this window has no field to type one in"))
	return text.String()
}

// quote is text as one YAML scalar on one line.
func quote(text string) string {
	raw, err := yaml.Marshal(text)
	if err != nil {
		return `"unprintable"`
	}
	return strings.Join(strings.Fields(strings.TrimSpace(string(raw))), " ")
}

// point is three coordinates as a flow sequence, each to the hundredth.
func point(at [3]float64) string {
	return fmt.Sprintf("[%.2f, %.2f, %.2f]", hundredth(at[0]), hundredth(at[1]), hundredth(at[2]))
}

// hundredth is value rounded to the hundredth. Adding zero turns a negative
// zero into a plain one, so no coordinate prints as -0.00.
func hundredth(value float64) float64 { return math.Round(value*percent)/percent + 0 }

// renderCamera writes the camera section.
func renderCamera(line func(string, ...any), camera Camera) {
	if camera.Why != "" {
		line("camera: %s", quote(camera.Why))
		return
	}
	line("camera:")
	line("  eye: %s", point(camera.Eye))
	line("  focus: %s", point(camera.Focus))
	line("  field_of_view_degrees: %.1f  # across the whole picture", camera.WideDegrees)
	line("  picture: %s  # the set's film size", quote(fmt.Sprintf("%dx%d", camera.Width, camera.Height)))
}

// renderSelected writes the selected thing, or says none is selected.
func renderSelected(line func(string, ...any), chosen *Selected) {
	if chosen == nil {
		line("selected: none")
		return
	}
	line("selected:")
	line("  name: %s", quote(chosen.Name))
	line("  kind: %s", quote(chosen.Kind))
	if chosen.SetKnown {
		line("  recipe: %s", quote(chosen.Recipe))
		line("  seed: %d", chosen.Seed)
		line("  detail: %s", quote(chosen.Detail))
	} else {
		line("  recipe: %s", quote("not known: the set could not be read"))
	}
	line("  stands: %s  # where the display has it now", point(chosen.Stands))
	if chosen.HeadingKnown {
		line("  heading_degrees: %.1f  # clockwise from north, as the set says", chosen.Heading)
	} else {
		line("  heading_degrees: %s", quote("not known: the set gives it none"))
	}
}

// renderPointer writes what the pointer's last place on the picture meets.
func renderPointer(line func(string, ...any), pointer Pointer) {
	if pointer.At == [2]float64{} && pointer.Why != "" {
		line("pointer: %s", quote(pointer.Why))
		return
	}
	line("pointer:")
	line("  at_picture: [%.3f, %.3f]  # from the left and the top, as fractions; where it last was over the picture", pointer.At[0], pointer.At[1])
	if pointer.Why != "" {
		line("  over: %s", quote(pointer.Why))
		return
	}
	if pointer.Over == "" {
		line("  over: %s", quote("no thing under the pointer"))
		return
	}
	line("  over: %s", quote(pointer.Over))
	line("  point: %s  # where its ray meets the surface", point(pointer.Point))
	line("  distance_m: %.2f  # from the eye", pointer.Distance)
	if pointer.Part == "" {
		line("  part: %s", quote("not known: the mesh names no material for this part"))
		return
	}
	line("  part: %s  # the material of the part", quote(pointer.Part))
}

// renderSeen writes the things in view.
func renderSeen(line func(string, ...any), described Look) {
	if described.SeenWhy != "" {
		line("in_view: %s", quote(described.SeenWhy))
		return
	}
	line("in_view:  # nearest first; share is of the picture, from each thing's bounding box, not pixels")
	if len(described.Seen) == 0 {
		line("  []")
	}
	for _, each := range described.Seen {
		line("  - name: %s", quote(each.Name))
		line("    kind: %s", quote(each.Kind))
		line("    recipe: %s", quote(each.Recipe))
		line("    distance_m: %.2f", each.Distance)
		line("    share_percent: %.2f", each.Share*percent)
	}
	line("more_in_view: %d", described.More)
}
