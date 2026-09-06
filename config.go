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
	var errs []error
	for i, line := range confFile {
		if strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			errs = append(errs, fmt.Errorf("line %v: does not contain \"=\"", i+1))
			continue
		}
		conf, ok := registry[key]
		if !ok {
			errs = append(errs, fmt.Errorf("line %v: unrecognized key: %q", i, key))
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
					errs = append(errs, fmt.Errorf("line %v: value of %q must be either \"ON\" or \"OFF\"", i, key))
					continue
				}
			case 1:
				num, err := strconv.Atoi(value)
				if err != nil {
					if errors.Is(err, strconv.ErrRange) {
						errs = append(errs, fmt.Errorf("line %v: value of %q too large", i, key))
						continue
					} else {
						errs = append(errs, fmt.Errorf("line %v: value of %q must be an integer", i, key))
						continue
					}
				}
				conf.Value = num
			case 2:
				conf.Value = value
			case 3:
				list := []string{}
				for j := i + 1; j < len(confFile); j++ {
					if strings.TrimSpace(confFile[j]) == "END" {
						break
					}
					list = append(list, confFile[j])
				}
				conf.Value = list
			default:
				panic(fmt.Sprintf("invalid conf value %q", conf.ValType))
			}
		}
	}
	return errs
}
