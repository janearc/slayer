package preview

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// invocation is one external Azazel client command; arguments never use a shell.
type invocation struct {
	Executable  string
	Arguments   []string
	Directory   string
	Environment []string
	Output      io.Writer
}

// String identifies a control call without logging inherited environment values.
func (call invocation) String() string { return fmt.Sprintf("%s %q", call.Executable, call.Arguments) }

// Runner is the subprocess boundary used by the external queue client.
type Runner interface {
	Run(context.Context, invocation) ([]byte, error)
}

type processRunner struct{}

// Run captures owner JSON separately from diagnostics and bounds client output.
func (processRunner) Run(ctx context.Context, call invocation) ([]byte, error) {
	command := exec.CommandContext(ctx, call.Executable, call.Arguments...)
	command.Dir, command.Env = call.Directory, call.Environment
	reply := limitedReply{}
	command.Stdout = &reply
	command.Stderr = call.Output
	command.WaitDelay = time.Second
	err := command.Run()
	return bytes.Clone(reply.Bytes()), err
}

// limitedReply bounds a full retained ledger response, not renderer output.
type limitedReply struct{ bytes.Buffer }

// Write refuses oversized control replies rather than accepting truncated JSON.
func (reply *limitedReply) Write(data []byte) (int, error) {
	const maximumReplyBytes = 64 * 1024 * 1024
	if reply.Len()+len(data) > maximumReplyBytes {
		return 0, fmt.Errorf("Azazel reply exceeds 64 MiB")
	}
	return reply.Buffer.Write(data)
}
