package workspace

import (
	"os/exec"
	"path/filepath"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

// imageCaption offers the received output alongside the image it belongs to.
func (work *Workspace) imageCaption(actions ...widget.Widget) *primitives.BoxWidget {
	children := []widget.Widget{primitives.Expanded(work.statusLine(func() string { return work.caption }))}
	children = append(children, actions...)
	if work.renderOutputPath() != "" {
		children = append(children, work.action("Show in Finder", false, func() bool {
			return work.renderOutputPath() == ""
		}, work.revealRender))
	}
	return primitives.HBox(children...).Gap(12)
}

// renderOutputPath excludes the bundled starting photograph.
func (work *Workspace) renderOutputPath() string {
	if work.output == filepath.Join(work.options.Starter, "preview.png") {
		return ""
	}
	return work.output
}

// revealRender resolves the current accepted image, even if the layout is unchanged.
func (work *Workspace) revealRender() {
	path := work.renderOutputPath()
	if path == "" {
		return
	}
	if err := finderRevealCommand(path).Run(); err != nil {
		work.message("Could not show render in Finder", path+"\n\n"+err.Error())
	}
}

// finderRevealCommand selects the image in its enclosing Finder folder.
func finderRevealCommand(path string) *exec.Cmd {
	return exec.Command("/usr/bin/open", "-R", path)
}
