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
		OutputRoot: work.evidenceFolder(), DirectoryPrefix: previewDirectoryPrefix(now, work.options.Tools.Agent),
		Profile: preview.Profile{Camera: work.working.cameraName, CameraWide: work.working.cameraWide, CameraPathRecipe: work.working.cameraPath},
	}
	if work.composition == "play" {
		request.Profile.Width, request.Profile.Height = 640, 400
		request.Profile = work.ownRender(request.Profile)
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
