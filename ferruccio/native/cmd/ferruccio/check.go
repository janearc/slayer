package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/ferruccio/native/internal/workspace"
	"github.com/janearc/slayer/session/placement"
)

// checkDefaultTimeout is how long one set may take to check when --timeout
// is not given.
const checkDefaultTimeout = time.Minute

// checkSetsPattern picks the set files out of a --sets directory.
const checkSetsPattern = "*.yaml"

// The verdicts a set can get from check.
const (
	checkTakes      = "takes"
	checkRefused    = "refused"
	checkTimedOut   = "timed out"
	checkNotChecked = "not checked"
)

// checkHelp is the one line --help gives for check.
const checkHelp = "say whether the working display would take each set"

// checkPage is the longer page for check.
const checkPage = `Open no window and draw nothing: run the checks the working
display runs when Ferruccio opens a set file, in the same order, and say
whether it would take the set or the refusal in the display's own words.

By default only the checks that cost nothing run: the file reads, its
schema accepts it, it is a set in space with a seed and nothing
unrendered, it has a film size, no rig and a fixed camera, and every
placement is not burning and has its recipe or kind. With --meshes the
display's own scene is also built, which is where a shape with no display
mesh or no hit test is found. --timeout bounds each set. A set that runs
out is reported as that, not as a refusal, and the sets after it are not
checked, because the display's build cannot be stopped and would go on
running beside the next.

The sets come from the FILE arguments and from --sets DIR, which takes the
yaml files directly in it. A tally follows, most common refusal first.`

// checkResult is what check says of one set.
type checkResult struct {
	Name, Verdict, Refusal, Reason string
	Elapsed                        time.Duration
}

// checkRun checks the file at path, as workspace.CheckWorking does.
type checkRun func(path string, meshes bool) error

// registerCheck declares check through Puzzlemouth beside open and proof.
func registerCheck(tool *pzm.Tool) {
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.FullCPU)
	tool.Verb("check", verb.NoShort, checkHelp, checkPage, "Example_check", facts, runCheck,
		pzm.Switch("meshes", verb.NoShort, "also build each thing's display mesh"),
		pzm.Value("sets", verb.NoShort, "a directory of set files to check"),
		pzm.Value("timeout", verb.NoShort, "longest one set may take, default 1m"))
}

// runCheck checks each set the call names and prints a line for each, then
// the tally. A refused set is a report, not an error.
func runCheck(call pzm.Call) error {
	paths, err := checkPaths(call.Args(), call.Value("sets"))
	if err != nil {
		return err
	}
	limit := checkDefaultTimeout
	if given := call.Value("timeout"); given != "" {
		limit, err = time.ParseDuration(given)
		if err != nil || limit <= 0 {
			return fmt.Errorf("timeout must be a positive duration such as 90s, not %q", given)
		}
	}
	meshes := call.On("meshes")
	results := make([]checkResult, 0, len(paths))
	stopped := false
	for _, path := range paths {
		result := checkResult{Name: path.name, Verdict: checkNotChecked}
		if !stopped {
			result = checkWith(path, limit, meshes, workspace.CheckWorking)
		}
		stopped = stopped || result.Verdict == checkTimedOut
		results = append(results, result)
		fmt.Fprintln(call.Out(), checkLine(result, meshes))
	}
	writeTally(call.Out(), results)
	return nil
}

// checkPath is a set file and the name check calls it by.
type checkPath struct{ path, name string }

// checkPaths lists the files named as arguments, then the yaml files in
// sets. It fails when it has none, or when sets cannot be read.
func checkPaths(args []string, sets string) ([]checkPath, error) {
	var paths []checkPath
	for _, each := range args {
		paths = append(paths, checkPath{path: each, name: each})
	}
	if sets != "" {
		if _, err := os.Stat(sets); err != nil {
			return nil, err
		}
		found, err := filepath.Glob(filepath.Join(sets, checkSetsPattern))
		if err != nil {
			return nil, err
		}
		for _, each := range found {
			paths = append(paths, checkPath{path: each, name: filepath.Base(each)})
		}
	}
	if len(paths) == 0 {
		return nil, errors.New("check needs FILE... or --sets DIR holding yaml files")
	}
	return paths, nil
}

// checkWith runs one check and gives it limit to finish. A check that panics
// is a refusal, since the display would have gone down with it.
func checkWith(path checkPath, limit time.Duration, meshes bool, run checkRun) checkResult {
	started := time.Now()
	done := make(chan error, 1)
	go func() {
		defer func() {
			if lost := recover(); lost != nil {
				done <- fmt.Errorf("panic: %v", lost)
			}
		}()
		done <- run(path.path, meshes)
	}()
	result := checkResult{Name: path.name}
	select {
	case err := <-done:
		result = checkFinished(result, path, err)
	case <-time.After(limit):
		result.Verdict, result.Refusal = checkTimedOut, "after "+limit.String()
	}
	result.Elapsed = time.Since(started)
	return result
}

// checkFinished fills result from the error a check returned.
func checkFinished(result checkResult, path checkPath, err error) checkResult {
	if err == nil {
		result.Verdict = checkTakes
		return result
	}
	result.Verdict = checkRefused
	result.Refusal = checkOneLine(err.Error())
	result.Reason = result.Refusal
	var named *placement.DisplayThingError
	if errors.As(err, &named) {
		result.Reason = checkOneLine(named.Err.Error())
	}
	if absolute, abs := filepath.Abs(path.path); abs == nil {
		result.Reason = strings.ReplaceAll(result.Reason, absolute, "FILE")
	}
	result.Reason = strings.ReplaceAll(result.Reason, path.path, "FILE")
	return result
}

// checkOneLine folds every run of white space, newlines included, to one
// space, so a refusal is one line.
func checkOneLine(text string) string { return strings.Join(strings.Fields(text), " ") }

// checkLine is the line for one set: its name, then takes or the refusal.
// A meshes check adds how long the set took.
func checkLine(result checkResult, meshes bool) string {
	line := result.Name + "\t" + result.Verdict
	switch result.Verdict {
	case checkRefused:
		line += ": " + result.Refusal
	case checkTimedOut:
		line += " " + result.Refusal
	}
	if meshes && result.Verdict != checkNotChecked {
		line += fmt.Sprintf(" (%s)", result.Elapsed.Round(time.Millisecond))
	}
	return line
}

// checkReasonCount is one refusal reason and how many sets it stops.
type checkReasonCount struct {
	Reason string
	Count  int
}

// checkTally counts the refusal reasons, most common first and, for equal
// counts, in alphabetical order.
func checkTally(results []checkResult) []checkReasonCount {
	counts := map[string]int{}
	for _, each := range results {
		if each.Verdict == checkRefused {
			counts[each.Reason]++
		}
	}
	tally := make([]checkReasonCount, 0, len(counts))
	for reason, count := range counts {
		tally = append(tally, checkReasonCount{Reason: reason, Count: count})
	}
	sort.Slice(tally, func(first, second int) bool {
		if tally[first].Count != tally[second].Count {
			return tally[first].Count > tally[second].Count
		}
		return tally[first].Reason < tally[second].Reason
	})
	return tally
}

// writeTally prints how many sets were checked, taken, refused, timed out
// and left unchecked, then each refusal reason with its count.
func writeTally(out io.Writer, results []checkResult) {
	totals := map[string]int{}
	for _, each := range results {
		totals[each.Verdict]++
	}
	fmt.Fprintf(out, "\n%d sets: %d takes, %d refused, %d timed out, %d not checked\n",
		len(results), totals[checkTakes], totals[checkRefused], totals[checkTimedOut], totals[checkNotChecked])
	for _, each := range checkTally(results) {
		fmt.Fprintf(out, "%5d  %s\n", each.Count, each.Reason)
	}
}
