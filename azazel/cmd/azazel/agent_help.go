package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
)

//go:embed agent-help.txt
var agentGuide string

// runQueue keeps Pzm's declarations and adds the complete agent workflow.
func runQueue(ctx context.Context, args []string, out, errs io.Writer) int {
	code := queueTool(ctx).Run(args, out, errs)
	if code != 0 || len(args) != 1 || args[0] != "--agent-help" {
		return code
	}
	if _, err := io.WriteString(out, agentGuide); err != nil {
		fmt.Fprintln(errs, err)
		return exitUsage
	}
	return 0
}
