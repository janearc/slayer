package door

// cache says what the disk caches hold and empties them: for each kind
// declared (diskcache.Kinds), its folder, entries, bytes against its
// cap and the oldest and newest use; with --clear, every kind's entries
// are removed, or one kind's.

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/diskcache"
)

// cacheHelp is cache's page.
const cacheHelp = `# cache

slayer cache says what the disk caches hold: for each kind, its
folder, how many entries and bytes it holds against its cap, and when
its least and most recently used entry was last used.

slayer cache --clear removes every entry of every kind; slayer cache
--clear KIND removes one kind's. A file being written is left to its
writer. With --dry-run it says what --clear would remove and removes
nothing.

The caches live under SLAYER_CACHE where it is set, cache/ under the
renders home (SLAYER_RENDERS) where that is, and the user's cache
folder otherwise. Everything in them is exact: a hit is what a fresh
run computes, byte for byte, and any change to what an entry was made
from is a miss.
`

// clearFlag names cache's switch that empties the caches.
const clearFlag = "clear"

// usedLayout is how cache says when an entry was last used.
const usedLayout = "2006-01-02 15:04"

// mebibyte is a mebibyte in bytes. Defined.
const mebibyte = 1 << 20

// declareCache adds cache to the tool.
func declareCache(made *pzm.Tool, env Env) {
	keeps := verb.NewFacts(verb.WritesTo("the disk caches' folders"),
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light)
	made.Verb("cache", verb.NoShort, "say what the disk caches hold, "+
		"or empty them", cacheHelp, "Example_cache", keeps,
		caching{env: env}.run,
		pzm.Switch(clearFlag, verb.NoShort,
			"remove every entry, or every entry of the kind named"))
}

// caching is the cache verb run under an env.
type caching struct{ env Env }

// run says the caches, or clears them.
func (made caching) run(call pzm.Call) error {
	home, err := diskcache.Home(made.env.lookup)
	if err != nil {
		return err
	}
	kinds, err := kindsOf(call.Args(), call.On(clearFlag))
	if err != nil {
		return err
	}
	for _, kind := range kinds {
		if err := made.each(call.Out(), home, kind, call.On(clearFlag),
			call.DryRun()); err != nil {
			return err
		}
	}
	return nil
}

// kindsOf is the kinds a call names: one with --clear, or every kind.
func kindsOf(args []string, clearing bool) ([]diskcache.Kind, error) {
	switch {
	case len(args) == nothing:
		return diskcache.Kinds(), nil
	case len(args) == one && clearing:
		kind, err := diskcache.KindNamed(args[nothing])
		return []diskcache.Kind{kind}, err
	}
	return nil, fmt.Errorf("cache takes no words, or --clear and one "+
		"kind: slayer cache [--clear [KIND]], not %s",
		strings.Join(args, " "))
}

// each says one kind, or clears it, or says what clearing would.
func (made caching) each(out io.Writer, home string,
	kind diskcache.Kind, clearing, dry bool) error {
	store, err := diskcache.Open(home, kind)
	if err != nil {
		return err
	}
	if clearing && dry {
		usage, err := store.Usage()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "%s: would remove %d entries from %s\n",
			kind.Name(), usage.Entries(), store.Folder())
		return err
	}
	if clearing {
		removed, err := store.Clear()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "%s: removed %d entries from %s\n",
			kind.Name(), removed, store.Folder())
		return err
	}
	usage, err := store.Usage()
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, said(usage))
	return err
}

// said is a kind's usage in two lines: what it holds, and where.
func said(usage diskcache.Usage) string {
	kind := usage.Kind()
	line := fmt.Sprintf("%s: %d entries, %.1f of %.0f MiB (%s)\n",
		kind.Name(), usage.Entries(), float64(usage.Bytes())/mebibyte,
		float64(kind.Cap())/mebibyte, kind.Why())
	if usage.Entries() > nothing {
		line += fmt.Sprintf("  used from %s to %s\n",
			local(usage.Oldest()), local(usage.Newest()))
	}
	return line + "  in " + usage.Folder() + "\n"
}

// local is a time as cache says it, in the machine's own zone.
func local(at time.Time) string { return at.In(time.Local).Format(usedLayout) }
