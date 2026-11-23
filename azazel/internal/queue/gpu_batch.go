package queue

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

// DispatchGame reserves Game's GPU lane for each nonempty GPU batch.
func (s Store) DispatchGame(ctx context.Context, executable string, serve bool, out io.Writer) error {
	var batch <-chan error
	tick := func(ctx context.Context) error {
		if batch != nil {
			select {
			case err := <-batch:
				batch = nil
				if err != nil {
					fmt.Fprintf(out, "GPU coordinator ended: %v; replacing batch from durable queue\n", err)
				}
			default:
				return nil
			}
		}
		state, err := s.Status(ctx)
		if err != nil {
			return err
		}
		if state.Paused != "" {
			return nil
		}
		for _, job := range state.Jobs {
			if job.Request.Lane == "gpu" && (job.State == Queued || job.State == Running) {
				batch, err = s.startGPUBatch(state.Config.Game, executable, out)
				return err
			}
		}
		return nil
	}
	return s.dispatch(ctx, GameExecutor{Store: s, Executable: executable}, NativeCapacityProbe{}, serve, cpuWork, tick)
}

// startGPUBatch detaches from worker signals; the durable ledger owns its lifetime.
func (s Store) startGPUBatch(game, executable string, out io.Writer) (<-chan error, error) {
	log, err := os.OpenFile(filepath.Join(s.root, "gpu-batches.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	defer log.Close()
	gpuLaneLimit := s.gpuLimit()
	command := exec.Command(game, "lane", "gpu", "--", executable, "work-gpu", "--queue", s.root, "--gpu-jobs", strconv.Itoa(gpuLaneLimit))
	command.Env = slices.DeleteFunc(os.Environ(), func(value string) bool { return strings.HasPrefix(value, "GAME_LANES=") })
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Stdout, command.Stderr = log, log
	if err := command.Start(); err != nil {
		return nil, err
	}
	fmt.Fprintf(out, "GPU batch coordinator pid %d; log %s\n", command.Process.Pid, log.Name())
	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()
	return finished, nil
}

// DispatchGPU shares its actual inherited Game lease among supervised renders.
func (s Store) DispatchGPU(ctx context.Context, executable string) error {
	if err := heldLane("gpu"); err != nil {
		return err
	}
	return s.dispatch(ctx, GameExecutor{Store: s, Executable: executable}, NativeCapacityProbe{}, false, gpuWork, nil)
}
