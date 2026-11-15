package manufacture

import (
	"strings"
	"testing"
)

// Face on, and what the tests below hold a surface's mirroring to.
const (
	faceOn       = 1.0  // defined: the cosine of a look straight at a surface
	goldRedOver  = 0.9  // picked: a bound under the 0.93 of red Johnson and Christy's gold mirrors face on
	goldBlueTo   = 0.5  // picked: a bound over the 0.40 of blue it mirrors, which is why gold is gold
	silverOver   = 0.85 // picked: a bound under the 0.91 or so of every colour Rakić's aluminium mirrors
	greyWithin   = 0.05 // picked: aluminium's three colours agree this nearly, and gold's do not
	surfacesFile = "webb-surfaces.yaml"
)

// dressedWebb is Webb's looks with its surfaces file laid over them.
func dressedWebb(t *testing.T, raw []byte) looks {
	t.Helper()
	dressed := spacecraftLooks("webb")
	if err := webbSurfaces(raw, dressed); err != nil {
		t.Fatal(err)
	}
	return dressed
}

// Webb's mirrors are gold: a conductor that mirrors nearly all of red and
// under half of blue face on, from gold's measured optical constants.
func TestWebbsMirrorsAreGold(t *testing.T) {
	dressed := dressedWebb(t, webbSurfacesRaw)
	for name, where := range map[string]slot{"primary": spaceMirror,
		"secondary": spaceSecondary} {
		look := dressed[where]
		if look.Metal == nil {
			t.Fatalf("webb's %s mirror is no metal", name)
		}
		mirrors := look.Reflects(faceOn)
		if mirrors.R < goldRedOver || mirrors.B > goldBlueTo {
			t.Errorf("webb's %s mirror mirrors %+v face on: not gold", name,
				mirrors)
		}
	}
}

// Webb's sunshield is aluminium: a conductor that mirrors most of every
// colour alike. It is the anti-test of the gold above: the same reading
// that calls the mirror gold calls the shield not gold.
func TestWebbsSunshieldIsAluminium(t *testing.T) {
	look := dressedWebb(t, webbSurfacesRaw)[spaceFoil]
	if look.Metal == nil {
		t.Fatal("webb's sunshield is no metal")
	}
	mirrors := look.Reflects(faceOn)
	if mirrors.R < silverOver || mirrors.B < silverOver ||
		mirrors.R-mirrors.B > greyWithin || mirrors.B-mirrors.R > greyWithin {
		t.Errorf("webb's sunshield mirrors %+v face on: not aluminium", mirrors)
	}
}

// What the file does not name keeps its picked look, and no other
// spacecraft is touched: Webb's structure is no metal, and Hubble's
// mirror, which has no surfaces file, is none either.
func TestOnlyWhatWebbsFileNamesIsRedressed(t *testing.T) {
	dressed := dressedWebb(t, webbSurfacesRaw)
	if dressed[spaceStructure].Metal != nil {
		t.Error("webb's structure became a metal though the file does not name it")
	}
	if spacecraftLooks("hubble")[spaceMirror].Metal != nil {
		t.Error("hubble's mirror became a metal")
	}
	if spacecraftLooks("webb")[spaceMirror].Metal != nil {
		t.Error("webb's picked look is a metal before its file is laid over it")
	}
}

// A surfaces file that cannot be built is refused, by the name of what
// is wrong with it.
func TestAWebbSurfaceThatCannotBeMadeIsRefused(t *testing.T) {
	whole := string(webbSurfacesRaw)
	refused := map[string]struct{ was, is, names string }{
		"a part webb has not":    {"part: sunshield", "part: warp nacelle", "warp nacelle"},
		"a material not known":   {"material: aluminium", "material: unobtainium", "unobtainium"},
		"a finish not known":     {"finish: satin", "finish: sparkly", "sparkly"},
		"a part named twice":     {"part: secondary mirror", "part: primary mirror", "twice"},
		"a file that is no yaml": {"surfaces:", "surfaces: [", surfacesFile},
	}
	for name, each := range refused {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		broken := strings.Replace(whole, each.was, each.is, 1)
		err := webbSurfaces([]byte(broken), spacecraftLooks("webb"))
		if err == nil {
			t.Errorf("%s: built", name)
			continue
		}
		if !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: refused as %q, which does not name %q", name, err,
				each.names)
		}
	}
}
