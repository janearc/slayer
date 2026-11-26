// Command ferruccio-stage makes an isolated, non-rendered working snapshot.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/ferruccio/native/internal/stagewire"
)

var build, built, who, sessions string

// main leaves command parsing and output routing to Puzzlemouth.
func main() { os.Exit(command().RunWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

// command declares the staging verb and its complete interface.
func command() *pzm.Tool {
	tool := pzm.New("ferruccio-stage", "Stage one Ferruccio working view", pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	facts := verb.NewFacts(verb.WritesTo("configured Slayer cache"), verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	tool.Verb("stage", verb.NoShort, "stage one unsaved scene", "Read a source buffer from standard input and write one display snapshot. No render is started.", "Example_stage", facts, func(call pzm.Call) error {
		path, kind := call.Value("path"), call.Value("kind")
		if len(call.Args()) != 0 || !filepath.IsAbs(path) || (kind != "set" && kind != "legacy-set" && kind != "job") {
			return fmt.Errorf("stage requires an absolute --path and a supported --kind")
		}
		if call.DryRun() {
			_, err := fmt.Fprintln(call.Out(), "stage the supplied buffer without rendering")
			return err
		}
		source, err := io.ReadAll(io.LimitReader(call.In(), stagewire.MaxSource+1))
		if err != nil {
			return err
		}
		if len(source) > stagewire.MaxSource {
			return fmt.Errorf("scene source exceeds %d bytes", stagewire.MaxSource)
		}
		snapshot, err := stage(context.Background(), path, string(source), kind)
		if err != nil {
			return err
		}
		return stagewire.Write(call.Out(), snapshot)
	}, pzm.Value("path", verb.NoShort, "absolute original document path"), pzm.Value("kind", verb.NoShort, "set, legacy-set or job"))
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("presets", verb.NoShort, "list producer film sizes", "Write the Slayer job producer's output-size presets as JSON. Physical film gauges are separate.", "Example_presets", reads, func(call pzm.Call) error {
		if len(call.Args()) != 0 {
			return fmt.Errorf("presets takes no arguments")
		}
		presets, err := filmPresets()
		if err != nil {
			return err
		}
		return json.NewEncoder(call.Out()).Encode(presets)
	})
	return tool
}
