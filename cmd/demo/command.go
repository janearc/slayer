package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"golang.org/x/mod/modfile"
)

// showNamed selects an output and presents the actual command's result.
func showNamed(ctx context.Context, call pzm.Call,
	given preview, preset preset) error {
	sizing := call.Given("width") || call.Given("height") ||
		call.Given("rays")
	if preset.SceneSized && sizing {
		return fmt.Errorf("%s uses film size and sampling from its set",
			preset.Name)
	}
	output, err := filepath.Abs(given.output)
	if err != nil {
		return err
	}
	target := filepath.Join(output, preset.Name+".png")
	if call.DryRun() {
		_, err := fmt.Fprintf(call.Out(),
			"would draw %s -> %s\n",
			preset.Name, target)
		return err
	}
	err = drawNamed(ctx, call.Out(), output, given, preset)
	if err != nil {
		return err
	}
	if preset.Plan != "" {
		first := filepath.Join(output, preset.Name, "frame-000000.png")
		raw, err := os.ReadFile(first)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, raw, 0644); err != nil {
			return err
		}
	}
	if call.On("terminal") {
		err := terminalFilePreview(call.Out(), target, given)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(call.Out(), target)
	return err
}

// drawNamed builds the selected scene command and runs it in the checkout.
func drawNamed(ctx context.Context, out io.Writer, output string,
	given preview, preset preset) error {
	root, err := checkoutRoot()
	if err != nil {
		return err
	}
	binary := filepath.Join(output, ".bin", preset.Command)
	if err := os.MkdirAll(filepath.Dir(binary), 0755); err != nil {
		return err
	}
	build := exec.CommandContext(ctx, "go", "build", "-p=1", "-o", binary,
		"./cmd/"+preset.Command)
	build.Dir = root
	build.Stdout, build.Stderr = out, out
	if err := build.Run(); err != nil {
		return fmt.Errorf("build demo %s: %w", preset.Name, err)
	}
	target := filepath.Join(output, preset.Name+".png")
	arguments := preset.arguments(given, target)
	return runDemo(ctx, root, binary, arguments, out, given.grace)
}

// arguments adds preview settings while respecting each command's output.
func (preset preset) arguments(given preview,
	target string) []string {
	if preset.Plan != "" {
		frames := filepath.Join(filepath.Dir(target), preset.Name)
		return []string{"sequence", "--plan", preset.Plan,
			"--output", frames, "--resume"}
	}
	arguments := append([]string(nil), preset.Arguments...)
	if !preset.SceneSized {
		arguments = append(arguments, "-w", fmt.Sprint(given.width),
			"-h", fmt.Sprint(given.height),
			"-n", fmt.Sprint(given.rays))
	}
	if preset.RawPreview {
		raw := target[:len(target)-len(filepath.Ext(target))] + ".pfm"
		return append(arguments, "-out", raw, "-preview", target)
	}
	return append(arguments, "-out", target)
}

// runDemo forwards cancellation to the owned renderer and bounds shutdown.
func runDemo(ctx context.Context, root, binary string,
	arguments []string, out io.Writer, grace time.Duration) error {
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Dir = root
	command.Stdout, command.Stderr = out, out
	stop := processStopper{command}
	command.Cancel = stop.interrupt
	command.WaitDelay = grace
	if err := command.Run(); err != nil {
		return fmt.Errorf("demo renderer: %w", err)
	}
	return nil
}

// processStopper owns cancellation after exec has started its process.
type processStopper struct{ command *exec.Cmd }

// interrupt lets the renderer handle its terminal cleanup before WaitDelay.
func (stop processStopper) interrupt() error {
	return stop.command.Process.Signal(os.Interrupt)
}

// checkoutRoot finds this module without depending on a private directory.
func checkoutRoot() (string, error) {
	here, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		raw, err := os.ReadFile(filepath.Join(here, "go.mod"))
		if err == nil &&
			modfile.ModulePath(raw) == "github.com/janearc/slayer" {
			return here, nil
		}
		parent := filepath.Dir(here)
		if parent == here {
			break
		}
		here = parent
	}
	return "", fmt.Errorf("named demos need the Slayer checkout; " +
		"run from its root")
}
