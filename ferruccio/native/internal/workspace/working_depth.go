package workspace

import (
	"fmt"

	"github.com/janearc/slayer/session/placement"
)

func farClipNotice(count int, groups string) string {
	return fmt.Sprintf("%s entirely beyond the nominal %d scene-unit depth limit at the staged camera (%d). Full renders use the scene’s geometry.",
		groups, placement.DisplayFar, count)
}

func omittedGiantNotice(count int) string {
	return fmt.Sprintf("Gas giant atmospheres are volumetric and omitted from the working view (%d).", count)
}
