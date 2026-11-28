// Command ferruccio-stage makes an isolated, non-rendered working snapshot.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

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
		camera := stagewire.Camera{Name: call.Value("camera"), PathRecipe: call.Value("still-camera")}
		if call.Given("frame") {
			frame, err := strconv.Atoi(call.Value("frame"))
			if err != nil || frame < 1 || kind != "job" {
				return fmt.Errorf("frame requires a positive film job frame number")
			}
			camera.Frame = frame
		}
		detail := call.Value("detail")
		if call.Given("detail") && detail == "" {
			return fmt.Errorf("detail requires an authored placement or stand ID")
		}
		if camera.PathRecipe != "" && (camera.Name != "" || kind == "job") {
			return fmt.Errorf("still-camera requires a scene set and no named camera override")
		}
		if call.Given("camera-wide") {
			wide, err := strconv.ParseFloat(call.Value("camera-wide"), 64)
			if err != nil || camera.Name == "" || !(wide > 0 && wide < 180) {
				return fmt.Errorf("camera-wide requires a named camera and an angle between zero and 180 degrees")
			}
			camera.Wide = wide
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
		snapshot, err := stage(context.Background(), path, string(source), kind, camera, detail)
		if err != nil {
			return err
		}
		return stagewire.Write(call.Out(), snapshot)
	}, pzm.Value("path", verb.NoShort, "absolute original document path"), pzm.Value("kind", verb.NoShort, "set, legacy-set or job"), pzm.Value("camera", verb.NoShort, "named camera for this working view"), pzm.Value("camera-wide", verb.NoShort, "horizontal angle of a named still camera in degrees"), pzm.Value("still-camera", verb.NoShort, "checked still camera path and scene time"), pzm.Value("detail", verb.NoShort, "authored placement or stand ID"), pzm.Value("frame", verb.NoShort, "one-based film job frame to inspect"))
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
