package sflag

import (
	"fmt"
	"slices"
	"strings"
)

// Parse reads a slice of arguments and parses it according to the flags registered
// using [Register] and/or [RegisterWithShort]. It takes a []string as
// input and returns a [Results], a []string with an entry for every leftover argument,
// and an error. In the event an error is returned, all other arguments will be
// returned nil.
//
// Parse walks the input []string one string at a time, marking long and short flags
// and their values as they are seen.
//
// Parse will reject values passed to short flags with an "=", (e.g. "-n=5")
// by returning the error [ErrShortEquals] and will reject values glued directly to long
// flags (e.g. "--passes5") by returning the error [ErrLongGlued]. Parse will also
// reject long flags passed glued to each other (e.g. "--verboseexact" where "verbose"
// and "exact" are seperate long flags) by returning the error [ErrLongGlued].
//
// Because of how Parse walks the input []string, and to prevent user errors, Parse
// will reject interspersed flags with [ErrFlagAfterNonFlag].
func Parse(inputArgs []string) (results Results, leftover []string, parseErr error) {
	if len(crntRegis.long) < 1 {
		panicExternal(ErrNeedRegis)
	}
	for longName, flag := range crntRegis.long {
		for conflict := range flag.conflictsWith {
			if conFlag, ok := crntRegis.long[conflict]; !ok {
				panicExternal(&FlagPanic{Long: longName, Problem: Conflict, Conflict: conflict, Err: ErrConflictNotRegis})
			} else {
				conFlag.conflictsWith[longName] = struct{}{}
			}
		}
	}
	defer func() {
		if parseErr != nil {
			ResetResults()
		} else {
			for long, result := range crntResults {
				flag, ok := crntRegis.long[long]
				if !ok {
					panicInternal(fmt.Sprintf("flag %q has a *Result but doesn't have a *flag", long))
				}
				if flag.valType == ValueBool && result.Seen && !result.ValueSeen {
					result.Value = true
				}
			}
		}
	}()
	var (
		consumedArgs = []int{}
		leftoverArgs = []string{}
		baldArg      string
	)
	// inputArgs = strings.Fields(QueryUser("Please enter the flags you would like to use."))
	for i, arg := range inputArgs {
		switch {
		case slices.Contains(consumedArgs, i):
			continue
		case arg == "--":
			return crntResults, inputArgs[i+1:], nil
		case strings.HasPrefix(arg, "--"):
			baldArg = strings.TrimPrefix(arg, "--")
			if len(baldArg) <= 1 {
				return nil, nil, &FlagError{Flag: baldArg, Err: ErrShortTwoHyphen}
			}
			c, ok := crntRegis.long[baldArg]
			switch {
			case ok:
				for conflict := range c.conflictsWith {
					if crntResults[conflict].Seen {
						return nil, nil, &ConflictError{Flag1: baldArg, Flag2: conflict, Err: ErrContradict}
					}
				}
				if c.valueRequired {
					if i == len(inputArgs)-1 {
						return nil, nil, &FlagError{Flag: baldArg, Err: ErrNeedValue}
					}
					err := checkValue(inputArgs[i+1], c)
					if err != nil {
						return nil, nil, &ValueError{inputArgs[i+1], baldArg, err}
					}
					crntResults[baldArg].ValueSeen = true
					consumedArgs = append(consumedArgs, i+1)
				}
				if err := markSeen(c); err != nil {
					return nil, nil, &FlagError{Flag: baldArg, Err: err}
				}
			case strings.Contains(baldArg, "="):
				flag, value, _ := strings.Cut(baldArg, "=")
				cEquals, ok := crntRegis.long[flag]
				if !ok {
					return nil, nil, &ArgError{baldArg, ErrUnknown}
				}
				for conflict := range cEquals.conflictsWith {
					if crntResults[conflict].Seen {
						return nil, nil, &ConflictError{Flag1: flag, Flag2: conflict, Err: ErrContradict}
					}
				}
				if cEquals.valType == ValueBoolCount {
					return nil, nil, &FlagError{baldArg, flag, ErrNoTakeValue}
				}
				if err := markSeen(cEquals); err != nil {
					return nil, nil, &FlagError{baldArg, flag, err}
				}
				if err := checkValue(value, cEquals); err != nil {
					return nil, nil, &ValueError{value, flag, err}
				}
				crntResults[flag].ValueSeen = true
			default:
				for flag := range crntRegis.long {
					if strings.Contains(baldArg, flag) {
						return nil, nil, &ArgError{baldArg, ErrLongGlued}
					}
				}
				return nil, nil, &ArgError{baldArg, ErrUnknown}
			}
		case strings.HasPrefix(arg, "-") && len(arg) > 1:
			baldArg = strings.TrimPrefix(arg, "-")
			for long := range crntRegis.long {
				if baldArg == long {
					return nil, nil, &ArgError{arg, ErrLongOneHyphen}
				}
			}
			if len(baldArg) > 1 {
				nonBoolFlagFound := false
				var nonBool string
				for j := 0; j < len(baldArg); j++ {
					cGlued, ok := crntRegis.short[baldArg[j]]
					if !ok {
						for flag := range crntRegis.long {
							if strings.Contains(baldArg, flag) {
								return nil, nil, &ArgError{arg, ErrLongOneHyphen}
							}
						}
						switch {
						case strings.Contains(baldArg, "="):
							return nil, nil, &ArgError{arg, ErrShortEquals}
						case j != 0:
							prevFlag, ok := crntRegis.short[baldArg[j-1]]
							if !ok {
								panicInternal(fmt.Sprintf("flag %q was not found in the short registry, but there should have been an error before reaching this point", string(baldArg[j-1])))
							}
							if prevFlag.valueRequired {
								panicInternal(fmt.Sprintf("flag %q requires a value, but this should have already been consumed as part of the value and the loop should have been broken", string(baldArg[j-1])))
							}
							return nil, nil, &FlagError{arg, string(baldArg[j]), ErrUnknownMayBeMalformed}
						default:
							return nil, nil, &FlagError{arg, string(baldArg[j]), ErrUnknown}
						}
					}
					for conflict := range cGlued.conflictsWith {
						if crntResults[conflict].Seen {
							return nil, nil, &ConflictError{Flag1: string(baldArg[j]), Flag2: conflict, Err: ErrContradict}
						}
					}
					if cGlued.valueRequired {
						nonBoolFlagFound = true
						nonBool = string(baldArg[j])
						switch {
						case len(baldArg[j+1:]) > 0:
							if err := checkValue(string(baldArg[j+1:]), cGlued); err != nil {
								if strings.HasPrefix(string(baldArg[j+1:]), "=") {
									return nil, nil, &FlagError{Flag: nonBool, Err: ErrShortEquals}
								} else {
									return nil, nil, &ValueError{string(baldArg[j+1:]), nonBool, err}
								}
							}
						case i < len(inputArgs)-1:
							if err := checkValue(inputArgs[i+1], cGlued); err != nil {
								return nil, nil, &ValueError{inputArgs[i+1], nonBool, err}
							}
							consumedArgs = append(consumedArgs, i+1)
						default:
							return nil, nil, &FlagError{arg, nonBool, ErrNeedValue}
						}
						crntResults[cGlued.formal].ValueSeen = true
					}
					if err := markSeen(cGlued); err != nil {
						return nil, nil, &FlagError{arg, nonBool, err}
					}
					if nonBoolFlagFound {
						break
					}
				}
			} else {
				c, ok := crntRegis.short[baldArg[0]]
				if ok {
					for conflict := range c.conflictsWith {
						if crntResults[conflict].Seen {
							return nil, nil, &ConflictError{Flag1: baldArg, Flag2: conflict, Err: ErrContradict}
						}
					}
					if c.valueRequired {
						if i == len(inputArgs)-1 {
							return nil, nil, &FlagError{Flag: baldArg, Err: ErrNeedValue}
						}
						if err := checkValue(inputArgs[i+1], c); err != nil {
							return nil, nil, &ValueError{inputArgs[i+1], baldArg, err}
						}
						consumedArgs = append(consumedArgs, i+1)
						crntResults[c.formal].ValueSeen = true
					}
					if err := markSeen(c); err != nil {
						return nil, nil, &FlagError{Flag: baldArg, Err: err}
					}
				} else {
					return nil, nil, &ArgError{arg, ErrUnknown}
				}
			}
		default:
			for j := i; j < len(inputArgs); j++ {
				if !(strings.HasPrefix(inputArgs[j], "-") && len(inputArgs[j]) > 1) && !(inputArgs[i] == "true" || inputArgs[i] == "false") {
					leftoverArgs = append(leftoverArgs, inputArgs[j])
				} else {
					switch {
					case i == 0:
						return nil, nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
					case strings.HasPrefix(inputArgs[i-1], "--"):
						test, ok := crntRegis.long[strings.TrimPrefix(inputArgs[i-1], "--")]
						if ok {
							switch {
							case test.valueRequired:
								panicInternal("if it requires a value, then it should have already been marked consumed or rejected.")
							case test.valType == ValueBool && (inputArgs[i] == "true" || inputArgs[i] == "false"):
								return nil, nil, &FlagError{Flag: inputArgs[i-1], Err: ErrMalformedValueTryHyphens}
							}
							return nil, nil, &FlagError{Flag: inputArgs[i-1], Err: ErrNoTakeValueTryHyphens}
						} else {
							return nil, nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
						}
					case strings.HasPrefix(inputArgs[i-1], "-"):
						test, ok := crntRegis.short[inputArgs[i-1][len(inputArgs[i-1])-1]] // Checks the last character of the last good argument.
						if ok && len(inputArgs[i-1]) == 1 && !slices.Contains(consumedArgs, i-1) {
							switch {
							case test.valueRequired:
								panicInternal("if it requires a value, then it should have already been marked consumed or rejected")
							case test.valType == ValueBool && (inputArgs[i] == "true" || inputArgs[i] == "false"):
								return nil, nil, &FlagError{Flag: inputArgs[i-1], Err: ErrMalformedValueTryHyphens}
							}
							return nil, nil, &FlagError{Flag: inputArgs[i-1], Err: ErrNoTakeValueTryHyphens}
						} else {
							return nil, nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
						}
					case slices.Contains(consumedArgs, i-1):
						return nil, nil, &ArgError{inputArgs[j], ErrFlagAfterNonFlag}
					default:
						panicInternal("last good arg must be a flag, flag group, or consumed value")
					}
				}
			}
			return crntResults, leftoverArgs, nil
		}
	}
	return crntResults, nil, nil
}
