package main

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
	sticky []string
}

type UnixPerms struct {
	dir, r, w, x, sticky bool
}

func CheckPerm(path string) (UnixPerms, error) {
	perms := UnixPerms{}
	info, err := os.Stat(path)
	if err != nil {
		return UnixPerms{}, err
	}
	if info.IsDir() {
		perms.dir = true
	}
	if PermQuery(path, 0) {
		perms.r = true
	}
	if PermQuery(path, 1) {
		perms.w = true
	}
	if PermQuery(path, 2) {
		perms.x = true
	}
	if QuerySticky(path) {
		perms.sticky = true
	}
	return perms, nil
}

func WalkPerms(walkingPath string) (WarnPermLists, error) {
	thePerms := WarnPermLists{
		DnRX:   []string{},
		DnW:    []string{},
		FnW:    []string{},
		sticky: []string{},
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
		if perms.dir {
			if !perms.r || !perms.x {
				thePerms.DnRX = append(thePerms.DnRX, path)
				if path != walkingPath {
					return fs.SkipDir
				} else {
					return fmt.Errorf("error: %q does not have read/exec permissions", walkingPath)
				}
			}
			if !perms.w {
				thePerms.DnW = append(thePerms.DnW, path)
			}
			if perms.sticky {
				thePerms.sticky = append(thePerms.sticky, path)
			}
			return nil
		} else {
			if !perms.w {
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
