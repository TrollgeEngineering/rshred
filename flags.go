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
	TakesValue    bool
	Seen          bool
	ValType       int
	Value         any
	CanBeMultiple bool
}

func (e *FlagError) Error() string {
	return fmt.Sprintf("error: flag %s: %v", e.Flag, e.Err)
}

func (e *ArgError) Error() string {
	return fmt.Sprintf("error: argument %q: %v", e.Arg, e.Err)
}

func (e *SizeError) Error() string {
	return fmt.Sprintf("error: invalid size %q: %v", e.Size, e.Err)
}

func (e *FlagError) Unwrap() error {
	return e.Err
}

func (e *ArgError) Unwrap() error {
	return e.Err
}

func (e *SizeError) Unwrap() error {
	return e.Err
}

type SizeError struct {
	Size string
	Err  error
}

var (
	ErrUnknown          = errors.New("flag not recognized")
	ErrFlagAfterNonFlag = errors.New("all arguments must come before paths. use \"--\" before paths beginning with \"-\"")
	ErrLongGlued        = errors.New("long flags cannot be passed together in the same argument and cannot take values in the same argument without being seperated by \"=\"")
	ErrNoTakeValue      = errors.New("flag cannot take a value")
	ErrNeedValue        = errors.New("flag requires a value")
	ErrDupe             = errors.New("flags can only be used once")
)

var (
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
		return &SizeError{input, ErrTooLarge}
	}
	if err != nil || bytes < 0 {
		if strings.HasPrefix(input, "-") {
			return &SizeError{input, ErrNegative}
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
					return &SizeError{input, ErrNoNum}
				}
			} else {
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
					return &SizeError{input, ErrBadSuffix}
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
				if cEquals.CanBeMultiple || !cEquals.Seen {
					cEquals.Seen = true
				} else {
					return nil, fmt.Errorf("error: flag %q can only be used once", splitEqualsFlag[0])
				}
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
						if cNonLong.CanBeMultiple || !cNonLong.Seen {
							cNonLong.Seen = true
						} else {
							return nil, fmt.Errorf("error: flag %q can only be used once", string(baldArg[i]))
						}
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
			if c.CanBeMultiple || !c.Seen {
				c.Seen = true
			} else {
				return nil, fmt.Errorf("error: flag %q can only be used once", arg)
			}
			if i == len(inputArgs)-1 {
				return nil, fmt.Errorf("error: flag %q requires a value", arg)
			}
			if err := CheckValue(arg, arg, inputArgs[i+1], c); err != nil {
				return nil, err
			} else {
				consumedArgs = append(consumedArgs, i+1)
			}
		default:
			if c.CanBeMultiple || !c.Seen {
				c.Seen = true
			} else {
				return nil, fmt.Errorf("error: flag %q can only be used once", arg)
			}
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
