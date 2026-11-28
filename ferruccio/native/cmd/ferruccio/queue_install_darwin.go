package main

import "golang.org/x/sys/unix"

// installPrivateQueue leaves even an empty existing destination intact.
func installPrivateQueue(source, destination string) error {
	return unix.RenamexNp(source, destination, unix.RENAME_EXCL)
}
