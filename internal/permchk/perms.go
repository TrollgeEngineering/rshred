package permchk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type WarnPermLists struct {
	DnRX   []string
	DnW    []string
	FnW    []string
	Sticky []string
}

type UnixPerms struct {
	Dir, R, W, X, Sticky bool
}

func CheckPerm(path string) (UnixPerms, error) {
	perms := UnixPerms{}
	info, err := os.Stat(path)
	if err != nil {
		return UnixPerms{}, err
	}
	if info.IsDir() {
		perms.Dir = true
	}
	if PermQuery(path, 0) {
		perms.R = true
	}
	if PermQuery(path, 1) {
		perms.W = true
	}
	if PermQuery(path, 2) {
		perms.X = true
	}
	if QuerySticky(path) {
		perms.Sticky = true
	}
	return perms, nil
}

func WalkPerms(walkingPath string) (WarnPermLists, error) {
	thePerms := WarnPermLists{
		DnRX:   []string{},
		DnW:    []string{},
		FnW:    []string{},
		Sticky: []string{},
	}
	errFull := filepath.WalkDir(walkingPath, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			if path != walkingPath {
				return fs.SkipDir
			} else {
				return err
			}
		}
		perms, err2 := CheckPerm(path)
		if err2 != nil {
			return nil
		}
		if perms.Dir {
			if !perms.R || !perms.X {
				thePerms.DnRX = append(thePerms.DnRX, path)
				if path != walkingPath {
					return fs.SkipDir
				} else {
					return fmt.Errorf("error: %q does not have read/exec permissions", walkingPath)
				}
			}
			if !perms.W {
				thePerms.DnW = append(thePerms.DnW, path)
			}
			if perms.Sticky {
				thePerms.Sticky = append(thePerms.Sticky, path)
			}
			return nil
		} else {
			if !perms.W {
				thePerms.FnW = append(thePerms.FnW, path)
			}
			return nil
		}
	})
	if errFull != nil {
		return WarnPermLists{}, errFull
	} else {
		return thePerms, nil
	}
}
