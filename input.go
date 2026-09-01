package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func QueryMultiple(query string, a ...any) []string {
	input := []string{}
	fmt.Printf(query, a...)
	for {
		if line := ReadStdin(); line != "" {
			input = append(input, line)
		} else {
			break
		}
	}
	return input
}

func YesOrNo(pref bool, question string, a ...any) bool {
	if pref {
		answer := QueryUser(fmt.Sprintf(question+" (Y/n)", a...))
		return answer != "N" && answer != "n"
	} else {
		answer := QueryUser(fmt.Sprintf(question+" (y/N)", a...))
		return !(answer != "Y" && answer != "y")
	}
}

func QueryUser(query string, a ...any) string {
	fmt.Printf(query, a...)
	return ReadStdin()
}

func ReadStdin() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		fmt.Println("fatal: failed to read stdin")
		os.Exit(1)
	}
	input = strings.TrimRight(input, "\r\n")
	return input
}
