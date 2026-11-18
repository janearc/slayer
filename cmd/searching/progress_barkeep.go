package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
)

// barkeepProgress uses Carrie's existing local API. Signals show current
// state; timestamped messages are occasional snapshots, not a live widget.
type barkeepProgress struct {
	address, kind, signal, state string
	client                       *http.Client
	last                         time.Time
}

// progressBar requires the owner's configured kind and signal explicitly.
// A loopback-only client ignores proxy environment settings and redirects.
func progressBar(call pzm.Call) (*barkeepProgress, error) {
	if !call.Given("barkeep") {
		return nil, nil
	}
	address, err := loopbackBar(call.Value("barkeep"))
	if err != nil {
		return nil, err
	}
	if call.Value("barkeep-kind") == "" || call.Value("barkeep-signal") == "" {
		return nil, fmt.Errorf("Barkeep needs its configured --barkeep-kind and --barkeep-signal")
	}
	return &barkeepProgress{address: address, kind: call.Value("barkeep-kind"), signal: call.Value("barkeep-signal"),
		client: &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil},
			CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("Barkeep redirects are refused") }}}, nil
}

// loopbackBar confines status updates to a literal local IP and port.
func loopbackBar(address string) (string, error) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Port() == "" ||
		parsed.Path != "" && parsed.Path != "/" || !net.ParseIP(parsed.Hostname()).IsLoopback() {
		return "", fmt.Errorf("Barkeep address must be http://LOOPBACK-IP:PORT without credentials or a path")
	}
	return strings.TrimSuffix(address, "/"), nil
}

// update leaves a busy message queue alone. In particular, an absent reader
// cannot accumulate hundreds of obsolete percentages. Signals still update.
func (bar *barkeepProgress) update(ctx context.Context, reading progressReading) error {
	status, err := bar.request(ctx, "/status", nil)
	if err != nil {
		return err
	}
	state := "running"
	if reading.Status == "completed movie" {
		state = "pass"
	} else if strings.HasPrefix(reading.Status, "stopped") {
		state = "fail"
	}
	if bar.state != state {
		if _, err := bar.request(ctx, "/signal", map[string]string{"name": bar.signal, "state": state}); err != nil {
			return err
		}
		bar.state = state
	}
	if status > 0 || !bar.last.IsZero() && reading.Updated.Sub(bar.last) < time.Minute && state == "running" {
		return nil
	}
	if _, err := bar.request(ctx, "/say", map[string]string{"text": reading.line(), "kind": bar.kind, "for": "5s"}); err != nil {
		return err
	}
	bar.last = reading.Updated
	return nil
}

// request bounds both waiting and response size; a refused allowlist or an
// unavailable listener is reported instead of claiming the bar was updated.
func (bar *barkeepProgress) request(ctx context.Context, path string, payload any) (int, error) {
	method, raw := http.MethodGet, []byte(nil)
	if payload != nil {
		var err error
		raw, err = json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		method = http.MethodPost
	}
	request, err := http.NewRequestWithContext(ctx, method, bar.address+path, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := bar.client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8193))
	if err != nil {
		return 0, err
	}
	if len(body) > 8192 || response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Barkeep %s returned %s", path, response.Status)
	}
	var status struct {
		Queued *int `json:"queued"`
	}
	if err := json.Unmarshal(body, &status); err != nil || status.Queued == nil || *status.Queued < 0 {
		return 0, fmt.Errorf("invalid Barkeep status response")
	}
	return *status.Queued, nil
}
