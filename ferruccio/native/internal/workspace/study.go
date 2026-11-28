package workspace

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/ferruccio/internal/preview"
)

var ErrStudyAssetConflict = errors.New("bundled study asset conflicts with an existing destination file")

type bundledStudyManifest struct {
	id, folder, entry, label string
	files                    []string
}

var oaklandLarsonStudy = bundledStudyManifest{
	id: "oakland-larson", folder: "oakland-larson", entry: "set.yaml",
	label: "Oakland Larson study",
	files: []string{"two-metre-human.gltf", "two-metre-human.bin", "LICENSE.txt"},
}

var larsonCampStudy = bundledStudyManifest{
	id: "larson-camp", folder: ".", entry: campExample, label: "Larson camp",
	files: []string{"larson-camp-visitor.glb", "larson-camp-visitor-LICENSE.txt"},
}

var larsonPlaygroundStudy = bundledStudyManifest{
	id: "larson-playground", folder: ".", entry: playgroundExample,
	label: "Larson playground", files: larsonCampStudy.files,
}

var bundledStudies = []bundledStudyManifest{oaklandLarsonStudy, larsonCampStudy, larsonPlaygroundStudy}

const posedVisitorCredit = "Meshcapade and Max Planck Institute for Intelligent Systems SMPL-X sample (Pavlakos et al. 2019). Slayer triangulated, posed, and exported the sample, and normalized it to two metres. Attribution and licence: https://smpl.is.tue.mpg.de/bodylicense.html"

func attributedVisitor(document *draft.Document) bool {
	model := larsonPlaygroundStudy.files[0]
	if document.Kind() != "set" || !strings.Contains(document.Text(), model) {
		return false
	}
	file, err := document.Value("scene", "world", "imports", "visitor", "file")
	if err != nil || file != model {
		return false
	}
	license, err := document.Value("scene", "world", "imports", "visitor", "license")
	if err != nil || license != "CC-BY-4.0" {
		return false
	}
	credit, err := document.Value("scene", "world", "imports", "visitor", "source")
	return err == nil && credit == posedVisitorCredit
}

func exactVisitorResources(folder string) map[string][]byte {
	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil
	}
	defer root.Close()
	verified := make(map[string][]byte, 2)
	for name, expected := range map[string]string{
		"larson-camp-visitor.glb":         "c73ccce5fb96ad88fe925f3bd411499d0a7c73aeafd6c84bb4c7bd73f2ea6663",
		"larson-camp-visitor-LICENSE.txt": "6c20b7860436398e9ebf3c1f10b2c8dc712ef703313e2ba3eb4fe4fe11214517",
	} {
		info, err := root.Lstat(name)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			return nil
		}
		data, err := root.ReadFile(name)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != expected {
			return nil
		}
		verified[name] = data
	}
	return verified
}

// studyForExample identifies a bundled study by its relative entry path.
func studyForExample(name string) (bundledStudyManifest, bool) {
	clean := filepath.ToSlash(filepath.Clean(name))
	for _, study := range bundledStudies {
		if clean == filepath.ToSlash(filepath.Join(study.folder, study.entry)) {
			return study, true
		}
	}
	return bundledStudyManifest{}, false
}

// studyManifest resolves a study identity retained by an editable draft.
func studyManifest(id string) (bundledStudyManifest, bool) {
	for _, study := range bundledStudies {
		if id == study.id {
			return study, true
		}
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
	if err := preview.CopySourceManifest(filepath.Join(sourceFolder, manifest.entry), targetFolder); err != nil {
		os.RemoveAll(targetFolder)
		return "", "", err
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
	cleanup, err := installBundledStudyResources(sourceFolder, filepath.Dir(path), study, nil)
	if err != nil {
		return err
	}
	if err := document.Save(path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// savePlayground keeps the visitor's notice and uses the ordinary draft saver
// for any further resources added while editing the starter.
func (work *Workspace) savePlayground(path string) (string, error) {
	for _, resource := range larsonPlaygroundStudy.files {
		if strings.EqualFold(filepath.Base(path), resource) {
			return "", fmt.Errorf("%w: %s is reserved for a playground resource", ErrStudyAssetConflict, resource)
		}
	}
	source := filepath.Dir(work.state.path)
	current := work.state.document
	if !attributedVisitor(current) && current.Check() == nil {
		return work.saveDraft(path)
	}
	verified := exactVisitorResources(source)
	if verified == nil {
		return work.saveDraft(path)
	}
	cleanup, err := installBundledStudyResources(source, filepath.Dir(path), larsonPlaygroundStudy.id, verified)
	if err != nil {
		return "", err
	}
	notice, err := work.saveDraft(path)
	if err != nil {
		cleanup()
	}
	return notice, err
}

type stagedStudyAsset struct {
	temporary string
	target    string
	data      []byte
}

// installBundledStudyResources copies missing exact-byte resources without replacing user files.
func installBundledStudyResources(sourceFolder, targetFolder, study string, verified map[string][]byte) (func(), error) {
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
	var source *os.Root
	if verified == nil {
		source, err = os.OpenRoot(sourceFolder)
		if err != nil {
			return nil, err
		}
		defer source.Close()
	}
	for _, name := range manifest.files {
		data, ok := verified[name]
		if verified == nil {
			data, err = readStudyResource(source, name)
			if err != nil {
				return nil, err
			}
		} else if !ok {
			return nil, fmt.Errorf("study resource %s was not verified", name)
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

func readStudyResource(source *os.Root, name string) ([]byte, error) {
	info, err := source.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("check study resource %s: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("study resource %s is not a regular file", name)
	}
	data, err := source.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read study resource %s: %w", name, err)
	}
	return data, nil
}
