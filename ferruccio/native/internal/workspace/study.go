package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/ferruccio/internal/draft"
)

var ErrStudyAssetConflict = errors.New("bundled study asset conflicts with an existing destination file")

type bundledStudyManifest struct {
	id, folder, entry string
	files             []string
}

var oaklandLarsonStudy = bundledStudyManifest{
	id: "oakland-larson", folder: "oakland-larson", entry: "set.yaml",
	files: []string{"two-metre-human.gltf", "two-metre-human.bin", "LICENSE.txt"},
}

// studyForExample identifies a bundled study by its relative entry path.
func studyForExample(name string) (bundledStudyManifest, bool) {
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == filepath.ToSlash(filepath.Join(oaklandLarsonStudy.folder, oaklandLarsonStudy.entry)) {
		return oaklandLarsonStudy, true
	}
	return bundledStudyManifest{}, false
}

// studyManifest resolves a study identity retained by an editable draft.
func studyManifest(id string) (bundledStudyManifest, bool) {
	if id == oaklandLarsonStudy.id {
		return oaklandLarsonStudy, true
	}
	return bundledStudyManifest{}, false
}

// copyBundledStudy materializes one trusted nested study and its exact local resources.
func copyBundledStudy(examples, name string) (string, string, error) {
	manifest, ok := studyForExample(name)
	if !ok {
		return "", "", fmt.Errorf("unknown bundled study example %q", name)
	}
	sourceFolder := filepath.Join(examples, manifest.folder)
	targetFolder, err := os.MkdirTemp("", "atelier-study-*")
	if err != nil {
		return "", "", err
	}
	for _, file := range append([]string{manifest.entry}, manifest.files...) {
		sourcePath := filepath.Join(sourceFolder, file)
		info, err := os.Lstat(sourcePath)
		if err != nil {
			os.RemoveAll(targetFolder)
			return "", "", err
		}
		if !info.Mode().IsRegular() {
			os.RemoveAll(targetFolder)
			return "", "", fmt.Errorf("bundled study resource is not a regular file: %s", file)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			os.RemoveAll(targetFolder)
			return "", "", err
		}
		if err := os.WriteFile(filepath.Join(targetFolder, file), data, 0600); err != nil {
			os.RemoveAll(targetFolder)
			return "", "", err
		}
	}
	return filepath.Join(targetFolder, manifest.entry), manifest.id, nil
}

// bundledStudyFiles returns only authored non-YAML resources for a known study.
func bundledStudyFiles(study string) []string {
	manifest, ok := studyManifest(study)
	if !ok {
		return nil
	}
	return append([]string(nil), manifest.files...)
}

// saveBundledStudyDocument preserves source-relative resources beside a Save As draft.
func saveBundledStudyDocument(document *draft.Document, path, sourceFolder, study string) error {
	cleanup, err := installBundledStudyResources(sourceFolder, filepath.Dir(path), study)
	if err != nil {
		return err
	}
	if err := document.Save(path); err != nil {
		cleanup()
		return err
	}
	return nil
}

type stagedStudyAsset struct {
	temporary string
	target    string
	data      []byte
}

// installBundledStudyResources copies missing exact-byte resources without replacing user files.
func installBundledStudyResources(sourceFolder, targetFolder, study string) (func(), error) {
	manifest, ok := studyManifest(study)
	if !ok {
		return nil, fmt.Errorf("unknown bundled study resource set %q", study)
	}
	info, err := os.Stat(targetFolder)
	if err != nil {
		return nil, fmt.Errorf("save folder: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("save folder is not a directory: %s", targetFolder)
	}
	assets := make([]stagedStudyAsset, 0, len(manifest.files))
	for _, name := range manifest.files {
		data, err := os.ReadFile(filepath.Join(sourceFolder, name))
		if err != nil {
			return nil, fmt.Errorf("read study resource %s: %w", name, err)
		}
		target := filepath.Join(targetFolder, name)
		current, err := os.Lstat(target)
		if err == nil {
			if !current.Mode().IsRegular() {
				return nil, fmt.Errorf("%w: %s is not a regular file", ErrStudyAssetConflict, target)
			}
			existing, readErr := os.ReadFile(target)
			if readErr != nil {
				return nil, readErr
			}
			if string(existing) != string(data) {
				return nil, fmt.Errorf("%w: %s", ErrStudyAssetConflict, target)
			}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		assets = append(assets, stagedStudyAsset{target: target, data: data})
	}
	cleanupStages := func() {
		for _, asset := range assets {
			if asset.temporary != "" {
				_ = os.Remove(asset.temporary)
			}
		}
	}
	for index := range assets {
		file, err := os.CreateTemp(targetFolder, ".atelier-study-resource-*")
		if err != nil {
			cleanupStages()
			return nil, err
		}
		assets[index].temporary = file.Name()
		if _, err := file.Write(assets[index].data); err != nil {
			file.Close()
			cleanupStages()
			return nil, err
		}
		if err := file.Sync(); err != nil {
			file.Close()
			cleanupStages()
			return nil, err
		}
		if err := file.Close(); err != nil {
			cleanupStages()
			return nil, err
		}
	}
	created := make([]string, 0, len(assets))
	rollback := func() {
		for _, path := range created {
			_ = os.Remove(path)
		}
	}
	for _, asset := range assets {
		if err := os.Link(asset.temporary, asset.target); err != nil {
			if errors.Is(err, fs.ErrExist) {
				existing, readErr := os.ReadFile(asset.target)
				if readErr == nil && string(existing) == string(asset.data) {
					continue
				}
				rollback()
				cleanupStages()
				return nil, fmt.Errorf("%w: %s", ErrStudyAssetConflict, asset.target)
			}
			rollback()
			cleanupStages()
			return nil, err
		}
		created = append(created, asset.target)
	}
	cleanupStages()
	return rollback, nil
}
