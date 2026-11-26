package preview

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/janearc/puzzlemouth/yamlfile"
)

//go:embed azazel-request.schema.json
var requestSchema []byte

type planState struct {
	mu        sync.Mutex
	active    bool
	uncertain bool
}

const requestName = "azazel-request.json"
const terminalName = "azazel-terminal.json"

// previewTimeoutSeconds limits a preview sent through the queue. The value
// was chosen: a preview that takes longer is not a preview.
const previewTimeoutSeconds = 300

// fullTimeoutSeconds is the limit sent with a full render. The request schema
// requires a limit, so this is the longest it accepts, seven days. It was
// chosen as the ceiling, not as a time a full render is expected to take.
const fullTimeoutSeconds = 604800

// timeoutSeconds is the queue limit for this plan.
func (plan *Plan) timeoutSeconds() int64 {
	if plan.Profile.Full {
		return fullTimeoutSeconds
	}
	return previewTimeoutSeconds
}

// freezeRequest persists exactly one immutable submission per prepared plan.
func (plan *Plan) freezeRequest(tools Tools) (queueRequest, error) {
	var request queueRequest
	if !filepath.IsAbs(tools.Azazel) || !filepath.IsAbs(tools.Queue) || tools.Agent == "" {
		return request, fmt.Errorf("configure absolute Azazel executable, shared queue and submitting agent")
	}
	if plan == nil || plan.state == nil || !filepath.IsAbs(plan.Input) || !filepath.IsAbs(plan.Directory) {
		return request, fmt.Errorf("prepare a frozen preview before queue submission")
	}
	argv, err := plan.arguments(tools)
	if err != nil {
		return request, err
	}
	inputs, err := plan.declaredInputs()
	if err != nil {
		return request, err
	}
	identity := sha256.Sum256([]byte(filepath.Clean(plan.Directory)))
	request = queueRequest{Agent: tools.Agent, Key: "ferruccio-" + hex.EncodeToString(identity[:]), Command: argv[0], Args: argv[1:],
		Directory: filepath.Dir(plan.Input), Lane: plan.lane(), TimeoutSeconds: plan.timeoutSeconds(),
		Inputs: inputs, Environment: queueEnvironment(tools)}
	raw, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return queueRequest{}, err
	}
	schema, err := yamlfile.NewSchema("azazel:job-request", requestSchema)
	if err != nil {
		return queueRequest{}, err
	}
	if _, err := schema.Read(requestName, raw); err != nil {
		return queueRequest{}, err
	}
	frozen, err := os.ReadFile(plan.Input)
	if err != nil {
		return queueRequest{}, err
	}
	audit, err := os.ReadFile(filepath.Join(plan.Directory, "scene.yaml"))
	if err != nil || !bytes.Equal(frozen, audit) {
		return queueRequest{}, fmt.Errorf("frozen preview input differs from retained audit YAML")
	}
	binding, err := json.Marshal(struct{ Azazel, Queue string }{tools.Azazel, tools.Queue})
	if err != nil {
		return queueRequest{}, err
	}
	if err := retainSubmissionFile(filepath.Join(plan.Directory, "azazel-client.json"), binding); err != nil {
		return queueRequest{}, fmt.Errorf("a retry must use the same Azazel executable and queue: %w", err)
	}
	path := filepath.Join(plan.Directory, requestName)
	if err := retainSubmissionFile(path, raw); err != nil {
		return queueRequest{}, fmt.Errorf("a retry must retain the same Azazel key and request content: %w", err)
	}
	if err := plan.writeRecipe(tools, request); err != nil {
		return queueRequest{}, err
	}

	return request, nil
}

// queueEnvironment forwards only the environment namespace accepted by Azazel.
func queueEnvironment(tools Tools) map[string]string {
	allowed := regexp.MustCompile("^SLAYER_[A-Z0-9_]+$")
	var result map[string]string
	for _, entry := range ChildEnvironment() {
		name, value, _ := strings.Cut(entry, "=")
		if allowed.MatchString(name) {
			if result == nil {
				result = map[string]string{}
			}
			result[name] = value
		}
	}
	if tools.Shelf != "" {
		if result == nil {
			result = map[string]string{}
		}
		result["SLAYER_SHELF"] = tools.Shelf
	}
	if tools.Cache != "" {
		if result == nil {
			result = map[string]string{}
		}
		result["SLAYER_CACHE"] = tools.Cache
	}
	if tools.Data != "" {
		if result == nil {
			result = map[string]string{}
		}
		result["SLAYER_DATA"] = tools.Data
	}
	workers := tools.Processors
	if workers == ProcessorsAll {
		workers = strconv.Itoa(min(runtime.NumCPU(), 1024))
	} else if count, err := strconv.Atoi(workers); err != nil || count < 1 || count > 1024 {
		workers = defaultProcessors
	}
	if result == nil {
		result = map[string]string{}
	}
	result["SLAYER_CPU_WORKERS"] = workers
	return result
}

// queuedInputNeeded is conservative across callers and process restarts.
func (plan *Plan) queuedInputNeeded() bool {
	raw, err := os.ReadFile(filepath.Join(plan.Directory, requestName))
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return true
	}
	var request queueRequest
	if yamlfile.Read(requestName, raw, &request) != nil {
		return true
	}
	terminal, err := os.ReadFile(filepath.Join(plan.Directory, terminalName))
	if err != nil {
		return true
	}
	job, err := decodeJob(terminal, request, 0)
	return err != nil || !terminalJob(job)
}

// sameRequest binds the full owner request, including producer arguments.
func sameRequest(first, second queueRequest) bool { return reflect.DeepEqual(first, second) }

// retainSubmissionFile never replaces an existing request or client binding.
func retainSubmissionFile(path string, raw []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		existing, readError := os.ReadFile(path)
		if readError != nil {
			return readError
		}
		if !bytes.Equal(existing, raw) {
			return fmt.Errorf("stored submission differs")
		}
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
