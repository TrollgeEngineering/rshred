package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	ValType int
	Value   any
}

type ConfigRegistry map[string]*Config

func ParseConfig(confFile []string, registry ConfigRegistry) []error {
	errorSlice := []error{}
	for i, line := range confFile {
		key, value, found := strings.Cut(line, "=")
		if !found {
			errorSlice = append(errorSlice, fmt.Errorf("line %v: does not contain \"=\"", i+1))
			continue
		}
		conf, ok := registry[key]
		if !ok {
			errorSlice = append(errorSlice, fmt.Errorf("line %v: unrecognized key: %q", i, key))
			continue
		} else {
			switch conf.ValType {
			case 0:
				switch value {
				case "ON":
					conf.Value = true
				case "OFF":
					conf.Value = false
				default:
					errorSlice = append(errorSlice, fmt.Errorf("line %v: value of %q must be either \"ON\" or \"OFF\"", i, key))
				}
			case 1:
				num, err := strconv.Atoi(value)
				if err != nil {
					if errors.Is(err, strconv.ErrRange) {
						errorSlice = append(errorSlice, fmt.Errorf("line %v: value of %q too large", i, key))
						continue
					} else {
						errorSlice = append(errorSlice, fmt.Errorf("line %v: value of %q must be an integer", i, key))
						continue
					}
				}
				conf.Value = num
			}
		}
	}
}
