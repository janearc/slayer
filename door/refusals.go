package door

// The refusals: assess FILE --refusals stages the job by draw's own
// runner, as draw stages it, and asks the GPU path of the first frame's
// world what it would refuse: every part, grouped by reason, not only
// the first, and draws nothing. It is for finding the whole gap between
// a scene and the GPU in one pass.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/gpu"
)

// refusalsFlag asks assess for what the GPU path would refuse.
const refusalsFlag = "refusals"

// refusalsTakes are the flags assess FILE --refusals takes: the ones a
// plain assess takes to stage the file, but no census.
var refusalsTakes = append([]string{"out", "width", "height", "rays", "at",
	"camera", "renderer", "quality", refusalsFlag}, switchNames()...)

// refuser is a source that says what the GPU path would refuse in a
// frame's world: a session.
type refuser interface {
	Refusals(index int) ([]gpu.Refusal, error)
}

// errUnrefused is the refusal to list what the GPU would refuse of
// frames already rendered.
var errUnrefused = errors.New("this scene is frames already rendered, " +
	"pictures with no staged world to ask the GPU path about")

// Refusals is what the GPU path would refuse in the session's frame.
func (made sessionSource) Refusals(index int) ([]gpu.Refusal, error) {
	return made.session.Refusals(index)
}

// refusals says what the GPU path would refuse in the file's first
// frame, and lets the scene go.
func (held assessing) refusals(call pzm.Call, given string) error {
	drawer := drawing(held)
	name := filepath.Base(given)
	run, err := drawer.planned(call, given)
	if err != nil {
		return fmt.Errorf("%s will not draw: %w", name, err)
	}
	tools, err := opening(run)
	if err != nil {
		return fmt.Errorf("%s will not draw: %w", name, err)
	}
	found, asked := refusedBy(tools.source)
	if err := errors.Join(asked, tools.source.Close()); err != nil {
		return err
	}
	refusalText := refusalLines(name, found)
	_, err = fmt.Fprintln(call.Out(), strings.Join(refusalText, "\n"))
	return err
}

// refusedBy asks a source what the GPU path would refuse in its first
// frame.
func refusedBy(opened source) ([]gpu.Refusal, error) {
	asking, asks := opened.(refuser)
	if !asks {
		return nil, errUnrefused
	}
	return asking.Refusals(censusFrame)
}

// reasonGroup is the parts refused for one reason: how many, and the
// first of them as an example.
type reasonGroup struct {
	reason  string
	count   int
	example gpu.Refusal
}

// grouped is the refusals by reason, in the order each reason first
// appears.
func grouped(found []gpu.Refusal) []reasonGroup {
	var groups []reasonGroup
	for _, each := range found {
		at := -1
		for index := range groups {
			if groups[index].reason == each.Reason {
				at = index
			}
		}
		if at < 0 {
			groups = append(groups, reasonGroup{reason: each.Reason, example: each})
			at = len(groups) - 1
		}
		groups[at].count++
	}
	return groups
}

// partWords says one refused part by its place, kind and material.
func partWords(each gpu.Refusal) string {
	words := each.Part
	if each.Index >= 0 {
		words = fmt.Sprintf("%s %d", each.Part, each.Index)
	}
	if each.Kind != "" {
		words += " " + each.Kind
	}
	if each.Material != "" {
		words += " (" + each.Material + ")"
	}
	return words
}

// refusalLines is what assess --refusals says: the verdict, then each
// reason with its count and an example, then every part refused.
func refusalLines(name string, found []gpu.Refusal) []string {
	if len(found) == nothing {
		return []string{name + ": the GPU path would take every part"}
	}
	groups := grouped(found)
	lines := []string{fmt.Sprintf("%s: the GPU path would refuse %d parts "+
		"for %d reasons; nothing was drawn", name, len(found), len(groups))}
	for _, group := range groups {
		exampleWords := partWords(group.example)
		lines = append(lines, fmt.Sprintf("%6d  %s (for example %s)",
			group.count, group.reason, exampleWords))
	}
	lines = append(lines, "each part:")
	for _, each := range found {
		lines = append(lines, "  "+partWords(each)+": "+each.Reason)
	}
	return lines
}
