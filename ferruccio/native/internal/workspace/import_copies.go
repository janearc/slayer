package workspace

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/janearc/slayer/oakland"
)

type importCopies struct {
	root    *os.Root
	created []importCopy
}

type importCopy struct {
	name string
	info fs.FileInfo
}

func (copies *importCopies) rollback() {
	for _, entry := range slices.Backward(copies.created) {
		info, err := copies.root.Lstat(entry.name)
		if err == nil && os.SameFile(info, entry.info) {
			_ = copies.root.Remove(entry.name)
		}
	}
}

func (copies *importCopies) directory(name string) error {
	if name == "." {
		return nil
	}
	info, err := copies.root.Lstat(name)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("import folder %q is not a directory", name)
		}
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := copies.directory(filepath.Dir(name)); err != nil {
		return err
	}
	if err := copies.root.Mkdir(name, 0700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return copies.directory(name)
		}
		return err
	}
	info, err = copies.root.Lstat(name)
	if err != nil {
		return err
	}
	copies.created = append(copies.created, importCopy{name, info})
	return nil
}

func (copies *importCopies) install(source *os.Root, name string, checked oakland.Source) error {
	if err := copies.directory(filepath.Dir(name)); err != nil {
		return err
	}
	if _, err := copies.root.Lstat(name); !errors.Is(err, fs.ErrNotExist) {
		if err != nil {
			return err
		}
		return readCheckedImport(copies.root, name, checked, io.Discard)
	}
	temporary := ".ferruccio-import-" + rand.Text()
	info, err := copies.writeTemporary(temporary, source, name, checked)
	if err != nil {
		return err
	}
	defer copies.root.Remove(temporary)

	if err := copies.root.Link(temporary, name); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return readCheckedImport(copies.root, name, checked, io.Discard)
		}
		return err
	}
	copies.created = append(copies.created, importCopy{name, info})
	return nil
}

func (copies *importCopies) writeTemporary(temporary string, source *os.Root, name string, checked oakland.Source) (fs.FileInfo, error) {
	file, err := copies.root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	writeErr := readCheckedImport(source, name, checked, file)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	info, statErr := file.Stat()
	if err := errors.Join(writeErr, statErr, file.Close()); err != nil {
		_ = copies.root.Remove(temporary)
		return nil, err
	}
	return info, nil
}

func readCheckedImport(root *os.Root, name string, checked oakland.Source, output io.Writer) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != checked.Size() {
		return fmt.Errorf("the resource differs from the verified import")
	}
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	digest := sha256.New()
	count, err := io.Copy(io.MultiWriter(output, digest), io.LimitReader(file, checked.Size()+1))
	if err != nil {
		return err
	}
	expected := checked.SHA256()
	if count != checked.Size() || !bytes.Equal(digest.Sum(nil), expected[:]) {
		return fmt.Errorf("the resource differs from the verified import")
	}
	return nil
}
