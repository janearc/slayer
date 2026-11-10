// Command island renders sourced geographic terrain in a declared sphere.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

var build, built, who, sessions string

// main runs the Pzm command tree with cancellation on interrupt or termination.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(tool(ctx).Run(os.Args[1:], os.Stdout, os.Stderr))
}

// tool declares render and sequence inputs and their local GPU effects.
func tool(ctx context.Context) *pzm.Tool {
	tool := pzm.New("island", "render a sourced island in a spherical frame",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("a new render directory"), verb.Changes,
		verb.ThisMachine, verb.NoOne, verb.FullGPU)
	tool.Verb("render", verb.NoShort, "prepare geographic terrain for one camera view",
		"# render\n\nRead a checked scene, verify its source hashes, prepare camera-relative terrain and render on native Metal. The destination must be new.\n",
		"Example_render", facts, func(call pzm.Call) error { return render(ctx, call) },
		pzm.Value("scene", verb.NoShort, "island scene YAML or JSON"),
		pzm.Value("out", verb.NoShort, "new output directory"))
	tool.Verb("sequence", verb.NoShort, "render a timed geographic ascent",
		"# sequence\n\nRead a checked YAML camera track, retain source and prototype caches, and write numbered native-Metal frames with exact effective settings. The destination must be new.\n",
		"Example_sequence", facts, func(call pzm.Call) error { return sequence(ctx, call) },
		pzm.Value("plan", verb.NoShort, "island sequence YAML or JSON"),
		pzm.Value("out", verb.NoShort, "new output directory"))
	return tool
}

// required distinguishes an explicit, nonempty flag value from an omitted one.
func required(call pzm.Call, name string) (string, error) {
	if !call.Given(name) || call.Value(name) == "" {
		return "", fmt.Errorf("island needs --%s", name)
	}
	return call.Value(name), nil
}
