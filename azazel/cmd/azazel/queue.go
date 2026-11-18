package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/azazel/internal/queue"
	"github.com/janearc/slayer/azazel/internal/telemetry"
	"github.com/janearc/slayer/internal/observability"
)

type queueApplication struct{ ctx context.Context }

// queueTool declares durable scheduling verbs through Puzzlemouth.
func queueTool(ctx context.Context) *pzm.Tool {
	made := stamped()
	app := queueApplication{ctx: ctx}
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)
	changes := verb.NewFacts(verb.WritesTo("the shared queue directory"), verb.Changes, verb.ThisMachine, verb.NoOne, verb.Light)
	runs := verb.NewFacts(verb.WritesTo("job outputs and queue logs"), verb.Changes, verb.ThisMachine, verb.NoOne, verb.FullGPU)
	root := pzm.Value("queue", verb.NoShort, "absolute shared queue directory")
	gpuJobs := pzm.Value("gpu-jobs", verb.NoShort, "maximum concurrent GPU attempts, 1 through 64 (default 4)")
	made.Verb("tune", verb.NoShort, "change running queue admission", "# tune\n\nApply durable live controls without restarting. Reducing concurrency lets active jobs finish.\n", "Example_queueTune", changes, previewMutation("tune", app.tune), root,
		pzm.Value("gpu-jobs", verb.NoShort, "live concurrent GPU jobs, 1 through 64"),
		pzm.Value("gpu-start-seconds", verb.NoShort, "seconds between GPU starts, 1 through 3600"),
		pzm.Value("pressure-policy", verb.NoShort, "stop or observe memory pressure"),
		pzm.Switch("reset", verb.NoShort, "restore worker defaults; use alone"))
	made.Verb("schema", verb.NoShort, "print the job request schema",
		"# schema\n\nPrint the schema for YAML and JSON job requests.\n", "Example_queueSchema", reads, func(call pzm.Call) error { _, err := call.Out().Write(requestSchema); return err })
	made.Verb("init", verb.NoShort, "create one shared queue",
		"# init\n\nCreate a new Azazel-owned render queue.\n", "Example_queueInit", changes, previewMutation("init", app.init), root,
		pzm.Value("game", verb.NoShort, "optional legacy Game executable"))
	made.Verb("submit", verb.NoShort, "submit a Slayer command request",
		"# submit\n\nSubmit REQUEST.yaml or REQUEST.json. Its command and arguments run unchanged.\n", "Example_queueSubmit", changes, previewMutation("submit", app.submit), root)
	made.Verb("status", verb.NoShort, "read ordered jobs and their results",
		"# status\n\nPrint a JSON snapshot, including held and failed work.\n", "Example_queueStatus", reads, app.status, root)
	made.Verb("cancel", verb.NoShort, "cancel a job from its submitting agent",
		"# cancel\n\nCancel SEQUENCE. Running work stops under its supervisor.\n", "Example_queueCancel", changes, previewMutation("cancel", app.cancel), root,
		pzm.Value("agent", verb.NoShort, "the agent that submitted this job"))
	made.Verb("pause", verb.NoShort, "pause work with a recorded reason",
		"# pause\n\nKeep pending jobs and stop active work.\n", "Example_queuePause", changes, previewMutation("pause", app.pause), root,
		pzm.Value("reason", verb.NoShort, "why work must stop"))
	made.Verb("resume", verb.NoShort, "explicitly clear the queue pause",
		"# resume\n\nClear the latch. Host conditions are checked before starting.\n", "Example_queueResume", changes, previewMutation("resume", app.resume), root)
	made.Verb("work", verb.NoShort, "fill available GPU and CPU capacity",
		"# work\n\nDrain eligible work with measured GPU and host admission.\nWith --serve, wait for later submissions. Azazel owns render admission.\nRestart adopts surviving supervisors and requeues unstarted attempts.\n", "Example_queueWork", runs, previewMutation("work", app.work), root,
		pzm.Switch("serve", verb.NoShort, "wait for more jobs after draining"), gpuJobs,
		pzm.Value("renders", verb.NoShort, "absolute render root for automatic publication"))
	made.Verb("work-gpu", verb.NoShort, "retired GPU coordinator",
		"# work-gpu\n\nRetired; work owns GPU and CPU render admission.\n", "Example_queueGPUWork", runs, previewMutation("work-gpu", app.gpuWork), root, gpuJobs)
	made.Verb("recover", verb.NoShort, "resolve an interrupted attempt",
		"# recover\n\nResolve SEQUENCE only after verifying its work and all descendants\nhave stopped. A dead worker alone is insufficient. No PID is signaled.\n", "Example_queueRecover", changes, previewMutation("recover", app.recover), root,
		pzm.Value("worker", verb.NoShort, "exact recorded worker identity"),
		pzm.Switch("stopped", verb.NoShort, "confirm that this attempt and its descendants have stopped"))
	made.Verb("execute", verb.NoShort, "retired Game supervisor entry",
		"# execute\n\nRetired; render supervisors use execute-owned.\n", "Example_queueExecute", runs, previewMutation("execute", app.execute), root,
		pzm.Value("worker", verb.NoShort, "the selecting worker identity"))
	made.Verb("execute-owned", verb.NoShort, "supervise an Azazel-owned attempt",
		"# execute-owned\n\nInternal worker entry for a claimed render lease.\n", "Example_queueExecuteOwned", runs, previewMutation("execute-owned", app.executeOwned), root,
		pzm.Value("worker", verb.NoShort, "the selecting worker identity"))
	publishVerb(made, app)
	pagesVerb(made, app, root)
	return made
}

// store requires every agent to name the same shared root explicitly.
func (app queueApplication) tune(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return fmt.Errorf("tune takes no positional arguments")
	}
	s, err := store(call)
	if err != nil {
		return err
	}
	change := queue.Tuning{Pressure: call.Value("pressure-policy")}
	for _, option := range []struct {
		name   string
		target *int
	}{{"gpu-jobs", &change.GPUJobs}, {"gpu-start-seconds", &change.GPUStartSeconds}} {
		if value := call.Value(option.name); value != "" {
			*option.target, err = strconv.Atoi(value)
			if err != nil || *option.target < 1 {
				return fmt.Errorf("invalid --%s", option.name)
			}
		}
	}
	if call.On("reset") {
		if change != (queue.Tuning{}) {
			return fmt.Errorf("--reset cannot be combined with tuning values")
		}
		if err := s.ResetTuning(app.ctx); err != nil {
			return err
		}
		return app.status(call)
	}
	if err := s.Tune(app.ctx, change); err != nil {
		return err
	}
	return app.status(call)
}

// store requires every agent to name the same shared root explicitly.
func store(call pzm.Call) (queue.Store, error) {
	if call.Value("queue") == "" {
		return queue.Store{}, fmt.Errorf("--queue is required")
	}
	return queue.At(call.Value("queue")), nil
}

// init creates rather than resets existing queue state.
func (app queueApplication) init(call pzm.Call) error {
	s, err := store(call)
	if err != nil {
		return err
	}
	if len(call.Args()) != 0 {
		return fmt.Errorf("init takes no positional arguments")
	}
	return s.Init(app.ctx, queue.Config{Game: call.Value("game")})
}

// readRequest refuses unknown fields, trailing documents and oversized input.
func readRequest(path string) (queue.Request, error) {
	var request queue.Request
	file, err := os.Open(path)
	if err != nil {
		return request, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil {
		return request, err
	}
	if len(data) > 1024*1024 {
		return request, fmt.Errorf("request exceeds one MiB")
	}
	return decodeRequest(data)
}

// submit prints the durable acknowledgement, including on an identical retry.
func (app queueApplication) submit(call pzm.Call) error {
	s, err := store(call)
	if err != nil {
		return err
	}
	if len(call.Args()) != 1 {
		return fmt.Errorf("submit needs one request file")
	}
	request, err := readRequest(call.Args()[0])
	if err != nil {
		return err
	}
	job, err := s.Submit(app.ctx, request)
	if err != nil {
		return err
	}
	return json.NewEncoder(call.Out()).Encode(job)
}

// status prints the complete retained job ledger.
func (app queueApplication) status(call pzm.Call) error {
	s, err := store(call)
	if err != nil {
		return err
	}
	if len(call.Args()) != 0 {
		return fmt.Errorf("status takes no positional arguments")
	}
	snapshot, err := s.Status(app.ctx)
	if err != nil {
		return err
	}
	return json.NewEncoder(call.Out()).Encode(snapshot)
}

// sequence reads exactly one positive queue sequence.
func sequence(call pzm.Call) (uint64, error) {
	if len(call.Args()) != 1 {
		return 0, fmt.Errorf("one job sequence is required")
	}
	value, err := strconv.ParseUint(call.Args()[0], 10, 64)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("invalid job sequence")
	}
	return value, nil
}

// cancel leaves active ownership intact until its group has ended.
func (app queueApplication) cancel(call pzm.Call) error {
	s, err := store(call)
	if err != nil {
		return err
	}
	seq, err := sequence(call)
	if err != nil {
		return err
	}
	return s.Cancel(app.ctx, seq, call.Value("agent"))
}

// pause records an operator-selected reason.
func (app queueApplication) pause(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return fmt.Errorf("pause takes no positional arguments")
	}
	s, err := store(call)
	if err != nil {
		return err
	}
	return s.Pause(app.ctx, call.Value("reason"))
}

// resume explicitly clears a durable stop.
func (app queueApplication) resume(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return fmt.Errorf("resume takes no positional arguments")
	}
	s, err := store(call)
	if err != nil {
		return err
	}
	return s.Resume(app.ctx)
}

// work owns render admission while keeping the queue's worker lock.
func (app queueApplication) work(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return fmt.Errorf("work takes no positional arguments")
	}
	s, err := workerStore(call)
	if err != nil {
		return err
	}
	collector := telemetry.NewCollector(s, call.Value("queue"))
	stopReporter := telemetry.StartReporter(app.ctx, collector, call.Out())
	defer stopReporter()
	if os.Getenv("SLAYER_METRICS_ADDR") != "" || os.Getenv("SLAYER_PPROF_ADDR") != "" {
		stop, err := observability.Start("azazel")
		if err != nil {
			fmt.Fprintln(call.Out(), "observability unavailable:", err)
		} else {
			defer stop()
			if os.Getenv("SLAYER_METRICS_ADDR") != "" {
				unregister := observability.RegisterCollector(collector.Collect)
				defer unregister()
			}
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	stopPages, err := app.workerPages(call)
	if err != nil {
		return err
	}
	defer stopPages()
	return s.Dispatch(app.ctx, queue.RenderExecutor{Store: s, Executable: executable}, queue.NativeCapacityProbe{}, call.On("serve"))
}

// gpuWork is the private Game-lane coordinator started by the main worker.
func (app queueApplication) gpuWork(call pzm.Call) error {
	if len(call.Args()) != 0 {
		return fmt.Errorf("work-gpu takes no positional arguments")
	}
	s, err := workerStore(call)
	if err != nil {
		return err
	}
	_ = s
	return fmt.Errorf("work-gpu is retired; use the Azazel-owned work command")
}

// workerStore validates concurrency before starting either coordinator.
func workerStore(call pzm.Call) (queue.Store, error) {
	s, err := store(call)
	if err != nil {
		return s, err
	}
	limit := 4
	if value := call.Value("gpu-jobs"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil {
			return queue.Store{}, fmt.Errorf("invalid --gpu-jobs: %w", err)
		}
	}
	return s.WithGPUJobs(limit)
}

// recover requires the operator's exact attempt and stopped acknowledgement.
func (app queueApplication) recover(call pzm.Call) error {
	s, err := store(call)
	if err != nil {
		return err
	}
	seq, err := sequence(call)
	if err != nil {
		return err
	}
	return s.Recover(app.ctx, seq, call.Value("worker"), call.On("stopped"))
}

// execute refuses the former Game supervisor entry for new queues.
func (app queueApplication) execute(call pzm.Call) error {
	return fmt.Errorf("execute is retired; render supervisors use execute-owned")
}

// executeOwned runs the sequence named by the call under its worker.
func (app queueApplication) executeOwned(call pzm.Call) error {
	s, err := store(call)
	if err != nil {
		return err
	}
	seq, err := sequence(call)
	if err != nil {
		return err
	}
	return s.ExecuteOwned(app.ctx, seq, call.Value("worker"), call.Out())
}

//go:embed request.schema.json
var requestSchema []byte

// decodeRequest uses the shared checked YAML/JSON contract.
func decodeRequest(data []byte) (queue.Request, error) {
	return decodeRequestSchema(data, requestSchema)
}

// decodeRequestSchema refuses duplicate fields, extra documents and unknown keys.
func decodeRequestSchema(data, rules []byte) (queue.Request, error) {
	var request queue.Request
	schema, err := yamlfile.NewSchema("azazel-request.schema.json", rules)
	if err != nil {
		return request, err
	}
	node, err := schema.Read("request", data)
	if err != nil {
		return request, err
	}
	err = node.Decode(&request)
	return request, err
}
