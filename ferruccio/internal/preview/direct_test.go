package preview

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSlayer writes a shell script that stands in for Slayer: it records its
// arguments in args.txt beside it, then runs body.
func fakeSlayer(t *testing.T, body string) (string, string) {
	t.Helper()
	root := t.TempDir()
	args := filepath.Join(root, "args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$0\" \"$@\" > " + args + "\n" + body + "\n"
	path := filepath.Join(root, "slayer")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path, args
}

func TestDirectStartsSlayerItselfAndKeepsItsWholeOutput(t *testing.T) {
	t.Setenv(CacheVariable, t.TempDir())
	plan, tools := queuedPlan(t)
	var slayerArgs string
	tools.Slayer, slayerArgs = fakeSlayer(t, "echo 'first line'\necho 'slayer draw: the reason' >&2\nexit 1")
	tools.Transport = "slayer"
	err := runDirect(context.Background(), tools, plan, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "exit status 1") {
		t.Fatalf("a failing child was not reported: %v", err)
	}
	raw, readError := os.ReadFile(slayerArgs)
	if readError != nil {
		t.Fatal("the child was not started:", readError)
	}
	if strings.Contains(string(raw), "lane") || !strings.Contains(string(raw), "draw") {
		t.Fatalf("the child was started through a lane or without draw: %q", raw)
	}
	kept, readError := os.ReadFile(filepath.Join(plan.Directory, TranscriptName))
	if readError != nil || string(kept) != "first line\nslayer draw: the reason\n" {
		t.Fatalf("the child's output was not kept whole: %q, %v", kept, readError)
	}
}

func TestDirectRequestNamesTheRealArgvAndNoLane(t *testing.T) {
	t.Setenv(CacheVariable, t.TempDir())
	plan, tools := queuedPlan(t)
	tools.Slayer, _ = fakeSlayer(t, "exit 1")
	_ = runDirect(context.Background(), tools, plan, io.Discard, nil)
	raw, err := os.ReadFile(filepath.Join(plan.Directory, "direct-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	producer, _ := request["producer"].([]any)
	if len(producer) == 0 || producer[0] != tools.Slayer {
		t.Fatalf("the producer is not the Slayer that was run: %v", request["producer"])
	}
	if _, exists := request["game"]; exists || request["lane"] != "none" {
		t.Fatalf("a Game or a lane is claimed that was not used: %v", request)
	}
	recipe, err := os.ReadFile(filepath.Join(plan.Directory, "recipe.txt"))
	if err != nil || strings.Contains(string(recipe), "through Game") {
		t.Fatalf("the recipe claims Game: %q, %v", recipe, err)
	}
}

func TestDirectNeedsAnAbsoluteSlayerButNoGame(t *testing.T) {
	t.Setenv(CacheVariable, t.TempDir())
	plan, tools := queuedPlan(t)
	tools.Slayer, tools.Game = "slayer", ""
	err := runDirect(context.Background(), tools, plan, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "absolute shared Slayer") {
		t.Fatalf("a relative Slayer was started: %v", err)
	}
}

func TestDirectRefusesWithoutACacheBeforeStartingAnything(t *testing.T) {
	t.Setenv(CacheVariable, "")
	plan, tools := queuedPlan(t)
	var slayerArgs string
	tools.Slayer, slayerArgs = fakeSlayer(t, "exit 0")
	err := runDirect(context.Background(), tools, plan, io.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "--cache") || !strings.Contains(err.Error(), CacheVariable) {
		t.Fatalf("the refusal does not name the flag and the variable: %v", err)
	}
	if _, statError := os.Stat(slayerArgs); statError == nil {
		t.Fatal("Slayer was started without a cache folder")
	}
	if _, statError := os.Stat(filepath.Join(plan.Directory, "direct-request.json")); statError == nil {
		t.Fatal("a request was written for a render that never started")
	}
}

func TestRequireCacheTakesTheFlagOrTheVariable(t *testing.T) {
	t.Setenv(CacheVariable, "")
	if err := RequireCache(Tools{}); err == nil {
		t.Fatal("no flag and no variable was taken")
	}
	if err := RequireCache(Tools{Cache: "/cache"}); err != nil {
		t.Fatalf("the flag was refused: %v", err)
	}
	t.Setenv(CacheVariable, "/from/the/environment")
	if err := RequireCache(Tools{}); err != nil {
		t.Fatalf("the variable was refused: %v", err)
	}
}

func TestChildGetsTheCacheTheFlagGave(t *testing.T) {
	t.Setenv(CacheVariable, "/ambient")
	found := ""
	for _, entry := range environment(Tools{Cache: "/from/the/flag"}) {
		if value, ok := strings.CutPrefix(entry, CacheVariable+"="); ok {
			found = value
		}
	}
	if found != "/from/the/flag" {
		t.Fatalf("the child would get %q", found)
	}
}

func TestDirectRejectsReferenceEditsBeforeStartingOrAccepting(t *testing.T) {
	for _, phase := range []string{"before", "during", "unchanged"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			setPath := filepath.Join(dir, "rig.yaml")
			changed := strings.Replace(rigFixture, "stop: 8", "stop: 4", 1)
			if err := os.WriteFile(setPath, []byte(rigFixture), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "changed.yaml"), []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
			plan, err := Prepare(Request{
				Path: filepath.Join(dir, "job.yaml"), Text: rigJobFixture,
				Profile: Profile{Full: true}, OutputRoot: dir,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()

			queuePNG(t, &Plan{Output: filepath.Join(dir, "fixture.png")}, 18, 12)
			body := "cp fixture.png \"$4\""
			switch phase {
			case "before":
				if err := os.WriteFile(setPath, []byte(changed), 0600); err != nil {
					t.Fatal(err)
				}
			case "during":
				body = "cp changed.yaml rig.yaml\n" + body
			}
			producer, args := fakeSlayer(t, body)
			var last Update
			err = runDirect(context.Background(), Tools{Slayer: producer, Cache: dir}, plan, io.Discard,
				func(update Update) { last = update })
			receipt := filepath.Join(plan.Directory, "receipt.json")
			if phase == "unchanged" {
				if err != nil || !last.Terminal || last.State != "succeeded" {
					t.Fatalf("unchanged reference gave %v, state %+v", err, last)
				}
				raw, err := os.ReadFile(receipt)
				if err != nil || !strings.Contains(string(raw), "\"f_stop\": 8") {
					t.Fatalf("accepted receipt lost the rig: %s, %v", raw, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "referenced set changed") {
				t.Fatalf("changed reference gave %v", err)
			}
			if _, err := os.Stat(receipt); !os.IsNotExist(err) || last.State == "succeeded" {
				t.Fatalf("accepted changed reference: receipt %v, state %+v", err, last)
			}
			if phase == "before" {
				if _, err := os.Stat(args); !os.IsNotExist(err) {
					t.Fatalf("producer started with a stale profile: %v", err)
				}
			}
			if phase == "during" {
				if _, err := os.Stat(plan.Output); err != nil || !last.Terminal || last.State != "failed" {
					t.Fatalf("completed output or refusal lost: %v, state %+v", err, last)
				}
			}
		})
	}
}
