//go:build unix

// Package permchk checks permissions before shredding
package permchk

import (
	"os"

	"golang.org/x/sys/unix"
)

func PermQuery(path string, perm int) bool {
	switch perm {
	case 0:
		return unix.Access(path, unix.R_OK) == nil
	case 1:
		return unix.Access(path, unix.W_OK) == nil
	case 2:
		return unix.Access(path, unix.X_OK) == nil
	default:
		panic("Invalid permission query!")
	}
}

func QuerySticky(path string) bool {
	if info, err := os.Stat(path); err != nil {
		return false
	} else {
		return info.Mode()&os.ModeSticky != 0
	}
}
