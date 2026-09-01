package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	verbose   = &Flag{}
	noRound   = &Flag{}
	zeroPass  = &Flag{}
	deallo    = &Flag{}
	force     = &Flag{}
	passes    = &Flag{TakesValue: true, ValType: 0, Value: 3}
	ShredSize = &Flag{TakesValue: true, ValType: 2, Value: -1}
)

var (
	fileNoWrite      []string
	dirNoReadAndExec []string
	dirNoWrite       []string
	dirSticky        []string
)

var rshredRegistry = flagRegistry{
	"v":       verbose,
	"verbose": verbose,

	"x":     noRound,
	"exact": noRound,

	"z":    zeroPass,
	"zero": zeroPass,

	"u":      deallo,
	"remove": deallo,

	"f":     force,
	"force": force,

	"n":      passes,
	"passes": passes,

	"s":    ShredSize,
	"size": ShredSize,
	// Non-interactive only flags below.
	"help": &Flag{},

	"version": &Flag{},

	"shut-up": &Flag{},
}

var (
	rVersion     string = "v4.0.0"
	shredVictims        = map[string]int{}
)

func DecideInteract(args []string) int {
	if len(args) > 0 {
		return cli(args)
	} else {
		return interact()
	}
}

func cli(args []string) int {
	fmt.Println("you did cli stuff")
	return 0
}

func interact() int {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGHUP)
	go func() {
		sig := <-sigChan
		if sysSig, ok := sig.(syscall.Signal); ok {
			exitCode := 128 + int(sysSig)
			fmt.Fprintf(os.Stderr, "\nInterrupt recieved!\n\n")
			fmt.Fprintln(os.Stderr, "Quitting...")
			os.Exit(exitCode)
		} else {
			os.Exit(2)
		}
	}()
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
			perm, err := CheckPerm(entry)
			switch {
			case err != nil:
				problemEntries = append(problemEntries, entry)
				fmt.Printf("%q does not exist.\n", entry)
			case perm.dir && !(perm.r && perm.x):
				problemEntries = append(problemEntries, entry)
				fmt.Printf("you do not have read/execute permissions on the directory %q.\n", entry)
			case !perm.dir && !perm.w:
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
		perms, err := WalkPerms(target)
		if err != nil {
			continue
		}
		fileNoWrite = append(fileNoWrite, perms.FnW...)
		dirNoReadAndExec = append(dirNoReadAndExec, perms.DnW...)
		dirNoWrite = append(dirNoWrite, perms.DnW...)
		dirSticky = append(dirSticky, perms.sticky...)
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
			log.Fatal("Shred aborted. No changes were made.")
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
		extraArgs, err := ParseFlags(interactArgs, rshredRegistry)
		if err != nil {
			fmt.Println(err)
		}
		if len(extraArgs) > 0 {
			extraFiles := []string{}
			fmt.Println("----------------------\nEXTRA ITEMS\n----------------------")
			var warnPrefix string
			var badFound bool
			for _, item := range extraArgs {
				warnPrefix = ""
				badFound = false
				perms, err := CheckPerm(item)
				if err != nil {
					fmt.Printf("UNRECOGNIZED FILE/DIR: %q\n", item)
				} else {
					if perms.dir {
						if !(perms.r && perms.x) {
							warnPrefix = "(no read/execute perms on dir) "
						} else {
							bad, err := WalkPerms(item)
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
						if !perms.w {
							warnPrefix = "(no write perm on file) "
						}
					}
					fmt.Printf("%v%v\n", warnPrefix, item)
					extraFiles = append(extraFiles, item)
				}
			}
			if YesOrNo(false, "The following extra arguments were found in your argument input: %v.\n\nThe above list shows all the files or directories, if any, that were able to be recognized from your extra arguments. Would you like to add them to the shred?", strings.Join(extraArgs, ", ")) {
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
	os.Exit(DecideInteract(os.Args[1:]))
}
