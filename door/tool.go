package door

import (
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// Tool is the slayer tool: its verbs, each declared through Puzzlemouth,
// drawing with the session, finish and movie.
func Tool(env Env) *pzm.Tool {
	return tool(env, pieces{stamp: env.build.hillside()}, machine{})
}

// tool is the slayer tool drawing with the workshop given and laying
// out panes on the machine given, so a test can give mocks in place of
// the renderer, the movie writer and the programs on PATH.
func tool(env Env, shop workshop, programs programs) *pzm.Tool {
	made := pzm.New(toolName(env), "the front door: draw a job, check "+
		"a file, see what there is, develop raws, hear tunes",
		pzm.Build(env.build.build), pzm.Built(env.build.built),
		pzm.Who(env.build.who), pzm.Sessions(env.build.sessions))
	declareDraw(made, drawing{env: env, shop: shop})
	declareList(made)
	declareTunes(made, env)
	declareAssess(made, env, shop)
	declareDevelop(made, env, shop)
	declareProve(made, env, shop)
	declareCache(made, env)
	live := watching{env: env, stager: sessions{stamp: env.build.hillside()}}
	interactive(made, editing{env: env, machine: programs, watcher: live})
	return made
}

// interactive declares the verbs a person uses at a terminal: watch,
// edit and schema; and edit as what bare slayer runs there.
func interactive(made *pzm.Tool, editor editing) {
	live, env := editor.watcher, editor.env
	watches := verb.NewFacts(verb.WritesTo("a status folder beside the "+
		"job, and the job when s is pressed"), verb.Changes,
		verb.ThisMachine, verb.NoOne, verb.FullGPU)
	made.Verb("watch", verb.NoShort, "draw a job live in this terminal "+
		"as it is edited", watchHelp, "Example_watch", watches,
		live.watch, jobFlags()...)
	edits := verb.NewFacts(verb.WritesTo("a new job, the agent's guides "+
		"beside it, and the schema in the cache"), verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.FullGPU)
	made.Verb("edit", verb.NoShort, "lay out watch, the editor and the "+
		"agent console in tmux", editHelp, "Example_edit", edits,
		editor.edit, jobFlags()...)
	schemas := verb.NewFacts(verb.WritesTo("a copy of the schema in "+
		"the cache, with --where"), verb.Changes, verb.ThisMachine,
		verb.NoOne, verb.Light)
	made.Verb("schema", verb.NoShort, "print the job schema, or where "+
		"its copy is", schemaHelp, "Example_schema", schemas,
		schemaing{env: env}.schema, pzm.Switch("where", verb.NoShort,
			"print the path of the schema's copy, writing it if stale"))
	made.Bare("edit")
}

// The tool's names: its own, and the one it gives on one day a year.
const (
	plainName = "slayer"
	dayName   = "sleigh"
)

// The day the tool gives its other name, by the local date. Picked
// 2026-11-12.
const (
	nameMonth = time.December
	nameDay   = 25
)

// toolName is the name the tool says itself by, on the env's clock's
// local date.
func toolName(env Env) verb.Name {
	today := env.now().Local()
	if today.Month() == nameMonth && today.Day() == nameDay {
		return dayName
	}
	return plainName
}
