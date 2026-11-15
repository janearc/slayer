package budget

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

var (
	errAdmissionBusy    = errors.New("admission transaction busy")
	errAdmissionPending = errors.New("admission intent requires recovery")
	errAdmissionUnsafe  = errors.New("unsafe admission transaction entry")
)

type admissionBinding struct {
	boot   string
	policy hostPolicy
	limit  byteCount
}

type admissionTransaction struct {
	mutex     sync.Mutex
	directory *os.File
	lock      *os.File
	state     admissionRecord
	digest    [sha256.Size]byte
	observed  bool
	owner     reservationOwner
	ready     bool
	attempted bool
	publish   func(string) error
	closed    bool
}

// admissionDirectory opens path as a directory, one part at a time without
// following symlinks. path must be absolute and already clean. The directory
// must be owned by the effective user with mode 0700, or the result is
// errAdmissionUnsafe.
func admissionDirectory(path string) (*os.File, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errAdmissionUnsafe
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if part == "" {
			continue
		}
		next, err := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(fd)
		if err != nil {
			return nil, err
		}
		fd = next
	}
	file := os.NewFile(uintptr(fd), path)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		_ = file.Close()
		return nil, err
	}
	if stat.Uid != uint32(os.Geteuid()) || stat.Mode&07777 != 0700 {
		_ = file.Close()
		return nil, errAdmissionUnsafe
	}
	return file, nil
}

// admissionOpen opens name in directory with flags, without following a
// symlink or blocking. Only a regular file with mode 0600, owned by the
// effective user, with one link and no more than the record size limit is
// accepted; otherwise the file is closed and the result is
// errAdmissionUnsafe.
func admissionOpen(directory *os.File, name string, flags int) (*os.File, error) {
	fd, err := unix.Openat(int(directory.Fd()), name, flags|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		_ = file.Close()
		return nil, err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Mode&07777 != 0600 || stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 || stat.Size > admissionRecordLimit {
		_ = file.Close()
		return nil, errAdmissionUnsafe
	}
	return file, nil
}

// admissionRead returns the contents of the file name in directory, opened
// through admissionOpen. More than the record size limit gives
// errAdmissionRecord.
func admissionRead(directory *os.File, name string) ([]byte, error) {
	file, err := admissionOpen(directory, name, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, admissionRecordLimit+1))
	err = errors.Join(readErr, file.Close())
	if len(data) > admissionRecordLimit {
		return nil, errors.Join(errAdmissionRecord, err)
	}
	return data, err
}

// stableLock checks that the open lock file is still the file named lock in
// the directory: same device and inode, one link, a regular file, mode 0600,
// owned by the effective user, and empty. If not, it returns
// errAdmissionUnsafe.
func (tx *admissionTransaction) stableLock() error {
	var opened, named unix.Stat_t
	if err := unix.Fstat(int(tx.lock.Fd()), &opened); err != nil {
		return err
	}
	if err := unix.Fstatat(int(tx.directory.Fd()), "lock", &named, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if opened.Dev != named.Dev || opened.Ino != named.Ino || named.Nlink != 1 || named.Mode&unix.S_IFMT != unix.S_IFREG || named.Mode&07777 != 0600 || named.Uid != uint32(os.Geteuid()) || named.Size != 0 {
		return errAdmissionUnsafe
	}
	return nil
}

// beginAdmissionTransaction opens directory, takes an exclusive lock on its
// lock file without waiting (errAdmissionBusy if it is held), and reads
// state, which must match binding. If intent-prepared, state-next, retiring
// or intent exists, an earlier transaction did not finish and it returns
// errAdmissionPending. The caller must close the result. On error, all it
// opened is closed.
func beginAdmissionTransaction(ctx context.Context, directory string, binding admissionBinding) (_ *admissionTransaction, resultErr error) {
	if ctx == nil {
		return nil, errAdmissionUnsafe
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := admissionDirectory(directory)
	if err != nil {
		return nil, err
	}
	tx := &admissionTransaction{directory: dir}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, tx.close())
		}
	}()
	tx.lock, err = admissionOpen(dir, "lock", unix.O_RDWR)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(tx.lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, errors.Join(errAdmissionBusy, err)
		}
		return nil, err
	}
	if err := tx.stableLock(); err != nil {
		return nil, err
	}
	data, err := admissionRead(dir, "state")
	if err != nil {
		return nil, err
	}
	tx.state, err = decodeAdmissionRecord(data)
	if err != nil {
		return nil, err
	}
	if tx.state.boot != binding.boot || tx.state.policy != binding.policy || tx.state.ledger.limit != binding.limit {
		return nil, errAdmissionUnsafe
	}
	tx.digest = sha256.Sum256(data)
	for _, name := range []string{"intent-prepared", "state-next", "retiring"} {
		_, err := admissionRead(dir, name)
		if err == nil {
			return nil, errAdmissionPending
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, errors.Join(errAdmissionPending, err)
		}
	}
	pending, err := admissionRead(dir, "intent")
	if err == nil {
		intent, decodeErr := decodeAdmissionIntent(pending)
		if decodeErr != nil {
			return nil, errors.Join(errAdmissionPending, decodeErr)
		}
		if intent.boot != binding.boot || intent.predecessor != tx.digest && intent.stateDigest != tx.digest {
			return nil, errors.Join(errAdmissionPending, errAdmissionRecord)
		}
		return nil, errAdmissionPending
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return tx, nil
}

// observe names owner as the one running this transaction. owner must be
// listed in the state. It writes an intent in the observing phase and syncs
// it and the directory. It runs at most once, even if it fails, and the
// transaction must not be closed. After it succeeds, admit or release may
// run.
func (tx *admissionTransaction) observe(ctx context.Context, owner reservationOwner) error {
	if tx == nil {
		return errAdmissionUnsafe
	}
	tx.mutex.Lock()
	defer tx.mutex.Unlock()
	if ctx == nil || tx.closed || tx.observed {
		return errAdmissionUnsafe
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	found := false
	for _, existing := range tx.state.owners {
		if existing == owner {
			found = true
			break
		}
	}
	if !found {
		return errAdmissionUnsafe
	}
	if err := tx.stableLock(); err != nil {
		return err
	}
	data, err := encodeAdmissionIntent(admissionIntent{version: admissionRecordVersion, phase: intentObserving, boot: tx.state.boot, owner: owner, predecessor: tx.digest})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx.observed = true
	file, err := admissionOpen(tx.directory, "intent", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return err
	}
	count, writeErr := file.Write(data)
	if writeErr == nil && count != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	err = errors.Join(writeErr, file.Close())
	if err != nil {
		return err
	}
	if err := unix.Fsync(int(tx.directory.Fd())); err != nil {
		return err
	}
	if err := tx.stableLock(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx.owner = owner
	tx.ready = true
	return nil
}

// close releases the lock and closes the lock file and the directory. A nil
// transaction, or one already closed, gives errAdmissionUnsafe.
func (tx *admissionTransaction) close() error {
	if tx == nil {
		return errAdmissionUnsafe
	}
	tx.mutex.Lock()
	defer tx.mutex.Unlock()
	if tx.closed {
		return errAdmissionUnsafe
	}
	tx.closed = true
	var err error
	if tx.lock != nil {
		err = errors.Join(unix.Flock(int(tx.lock.Fd()), unix.LOCK_UN), tx.lock.Close())
	}
	if tx.directory != nil {
		err = errors.Join(err, tx.directory.Close())
	}
	return err
}
