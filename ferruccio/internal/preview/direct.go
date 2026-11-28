package preview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// runDirect is an explicitly authorized foreground shared-Slayer invocation.
// It starts Slayer itself, with no Game lane in between: the person at the
// keyboard does not queue behind batch work. Azazel's road is another one.
func runDirect(ctx context.Context, tools Tools, plan *Plan, output io.Writer, notify func(Update)) error {
	if ctx == nil || plan == nil || plan.state == nil {
		return fmt.Errorf("prepare a preview before rendering")
	}
	if !filepath.IsAbs(tools.Slayer) {
		return fmt.Errorf("configure an absolute shared Slayer executable path")
	}
	if plan.Kind == "catalogue" {
		return fmt.Errorf("direct Slayer preview does not interpret catalogue settings")
	}
	if err := RequireCache(tools); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(plan.Directory, requestName)); err == nil {
		return fmt.Errorf("prepare a new direct preview; this plan already belongs to Azazel")
	}
	if _, err := os.Lstat(plan.Output); err == nil {
		return fmt.Errorf("prepare a fresh preview output before rendering")
	}
	argv, err := plan.arguments(tools)
	if err != nil {
		return err
	}
	plan.state.mu.Lock()
	if plan.state.active || plan.state.uncertain {
		plan.state.mu.Unlock()
		return fmt.Errorf("this preview is already active or its prior stop is unconfirmed")
	}
	plan.state.active = true
	plan.state.mu.Unlock()
	defer func() { plan.state.mu.Lock(); plan.state.active = false; plan.state.mu.Unlock() }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plan.Profile.reference.checkCurrent(); err != nil {
		return err
	}
	if err := plan.writeDirectIntent(tools, argv); err != nil {
		return err
	}
	if output == nil {
		output = io.Discard
	}
	transcript, err := os.OpenFile(filepath.Join(plan.Directory, TranscriptName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("keep the render's output: %w", err)
	}
	defer transcript.Close()
	output = io.MultiWriter(output, transcript)
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Dir, command.Env = filepath.Dir(plan.Input), environment(tools)
	command.Stdout, command.Stderr = output, output
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		err := syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	command.WaitDelay = 3 * time.Second
	if err := command.Start(); err != nil {
		if notify != nil {
			notify(Update{State: "failed", Terminal: true})
		}
		return fmt.Errorf("start shared Slayer preview: %w", err)
	}
	plan.setDirectUncertain(true)
	runError := command.Wait()
	remaining, groupError := settleDirectGroup(command.Process.Pid)
	if groupError != nil {
		if notify != nil {
			notify(Update{State: "interrupted"})
		}
		return errors.Join(runError, ctx.Err(), fmt.Errorf("preview process-group stop unconfirmed; frozen input retained: %w", groupError))
	}
	plan.setDirectUncertain(false)
	if remaining && runError == nil {
		runError = fmt.Errorf("foreground renderer left active descendants")
	}
	if ctx.Err() != nil {
		if notify != nil {
			notify(Update{State: "cancelled", Terminal: true})
		}
		return ctx.Err()
	}
	if runError != nil {
		if notify != nil {
			notify(Update{State: "failed", Terminal: true})
		}
		return fmt.Errorf("shared Slayer preview failed: %w", runError)
	}
	err = plan.acceptDirect(tools, argv)
	if notify != nil {
		state := "succeeded"
		if err != nil {
			state = "failed"
		}
		notify(Update{State: state, Terminal: true})
	}
	return err
}

// setDirectUncertain records whether a started shared Slayer preview still
// has an unsettled process group. A true value blocks new direct previews.
func (plan *Plan) setDirectUncertain(value bool) {
	plan.state.mu.Lock()
	plan.state.uncertain = value
	plan.state.mu.Unlock()
}

// settleDirectGroup checks only the process group created by this invocation.
// After its foreground leader exits, any surviving descendants are stopped.
func settleDirectGroup(group int) (bool, error) {
	alive := syscall.Kill(-group, 0)
	if alive == syscall.ESRCH {
		return false, nil
	}
	if alive != nil {
		return false, alive
	}
	if err := syscall.Kill(-group, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		return true, err
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := syscall.Kill(-group, 0)
		if err == syscall.ESRCH {
			return true, nil
		}
		if err != nil {
			return true, err
		}
		if time.Now().After(deadline) {
			return true, fmt.Errorf("owned process group still exists")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
