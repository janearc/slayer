package main

import (
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/azazel/pages"
)

// publishHelp is the publish verb's page: whose it is, and whose it is
// not.
const publishHelp = "# publish\n\nPublish one render folder's pages once, " +
	"as when a job ends: its page,\nits film so far, latest.html, the " +
	"indexes and, given --pictures, Barkeep's\nline. It is the operator's, " +
	"for a render begun before azazel kept its\npages. Agents do not run " +
	"it for their jobs.\n"

// publishVerb declares the operator's one-off publication of a folder.
func publishVerb(made *pzm.Tool, app queueApplication) {
	writes := verb.NewFacts(verb.WritesTo("a render folder's pages and the "+
		"renders gallery"), verb.Changes, verb.ThisMachine, verb.NoOne,
		verb.Light)
	made.Verb("publish", verb.NoShort, "publish one render folder's pages once",
		publishHelp, "Example_queuePublish", writes,
		previewPublish(app.publish),
		pzm.Value("renders", verb.NoShort, "the absolute renders root"),
		pzm.Value("folder", verb.NoShort, "the render folder: frames/X or "+
			"animations/X"),
		pzm.Value("pictures", verb.NoShort, "the pictures the folder holds "+
			"when the whole render is done"))
}

// previewPublish says what publish would do, under --dry-run, without
// reading or writing any page. Publish names no queue, so it has its own.
func previewPublish(apply pzm.Do) pzm.Do {
	return func(call pzm.Call) error {
		if !call.DryRun() {
			return apply(call)
		}
		preview := struct {
			DryRun   bool   `json:"dry_run"`
			Action   string `json:"action"`
			Renders  string `json:"renders"`
			Folder   string `json:"folder"`
			Pictures string `json:"pictures"`
			Notice   string `json:"notice"`
		}{DryRun: true, Action: "publish", Renders: call.Value("renders"),
			Folder: call.Value("folder"), Pictures: call.Value("pictures"),
			Notice: "intent only; nothing was read or written"}
		if err := json.MarshalWrite(call.Out(), preview); err != nil {
			return err
		}
		_, err := fmt.Fprintln(call.Out())
		return err
	}
}

// picturesGiven is --pictures, a positive count, or nought when absent.
func picturesGiven(call pzm.Call) (int64, error) {
	if !call.Given("pictures") {
		return 0, nil
	}
	count, err := strconv.ParseInt(call.Value("pictures"), 10, 64)
	if err != nil || count <= 0 {
		return 0, fmt.Errorf("--pictures must be a positive whole number")
	}
	return count, nil
}

// publish publishes one render folder once and says what it was told.
func (app queueApplication) publish(call pzm.Call) error {
	root := call.Value("renders")
	if len(call.Args()) != 0 || !filepath.IsAbs(root) {
		return fmt.Errorf("publish needs an absolute --renders and no " +
			"positional arguments")
	}
	folder, err := pages.ParseFolder(root, call.Value("folder"))
	if err != nil {
		return err
	}
	pictures, err := picturesGiven(call)
	if err != nil {
		return err
	}
	told, err := pages.Publish(app.ctx, pages.Settings{Renders: root,
		Encoder: pages.FoundFfmpeg(), Now: time.Now,
		Location: time.Local, Every: pages.TickEvery}, folder, pictures)
	for _, notice := range told {
		fmt.Fprintln(call.Out(), notice)
	}
	return err
}
