package main

import (
	"encoding/json"

	"github.com/janearc/puzzlemouth/pzm"
)

// previewMutation reports intent without invoking any stateful handler.
func previewMutation(action string, apply pzm.Do) pzm.Do {
	return func(call pzm.Call) error {
		if !call.DryRun() {
			return apply(call)
		}
		preview := struct {
			DryRun  bool           `json:"dry_run"`
			Action  string         `json:"action"`
			Queue   string         `json:"queue"`
			Args    []string       `json:"args"`
			Options map[string]any `json:"options"`
			Notice  string         `json:"notice"`
		}{
			DryRun: true, Action: action, Queue: call.Value("queue"), Args: call.Args(),
			Options: previewOptions(action, call),
			Notice:  "intent only; inputs and queue state are not validated; nothing was submitted or executed",
		}
		return json.NewEncoder(call.Out()).Encode(preview)
	}
}

// previewOptions reads only flags declared by this action.
func previewOptions(action string, call pzm.Call) map[string]any {
	options := map[string]any{}
	if action == "tune" {
		options["reset"] = call.On("reset")
		for _, key := range []string{"gpu-jobs", "gpu-start-seconds", "pressure-policy"} {
			options[key] = call.Value(key)
		}
	}
	value := map[string]string{
		"init": "game", "cancel": "agent", "pause": "reason",
		"recover": "worker", "execute": "worker", "execute-owned": "worker",
		"work": "gpu-jobs", "work-gpu": "gpu-jobs",
	}[action]
	if value != "" {
		options[value] = call.Value(value)
	}
	flag := map[string]string{"work": "serve", "recover": "stopped"}[action]
	if flag != "" {
		options[flag] = call.On(flag)
	}
	return options
}
