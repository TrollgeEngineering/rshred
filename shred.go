package main

import (
	"crypto/rand"
	"fmt"
	"os"
)

func Shred(file *os.File, zero bool, depth int, bufSize int) error {
	var written int64 = 0
	buf := make([]byte, bufSize)
	var amount int64
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info")
	} else if info.IsDir() {
		return fmt.Errorf("error: target is a directory")
	}
	if (info.Size() > int64(depth)) && depth != -1 {
		amount = int64(depth)
	} else {
		amount = info.Size()
	}
	for written < amount {
		var n []byte
		if !zero {
			rand.Read(buf)
		} else {
			for i := range buf {
				buf[i] = 0x00
			}
		}
		if remaining := amount - written; remaining < int64(bufSize) {
			n = buf[:remaining]
		} else {
			n = buf
		}
		if wrote, err := file.Write(n); err != nil {
			return fmt.Errorf("error occured writing to file")
		} else {
			written = written + int64(wrote)
		}
		if err := file.Sync(); err != nil {
			return fmt.Errorf("error occured syncing file")
		}
	}
	return nil
}
