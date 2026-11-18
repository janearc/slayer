package door

// cache says what the disk caches hold and empties them: for each kind
// declared (diskcache.Kinds), its folder, entries, bytes against its
// cap and the oldest and newest use; with --clear, every kind's entries
// are removed, or one kind's.

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/treebolt"
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

The trees kind is kept two ways, a file for each tree or one bbolt file,
trees.bolt, and cache says and clears both under the name trees. The one
file is kept to its cap by each run's end: the least recently used trees
go and the file is written fresh, which is what gives the disc back.

The caches live under SLAYER_CACHE, an absolute folder, and nowhere
else; where it is not set, slayer refuses by name. Everything in them
is exact: a hit is what a fresh run computes, byte for byte, and any
change to what an entry was made from is a miss.
`

// clearFlag names cache's switch that empties the caches.
const clearFlag = "clear"

// clearWait is how long a clear of the trees' bolt file waits for a
// process using it, as long as the file's users wait for each other.
const clearWait = 6 * time.Minute

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
	if err := made.folder(out, home, kind, clearing, dry); err != nil {
		return err
	}
	if kind.Name() != treebolt.KindName {
		return nil
	}
	return made.file(out, home, clearing, dry)
}

// file says the trees' bolt file, or clears it, or says what clearing
// would. It is the same kind as the trees folder, kept the other way:
// SLAYER_TREE_STORE picks one of the two, and a clear that left the
// other behind would leave its gigabytes to be found by hand.
func (made caching) file(out io.Writer, home string, clearing,
	dry bool) error {
	path := filepath.Join(home, treebolt.File)
	if clearing && !dry {
		result, err := treebolt.Clear(path, clearWait)
		_, said := fmt.Fprintf(out, "%s: removed %d entries from %s, "+
			"%.1f MiB back\n", treebolt.KindName, result.Dropped, path,
			float64(result.Before-result.After)/mebibyte)
		return errors.Join(err, said)
	}
	usage, err := treebolt.Look(path, treebolt.ListWait)
	if err != nil {
		return err
	}
	if clearing {
		_, err = fmt.Fprintf(out, "%s: would remove %d entries from %s\n",
			treebolt.KindName, usage.Entries, path)
		return err
	}
	_, err = io.WriteString(out, saidFile(usage))
	return err
}

// saidFile is the bolt file's usage in the lines a kind's takes: what
// it holds, on disc and live, against its cap, when it was used, and
// where.
func saidFile(usage treebolt.Usage) string {
	line := fmt.Sprintf("%s (one file): %d entries, %.1f of %.0f MiB on "+
		"disc, %.1f MiB of them live (%s)\n", treebolt.KindName,
		usage.Entries, float64(usage.Disc)/mebibyte,
		float64(treebolt.Cap)/mebibyte, float64(usage.Live)/mebibyte,
		treebolt.Why)
	if !usage.Oldest.IsZero() {
		line += fmt.Sprintf("  used from %s to %s\n", local(usage.Oldest),
			local(usage.Newest))
	}
	if usage.Unrecorded > nothing {
		line += fmt.Sprintf("  %d entries have no recorded use and go "+
			"first when it is over its cap\n", usage.Unrecorded)
	}
	return line + "  in " + usage.Path + "\n"
}

// folder says a kind's folder, or clears it, or says what clearing
// would.
func (made caching) folder(out io.Writer, home string,
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
