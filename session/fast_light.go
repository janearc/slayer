package session

import (
	"errors"

	"github.com/janearc/slayer/job"
)

// Fast light is the gpu's approximations of how light is found
// (doc/fast-light.md), each a shortcut with a name: lamp-store, boxed
// mist and smoke lit from a store of each lamp's light, and sky-looks,
// the sky's sixteen looks from a surface shared among a pixel's rays.
//
// A shortcut is off unless something says it. job/README.md, "flags
// and shortcuts", says the order: the shortcut's own word, then the
// fastlight flag, then vibes; each from the command line or the job,
// then from the set. The cpu tracer stays exact, so a shortcut asked
// of it is refused, and so is one asked beside a proof, which replays
// pixels on the exact tracer.

// noSetFlags is the flags of no set: none said.
func noSetFlags(string) (bool, bool) { return false, false }

// switchedBy is what was asked with every flag and shortcut a job says
// noted by name, over any the command line noted.
func (given asked) switchedBy(wanted job.Job) asked {
	said := map[string]bool{}
	for name, on := range given.switches {
		said[name] = on
	}
	for _, each := range job.Switches() {
		if on, has := wanted.Said(each.Name()); has {
			said[each.Name()] = on
		}
	}
	given.switches = said
	return given
}

// flagged is what was asked with the command line's own two flags
// noted, each on where it was given.
func (given asked) flagged() asked {
	said := map[string]bool{}
	for name, on := range map[string]bool{
		job.FlagFastlight: given.flagFastlight,
		job.FlagVibes:     given.flagVibes} {
		if on {
			said[name] = true
		}
	}
	if given.exactSky {
		said[job.ShortcutSkyLooks] = false
	}
	given.switches = said
	return given
}

// shortcut is whether a shortcut is on: by the first of the names that
// decide it which the command line, the job or then the set says.
func (given asked) shortcut(name string,
	setFlag func(string) (bool, bool)) bool {
	for _, each := range job.Above(name) {
		if on, said := given.switches[each]; said {
			return on
		}
		if on, said := setFlag(each); said {
			return on
		}
	}
	return false
}

// resolved is what was asked with each light shortcut worked out, the
// set's flags read by setFlag.
func (given asked) resolved(setFlag func(string) (bool, bool)) asked {
	given.fastLighting = given.shortcut(job.ShortcutLampStore, setFlag)
	given.skyLooks = given.shortcut(job.ShortcutSkyLooks, setFlag)
	given.restir = given.shortcut(job.ShortcutRestir, setFlag)
	return given
}

// What a shortcut is refused with, each by name.
var (
	// errFastOnCPU is a shortcut asked of the cpu tracer.
	errFastOnCPU = errors.New("fast light's shortcuts are drawn by the " +
		"gpu and the cpu tracer stays exact; say renderer: gpu-native, " +
		"or turn fastlight off")
	// errFastThroughGlass is a shortcut asked through a rig's lens.
	errFastThroughGlass = errors.New("fast light's shortcuts require a " +
		"pinhole film; turn fastlight off to draw through this rig")
	// errFastAndCached is lamp-store asked beside the cached smoke grid.
	errFastAndCached = errors.New("lamp-store and volume-lighting: " +
		"cached are two stores for the same light; say one")
	// errFastNamed is a shortcut asked of a scene built in Go by name,
	// which a session draws with the gpu's own traversal and no store.
	errFastNamed = errors.New("fast light's shortcuts draw a set or a " +
		"smoke, not yet a named scene; turn fastlight off")
	// errFastProved is a shortcut asked beside a proof.
	errFastProved = errors.New("a proof replays pixels on the exact " +
		"tracer, which a picture drawn with a shortcut is not; turn " +
		"vibes, fastlight and every shortcut off to prove it")
	// errRestirCached is restir asked beside the cached smoke grid.
	errRestirCached = errors.New("restir and volume-lighting: cached " +
		"are two lightings for one frame; say one")
	// errRestirShutter is restir asked of a film with an open shutter or
	// an aperture, whose rays the kernel makes another way.
	errRestirShutter = errors.New("restir draws a pinhole with a closed " +
		"shutter; turn restir off to draw this film")
	// errSkyAlone is sky-looks asked without lamp-store.
	errSkyAlone = errors.New("sky-looks is drawn by the lamp store's " +
		"mode; turn lamp-store on, or sky-looks off")
)

// fastRefusal is why what was asked cannot be drawn with the shortcuts
// it asks, or nil where it can or where none is asked.
func fastRefusal(given asked) error {
	if given.skyLooks && !given.fastLighting {
		return errSkyAlone
	}
	for _, refusal := range []struct {
		holds bool
		err   error
	}{{given.prove != "", errFastProved},
		{!given.onGPU, errFastOnCPU},
		{given.glass != nil, errFastThroughGlass}} {
		if (given.fastLighting || given.restir) && refusal.holds {
			return refusal.err
		}
	}
	if given.fastLighting && given.cachedLighting {
		return errFastAndCached
	}
	if given.restir && given.cachedLighting {
		return errRestirCached
	}
	if given.restir && given.optics.closing != nil {
		return errRestirShutter
	}
	return nil
}
