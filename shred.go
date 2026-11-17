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

func shred(file *os.File, zero bool, depth uint64, bufSize uint64, reportEvery uint) (chan logMessage, chan error, error) {
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
	progTick := time.NewTicker(time.Duration(reportEvery * uint(time.Second)))
	go func() {
		size := uint64(info.Size())
		var dirty uint64
		if (size > depth) && depth != 0 {
			amount = depth
		} else {
			if depth == 0 {
				progress <- logMessage{fmt.Sprintf("specified depth is 0B, so shredding the entire file of size %s", sflag.Size(size)), logDebug}
			} else {
				progress <- logMessage{fmt.Sprintf("specified depth (%s) is larger than size of file (%s); clamping depth to file size", sflag.Size(depth), sflag.Size(size)), logDebug}
			}
			amount = size
		}
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
				if remaining := amount - (written.Load() + dirty); remaining < bufSize {
					n = buf[:remaining]
				} else {
					n = buf
				}
				if wrote, err := file.Write(n); err != nil {
					finalErr <- fmt.Errorf("failed to write to file")
					return
				} else {
					dirty += uint64(wrote)
					if time.Since(lastSync) > 2*time.Second {
						if syncErr := file.Sync(); syncErr != nil {
							finalErr <- fmt.Errorf("failed to sync file to disk")
							return
						}
						lastSync = time.Now()
						written.Add(dirty)
						dirty = 0
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
		var bytesWritten atomic.Uint64
		var avgSpeed float64
		var avgSpeedText string
		type sample struct {
			time         time.Time
			bytesWritten uint64
		}
		samples := []sample{}
		for {
			select {
			case <-done:
				return
			case <-progTick.C:
				bytesWritten.Store(written.Load())
				samples = append(samples, sample{time.Now(), bytesWritten.Load()})
				if len(samples) > 10 {
					samples = samples[1:]
				}
				if time.Since(samples[0].time) >= 25*time.Second {
					avgSpeed = float64(samples[len(samples)-1].bytesWritten-samples[0].bytesWritten) / samples[len(samples)-1].time.Sub(samples[0].time).Seconds()
					switch {
					case avgSpeed < 1024.0:
						avgSpeedText = fmt.Sprintf("%.0fB/s", avgSpeed)
					case avgSpeed < 1024.0*1024.0:
						avgSpeedText = fmt.Sprintf("%.2fKiB/s", avgSpeed/float64(1<<10))
					default:
						avgSpeedText = fmt.Sprintf("%.2fMiB/s", avgSpeed/float64(1<<20))
					}
					progress <- logMessage{fmt.Sprintf("shredded %s out of %s bytes. (speed: %s)", sflag.Size(bytesWritten.Load()), sflag.Size(amount), avgSpeedText), logInfo}
				} else {
					progress <- logMessage{"calculating speed...", logInfo}
				}
			}
		}
	}()
	return progress, finalErr, nil
}
