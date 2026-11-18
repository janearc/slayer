// Command report prints a saved lighting census as JSON without rendering.
package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	census "github.com/janearc/slayer/tools/lighting-census"
)

// main runs the lighting-report command and exits with its status.
func main() {
	os.Exit(tool().Run(os.Args[1:], os.Stdout, os.Stderr))
}

// tool builds the lighting-report command with its read verb.
func tool() *pzm.Tool {
	made := pzm.New("lighting-report", "read a saved lighting census",
		pzm.Build(""), pzm.Built(""), pzm.Who(""), pzm.Sessions(""))
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.FullCPU)
	made.Verb("read", verb.NoShort, "print a saved census as JSON",
		"# read\n\nCompare saved linear frames without rendering.\n",
		"Example_read", facts, read,
		pzm.Value("root", verb.NoShort, "census output directory"),
		pzm.Value("reference", verb.NoShort, "cpu or gpu-native-meshes"))
	return made
}

// read prints a saved census as JSON for the root and reference given.
func read(call pzm.Call) error {
	root, reference := call.Value("root"), call.Value("reference")
	if root == "" || len(call.Args()) != 0 {
		return errors.New("read needs --root DIRECTORY and no positional arguments")
	}
	if reference == "" {
		reference = "cpu"
	}
	report, err := census.Summarize(root, reference)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(call.Out())
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
