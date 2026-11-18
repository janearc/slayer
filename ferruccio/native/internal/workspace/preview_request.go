package workspace

import (
	"strings"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/preview"
)

// previewRequest keeps native outputs under the configured shared evidence root.
func (work *Workspace) previewRequest(now time.Time) preview.Request {
	request := preview.Request{
		Path: work.state.path, Text: work.state.document.Text(), Specimen: work.specimen,
		OutputRoot: work.options.Evidence, DirectoryPrefix: previewDirectoryPrefix(now, work.options.Tools.Agent),
	}
	if work.composition == "play" {
		request.Profile = preview.Profile{Width: 640, Height: 400, CPURenderer: true}
	}
	return request
}

// previewDirectoryPrefix uses a UTC date and a bounded safe agent basename.
func previewDirectoryPrefix(now time.Time, agent string) string {
	name := strings.Map(func(letter rune) rune {
		if letter >= 'a' && letter <= 'z' || letter >= '0' && letter <= '9' || letter == '-' || letter == '_' {
			return letter
		}
		return '-'
	}, strings.ToLower(agent))
	name = strings.Trim(name, "-_")
	if len(name) > 48 {
		name = strings.Trim(name[:48], "-_")
	}
	if name == "" {
		name = "unattributed"
	}
	return now.UTC().Format("2006-01-02") + "-" + name + "-ferruccio-preview-"
}
