package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type Flag struct {
	TakesValue bool
	Seen       bool
	ValType    int
	Value      any
}

/* var (
	verbose   = &Flag{}
	noRound   = &Flag{}
	zeroPass  = &Flag{}
	deallo    = &Flag{}
	force     = &Flag{}
	passes    = &Flag{TakesValue: true, ValType: 0, Value: 3}
	ShredSize = &Flag{TakesValue: true, ValType: 1, Value: -1}
) */

/* var (
	flagRegistry = map[string]*Flag{
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
	}
) */

type flagRegistry map[string]*Flag

/* var behaviors struct {
	Verbose    bool // -v flag
	NoRound    bool // -x flag
	ZeroPass   bool // -z flag
	Deallocate bool // -u flag
	Force      bool // -f flag
	Passes     int  // -n flag
	ShredSize  int  // -s flag
} */

func ParseSize(input string, s *Flag) error {
	bytes, err := strconv.Atoi(input)
	if errors.Is(err, strconv.ErrRange) {
		return fmt.Errorf("error: value %q is too large", input)
	}
	if err != nil || bytes < 0 {
		if strings.HasPrefix(input, "-") {
			return fmt.Errorf("error: invalid value %q. value of -s cannot be negative", input)
		}
		allNums := false
		val := []byte{}
		for i := 0; i < len(input); i++ {
			if !allNums {
				if unicode.IsDigit(rune(input[i])) {
					val = append(val, input[i])
					if !unicode.IsDigit(rune(input[i+1])) {
						allNums = true
					}
				} else {
					return fmt.Errorf("error: invalid value %q. value of -s must begin with a number", input)
				}
			} else {
				if i == len(input)-1 {
					baseNum, _ := strconv.Atoi(string(val))
					switch string(input[i]) {
					case "B":
						s.Value = baseNum
						return nil
					case "K":
						s.Value = 1000 * baseNum
						return nil
					case "M":
						s.Value = 1000 * 1000 * baseNum
						return nil
					case "G":
						s.Value = 1000 * 1000 * 1000 * baseNum
						return nil
					case "T":
						s.Value = 1000 * 1000 * 1000 * 1000 * baseNum
						return nil
					case "P":
						s.Value = 1000 * 1000 * 1000 * 1000 * 1000 * baseNum
						return nil
					default:
						return fmt.Errorf("error: invalid value %q. unrecognized suffix %q. suffixes must be uppercase", input, string(input[i]))
					}
				} else {
					return fmt.Errorf("error: invalid value %q. suffix for size must be one uppercase character", input)
				}
			}
		}
		return nil // This will be never be hit, but my IDE wanted this return anyway.
	} else {
		s.Value = bytes
		return nil
	}
}

func ParseFlags(inputArgs []string, registry flagRegistry) ([]string, error) {
	var (
		long         bool
		consumedArgs = []int{}
		leftoverArgs = []string{}
		baldArg      string
	)
	// inputArgs = strings.Fields(QueryUser("Please enter the flags you would like to use."))
	for i, arg := range inputArgs {
		switch {
		case arg == "--":
			for i := i + 1; i < len(inputArgs); i++ {
				leftoverArgs = append(leftoverArgs, inputArgs[i])
			}
			return leftoverArgs, nil
		case slices.Contains(consumedArgs, i):
			continue
		case strings.HasPrefix(arg, "--"):
			long = true
			baldArg = strings.TrimPrefix(arg, "--")
		case strings.HasPrefix(arg, "-"):
			long = false
			baldArg = strings.TrimPrefix(arg, "-")
		default:
			for i := i; i < len(inputArgs); i++ {
				if !strings.HasPrefix(inputArgs[i], "-") {
					leftoverArgs = append(leftoverArgs, inputArgs[i])
				} else {
					return nil, fmt.Errorf("error: invalid argument %q. all arguments must come before paths. use \"--\" before paths beginning with \"-\"", leftoverArgs[0])
				}
			}
			return leftoverArgs, nil
		}
		c, ok := registry[baldArg]
		switch {
		case !ok:
			if len(baldArg) <= 1 {
				return nil, fmt.Errorf("error: flag %q not recognized", baldArg)
			}
			switch {
			case strings.Contains(arg, "="):
				splitEqualsFlag := strings.SplitN(baldArg, "=", 2)
				cEquals, ok := registry[splitEqualsFlag[0]]
				if !ok {
					return nil, fmt.Errorf("error: invalid argument %q: flag %q not recognized", arg, splitEqualsFlag[0])
				}
				cEquals.Seen = true
				err := CheckValue(splitEqualsFlag[0], baldArg, splitEqualsFlag[1], cEquals)
				if err != nil {
					return nil, err
				}
			case !long:
				potentFlagValue := []byte{}
				nonBoolFlagFound := false
				var nonBool string
				for i := 0; i < len(baldArg); i++ {
					if !nonBoolFlagFound {
						cNonLong, ok := registry[string(baldArg[i])]
						if !ok {
							if i == 0 {
								return nil, fmt.Errorf("error: invalid argument %q. unrecognized flag %q", arg, string(baldArg[i]))
							} else {
								return nil, fmt.Errorf("error: invalid argument %q. flag %q cannot take a value", arg, string(baldArg[i-1]))
							}
						}
						if cNonLong.TakesValue {
							nonBoolFlagFound = true
							nonBool = string(baldArg[i])
						}
						cNonLong.Seen = true
					} else {
						potentFlagValue = append(potentFlagValue, baldArg[i])
					}
				}
				if nonBoolFlagFound {
					err := CheckValue(baldArg, nonBool, string(potentFlagValue), registry[nonBool])
					if err != nil {
						return nil, err
					}
				}
			default:
				return nil, fmt.Errorf("error: invalid argument %q. long flags cannot be passed together in the same argument and cannot take values in the same argument without being seperated by \"=\"", baldArg)
			}
		case c.TakesValue:
			c.Seen = true
			if i == len(inputArgs)-1 {
				return nil, fmt.Errorf("error: flag %q requires a value", arg)
			}
			if err := CheckValue(arg, arg, inputArgs[i+1], c); err != nil {
				return nil, err
			} else {
				consumedArgs = append(consumedArgs, i+1)
			}
		default:
			c.Seen = true
		}
	}
	return nil, nil
}

func CheckValue(argIn string, flg string, value string, flgEntry *Flag) error {
	switch flgEntry.ValType {
	case 0:
		flagNum, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("error: invalid argument %q: flag %q requires an integer value", argIn, flg)
		}
		flgEntry.Value = flagNum
		return nil
	case 1:
		flgEntry.Value = value
		return nil
	case 2:
		err := ParseSize(value, flgEntry)
		if err != nil {
			return err
		}
	default:
		panic(fmt.Sprintf("THIS SHOULD HAVE NEVER BEEN HIT!!! FLAG TYPE SHOULD NEVER BE %v!!!!", flgEntry.ValType))
	}
	return nil // Shouldn't be reached.
}
