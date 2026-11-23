package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

func oceanSetText(t *testing.T) string {
	t.Helper()
	text := strings.Replace(built(t),
		"colour: {srgb_linear: [0.05, 0.06, 0.04]}",
		"clear: {through: 1, index: 1.33, gloss: 1, rank: 0}", 1)
	return strings.Replace(text, "    materials:\n", "    ocean:\n"+
		"      material: grass\n"+
		"      middle: [10, 20, 30]\n"+
		"      extent: {value: 64, unit: m}\n"+
		"      samples: 9\n"+
		"      sea:\n"+
		"        wind: {value: 0, unit: m/s}\n"+
		"        fetch: {value: 0, unit: m}\n"+
		"        depth: {value: 0, unit: m}\n"+
		"        toward: {value: 0, unit: rad}\n"+
		"      seed: 7\n"+
		"      time: {value: 0, unit: s}\n"+
		"      source: eye\n"+
		"    materials:\n", 1)
}

func TestOceanInterfaceEventsReadOnlyForExactContinuation(t *testing.T) {
	text := strings.Replace(oceanSetText(t), "      material: grass\n",
		"      material: grass\n      interface-events: 2\n"+
			"      flat-mean:\n"+
			"        mean-level: {value: 0, unit: m}\n"+
			"        collar-width: {value: 16, unit: m}\n"+
			"        extends-to-infinity: true\n", 1)
	read, err := set.Read("ocean-events.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	ocean, _ := world.OceanSurface()
	if ocean.InterfaceEvents() != 2 {
		t.Fatalf("ocean event budget=%d, want 2", ocean.InterfaceEvents())
	}
	for _, bad := range []string{"-1", "1.5", "17"} {
		broken := strings.Replace(text, "interface-events: 2",
			"interface-events: "+bad, 1)
		if _, err := set.Read("bad-events.yaml", []byte(broken)); err == nil {
			t.Errorf("accepted interface-events %s", bad)
		}
	}
	finite := strings.Replace(text, "extends-to-infinity: true",
		"extends-to-infinity: false", 1)
	if _, err := set.Read("finite-events.yaml", []byte(finite)); err == nil {
		t.Fatal("accepted repeated interface events on legacy finite geometry")
	}
	source := oceanSetText(t)
	legacy, err := set.Read("legacy-ocean.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	legacyWorld, _ := legacy.World()
	legacyOcean, _ := legacyWorld.OceanSurface()
	if legacyOcean.InterfaceEvents() != 0 {
		t.Fatal("default ocean gained interface traversal events")
	}
}

func TestWorldReadsAuthoredOceanSurface(t *testing.T) {
	text := oceanSetText(t)
	made, err := set.Read("ocean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	world, ok := made.World()
	if !ok {
		t.Fatal("world missing")
	}
	surface, ok := world.OceanSurface()
	if !ok || surface.Material() != "grass" || surface.Middle() != maths.V(10, 30, -20) ||
		surface.Extent() != 64 || surface.Samples() != 9 || surface.Seed() != 7 || surface.Seconds() != 0 {
		t.Fatalf("surface = %+v, present %t", surface, ok)
	}
	if surface.Source() != "eye" {
		t.Fatalf("source = %q, want eye", surface.Source())
	}
	if wind, fetch, depth, toward := surface.Sea(); wind != 0 || fetch != 0 || depth != 0 || toward != 0 {
		t.Fatalf("sea = %g, %g, %g, %g", wind, fetch, depth, toward)
	}
	if got, want := world.Order(), []string{"ocean", "materials", "boxes", "lamps", "floods", "sky", "bodies", "suns"}; !equalStrings(got, want) {
		t.Fatalf("world order = %v, want %v", got, want)
	}
}

func TestWorldReadsOptInSceneClockOceanMotion(t *testing.T) {
	text := strings.Replace(oceanSetText(t), "    ocean:\n",
		"    ocean:\n      motion: scene-clock\n", 1)
	v2 := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n"
	read, err := set.Read("scene-clock-ocean.yaml", []byte(v2+text))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	surface, has := world.OceanSurface()
	if !has || surface.Motion() != "scene-clock" {
		t.Fatalf("v2 ocean motion = %q, present %t", surface.Motion(), has)
	}
	if _, err := set.Read("scene-clock-ocean-v1.yaml", []byte(text)); err == nil || !strings.Contains(err.Error(), "not a key here") {
		t.Fatalf("v1 motion error = %v, want explicit version-two refusal", err)
	}
	for _, mode := range []string{"frozen", "camera", "true"} {
		bad := strings.Replace(text, "motion: scene-clock", "motion: "+mode, 1)
		if _, err := set.Read("bad-ocean-motion.yaml", []byte(v2+bad)); err == nil {
			t.Errorf("accepted unsupported ocean motion %q", mode)
		}
	}
}

func TestWorldReadsShotScopedSurfaceWakeAndShadingChoice(t *testing.T) {
	text := strings.Replace(oceanSetText(t),
		"      time: {value: 0, unit: s}\n      source: eye\n",
		"      time: {value: 0, unit: s}\n"+
			"      shading-normal: spectral-slopes\n"+
			"      surface-wake:\n"+
			"        origin: [10, 20, 30]\n"+
			"        heading: {value: 1.2, unit: rad}\n"+
			"        speed: {value: 6, unit: m/s}\n"+
			"        amplitude: {value: 0.12, unit: m}\n"+
			"        half-width: {value: 7, unit: m}\n"+
			"        length: {value: 48, unit: m}\n"+
			"        wavelength: {value: 16, unit: m}\n"+
			"        add-to-spectral-shading: true\n"+
			"        foam:\n"+
			"          coverage: 0.9\n"+
			"          band: 0.3\n"+
			"          reflectance: [0.7, 0.8, 0.9]\n"+
			"          satin: 0.8\n"+
			"      source: eye\n", 1)
	v2 := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n"
	read, err := set.Read("surface-wake.yaml", []byte(v2+text))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	surface, _ := world.OceanSurface()
	wake, has := surface.SurfaceWake()
	if !has || wake.Origin() != [3]float64{10, 20, 30} ||
		wake.HeadingRadians() != 1.2 || wake.SpeedMetresPerSecond() != 6 ||
		wake.AmplitudeMetres() != .12 || wake.HalfWidthMetres() != 7 ||
		wake.LengthMetres() != 48 || wake.WavelengthMetres() != 16 ||
		!wake.AddToSpectralShading() {
		t.Fatalf("surface wake = %+v, present %t", wake, has)
	}
	foam, hasFoam := wake.Foam()
	if !hasFoam || foam.Coverage() != .9 || foam.Band() != .3 ||
		foam.Reflectance() != [3]float64{.7, .8, .9} || foam.Satin() != .8 {
		t.Fatalf("surface wake foam = %+v, present %t", foam, hasFoam)
	}
	withoutSlope := strings.Replace(text, "        add-to-spectral-shading: true", "        add-to-spectral-shading: false", 1)
	if _, err := set.Read("surface-wake-without-slope.yaml", []byte(v2+withoutSlope)); err == nil || !strings.Contains(err.Error(), "requires surface-wake add-to-spectral-shading") {
		t.Fatalf("spectral wake without its explicit slopes gave %v", err)
	}
}

func TestWorldReadsAuthoredOceanSurfaceUnderVersionTwoSchema(t *testing.T) {
	text := `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"
` + oceanSetText(t)
	made, err := set.Read("ocean-v2.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	world, has := made.World()
	surface, hasSurface := world.OceanSurface()
	if !has || !hasSurface || surface.Material() != "grass" ||
		surface.Middle() != maths.V(10, 30, -20) || surface.Samples() != 9 {
		t.Fatalf("v2 world/ocean = %+v/%+v, present %t/%t", world, surface, has, hasSurface)
	}
}

func TestWorldReadsDiffuseSkyInscatterBudgetOnlyWithInfinitePlane(t *testing.T) {
	text := strings.Replace(oceanSetText(t),
		"      middle: [10, 20, 30]\n",
		"      direct-sun: mean-plane\n"+
			"      diffuse-sky-samples: 32\n"+
			"      clear-cloud-sky-samples: 16\n"+
			"      flat-mean:\n"+
			"        mean-level: {value: 0, unit: m}\n"+
			"        collar-width: {value: 16, unit: m}\n"+
			"        extends-to-infinity: true\n"+
			"      middle: [10, 20, 30]\n", 1)
	for _, schema := range []string{"v1", "v2"} {
		t.Run(schema, func(t *testing.T) {
			written := text
			if schema == "v2" {
				written = `"$schema": "https://github.com/janearc/slayer/schemas/ecosystem-v2.json"` + "\n" + written
			}
			read, err := set.Read("diffuse-sky-ocean.yaml", []byte(written))
			if err != nil {
				t.Fatal(err)
			}
			world, _ := read.World()
			surface, _ := world.OceanSurface()
			if surface.DiffuseSkySamples() != 32 || surface.ClearCloudSkySamples() != 16 {
				t.Fatalf("diffuse/cloud sky samples=%d/%d, want 32/16", surface.DiffuseSkySamples(), surface.ClearCloudSkySamples())
			}
		})
	}
	for _, bad := range []struct {
		name, old, replacement string
	}{
		{"finite mean collar", "extends-to-infinity: true", "extends-to-infinity: false"},
	} {
		broken := strings.Replace(text, bad.old, bad.replacement, 1)
		if _, err := set.Read("bad-diffuse-sky-ocean.yaml", []byte(broken)); err == nil {
			t.Errorf("accepted diffuse-sky ocean %s", bad.name)
		}
	}
	withoutSun := strings.Replace(text, "      direct-sun: mean-plane\n", "", 1)
	if _, err := set.Read("diffuse-sky-without-sun.yaml",
		[]byte(withoutSun)); err != nil {
		t.Errorf("sky-only ocean should not require a direct sun: %v", err)
	}
}

func TestOceanShadingNormalApproximationIsExplicitInBothSchemas(t *testing.T) {
	text := strings.Replace(oceanSetText(t),
		"      time: {value: 0, unit: s}\n      source: eye\n",
		"      time: {value: 0, unit: s}\n      source: eye\n"+
			"      shading-normal: "+
			arid.SmoothNormalBilinearVertexSlopes+"\n", 1)
	for _, schema := range []string{"", "\"$schema\": \"https://github.com/janearc/slayer/schemas/ecosystem-v2.json\"\n"} {
		written := text
		if schema != "" {
			written = schema + written
		}
		read, err := set.Read("ocean-shading-normal.yaml", []byte(written))
		if err != nil {
			t.Fatalf("schema %q: %v", schema, err)
		}
		world, _ := read.World()
		surface, has := world.OceanSurface()
		if !has || surface.ShadingNormalApproximation() !=
			arid.SmoothNormalBilinearVertexSlopes {
			t.Errorf("schema %q: shading-normal = %q, present %t", schema,
				surface.ShadingNormalApproximation(), has)
		}
	}
	for _, value := range []string{"smooth", "exact", "true"} {
		bad := strings.Replace(text,
			"shading-normal: "+arid.SmoothNormalBilinearVertexSlopes,
			"shading-normal: "+value, 1)
		if _, err := set.Read("bad-ocean-shading-normal.yaml", []byte(bad)); err == nil {
			t.Errorf("accepted shading-normal %q", value)
		}
	}
	withFlatMean := strings.Replace(text, "      shading-normal:",
		"      flat-mean:\n        mean-level: {value: 0, unit: m}\n"+
			"        collar-width: {value: 2, unit: m}\n      shading-normal:", 1)
	if _, err := set.Read("bad-ocean-shading-normal-flat-mean.yaml",
		[]byte(withFlatMean)); err == nil ||
		!strings.Contains(err.Error(), "cannot be combined with flat-mean") {
		t.Errorf("smooth normals with flat-mean gave %v", err)
	}
}

func TestSpectralSlopeShadingNormalIsExplicitInBothSchemas(t *testing.T) {
	text := strings.Replace(oceanSetText(t),
		"      time: {value: 0, unit: s}\n      source: eye\n",
		"      time: {value: 0, unit: s}\n      source: eye\n"+
			"      shading-normal: "+arid.SpectralSlopeNormals+"\n", 1)
	for _, schema := range []string{"", "\"$schema\": \"https://github.com/janearc/slayer/schemas/ecosystem-v2.json\"\n"} {
		written := text
		if schema != "" {
			written = schema + written
		}
		read, err := set.Read("ocean-spectral-normal.yaml", []byte(written))
		if err != nil {
			t.Fatalf("schema %q: %v", schema, err)
		}
		world, _ := read.World()
		surface, has := world.OceanSurface()
		if !has || surface.ShadingNormalApproximation() != arid.SpectralSlopeNormals {
			t.Errorf("schema %q: shading-normal = %q, present %t", schema,
				surface.ShadingNormalApproximation(), has)
		}
	}
	withFlatMean := strings.Replace(text, "      shading-normal:",
		"      flat-mean:\n        mean-level: {value: 0, unit: m}\n"+
			"        collar-width: {value: 2, unit: m}\n      shading-normal:", 1)
	if _, err := set.Read("bad-ocean-spectral-normal-flat-mean.yaml",
		[]byte(withFlatMean)); err == nil ||
		!strings.Contains(err.Error(), "cannot be combined with flat-mean") {
		t.Errorf("spectral normals with flat-mean gave %v", err)
	}
}

func TestWorldReadsBoundedFlatMeanOceanContinuation(t *testing.T) {
	text := strings.Replace(oceanSetText(t),
		"      time: {value: 0, unit: s}\n      source: eye\n",
		"      time: {value: 0, unit: s}\n      source: eye\n"+
			"      flat-mean:\n"+
			"        mean-level: {value: -0.5, unit: m}\n"+
			"        collar-width: {value: 8, unit: m}\n", 1)
	read, err := set.Read("ocean-flat-mean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	surface, _ := world.OceanSurface()
	continuation, has := surface.FlatMean()
	if !has || continuation.MeanLevelMetres() != -.5 ||
		continuation.CollarWidthMetres() != 8 {
		t.Fatalf("flat-mean = %+v, present %t", continuation, has)
	}
	for _, change := range []struct{ was, now string }{
		{"collar-width: {value: 8, unit: m}", "collar-width: {value: 65, unit: m}"},
		{"collar-width: {value: 8, unit: m}", "collar-width: {value: 0, unit: m}"},
		{"mean-level: {value: -0.5, unit: m}", "mean-level: {value: 1, unit: km}"},
		{"collar-width: {value: 8, unit: m}", "collar-wdith: {value: 8, unit: m}"},
	} {
		bad := strings.Replace(text, change.was, change.now, 1)
		if _, err := set.Read("bad-ocean-flat-mean.yaml", []byte(bad)); err == nil {
			t.Errorf("accepted invalid flat-mean change %q => %q", change.was, change.now)
		}
	}
}

func TestWorldReadsInfiniteMeanPlaneAsExplicitFlatMeanOption(t *testing.T) {
	text := strings.Replace(oceanSetText(t),
		"      time: {value: 0, unit: s}\n      source: eye\n",
		"      time: {value: 0, unit: s}\n      source: eye\n"+
			"      flat-mean:\n"+
			"        mean-level: {value: 0, unit: m}\n"+
			"        collar-width: {value: 4, unit: m}\n"+
			"        extends-to-infinity: true\n", 1)
	read, err := set.Read("ocean-infinite-mean.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	surface, _ := world.OceanSurface()
	continuation, has := surface.FlatMean()
	if !has || !continuation.ExtendsToInfinity() ||
		continuation.CollarWidthMetres() != 4 {
		t.Fatalf("infinite mean-plane = %+v, present %t", continuation, has)
	}
	edited := strings.Replace(text, "extends-to-infinity: true",
		"extends-to-infinity: yes", 1)
	if _, err := set.Read("bad-ocean-infinite-mean.yaml",
		[]byte(edited)); err == nil {
		t.Fatal("accepted a non-boolean infinite mean-plane option")
	}
}

func TestOceanDirectSunMeanPlaneIsExplicitAndSchemaSupported(t *testing.T) {
	text := strings.Replace(oceanSetText(t), "    ocean:\n",
		"    ocean:\n      direct-sun: mean-plane\n", 1)
	for _, schema := range []string{"", "\"$schema\": \"https://github.com/janearc/slayer/schemas/ecosystem-v2.json\"\n"} {
		written := text
		if schema != "" {
			written = schema + written
		}
		read, err := set.Read("direct-sun.yaml", []byte(written))
		if err != nil {
			t.Fatalf("schema %q: %v", schema, err)
		}
		world, _ := read.World()
		surface, has := world.OceanSurface()
		if !has || surface.DirectSunMode() != "mean-plane" {
			t.Fatalf("schema %q: direct sun mode = %q, present %t",
				schema, surface.DirectSunMode(), has)
		}
	}
	for _, mode := range []string{"off", "wavy", "true"} {
		bad := strings.Replace(text, "direct-sun: mean-plane",
			"direct-sun: "+mode, 1)
		if _, err := set.Read("bad-direct-sun.yaml", []byte(bad)); err == nil {
			t.Errorf("accepted unsupported direct-sun mode %q", mode)
		}
	}
}

// glintSetText is the ocean fixture with a glint block said under it,
// with and without the version 2 schema.
func glintSetText(t *testing.T, glint string, two bool) string {
	t.Helper()
	const said = "      time: {value: 0, unit: s}\n      source: eye\n"
	text := strings.Replace(oceanSetText(t), said,
		said+"      glint: "+glint+"\n", 1)
	if two {
		text = "\"$schema\": \"https://github.com/janearc/slayer/schemas/ecosystem-v2.json\"\n" + text
	}
	return text
}

func TestOceanGlintIsReadOnlyWhenSaid(t *testing.T) {
	for _, two := range []bool{false, true} {
		source := glintSetText(t, "{model: artistic, roughness: 0.2}", two)
		read, err := set.Read("glint.yaml", []byte(source))
		if err != nil {
			t.Fatalf("schema two %t: %v", two, err)
		}
		world, _ := read.World()
		surface, _ := world.OceanSurface()
		glint := surface.Glint()
		if glint.Model() != set.GlintArtistic || glint.Roughness() != 0.2 {
			t.Fatalf("schema two %t: glint = %+v", two, glint)
		}
	}
	for _, two := range []bool{false, true} {
		source := glintSetText(t, "{model: cox-munk}", two)
		read, err := set.Read("cox-munk.yaml", []byte(source))
		if err != nil {
			t.Fatalf("schema two %t: %v", two, err)
		}
		world, _ := read.World()
		surface, _ := world.OceanSurface()
		glint := surface.Glint()
		if glint.Model() != set.GlintCoxMunk || glint.Roughness() != 0 {
			t.Fatalf("schema two %t: glint = %+v", two, glint)
		}
	}
	source := oceanSetText(t)
	read, err := set.Read("plain.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	surface, _ := world.OceanSurface()
	if surface.Glint() != (set.OceanGlint{}) {
		t.Fatalf("an ocean that says no glint has %+v", surface.Glint())
	}
}

func TestOceanGlintRefusesWhatItDoesNotSay(t *testing.T) {
	for _, glint := range []string{
		"{roughness: 0.2}",
		"{model: artistic}",
		"{model: artistic, roughness: 0}",
		"{model: artistic, roughness: 1.5}",
		"{model: artistic, roughness: .nan}",
		"{model: shiny, roughness: 0.2}",
		"{model: artistic, roughness: 0.2, sheen: 1}",
		"{model: cox-munk, roughness: 0.2}",
		"{model: cox-munk, wind: 7}",
	} {
		for _, two := range []bool{false, true} {
			text := glintSetText(t, glint, two)
			if _, err := set.Read("bad-glint.yaml", []byte(text)); err == nil {
				t.Errorf("schema two %t: accepted glint %s", two, glint)
			}
		}
	}
}

func TestOceanSurfaceRejectsBadReferencesAndBounds(t *testing.T) {
	valid := oceanSetText(t)
	for _, change := range []struct{ was, now string }{
		{"material: grass", "material: absent"},
		{"material: grass", "material: lit-window"},
		{"samples: 9", "samples: 1"},
		{"samples: 9", "samples: 1025"},
		{"extent: {value: 64, unit: m}", "extent: {value: 0, unit: m}"},
		{"wind: {value: 0, unit: m/s}", "wind: {value: .nan, unit: m/s}"},
		{"toward: {value: 0, unit: rad}", "toward: {value: 7, unit: rad}"},
		{"time: {value: 0, unit: s}", "time: {value: -1, unit: s}"},
		{"source: eye", "surprise: 1"},
	} {
		if !strings.Contains(valid, change.was) {
			t.Fatalf("fixture lacks %q", change.was)
		}
		edited := strings.Replace(valid, change.was, change.now, 1)
		if _, err := set.Read("bad-ocean.yaml", []byte(edited)); err == nil {
			t.Errorf("accepted %q", change.now)
		}
	}
}

func TestSetsWithoutOceanSurfaceKeepTheirWorldOrder(t *testing.T) {
	source := built(t)
	read, err := set.Read("built.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	world, _ := read.World()
	if _, has := world.OceanSurface(); has {
		t.Fatal("legacy world has an ocean")
	}
	if got, want := world.Order(), []string{"materials", "boxes", "lamps", "floods", "sky", "bodies", "suns"}; !equalStrings(got, want) {
		t.Fatalf("legacy world order = %v, want %v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestOceanSpotInscatterIsExplicitAndSchemaSupported(t *testing.T) {
	text := strings.Replace(oceanSetText(t), "      material: grass\n",
		"      material: grass\n      in-scatter: spot-lamps\n", 1)
	for _, schema := range []string{set.EcosystemSchema,
		"https://github.com/janearc/slayer/schemas/ecosystem-v2.json"} {
		written := strings.Replace(text, "scene:\n",
			"$schema: "+schema+"\nscene:\n", 1)
		read, err := set.Read("ocean-spot-scatter.yaml", []byte(written))
		if err != nil {
			t.Fatalf("schema %s: %v", schema, err)
		}
		world, _ := read.World()
		surface, _ := world.OceanSurface()
		if surface.InscatterSource() != "spot-lamps" {
			t.Fatalf("in-scatter source = %q", surface.InscatterSource())
		}
	}
	for _, source := range []string{"flood", "all-lights", "true"} {
		bad := strings.Replace(text, "in-scatter: spot-lamps",
			"in-scatter: "+source, 1)
		if _, err := set.Read("bad-ocean-scatter-source.yaml", []byte(bad)); err == nil {
			t.Errorf("accepted in-scatter source %q", source)
		}
	}
}

func TestWorldReadsClearCloudWaterStepMode(t *testing.T) {
	for _, mode := range []string{"per-gap", "ray-spacing"} {
		text := strings.Replace(oceanSetText(t), "      middle: [10, 20, 30]\n", "      clear-cloud-water-step-mode: "+mode+"\n      middle: [10, 20, 30]\n", 1)
		made, err := set.Read("water-step-mode.yaml", []byte(text))
		if err != nil {
			t.Fatal(err)
		}
		world, _ := made.World()
		ocean, _ := world.OceanSurface()
		if ocean.ClearCloudWaterStepMode() != mode {
			t.Fatalf("mode=%q want=%q", ocean.ClearCloudWaterStepMode(), mode)
		}
	}
	bad := strings.Replace(oceanSetText(t), "      middle: [10, 20, 30]\n", "      clear-cloud-water-step-mode: cheap\n      middle: [10, 20, 30]\n", 1)
	if _, err := set.Read("bad-mode.yaml", []byte(bad)); err == nil {
		t.Fatal("accepted unknown optimization mode")
	}
}
