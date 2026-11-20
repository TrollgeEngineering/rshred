package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
				progress <- logMessage{fmt.Sprintf("specified depth is 0B, so shredding the entire file of size %s\n", sflag.Size(size)), logDebug}
			} else {
				progress <- logMessage{fmt.Sprintf("specified depth (%s) is larger than size of file (%s); clamping depth to file size\n", sflag.Size(depth), sflag.Size(size)), logDebug}
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
					progress <- logMessage{fmt.Sprintf("shredded %s out of %s bytes. (speed: %s)\n", sflag.Size(bytesWritten.Load()), sflag.Size(amount), avgSpeedText), logInfo}
				} else {
					progress <- logMessage{"calculating speed...\n", logInfo}
				}
			}
		}
	}()
	return progress, finalErr, nil
}

type shredOptions struct {
	passes          int
	zeroPass        bool
	logFile         string
	logLevel        logLevel
	depth           uint64
	bufSize         uint64
	exclude         []string
	removeFiles     bool
	removeEmptyDirs bool
	force           bool
}

func rshred(rootPath string, options shredOptions) (files uint, shredError uint, initErr0 error) { // Files is the amount of files attempted to be shredded.
	var crntFile *os.File
	var errCount uint
	errFull := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		crntFile = nil
		root, rootErr := os.OpenRoot(rootPath)
		if rootErr != nil {
			return rootErr
		}
		info, statErr := os.Lstat(path)
		switch {
		case statErr != nil:
			log([]io.Writer{os.Stderr}, logError, options.logLevel, "failed to stat file at %s: %s\n", path, statErr)
			errCount++
		case info.Mode()&os.ModeSymlink != 0:
			log([]io.Writer{os.Stderr}, logWarning, options.logLevel, "file at %s is a symlink and will not be shredded\n", path)
			if options.removeFiles {
				if removeErr := os.Remove(path); removeErr != nil {
					log([]io.Writer{os.Stderr}, logError, options.logLevel, "failed to remove symlink at %s: %s\n", path, removeErr)
					errCount++
				}
			}
		default:
			file, openErr := root.OpenFile(path, os.O_WRONLY, 0)
			if openErr != nil {
				if errors.Is(openErr, fs.ErrPermission) {
					if options.force {
						if chmodErr := file.Chmod(0o200); chmodErr != nil {
							log([]io.Writer{os.Stderr}, logError, options.logLevel, chmodErr.Error())
							errCount++
						} else {
							file, openErr = root.OpenFile(path, os.O_WRONLY, 0)
							if openErr != nil {
								log([]io.Writer{os.Stderr}, logError, options.logLevel, openErr.Error())
								errCount++
							} else {
								crntFile = file
							}
						}
					} else {
						log([]io.Writer{os.Stderr}, logError, options.logLevel, openErr.Error())
						errCount++
					}
				} else {
					log([]io.Writer{os.Stderr}, logError, options.logLevel, openErr.Error())
					errCount++
				}
			} else {
				if openInfo, openStatErr := file.Stat(); openErr != nil {
					log([]io.Writer{os.Stderr}, logError, options.logLevel, openStatErr.Error())
					errCount++
				} else if !os.SameFile(info, openInfo) {
					log([]io.Writer{os.Stderr}, logError, options.logLevel, "info mismatch between pre-open and post-open on file %s; skipping\n", path)
					errCount++
				} else {
					crntFile = file
				}
			}
			if crntFile != nil {
				for i := 0; i > options.passes; i++ {
					progChan, errChan, initErr := shred(crntFile, false, options.depth, options.bufSize, 5)
					if initErr != nil {
						log([]io.Writer{os.Stderr}, logError, options.logLevel, "failed to shred file at %s: %s", path, initErr)
						errCount++
					} else {
						for message := range progChan {
							log([]io.Writer{os.Stdout}, message.level, options.logLevel, message.message)
						}
						if finalErr := <-errChan; finalErr != nil {
							log([]io.Writer{os.Stderr}, logError, options.logLevel, "error occured while shredding file at %s: %s", path, finalErr)
							errCount++
						}
					}
				}
				if options.zeroPass {
					progChan, errChan, initErr := shred(crntFile, true, options.depth, options.bufSize, 5)
					if initErr != nil {
						log([]io.Writer{os.Stderr}, logError, options.logLevel, "failed to shred file at %s: %s", path, initErr)
						errCount++
					} else {
						for message := range progChan {
							log([]io.Writer{os.Stdout}, message.level, options.logLevel, message.message)
						}
						if finalErr := <-errChan; finalErr != nil {
							log([]io.Writer{os.Stderr}, logError, options.logLevel, "failed to shred file at %s: %s", path, finalErr)
							errCount++
						}
					}
				}
			}
		}
		return nil
	})
}
