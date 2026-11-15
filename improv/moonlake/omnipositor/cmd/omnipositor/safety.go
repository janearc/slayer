package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

type hostUsage struct {
	load float64
	swap int64
}

// parseHostUsage reads the text of the sysctl vm.loadavg and vm.swapusage
// values. It keeps the first of the three load averages and the used swap in
// bytes, rounded up, with the suffix B, K, M, G or T meaning powers of 1024.
// It refuses loads that are not three finite non-negative numbers, and swap
// text with no valid used figure.
func parseHostUsage(load, swap string) (hostUsage, error) {
	var usage hostUsage
	fields := strings.Fields(strings.Trim(load, "{} \n\t"))
	if len(fields) != 3 {
		return usage, errors.New("host load: expected three load averages")
	}
	for i, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return usage, errors.New("host load: invalid load average")
		}
		if i == 0 {
			usage.load = value
		}
	}
	fields = strings.Fields(swap)
	for i := 0; i+2 < len(fields); i++ {
		if fields[i] != "used" || fields[i+1] != "=" {
			continue
		}
		text := fields[i+2]
		if len(text) < 2 {
			break
		}
		multiplier, ok := map[byte]float64{'B': 1, 'K': 1024, 'M': 1 << 20, 'G': 1 << 30, 'T': 1 << 40}[text[len(text)-1]]
		value, err := strconv.ParseFloat(text[:len(text)-1], 64)
		value *= multiplier
		if !ok || err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= math.Exp2(63) {
			break
		}
		usage.swap = int64(math.Ceil(value))
		return usage, nil
	}
	return usage, errors.New("host swap: invalid used-byte statistic")
}

type hostOutput struct{ bytes.Buffer }

// Write adds p to the captured output, or refuses all of it if the total would
// pass 4096 bytes.
func (b *hostOutput) Write(p []byte) (int, error) {
	if len(p) > 4096-b.Len() {
		return 0, errors.New("host statistics exceed 4096 bytes")
	}
	return b.Buffer.Write(p)
}

// readHostUsage asks /usr/sbin/sysctl for vm.loadavg and vm.swapusage, each
// with five seconds to answer, and parses them. It refuses on any system other
// than Darwin.
func readHostUsage(ctx context.Context) (hostUsage, error) {
	if runtime.GOOS != "darwin" {
		return hostUsage{}, errors.New("host admission: this command currently supports Darwin host statistics only")
	}
	query := func(key string) (string, error) {
		child, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(child, "/usr/sbin/sysctl", "-n", key)
		var output hostOutput
		cmd.Stdout = &output
		cmd.Stderr = &output
		cmd.WaitDelay = time.Second
		if err := cmd.Run(); err != nil {
			if child.Err() != nil {
				err = child.Err()
			}
			return "", fmt.Errorf("host admission %s: %w", key, err)
		}
		return output.String(), nil
	}
	load, err := query("vm.loadavg")
	if err != nil {
		return hostUsage{}, err
	}
	swap, err := query("vm.swapusage")
	if err != nil {
		return hostUsage{}, err
	}
	return parseHostUsage(load, swap)
}

// gatePause waits one second, or less if ctx ends first, in which case it
// returns the error of ctx.
func gatePause(ctx context.Context) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// gibibyte is how many bytes a GiB is, for saying swap as a person reads it
const gibibyte = 1 << 30

// held is why the gate holds a host at this usage: which of the load and
// the swap is over its limit, each with its reading and the setting that
// limits it. It is empty when the gate does not hold.
func held(usage hostUsage, g settings.Gates) string {
	var over []string
	if usage.load > g.LoadLimit {
		over = append(over, fmt.Sprintf("the load is %.2f, over gates.load_limit of %.2f", usage.load, g.LoadLimit))
	}
	if usage.swap > g.SwapLimitBytes {
		over = append(over, fmt.Sprintf("swap in use is %.2f GiB, over gates.swap_limit_bytes of %.2f GiB",
			float64(usage.swap)/gibibyte, float64(g.SwapLimitBytes)/gibibyte))
	}
	return strings.Join(over, ", and ")
}

// waitGate waits until the host is under its load and swap limits. It
// never waits without saying so: the first time it holds, and again
// whenever what holds it changes, it tells told which limit is over, by
// how much, and how to go on. A gate that held in silence looked like a
// hang, on 2026-11-11, for ten minutes.
func waitGate(ctx context.Context, g settings.Gates, read func(context.Context) (hostUsage, error), pause func(context.Context) error, told io.Writer) error {
	if math.IsNaN(g.LoadLimit) || math.IsInf(g.LoadLimit, 0) || g.LoadLimit <= 0 || g.SwapLimitBytes < 0 {
		return errors.New("host admission: invalid load or swap limit")
	}
	if told == nil {
		return errors.New("host admission: nowhere to say why it waits")
	}
	var saidLoad, saidSwap bool
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		usage, err := read(ctx)
		if err != nil {
			return err
		}
		if math.IsNaN(usage.load) || math.IsInf(usage.load, 0) || usage.load < 0 || usage.swap < 0 {
			return errors.New("host admission: invalid host statistics")
		}
		if usage.load <= g.LoadLimit && usage.swap <= g.SwapLimitBytes {
			return ctx.Err()
		}
		overLoad, overSwap := usage.load > g.LoadLimit, usage.swap > g.SwapLimitBytes
		if overLoad != saidLoad || overSwap != saidSwap {
			if _, err := fmt.Fprintf(told, "omnipositor: waiting, and compositing nothing: %s. "+
				"It looks again every second and goes on by itself once the machine is under its limits. "+
				"To go on now, raise the limit in the settings' gates; to stop, interrupt it.\n",
				held(usage, g)); err != nil {
				return fmt.Errorf("host admission: saying why it waits: %w", err)
			}
			saidLoad, saidSwap = overLoad, overSwap
		}
		if err := pause(ctx); err != nil {
			return err
		}
	}
}

// readInput reads the whole file at path, which must be a regular file of at
// most limit bytes, and refuses one that is not, or that grows past limit as
// it is read. It opens without blocking so a pipe cannot stall it, and checks
// ctx between reads of 32 KiB.
func readInput(ctx context.Context, path string, limit int64) (data []byte, err error) {
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("input %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input %s: expected regular file", path)
	}
	if limit <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("input %s: exceeds %d bytes", path, limit)
	}
	data = make([]byte, 0, int(min(info.Size(), limit)))
	buffer := make([]byte, 32768)
	reader := io.LimitReader(f, limit+1)
	for {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		n, e := reader.Read(buffer)
		data = append(data, buffer[:n]...)
		if int64(len(data)) > limit {
			return nil, fmt.Errorf("input %s: exceeds %d bytes", path, limit)
		}
		if e == io.EOF {
			return data, nil
		}
		if e != nil {
			return nil, e
		}
	}
}

// outputDiskSpace runs check with floor on the output directory, then on the
// directories holding the movie and the trace when those are set. It returns
// the first error.
func outputDiskSpace(o stretchOptions, floor int64, check func(string, int64) error) error {
	if err := check(o.out, floor); err != nil {
		return err
	}
	if o.encode != "" {
		if err := check(filepath.Dir(o.encode), floor); err != nil {
			return err
		}
	}
	if o.trace != "" {
		if err := check(filepath.Dir(o.trace), floor); err != nil {
			return err
		}
	}
	return nil
}

type admittedLight struct {
	stage compositor.LightStage
	check func() error
}

// Light runs the wrapped stage on r only after check allows it, and returns
// the error from check, lighting nothing, if it does not.
func (s admittedLight) Light(ctx context.Context, r compositor.LightRequest) (compositor.LightResult, error) {
	if err := s.check(); err != nil {
		return compositor.LightResult{}, err
	}
	return s.stage.Light(ctx, r)
}
