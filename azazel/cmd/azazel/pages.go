package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/azazel/pages"
)

// workerPages owns the publisher for the main worker's lifetime. Restarting
// work rebuilds publication from the ledger and files; render supervisors
// remain independent of publication failures.
func (app queueApplication) workerPages(call pzm.Call) (func(), error) {
	root := call.Value("renders")
	if root == "" {
		root = os.Getenv("SLAYER_RENDERS")
	}
	if root == "" {
		return func() {}, nil
	}
	if !filepath.IsAbs(root) {
		return nil, errors.New("publication requires an absolute renders root")
	}
	log, err := pages.OpenLog(call.Value("queue"))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(app.ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer log.Close()
		if err := pages.Serve(ctx, call.Value("queue"), pages.Settings{Renders: root, Encoder: pages.FoundFfmpeg(), Now: time.Now, Location: time.Local, Every: pages.TickEvery}, log); err != nil {
			fmt.Fprintln(log, "publication:", err)
		}
	}()
	return func() { cancel(); <-done }, nil
}

// pagesHelp is the pages verb's page.
const pagesHelp = "# pages\n\nServe the queue's one page coordinator until " +
	"stopped: every 10 s it reads\nthe ledger as status does, and keeps each " +
	"running or just finished job's\nfolder page, film so far, latest.html, " +
	"the indexes and Barkeep's line.\nA second one for the same queue is " +
	"refused. Its log is QUEUE/pages.log.\nwork --renders ROOT starts it " +
	"itself; this verb serves a queue without one.\n"

// pagesVerb declares the queue's one page coordinator.
func pagesVerb(made *pzm.Tool, app queueApplication, root pzm.Flag) {
	writes := verb.NewFacts(verb.WritesTo("render folders' pages, the renders "+
		"gallery, Barkeep's line, and pages.lock and pages.log in the queue"),
		verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light)
	made.Verb("pages", verb.NoShort, "serve the queue's one page coordinator",
		pagesHelp, "Example_queuePages", writes,
		previewMutation("pages", app.pages), root,
		pzm.Value("renders", verb.NoShort, "the absolute renders root"),
		pzm.Switch("serve", verb.NoShort, "run until stopped"))
}

// pages serves the coordinator for the queue until the call's context
// ends.
func (app queueApplication) pages(call pzm.Call) error {
	queueRoot, renders := call.Value("queue"), call.Value("renders")
	if len(call.Args()) != 0 || !call.On("serve") ||
		!filepath.IsAbs(queueRoot) || !filepath.IsAbs(renders) {
		return errors.New("pages needs --serve, an absolute --queue and " +
			"--renders, and no positional arguments")
	}
	log, err := pages.OpenLog(queueRoot)
	if err != nil {
		return err
	}
	defer log.Close()
	fmt.Fprintf(call.Out(), "pages: serving %s for %s; the log is %s\n",
		renders, queueRoot, log.Name())
	return pages.Serve(app.ctx, queueRoot, pages.Settings{Renders: renders,
		Encoder: pages.FoundFfmpeg(), Now: time.Now,
		Location: time.Local, Every: pages.TickEvery}, log)
}
