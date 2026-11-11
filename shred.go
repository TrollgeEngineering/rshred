package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"sync/atomic"
)

func Shred(file *os.File, zero bool, depth uint64, bufSize uint64) error {
	var written atomic.Uint64
	buf := make([]byte, bufSize)
	var amount uint64
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info")
	} else if info.IsDir() {
		return fmt.Errorf("target is a directory")
	}
	if depth&(1<<63) != 0 {
		return fmt.Errorf("size too big")
	}
	size := uint64(info.Size())
	if (size > depth) && depth != 0 {
		amount = depth
	} else {
		amount = size
	}
	for written.Load() < amount {
		var n []byte
		if !zero {
			rand.Read(buf)
		} else {
			for i := range buf {
				buf[i] = 0x00
			}
		}
		if remaining := amount - written.Load(); remaining < bufSize {
			n = buf[:remaining]
		} else {
			n = buf
		}
		if wrote, err := file.Write(n); err != nil {
			return fmt.Errorf("failed to write to file")
		} else {
			written.Add(uint64(wrote))
		}
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file writes")
	}
	return nil
}
