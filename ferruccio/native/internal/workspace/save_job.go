package workspace

import "path/filepath"

func (work *Workspace) saveJob(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	document := work.state.document
	from := work.state.path
	if !filepath.IsAbs(from) || filepath.Dir(from) == filepath.Dir(absolute) {
		return document.Save(path)
	}
	original := document.Text()
	current, err := document.RebasedJobText(original, from, absolute)
	if err != nil {
		return err
	}
	document.Replace(current)
	if err := document.Save(path); err != nil {
		document.Replace(original)
		return err
	}
	work.editor.RelocateText(current, func(snapshot string) string {
		rebased, err := document.RebasedJobText(snapshot, from, absolute)
		if err != nil {
			return snapshot
		}
		return rebased
	})
	return nil
}
