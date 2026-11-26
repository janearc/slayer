package door

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/render/regioncompare"
	"github.com/janearc/slayer/render/regioninput"
	"github.com/janearc/slayer/render/regionset"
)

type regionComparison struct{ env Env }

// declareRegionComparison declares the verb and all flags through Puzzlemouth.
func declareRegionComparison(made *pzm.Tool, env Env) {
	writes := verb.WritesTo("the cache, reference directory and a new result directory")
	draws := verb.NewFacts(writes, verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	made.Verb("compare-regions", verb.NoShort, "compare regions of a picture",
		regionComparisonHelp, "Example_compareRegions", draws, regionComparison{env: env}.run,
		pzm.Value("snapshot", verb.NoShort, "absolute path to the frozen input snapshot"),
		pzm.List("inputs", verb.NoShort, "all input names relative to the snapshot"),
		pzm.Switch("complete-inputs", verb.NoShort, "assert the named inputs are complete and stable"),
		pzm.Value("references", verb.NoShort, "absolute path to an existing reference directory"),
		pzm.Value("reference-plan", verb.NoShort, "reuse the manifests from an earlier comparison"),
		pzm.Value("out", verb.NoShort, "new directory for the report and sheet"),
		pzm.Value("width", verb.NoShort, "full frame width in pixels"),
		pzm.Value("height", verb.NoShort, "full frame height in pixels"),
		pzm.Value("rays", verb.NoShort, "candidate rays per side"),
		pzm.Value("reference-rays", verb.NoShort, "reference rays per side"),
		pzm.Value("repeat", verb.NoShort, "candidate draws per region, from 1 to 8"))
}

// run validates the request, draws its regions and writes a completed result.
func (comparison regionComparison) run(call pzm.Call) error {
	args := call.Args()
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("compare-regions takes one set path")
	}
	for _, name := range []string{"snapshot", "inputs", "complete-inputs", "references", "out", "width", "height", "rays", "reference-rays"} {
		if !call.Given(name) {
			return fmt.Errorf("compare-regions: --%s is required", name)
		}
	}
	numbers := make([]int, 0, 4)
	for _, name := range []string{"width", "height", "rays", "reference-rays"} {
		value, err := strconv.Atoi(call.Value(name))
		if err != nil || value <= 0 {
			return fmt.Errorf("compare-regions: --%s must be a positive integer", name)
		}
		numbers = append(numbers, value)
	}
	repeats := 1
	if call.Given("repeat") {
		var err error
		repeats, err = strconv.Atoi(call.Value("repeat"))
		if err != nil || repeats < 1 || repeats > 8 {
			return fmt.Errorf("compare-regions: --repeat must be between 1 and 8")
		}
	}
	root, refs, out := call.Value("snapshot"), call.Value("references"), call.Value("out")
	if !filepath.IsAbs(root) || !filepath.IsAbs(refs) || !filepath.IsAbs(out) {
		return fmt.Errorf("compare-regions: snapshot, references and out must be absolute")
	}
	if numbers[3] <= numbers[2] || numbers[3] > 64 {
		return fmt.Errorf("compare-regions: reference rays must be greater than candidate rays and at most 64 per side")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	resolvedRefs, err := filepath.EvalSymlinks(refs)
	if err != nil {
		return err
	}
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(out))
	if err != nil {
		return err
	}
	if beneath(resolvedRoot, resolvedRefs) || beneath(resolvedRoot, filepath.Join(resolvedParent, filepath.Base(out))) {
		return fmt.Errorf("compare-regions: references and output must be outside the snapshot")
	}
	if _, err := os.Lstat(out); err == nil {
		return fmt.Errorf("compare-regions: output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	entries, err := regioninput.Inspect(root, call.List("inputs"))
	if err != nil {
		return err
	}
	data, err := os.ReadFile(absolute(args[0], comparison.env.directory))
	if err != nil {
		return err
	}
	set, err := regionset.Read(strings.NewReader(string(data)), numbers[0], numbers[1])
	if err != nil {
		return err
	}
	if !filepath.IsLocal(set.Job) {
		return fmt.Errorf("compare-regions: job must be relative to the snapshot")
	}
	shelf, shelfOK := comparison.env.lookup("SLAYER_SHELF")
	cache, cacheOK := comparison.env.lookup("SLAYER_CACHE")
	if !shelfOK || !cacheOK {
		return fmt.Errorf("compare-regions: SLAYER_SHELF and SLAYER_CACHE are required")
	}
	resolvedCache, err := filepath.EvalSymlinks(cache)
	if err != nil {
		return err
	}
	if beneath(resolvedRoot, resolvedCache) {
		return fmt.Errorf("compare-regions: cache must be outside the snapshot")
	}
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(), "would compare %d regions of %s with %d candidate draws each and write %s\n", len(set.Regions), set.Job, repeats, out)
		return err
	}
	source := &regioncompare.SessionSource{Context: comparison.env.ctx, Stamp: comparison.env.build.hillside(), Snapshot: root,
		Entries: entries, JobName: set.Job, JobFile: set.Job, Shelf: shelf, Cache: cache,
		Width: numbers[0], Height: numbers[1], Rays: numbers[2]}
	if err := source.PlanSession(set.Regions, numbers[3], refs, true); err != nil {
		return err
	}
	if call.Given("reference-plan") {
		plan, err := os.Open(absolute(call.Value("reference-plan"), comparison.env.directory))
		if err != nil {
			return err
		}
		err = regioncompare.ApplyReferencePlan(plan, source, refs)
		if err = errors.Join(err, plan.Close()); err != nil {
			return err
		}
		for _, ref := range source.References {
			if ref.Rays != numbers[3] {
				return fmt.Errorf("compare-regions: pinned reference rays differ from --reference-rays")
			}
		}
	}
	references := make(map[string]regioncompare.Identity, len(set.Regions))
	for _, region := range set.Regions {
		references[region.Name] = source.References[region.Name].Identity
	}
	result, err := regioncompare.RunRepeated(strings.NewReader(string(data)), numbers[0], numbers[1], source.Candidates, references, source, repeats)
	if err != nil {
		return err
	}
	if err := writeRegionComparison(out, result, source); err != nil {
		return err
	}
	_, err = fmt.Fprintf(call.Out(), "wrote %s\n", out)
	return err
}

// beneath reports whether path is the root or lies within it.
func beneath(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && (relative == "." || filepath.IsLocal(relative))
}

// writeRegionComparison installs a report and sheet in a new directory.
func writeRegionComparison(out string, result regioncompare.Result, source *regioncompare.SessionSource) (resultErr error) {
	if err := os.Mkdir(out, 0700); err != nil {
		return err
	}
	defer func() {
		if resultErr != nil {
			for _, name := range []string{"report.json", "sheet.png", "references.json"} {
				if err := os.Remove(filepath.Join(out, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
					resultErr = errors.Join(resultErr, err)
				}
			}
			resultErr = errors.Join(resultErr, os.Remove(out))
		}
	}()
	for _, item := range []struct {
		name  string
		write func(*os.File) error
	}{
		{"report.json", func(file *os.File) error { return regioncompare.WriteReport(file, result) }},
		{"sheet.png", func(file *os.File) error { return regioncompare.WriteSheet(file, result) }},
		{"references.json", func(file *os.File) error { return regioncompare.WriteReferencePlan(file, source) }},
	} {
		file, err := os.OpenFile(filepath.Join(out, item.name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		err = item.write(file)
		if err = errors.Join(err, file.Close()); err != nil {
			return err
		}
	}
	return nil
}

const regionComparisonHelp = `# compare-regions

compare-regions SET draws named full-frame regions and compares each with a
finer retained reference. Supply an absolute frozen --snapshot, all its input
names with --inputs, and --complete-inputs to assert that list is complete and
stable. SLAYER_SHELF must be inside the snapshot; SLAYER_CACHE must be stable.
The command cannot discover missing or environment-selected inputs. Supply
--width, --height, --rays and --reference-rays, an existing absolute
--references directory, and a new absolute --out directory. A reference is a
finite estimate, not proof of convergence. The result includes a labelled PNG
sheet, linear-light report and reference plan. To hold those reference
manifests fixed on another run, pass the plan with --reference-plan. Native
GPU, bloom and overlays are not supported. --repeat asks for up to eight
candidate draws per region and reports each repeat against the first in raw
linear light.
`
