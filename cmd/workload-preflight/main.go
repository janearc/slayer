// Command workload-preflight compares a stored exact-key workload peak with
// explicitly supplied host capacity, current use, process count and headroom.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/render/workload"
)

var build, built, who, sessions string

// main runs the tool with the command line arguments and exits with its
// status.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run declares the report verb and runs it on args, returning the exit
// status.
func run(args []string, out, errs io.Writer) int {
	tool := pzm.New("workload-preflight", "compare exact-key render peaks with explicit host inputs",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("report", verb.NoShort, "report evidence-based memory preflight advice",
		"Loads an exact workload identity and stored profile, then compares its process lifetime peak with explicit host inputs. Unknown evidence stays unknown; this does not admit work or change render settings.",
		"Example_report", reads, report,
		pzm.Value("cache-home", verb.NoShort, "absolute workload cache directory"),
		pzm.Value("identity-file", verb.NoShort, "JSON file containing the complete workload identity"),
		pzm.Value("capacity-bytes", verb.NoShort, "host memory capacity in bytes"),
		pzm.Value("current-use-bytes", verb.NoShort, "current host use in bytes, including running work"),
		pzm.Value("processes", verb.NoShort, "additional independent process copies to consider"),
		pzm.Value("frames-per-process", verb.NoShort, "frames/workers per process, reported but not multiplied"),
		pzm.Value("headroom-bytes", verb.NoShort, "explicit additional margin in bytes"))
	return tool.RunWith(args, bytes.NewReader(nil), out, errs)
}

// report reads the identity file and the host inputs, then writes the
// estimate as JSON. A missing or unreadable input gives an unknown estimate.
func report(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return fmt.Errorf("workload-preflight report takes no positional arguments")
	}
	cacheHome := call.Value("cache-home")
	identityPath := call.Value("identity-file")
	capacity, err := uintFlag(call, "capacity-bytes")
	if err != nil {
		return err
	}
	current, err := uintFlag(call, "current-use-bytes")
	if err != nil {
		return err
	}
	processes, err := uintFlag(call, "processes")
	if err != nil {
		return err
	}
	frames, err := uintFlag(call, "frames-per-process")
	if err != nil {
		return err
	}
	headroom, err := uintFlag(call, "headroom-bytes")
	if err != nil {
		return err
	}
	inputs := workload.EstimateInputs{HostCapacityBytes: &capacity,
		CurrentUseBytes: &current, AdditionalProcesses: processes,
		FramesPerProcess: frames, HeadroomBytes: &headroom}
	identity, readErr := readIdentity(identityPath)
	if readErr != nil {
		return encodeUnknown(call.Out(), "cannot read complete workload identity: "+readErr.Error())
	}
	inputs.Identity = identity
	if _, err := identity.Key(); err != nil {
		report := workload.Estimate(workload.Profile{}, inputs)
		return json.NewEncoder(call.Out()).Encode(report)
	}
	store, err := workload.OpenReadOnly(cacheHome, 2*time.Second)
	if err != nil {
		return encodeUnknown(call.Out(), "cannot open workload profile cache read-only: "+err.Error())
	}
	profile, found, err := store.Lookup(identity)
	if err != nil {
		return encodeUnknown(call.Out(), "workload profile lookup failed: "+err.Error())
	}
	if !found {
		profile = workload.Profile{Identity: identity}
	}
	return json.NewEncoder(call.Out()).Encode(workload.Estimate(profile, inputs))
}

// uintFlag returns the named flag as an unsigned integer. It fails when the
// flag is absent or is not a number.
func uintFlag(call pzm.Call, name string) (uint64, error) {
	if !call.Given(name) {
		return 0, fmt.Errorf("--%s is required", name)
	}
	value, err := strconv.ParseUint(call.Value(name), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("--%s must be an unsigned integer number of bytes/count: %w", name, err)
	}
	return value, nil
}

// readIdentity decodes the single JSON identity object in the file at path
// and checks that its key is complete.
func readIdentity(path string) (workload.Identity, error) {
	if path == "" {
		return workload.Identity{}, errors.New("--identity-file is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return workload.Identity{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var identity workload.Identity
	if err := decoder.Decode(&identity); err != nil {
		return workload.Identity{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return workload.Identity{}, errors.New("identity file has multiple JSON values")
		}
		return workload.Identity{}, err
	}
	if _, err := identity.Key(); err != nil {
		return workload.Identity{}, err
	}
	return identity, nil
}

// encodeUnknown writes an unknown estimate that gives reason and no numbers.
func encodeUnknown(output io.Writer, reason string) error {
	return json.NewEncoder(output).Encode(struct {
		Status  workload.EstimateStatus `json:"status"`
		Unknown []string                `json:"unknown"`
	}{workload.EstimateUnknown, []string{reason}})
}
