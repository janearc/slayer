//go:build darwin

package gpu

import (
	_ "embed"
	"fmt"
	"strings"
)

// restirSource is what restir changes of trace.metal, embedded as two
// parts: a prelude that goes before `falling` and the passes that go at
// the end (restir.metal).
//
// restir does not edit trace.metal. It is put into the kernel's text
// here, at the few places it must be, only for the restir mode, so every
// other mode compiles byte for byte the text it compiled without restir
// (TestNoOpticsShaderSourceMatchesBaselineWithSpectralNormals holds the
// legacy text to its hash, and restir_source_test.go holds the places
// below to be found once each).
//
//go:embed restir.metal
var restirSource string

// A restirEdit is one place in the kernel's text, said as the text there
// now and what it becomes.
type restirEdit struct {
	name, anchor, into string
}

// restirParts splits the embedded text at its `//// name` lines.
func restirParts() map[string]string {
	parts := map[string]string{}
	name := ""
	for _, line := range strings.SplitAfter(restirSource, "\n") {
		if rest, found := strings.CutPrefix(line, "//// "); found {
			name = strings.TrimSpace(rest)
			continue
		}
		parts[name] += line
	}
	return parts
}

// restirEdits is the places, in the order they are made.
func restirEdits() []restirEdit {
	parts := restirParts()
	const fall = "static float3 falling(World world, float3 point, float3 normal,\n" +
		"                      int leaving_sphere) {\n"
	return []restirEdit{
		{"the world's two fields", "\tthread uint *work;\n#endif\n};\n",
			"\tthread uint *work;\n#endif\n" +
				"\t// restir (doc/restir.md): whether the primary hit's lamp comes\n" +
				"\t// from this sample's reservoir, and where it is among the\n" +
				"\t// rays' floats. Nought, as every initializer leaves it, is off.\n" +
				"\tint restir_on;\n\tint restir_at;\n};\n"},
		{"falling takes the reservoir's lamp", fall + "\tfloat3 sum = skylit(world, point, normal, leaving_sphere);\n",
			parts["prelude"] + fall +
				"\tif (world.restir_on != 0) {\n" +
				"\t\treturn falling_restir(world, point, normal, leaving_sphere);\n\t}\n" +
				"\tfloat3 sum = skylit(world, point, normal, leaving_sphere);\n"},
		{"only the primary hit reads its reservoir",
			"\tSeen seen = surfaced(world, hit, point, normal);\n\tfloat clearness = clamp(made.mirror.z, 0.0f, 1.0f);\n",
			"\tSeen seen = surfaced(world, hit, point, normal);\n" +
				"\t// later bounces light as ever\n\tworld.restir_on = 0;\n" +
				"\tfloat clearness = clamp(made.mirror.z, 0.0f, 1.0f);\n"},
		{"the trace kernel names its reservoir", "\tworld.shared = sky_share(header, pixel);\n",
			"\tworld.shared = sky_share(header, pixel);\n\tworld.restir_on = 1;\n" +
				"\tworld.restir_at = rc_int(rays, rc_res_b) + 4 * restir_pixel(rays, pixel);\n"},
	}
}

// restirKernel is the kernel's text with restir in it, or an error that
// names the place that is not there exactly once.
func restirKernel(text string) (string, error) {
	for _, edit := range restirEdits() {
		if count := strings.Count(text, edit.anchor); count != 1 {
			return "", fmt.Errorf("gpu: restir: %s: the kernel has %d "+
				"places to do it, want one; trace.metal changed under restir",
				edit.name, count)
		}
		text = strings.Replace(text, edit.anchor, edit.into, 1)
	}
	return text + "\n" + restirParts()["kernels"], nil
}
