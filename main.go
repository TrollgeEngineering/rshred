package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"codeberg.org/TrollgeEngineering/rshred/internal/permchk"
	"codeberg.org/TrollgeEngineering/sflag"
)

func registerCliFlags() {
	sflag.RegisterWithShort("passes", 'n', sflag.Options{ValueRequired: true, ValueType: sflag.ValueInt, Default: 3})
	sflag.RegisterWithShort("verbose", 'v', sflag.Options{ValueType: sflag.ValueBoolCount, ConflictsWith: []string{"quiet"}})
	sflag.RegisterWithShort("quiet", 'q', sflag.Options{ValueType: sflag.ValueBoolCount, ConflictsWith: []string{"verbose"}})
	sflag.RegisterWithShort("logs", 'l', sflag.Options{Default: false})
	sflag.RegisterWithShort("size", 's', sflag.Options{ValueType: sflag.ValueSize, ValueRequired: true, Default: 0})
	sflag.RegisterWithShort("zero", 'z', sflag.Options{Default: true})
	sflag.RegisterWithShort("interactive", 'i', sflag.Options{})
	sflag.Register("log-level", sflag.Options{ValueRequired: true, ValueType: sflag.ValueString, Default: true, ConflictsWith: []string{"quiet", "verbose"}})
	sflag.Register("exclude", sflag.Options{ValueType: sflag.ValueString, ValueRequired: true})
	sflag.Register("log-path", sflag.Options{ValueType: sflag.ValueString, ValueRequired: true})
	sflag.Register("dry-run", sflag.Options{})
	sflag.Register("help", sflag.Options{})
	sflag.Register("version", sflag.Options{})
	sflag.Register("buffer", sflag.Options{ValueType: sflag.ValueInt, ValueRequired: true, Default: sflag.Size(4096)})
	sflag.Register("gen-config", sflag.Options{})
}

var (
	fileNoWrite      []string
	dirNoReadAndExec []string
	dirNoWrite       []string
	dirSticky        []string
)

const rVersion = "v4.0.0"

var shredVictims = map[string]int{}

var rshredConf = ConfigSpec{
	&Config{Key: "CheckForUpdates", Default: "OFF", Comment: []string{"Checks for updates on boot. (ON/OFF) Default:"}},
	&Config{Key: "LogPath", ValType: confString, Default: "~/.rshred/logs", Comment: []string{"Full path for the logfiles. Default: ~/.rshred/logs"}},
	&Config{Key: "ExcludeDirsPrompt", Default: "OFF", Comment: []string{"Displays a prompt after flag selection for directories to exclude from shredding on a per-run basis."}},
	&Config{Key: "ExcludeConfigDirs", Default: "ON", Comment: []string{"Enables the exclusion of the directories set below. (ON/OFF) Default: ON"}},
	&Config{
		Key: "ExcludeDirectories", ValType: confList, Comment: []string{
			"Full paths of directories you always want to exlude from shredding.",
			"Put the FULL, ABSOLUTE PATHS of the directories you want between the brackets (\"{\" and \"}\")",
			"Put one directory path per line.",
			"Do not quote or escape any paths, even if they contain spaces or special characters.",
		},
	},
	&Config{Key: "RemoveDirectories", Default: "OFF", Comment: []string{"Attempts to remove all empty directories if the -u flag is used."}},
	&Config{
		Key: "LogLevel", ValType: confString, Default: "info", Comment: []string{
			"The threshold for which messages will be logged to the screen or file.",
			"Setting this option to one of the below levels will cause messages of that level and above to be logged.",
			"Fatal - Faliures that cause rshred to crash.",
			"Error - Faliure of a specific operation that is not fatal to the program.",
			"Warning - Not a faliure, but important information the user should know.",
			"Info - Regular information about how rshred is running, such as shred progress.",
			"Debug - Fine-grained detail of everything rshred is doing.",
		},
	},
}

func decideInteract(args []string) int {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGHUP)
	go func() {
		sig := <-sigChan
		select {
		case done <- struct{}{}:
		default:
		}
		fmt.Fprintf(os.Stderr, "\nInterrupt recieved!\n\n")
		fmt.Fprintln(os.Stderr, "Quitting...")
		if sysSig, ok := sig.(syscall.Signal); ok {
			exitCode := 128 + int(sysSig)
			os.Exit(exitCode)
		} else {
			os.Exit(2)
		}
	}()
	registerCliFlags()
	results, leftovers, err := sflag.Parse(args)
	if err != nil {
		log([]io.Writer{os.Stderr}, logFatal, logInfo, err.Error())
		return 1
	}
	if results["interactive"].Seen {
		return interact()
	}
	return cli(results, leftovers)
}

func cli(results sflag.Results, leftovers []string) int {
	var crntLogLevel logLevel
	if results["log-level"].ValueSeen {
		switch results["log-level"].Value.(string) {
		case "fatal":
			crntLogLevel = logFatal
		case "error":
			crntLogLevel = logError
		case "warning":
			crntLogLevel = logWarning
		case "info":
			crntLogLevel = logInfo
		case "debug":
			crntLogLevel = logDebug
		default:
			log([]io.Writer{os.Stderr}, logFatal, logInfo, "invalid logging level %s", results["log-level"].Value.(string))
			return 1
		}
	} else {
		crntLogLevel = logInfo
	}
	confDir, err := os.UserConfigDir()
	if err != nil {
		log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "error getting config path")
		return 1
	}
	rshredConfDir := filepath.Join(confDir, "rshred")
	rshredConfFile := filepath.Join(rshredConfDir, "rshred.conf")
	openConfFile, err := os.OpenFile(rshredConfDir, os.O_RDONLY, 0)
	switch {
	case err == nil:
		scanner := bufio.NewScanner(openConfFile)
		confFile := []string{}
		for scanner.Scan() {
			confFile = append(confFile, scanner.Text())
		}
		if scanner.Err() != nil {
			log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "error reading config file at %s", rshredConfFile)
			return 1
		}
		if len(confFile) < 3 {
			log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "log file at %s is less than two lines", rshredConfFile)
			return 1
		}
		err := ValidateConfig(confFile[0])
		switch {
		case err == nil:
			if errs := ParseConfig(confFile[2:], rshredConf); errs != nil {
				log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "the config file at %s has the below errors:\n", rshredConfFile)
				for _, confErr := range errs {
					fmt.Println(confErr)
				}
				return 1
			}
		case errors.Is(err, ErrConfVersionTooLow):
			newConf, transferErr := TransferConf(rshredConf, confFile)
			if transferErr != nil {
				log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "failed to parse config file at %s", rshredConfFile)
				return 1
			}
			if err := ParseConfig(newConf, rshredConf); err != nil {
				panic("an invalid config was generated")
			}
			log([]io.Writer{os.Stderr}, logWarning, crntLogLevel, err.Error())
		case errors.Is(err, ErrCorruptHeader):
			log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "config file at %s has a corrupt header", rshredConfFile)
			return 1
		default:
			panic(fmt.Sprintf("got unrecognized error %q", err))
		}
	case errors.Is(err, fs.ErrNotExist):
		log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "the config file at %s does not exist. pass --gen-config or -i to generate it. if you think this is a mistake, please check the path or pass a config file with --config-file", rshredConfFile)
		return 1
	case errors.Is(err, fs.ErrPermission):
		log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "permission denied while attempting to open config file at %s", rshredConfFile)
	default:
		log([]io.Writer{os.Stderr}, logFatal, crntLogLevel, "unknown error while attempting to open config file at %s", rshredConfFile)
	}
	switch rshredConf.QueryConf("LogLevel").Value.(string) {
	}
	if results["help"].Seen {
		fmt.Println("help")
		return 0
	}
	if results["version"].Seen {
		fmt.Println(rVersion)
		return 0
	}
	for _, leftover := range leftovers {
		fmt.Println(leftover)
	}
	return 0
}

func interact() int {
	confDir, err := os.UserConfigDir()
	if err != nil {
		PrintExtraNewline("error getting config directory. loading default settings.")
		LoadDefault(rshredConf)
	} else {
		rshredConfDir := filepath.Join(confDir, "rshred")
		confFilePath := filepath.Join(rshredConfDir, "rshred.conf")
		openConfFile, err := os.OpenFile(confFilePath, os.O_RDONLY, 0)
		if err != nil {
			switch {
			case errors.Is(err, fs.ErrNotExist):
				if YesOrNo(true, "The config file at %v does not exist. Would you like to create it?", confFilePath) {
					err := os.MkdirAll(rshredConfDir, 0o700)
					if err != nil {
						if errors.Is(err, fs.ErrPermission) {
							PrintExtraNewline("permission denied while attempting to create config directories. loading default settings.")
						} else {
							PrintExtraNewline("unknown error while attempting to create config directories. loading default settings.")
						}
					} else {
						createdConfFile, err := os.OpenFile(confFilePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o700)
						if err != nil {
							switch {
							case errors.Is(err, fs.ErrExist):
								PrintExtraNewline("error creating config file: file exists. loading default config...")
							case errors.Is(err, fs.ErrPermission):
								PrintExtraNewline("error creating config file: permission denied. loading default config...")
							default:
								PrintExtraNewline("unknown error creating config file. loading default config...")
							}
						} else {
							if _, err := fmt.Fprintf(createdConfFile, "Generated with rshred %v\n\n", rVersion); err != nil {
								PrintExtraNewline("error writing to newly created config file. loading default config...")
							} else {
								newConf := GenConfig(rshredConf)
								for _, line := range newConf {
									if _, confLineErr := fmt.Fprintln(createdConfFile, line); confLineErr != nil {
										PrintExtraNewline("error writing to newly created config file. loading default config...")
										break
									}
								}
							}
						}
						if closeErr := createdConfFile.Close(); closeErr != nil {
							PrintExtraNewline("error closing config file at %s", confFilePath)
						}
					}
				} else {
					PrintExtraNewline("loading default config...")
				}
			case errors.Is(err, fs.ErrPermission):
				PrintExtraNewline("error reading config file: permission denied. loading default config...")
			default:
				PrintExtraNewline("unknown error reading config file. loading default config...")
			}
			LoadDefault(rshredConf)
		} else {
			scanner := bufio.NewScanner(openConfFile)
			var confFile []string
			for scanner.Scan() {
				confFile = append(confFile, scanner.Text())
			}
			if scanner.Err() != nil {
				PrintExtraNewline("unknown error reading config file. loading default config...")
				LoadDefault(rshredConf)
			} else {
				if len(confFile) < 3 {
					PrintExtraNewline("config file too short. loading default config...")
					LoadDefault(rshredConf)
				} else {
					err := ValidateConfig(confFile[0])
					if err != nil {
						if YesOrNo(true, "%v\nWould you like to regenerate it? Your settings will be preserved where possible.", err) {
							newConf, _ := TransferConf(rshredConf, confFile)
							openNewConf, openErr := os.OpenFile(confFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
							if openErr != nil {
								switch {
								case errors.Is(openErr, fs.ErrPermission):
									PrintExtraNewline("error regenerating config file: permission denied. your preserved config will be loaded for this run, but could not be saved.")
								default:
									PrintExtraNewline("unknown error regenerating config file. your preserved config will be loaded for this run, but could not be saved.")
								}
							} else {
								if _, confWriteErr := fmt.Fprintf(openNewConf, "Generated with rshred %v\n\n", rVersion); confWriteErr != nil {
									PrintExtraNewline("error writing to newly created config file. your preserved config will be loaded for this run, but could not be saved.")
								} else {
									for _, item := range newConf {
										if _, confWriteErr := fmt.Fprintln(openNewConf, item); confWriteErr != nil {
											PrintExtraNewline("error writing to newly created config file. your preserved config will be loaded for this run, but could not be saved.")
											break
										}
									}
								}
							}
							if preserveError := ParseConfig(newConf, rshredConf); preserveError != nil {
								panic("an invalid config file was generated")
							}
						} else {
							PrintExtraNewline("loading default config...")
							LoadDefault(rshredConf)
						}
					} else {
						parseErr := ParseConfig(confFile[2:], rshredConf)
						if parseErr != nil {
							for _, confError := range parseErr {
								fmt.Println(confError)
							}
							if YesOrNo(true, "\nThe above lines in the config file at %s are invalid.\nWould you like to regenerate it? Your settings will be preserved where possible.", confFilePath) {
								newConf, _ := TransferConf(rshredConf, confFile)
								openNewConf, openErr := os.OpenFile(confFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
								if openErr != nil {
									switch {
									case errors.Is(openErr, fs.ErrPermission):
										PrintExtraNewline("error regenerating config file: permission denied. your preserved config will be loaded for this run, but could not be saved.")
									default:
										PrintExtraNewline("unknown error regenerating config file. your preserved config will be loaded for this run, but could not be saved.")
									}
								} else {
									if _, confWriteErr := fmt.Fprintf(openNewConf, "Generated with rshred %v\n\n", rVersion); confWriteErr != nil {
										PrintExtraNewline("failed to write to newly generated config file. your preserved config will be loaded for this run, but could not be saved.")
									} else {
										for _, item := range newConf {
											if _, confWriteErr := fmt.Fprintln(openNewConf, item); confWriteErr != nil {
												PrintExtraNewline("failed to write to newly created config file. your preserved config will be loaded for this run, but could not be saved.")
												break
											}
										}
									}
								}
								if newParseErr := ParseConfig(newConf, rshredConf); newParseErr != nil {
									panic("generated invalid config file")
								}
							} else {
								PrintExtraNewline("loading default config...")
								LoadDefault(rshredConf)
							}
						}
					}
				}
			}
			if closeErr := openConfFile.Close(); closeErr != nil {
				PrintExtraNewline("error closing config file")
			}
		}
	}
	fmt.Printf("rshred Copyright 2026 TrollgeEngineering\n\n")
	fmt.Printf("This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation,\neither version 3 of the License, or (at your option) any later version.\n\n")
	fmt.Printf("This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.\nSee the GNU General Public License for more details.\n\n")
	fmt.Printf("You should have received a copy of the GNU General Public License along with this program. If not, see <https://www.gnu.org/licenses/>.\n\n")
	fmt.Printf("Welcome to rshred %v! ", rVersion)
	goodEntries := []string{}
	for {
		entries := QueryMultiple("Please enter the path of the directory or file you would like to shred below.\n")
		problemEntries := []string{}
		goodEntries = []string{}
		for _, entry := range entries {
			perm, err := permchk.CheckPerm(entry)
			switch {
			case err != nil:
				problemEntries = append(problemEntries, entry)
				fmt.Printf("%q does not exist.\n", entry)
			case perm.Dir && !(perm.R && perm.X):
				problemEntries = append(problemEntries, entry)
				fmt.Printf("you do not have read/execute permissions on the directory %q.\n", entry)
			case !perm.Dir && !perm.X:
				problemEntries = append(problemEntries, entry)
				fmt.Printf("you do not have write permissions on the file %q.\n", entry)
			default:
				goodEntries = append(goodEntries, entry)
			}
		}
		if len(problemEntries) > 0 {
			if len(goodEntries) == 0 {
				fmt.Println("------------------------------------")
				fmt.Println("The above items have the above problems. Please choose different items.")
				continue
			} else {
				fmt.Println("------------------------------------")
				if YesOrNo(false, "The above items have the above problems. Continue without them?") {
					break
				} else {
					continue
				}
			}
		} else {
			break
		}
	}
	for _, entry := range goodEntries {
		shredVictims[entry] = 0
	}
	for target := range shredVictims {
		perms, err := permchk.WalkPerms(target)
		if err != nil {
			continue
		}
		fileNoWrite = append(fileNoWrite, perms.FnW...)
		dirNoReadAndExec = append(dirNoReadAndExec, perms.DnW...)
		dirNoWrite = append(dirNoWrite, perms.DnW...)
		dirSticky = append(dirSticky, perms.Sticky...)
	}
	if len(fileNoWrite) > 0 || len(dirNoReadAndExec) > 0 {
		if len(fileNoWrite) > 0 {
			fmt.Printf("--------------------------------\nFILES WITHOUT WRITE PERMISSIONS\n--------------------------------\n\n")
			for _, file := range fileNoWrite {
				fmt.Println(file)
			}
		}
		if len(dirNoReadAndExec) > 0 {
			fmt.Printf("--------------------------------\nDIRECTORIES WITHOUT READ AND/OR EXECUTE PERMISSIONS\n--------------------------------\n\n")
			for _, dir := range dirNoReadAndExec {
				fmt.Println(dir)
			}
		}
		if !YesOrNo(false, "You do not have the neccesary permissions to shred the above files/directories in the paths you listed. Continue anyway?") {
			fmt.Fprintln(os.Stderr, "Shred aborted. No changes were made.")
		}
	}
	if len(dirNoWrite) > 0 || len(dirSticky) > 0 {
		fmt.Println("--------------------------------")
		for _, dir := range dirNoWrite {
			fmt.Println(dir)
		}
		for _, dir := range dirSticky {
			fmt.Printf("(sticky dir: you can only remove files you own) %v\n", dir)
		}
		fmt.Println("--------------------------------\nYou may not have write permissions in some of the the above directories! -u will not work on files within them!")
	}
	fmt.Println("FLAG LIST")
	fmt.Println("--------------------------")
	fmt.Println("-f, --force -- changes permissions when needed and possible to shred files.")
	fmt.Println("-n, --passes -- the number of times rshred will overwrite each file. accepts a positive integer")
	fmt.Println("-s, --size -- the amount of each file to shred. accepts a size value such as 37K or 4G. valid suffixes: B, K, M, G, T, P.")
	fmt.Println("-u, --remove -- removes each file after they have been shredded")
	fmt.Println("-v, --verbose -- show the progress of the shred")
	fmt.Println("-x, --exact -- do not round up overwrite to the next filesystem block")
	fmt.Println("-z, --zero -- overwrite the file with null bytes (0x00 or 00000000) after random shredding.")
	for {
		interactArgs := strings.Fields(QueryUser("\nPlease enter the flags you would like to use."))
		_, leftover, err := sflag.Parse(interactArgs)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if len(leftover) > 0 {
			extraFiles := []string{}
			fmt.Println("----------------------\nEXTRA ITEMS\n----------------------")
			var warnPrefix string
			var badFound bool
			for _, item := range leftover {
				warnPrefix = ""
				badFound = false
				perms, err := permchk.CheckPerm(item)
				if err != nil {
					fmt.Printf("UNRECOGNIZED FILE/DIR: %q\n", item)
				} else {
					if perms.Dir {
						if !(perms.R && perms.X) {
							warnPrefix = "(no read/execute perms on dir) "
						} else {
							bad, err := permchk.WalkPerms(item)
							if err != nil {
								fmt.Printf("(error checking permissions of children of dir) %v\n", item)
								continue
							}
							if len(bad.DnRX) > 0 {
								warnPrefix = warnPrefix + "R/X perms on some directories "
								badFound = true
							}
							if len(bad.FnW) > 0 {
								if badFound {
									warnPrefix = warnPrefix + "and "
								}
								warnPrefix = warnPrefix + "W perms on some files "
								badFound = true
							}
						}
						if badFound {
							warnPrefix = "(no " + warnPrefix + "in dir) "
						}
					} else {
						if !perms.W {
							warnPrefix = "(no write perm on file) "
						}
					}
					fmt.Printf("%v%v\n", warnPrefix, item)
					extraFiles = append(extraFiles, item)
				}
			}
			if YesOrNo(false, "The following extra arguments were found in your argument input: %v.\n\nThe above list shows all the files or directories, if any, that were able to be recognized from your extra arguments. Would you like to add them to the shred?", strings.Join(leftover, ", ")) {
				for _, file := range extraFiles {
					shredVictims[file] = 0
				}
			}
		}
		extracted := []string{}
		for item := range shredVictims {
			extracted = append(extracted, item)
		}
		if QueryUser("WARNING! You are about to RECURSIVELY DESTROY all files within the following: %q. THIS ACTION IS IRREVERSIBLE! Please type \"YES\" to confirm you understand what you are doing.", strings.Join(extracted, ", ")) == "YES" {
			break
		} else {
			continue
		}
	}
	fmt.Println("yeah the shredding went good bruh")
	return 0
}

func main() {
	os.Exit(decideInteract(os.Args[1:]))
}
