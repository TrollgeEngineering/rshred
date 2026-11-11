package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"codeberg.org/TrollgeEngineering/sflag"
)

type logMessage struct {
	message string
	level   logLevel
}

var done = make(chan struct{})

func Shred(file *os.File, zero bool, depth uint64, bufSize uint64, reportEvery uint) (chan logMessage, chan error, error) {
	if bufSize > 50*(1<<20) {
		return nil, nil, fmt.Errorf("buffer size of %s is too big (maximum is 50 MiB)", sflag.Size(bufSize>>20))
	}
	var written atomic.Uint64
	buf := make([]byte, bufSize)
	var amount uint64
	info, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file info")
	} else if info.IsDir() {
		return nil, nil, fmt.Errorf("target is a directory")
	}
	if depth&(1<<63) != 0 {
		return nil, nil, fmt.Errorf("specified depth of %s is too large", sflag.Size(depth))
	}
	progress, finalErr := make(chan logMessage), make(chan error, 1)
	size := uint64(info.Size())
	if (size > depth) && depth != 0 {
		amount = depth
	} else {
		progress <- logMessage{fmt.Sprintf("specified depth (%s) is larger than size of file (%s); clamping depth to file size", sflag.Size(depth), sflag.Size(size)), logDebug}
		amount = size
	}
	progTick := time.NewTicker(time.Duration(reportEvery * uint(time.Second)))
	start := time.Now()
	go func() {
		defer func() {
			select {
			case done <- struct{}{}:
			default:
			}
		}()
		lastSync := time.Now()
		for written.Load() < amount {
			select {
			case <-done:
				return
			default:
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
					finalErr <- fmt.Errorf("failed to write to file")
					return
				} else {
					if time.Since(lastSync) > 2*time.Second {
						if syncErr := file.Sync(); syncErr != nil {
							finalErr <- fmt.Errorf("failed to sync file to disk")
							return
						}
						lastSync = time.Now()
						written.Add(uint64(wrote))
					}
				}
			}
		}
		if syncErr := file.Sync(); syncErr != nil {
			finalErr <- fmt.Errorf("failed to sync file to disk")
		} else {
			finalErr <- nil
		}
	}()
	go func() {
		defer func() {
			close(progress)
		}()
		var old atomic.Uint64
		var delta atomic.Uint64
		var bytesWritten atomic.Uint64
		var speed float64
		var avgSpeed float64
		var speedText string
		var avgSpeedText string
		for {
			select {
			case <-done:
				return
			case <-progTick.C:
				bytesWritten.Store(written.Load())
				delta.Store(bytesWritten.Load() - old.Load())
				speed = float64(delta.Load()) / float64(reportEvery)
				avgSpeed = float64(bytesWritten.Load()) / float64(time.Since(start).Seconds())
				switch {
				case speed < 1024.0:
					speedText = fmt.Sprintf("%.0fB/s", speed)
				case speed < 1024.0*1024.0:
					speedText = fmt.Sprintf("%.2fKiB/s", speed/1024.0)
				default:
					speedText = fmt.Sprintf("%.2fMiB/s", speed/(1024.0*1024.0))
				}
				switch {
				case avgSpeed < 1024.0:
					avgSpeedText = fmt.Sprintf("%.0fB/s", avgSpeed)
				case avgSpeed < 1024.0*1024.0:
					avgSpeedText = fmt.Sprintf("%.2fKiB/s", avgSpeed/1024.0)
				default:
					avgSpeedText = fmt.Sprintf("%.2fMiB/s", avgSpeed/(1024.0*1024.0))
				}
				select {
				case progress <- logMessage{fmt.Sprintf("shredded %d out of %d bytes. (current speed: %s) (average speed: %s)", bytesWritten.Load(), amount, speedText, avgSpeedText), logInfo}:
				default:
				}
				old.Store(bytesWritten.Load())
			}
		}
	}()
	return progress, finalErr, nil
}
