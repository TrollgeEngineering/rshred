//go:build windows

package main

import (
	"crypto/rand"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func PermQuery(path string, perm int) bool {
	switch perm {
	case 0:
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				_, inErr := os.ReadDir(path)
				return inErr == nil
			} else {
				f, inErr := os.OpenFile(path, os.O_RDONLY, 0)
				defer f.Close()
				return inErr == nil
			}
		} else {
			return false
		}
	case 1:
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				var testWrite *os.File
				var writeErr error
				var randName string
				for {
					randName = rand.Text()
					testWrite, writeErr = os.OpenFile(filepath.Join(path, randName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o200)
					if !errors.Is(writeErr, fs.ErrExist) {
						break
					}
				}
				defer func() {
					testWrite.Close()
					os.Remove(filepath.Join(path, randName))
				}()
				if writeErr != nil {
					return false
				} else {
					testWrite.Close()
					err := os.Remove(filepath.Join(path, randName))
					return err == nil
				}
			} else {
				f, err := os.OpenFile(path, os.O_WRONLY, 0)
				defer f.Close()
				return err == nil
			}
		} else {
			return false
		}
	case 2:
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				_, inErr := os.ReadDir(path)
				return inErr == nil
			} else {
				return true /* There is no reason to check a file for execute permissions for shredding,
				and I don't know how to check on Windows. */
			}
		} else {
			return false
		}
	default:
		panic("Invalid permission query!")
	}
}

func QuerySticky(path string) bool {
	return false
}
