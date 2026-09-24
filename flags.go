package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type FlagValueType int

const (
	ValueInt FlagValueType = iota
	ValueString
	ValueSize
)

type Flag struct {
	TakesValue    bool
	Seen          bool
	ValType       FlagValueType
	Default       any
	Value         any
	CanBeMultiple bool
}
type FlagRegistry map[string]*Flag

func (e *FlagError) Error() string {
	if e.Arg != "" {
		return fmt.Sprintf("error: argument %q: flag %s: %v", e.Arg, e.Flag, e.Err)
	} else {
		return fmt.Sprintf("error: flag %s: %v", e.Flag, e.Err)
	}
}

func (e *ArgError) Error() string {
	return fmt.Sprintf("error: argument %q: %v", e.Arg, e.Err)
}

func (e *ValueError) Error() string {
	return fmt.Sprintf("error: flag %s: invalid value %q: %v", e.Flag, e.Value, e.Err)
}

func (e *FlagError) Unwrap() error {
	return e.Err
}

func (e *ArgError) Unwrap() error {
	return e.Err
}

func (e *ValueError) Unwrap() error {
	return e.Err
}

type ValueError struct {
	Value string
	Flag  string
	Err   error
}

var (
	ErrUnknown               = errors.New("flag not recognized")
	ErrFlagAfterNonFlag      = errors.New("all arguments must come before paths. use \"--\" before paths beginning with \"-\"")
	ErrLongGlued             = errors.New("long flags cannot be passed together in the same argument and cannot take values in the same argument without being seperated by \"=\"")
	ErrNoTakeValue           = errors.New("flag cannot take a value")
	ErrNoTakeValueTryHyphens = errors.New("flag cannot take a value. all arguments must come before paths. use \"--\" before paths beginning with \"-\"")
	ErrNeedValue             = errors.New("flag requires a value")
	ErrDupe                  = errors.New("flags can only be used once")
)

var (
	ErrNeedInt   = errors.New("flag needs an integer value")
	ErrTooLarge  = errors.New("value is too large")
	ErrNegative  = errors.New("value cannot be negative")
	ErrNoNum     = errors.New("value must begin with a number")
	ErrBadSuffix = errors.New("invalid suffix. suffixes must be uppercase")
)

type ArgError struct {
	Arg string
	Err error
}

type FlagError struct {
	Arg  string
	Flag string
	Err  error
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
	FlagRegistry = map[string]*Flag{
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

/*
	 var behaviors struct {
		Verbose    bool // -v flag
		NoRound    bool // -x flag
		ZeroPass   bool // -z flag
		Deallocate bool // -u flag
		Force      bool // -f flag
		Passes     int  // -n flag
		ShredSize  int  // -s flag
	}
*/
func reset(registry FlagRegistry) {
	for _, flg := range registry {
		flg.Value = flg.Default
		flg.Seen = false
	}
}

func parseSize(input string) (int, error) {
	bytes, err := strconv.Atoi(input)
	if errors.Is(err, strconv.ErrRange) {
		return 0, ErrTooLarge
	}
	if err != nil || bytes < 0 {
		if strings.HasPrefix(input, "-") {
			return 0, ErrNegative
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
					return 0, ErrNoNum
				}
			} else {
				baseNum, _ := strconv.Atoi(string(val))
				switch string(input[i]) {
				case "B":
					return baseNum, nil
				case "K":
					return 1000 * baseNum, nil
				case "M":
					return 1000 * 1000 * baseNum, nil
				case "G":
					return 1000 * 1000 * 1000 * baseNum, nil
				case "T":
					return 1000 * 1000 * 1000 * 1000 * baseNum, nil
				case "P":
					return 1000 * 1000 * 1000 * 1000 * 1000 * baseNum, nil
				default:
					return 0, ErrBadSuffix
				}
			}
		}
		panic("uncaught exception while parsing size")
	} else {
		return bytes, nil
	}
}

func ParseFlags(inputArgs []string, registry FlagRegistry) (leftover []string, parseErr error) {
	defer func() {
		if parseErr != nil {
			reset(registry)
		}
	}()
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
			for j := i; j < len(inputArgs); j++ {
				if !strings.HasPrefix(inputArgs[j], "-") {
					leftoverArgs = append(leftoverArgs, inputArgs[j])
				} else {
					switch {
					case i == 0:
						return nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
					case strings.HasPrefix(inputArgs[i-1], "--"):
						test, ok := registry[strings.TrimPrefix(inputArgs[i-1], "--")]
						if ok {
							if test.TakesValue {
								panic("if it takes a value, then it should have been marked consumed or rejected.")
							}
							return nil, &FlagError{Flag: inputArgs[i-1], Err: ErrNoTakeValueTryHyphens}
						} else {
							return nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
						}
					case strings.HasPrefix(inputArgs[i-1], "-"):
						test, ok := registry[string(inputArgs[i-1][len(inputArgs[i-1])-1])] // Checks the last character of the last good argument.
						if ok {
							if test.TakesValue {
								panic("if it takes a value, then it should have already been marked consumed or rejected")
							}
							return nil, &FlagError{Flag: inputArgs[i-1], Err: ErrNoTakeValueTryHyphens}
						} else {
							return nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
						}
					case slices.Contains(consumedArgs, i-1):
						return nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
					default:
						panic("all args before the last good arg need to either be flag arguments or consumed values")
					}
				}
			}
			return leftoverArgs, nil
		}
		c, ok := registry[baldArg]
		switch {
		case !ok:
			if len(baldArg) <= 1 {
				return nil, &ArgError{baldArg, ErrUnknown}
			}
			switch {
			case strings.Contains(arg, "="):
				splitEqualsFlag := strings.SplitN(baldArg, "=", 2)
				cEquals, ok := registry[splitEqualsFlag[0]]
				if !ok {
					return nil, &ArgError{baldArg, ErrUnknown}
				}
				if cEquals.CanBeMultiple || !cEquals.Seen {
					cEquals.Seen = true
				} else {
					return nil, &FlagError{Flag: baldArg, Err: ErrDupe}
				}
				err := checkValue(splitEqualsFlag[1], cEquals)
				if err != nil {
					return nil, &ValueError{splitEqualsFlag[1], splitEqualsFlag[0], err}
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
								return nil, &FlagError{baldArg, string(baldArg[i]), ErrUnknown}
							} else {
								return nil, &FlagError{baldArg, string(baldArg[i-1]), ErrNoTakeValue}
							}
						}
						if cNonLong.TakesValue {
							nonBoolFlagFound = true
							nonBool = string(baldArg[i])
						}
						if cNonLong.CanBeMultiple || !cNonLong.Seen {
							cNonLong.Seen = true
						} else {
							return nil, &FlagError{baldArg, string(baldArg[i]), ErrDupe}
						}
					} else {
						potentFlagValue = append(potentFlagValue, baldArg[i])
					}
				}
				if nonBoolFlagFound {
					var valErr error
					if len(potentFlagValue) < 1 {
						if i == len(inputArgs)-1 {
							return nil, &FlagError{baldArg, nonBool, ErrNeedValue}
						}
						consumedArgs = append(consumedArgs, i+1)
						valErr = checkValue(inputArgs[i+1], registry[nonBool])
					} else {
						valErr = checkValue(string(potentFlagValue), registry[nonBool])
					}
					if valErr != nil {
						return nil, &FlagError{baldArg, nonBool, valErr}
					}
				}
			default:
				return nil, &ArgError{baldArg, ErrLongGlued}
			}
		case c.TakesValue:
			if c.CanBeMultiple || !c.Seen {
				c.Seen = true
			} else {
				return nil, &FlagError{Flag: baldArg, Err: ErrDupe}
			}
			if i == len(inputArgs)-1 {
				return nil, &FlagError{Flag: baldArg, Err: ErrNeedValue}
			}
			if err := checkValue(inputArgs[i+1], c); err != nil {
				return nil, &ValueError{inputArgs[i+1], baldArg, err}
			} else {
				consumedArgs = append(consumedArgs, i+1)
			}
		default:
			if c.CanBeMultiple || !c.Seen {
				c.Seen = true
			} else {
				return nil, &FlagError{Flag: baldArg, Err: ErrDupe}
			}
		}
	}
	return nil, nil
}

func checkValue(value string, flgEntry *Flag) error {
	if value == "" {
		return ErrNeedValue
	}
	switch flgEntry.ValType {
	case ValueInt:
		flagNum, err := strconv.Atoi(value)
		if err != nil {
			if errors.Is(err, strconv.ErrRange) {
				return ErrTooLarge
			} else {
				return ErrNeedInt
			}
		}
		flgEntry.Value = flagNum
		return nil
	case ValueString:
		flgEntry.Value = value
		return nil
	case ValueSize:
		bytes, err := parseSize(value)
		if err != nil {
			return err
		}
		flgEntry.Value = bytes
	default:
		panic(fmt.Sprintf("THIS SHOULD HAVE NEVER BEEN HIT!!! FLAG TYPE SHOULD NEVER BE %v!!!!", flgEntry.ValType))
	}
	return nil // Shouldn't be reached.
}
