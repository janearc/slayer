package workspace

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/oakland"
	"github.com/janearc/slayer/set"
)

func (work *Workspace) saveDraft(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	from := work.state.path
	document := work.state.document
	if !filepath.IsAbs(from) || filepath.Dir(from) == filepath.Dir(absolute) {
		return "", document.Save(path)
	}
	resources, err := checkedImportResources(document, from)
	if err != nil {
		notice := fmt.Sprintf("Draft saved as written. Its dependencies could not be checked or copied: %v", err)
		return notice, document.Save(absolute)
	}
	original := document.Text()
	if err := document.RebaseProfileBooks(from, absolute); err != nil {
		return "", err
	}
	if err := saveSetCopy(document, from, absolute, resources); err != nil {
		document.Replace(original)
		return "", err
	}
	if document.Text() != original {
		work.editor.ApplyText(document.Text())
	}
	return "", nil
}

func checkedImportResources(document *draft.Document, path string) (map[string]oakland.Source, error) {
	found, err := document.HasImportedGeometry()
	if err != nil || !found {
		return nil, err
	}
	checked, err := set.Read(path, []byte(document.Text()))
	if err != nil {
		return nil, err
	}
	return importResources(checked), nil
}

func saveSetCopy(document *draft.Document, from, to string, originalResources map[string]oakland.Source) error {
	if len(originalResources) == 0 {
		return document.Save(to)
	}
	source, err := os.OpenRoot(filepath.Dir(from))
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.OpenRoot(filepath.Dir(to))
	if err != nil {
		return err
	}
	defer destination.Close()

	copies := importCopies{root: destination}
	defer copies.rollback()
	for _, name := range slices.Sorted(maps.Keys(originalResources)) {
		if strings.EqualFold(name, filepath.Base(to)) {
			return fmt.Errorf("the save filename conflicts with imported file %q", name)
		}
		if err := copies.install(source, name, originalResources[name]); err != nil {
			return fmt.Errorf("copy imported file %q: %w", name, err)
		}
	}
	checked, err := set.Read(to, []byte(document.Text()))
	if err != nil {
		return fmt.Errorf("check copied imports: %w", err)
	}
	if !maps.Equal(originalResources, importResources(checked)) {
		return fmt.Errorf("copied imports resolve to different files")
	}
	if err := document.Save(to); err != nil {
		return err
	}
	copies.created = nil
	return nil
}

func importResources(imports set.Set) map[string]oakland.Source {
	resources := make(map[string]oakland.Source)
	world, _ := imports.World()
	for _, part := range world.Imports() {
		for _, source := range part.Asset().SourceClosure() {
			resources[filepath.FromSlash(source.Path())] = source
		}
	}
	return resources
}
