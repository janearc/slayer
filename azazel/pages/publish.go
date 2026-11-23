package pages

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
)

// Publish publishes one render folder once, as at a job's end: its last
// partial film, its page, the gallery and, given its pictures, barkeep's
// line. It is the operator's, for a render begun before azazel kept its
// pages, and returns what the operator should be told.
func Publish(ctx context.Context, settings Settings, folder Folder,
	pictures int64) ([]string, error) {
	made := publisherOf(settings)
	dir := folder.path(made.root)
	found, err := scanFolder(dir)
	if err != nil {
		return nil, err
	}
	look, err := made.view(folder, nil)
	failures := []error{err}
	if look.due(true) {
		failures = append(failures, made.encode(ctx, dir, look))
		look, err = made.view(folder, nil)
		failures = append(failures, err)
	}
	notice, err := made.publishFolder(folder, found, look)
	failures = append(failures, err, made.publishGallery())
	told := []string{notice}
	if pictures > 0 {
		pictureCount := int64(len(found.pictures))
		said, err := made.publishLine(pictureCount, pictures)
		told, failures = append(told, said), append(failures, err)
	}
	silent := func(word string) bool { return word == "" }
	return slices.DeleteFunc(told, silent), errors.Join(failures...)
}

// publishLine writes barkeep's line once, unless someone else's live
// render holds it. With no pace to take, it says nought a minute.
func (p publisher) publishLine(done, total int64) (string, error) {
	keeper := barkeep{path: filepath.Join(p.root, progressName)}
	now := p.now().Unix()
	if !keeper.free(now) {
		return heldLine, nil
	}
	left := int64(unknownLeft)
	if done >= total {
		left = 0
	}
	return "", keeper.write(done, total, 0, left, now)
}
