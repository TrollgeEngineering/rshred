package sflag

import (
	"slices"
	"strings"
)

type flag struct {
	valueRequired bool
	valType       valueType
	canBeMultiple bool
	conflictsWith map[string]struct{}
	formal        string
	short         byte
	defaultVal    any
}

type (
	longRegis  map[string]*flag
	shortRegis map[byte]*flag
)

type flagRegistry struct {
	long  longRegis
	short shortRegis
}

var crntRegis = flagRegistry{
	make(longRegis),
	make(shortRegis),
}

// Options is a struct of configuration options passed
// to [Register] or [RegisterWithShort] configure a flag.
// Passing an empty Options will result in the registration
// of a flag of ValueType [ValueBool] that cannot be repeated,
// may take an optional boolean value, and has no conflicts.
type Options struct {
	// ValueRequired is a configuration bool used to set
	// whether or not a flag is required to take a value.
	// If a flag without a corresponding required value
	// is passed in the inputArgs of [Parse], [ErrNeedValue]
	// will be returned. Both [Register] and [RegisterWithShort]
	// will panic if a ValueRequired is set to true and
	// ValueType is set to [ValueBoolCount].
	ValueRequired bool
	// CanBeMultiple is a configuration bool used to set
	// whether or not a flag can be passed multiple times.
	// Both [Register] and [RegisterWithShort] panic
	// if CanBeMultiple is set to true and ValueType
	// is set to [ValueBool].
	CanBeMultiple bool
	// ValueType is used to configure the types of values,
	// a flag can take. The constants above should be passed
	// to ValueType.
	ValueType valueType
	// ConflictsWith is a []string used at registration to
	// list flags by long name that should not be passed in
	// the same line as the registered flag. Conflicting flags
	// do not have to be registered symetrically, and conflicts
	// will still be handled no matter the order the flags are
	// passed. For example, if "verbose" conflicts with "logs",
	// only "logs" needs to be registered in verbose's
	// ConflictsWith, or vice versa. The conflicting flag
	// does not need to exist at registration time, but
	// [Parse] will panic if a registered flag lists a flag
	// in ConflictsWith that was not registered.
	ConflictsWith []string
	Default       any
}

func isAlphaNumASCII(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func validateLongName(long string) error {
	if len(long) < 2 {
		return ErrLongTooShort
	}
	if strings.HasPrefix(long, "-") || strings.HasSuffix(long, "-") {
		return ErrLeadTrailHyphen
	}
	for i := 0; i < len(long); i++ {
		if !isAlphaNumASCII(long[i]) && long[i] != '-' {
			return ErrLongIllegalChar
		}
	}
	return nil
}

// Register registers a flag with the specified long
// name and the options designated in the [Options]. Register
// will panic if the string passed has a length less than
// 2, has leading or trailing hyphens, or if it contains
// characters that are not alphanumeric ASCII characters
// or hyphens. Register will also panic if the long name
// supplied has already been registered, if [ConflictsWith]
// contains the name of the flag currently being registered,
// or if the [ValueType] is invalid.
func Register(long string, options Options) {
	register(long, 0, options)
}

// RegisterWithShort registers a flag with a long name and
// a corresponding short name using the configuration options
// supplied in the [Options]. RegisterWithShort will panic
// for the same reasons as [Register], with the addition of
// panicking when the short name supplied has already been
// registered and if the short name is not an alphanumeric
// ASCII character.
func RegisterWithShort(long string, short byte, options Options) {
	if !isAlphaNumASCII(short) {
		panicExternal(&FlagPanic{Long: long, Short: short, Problem: ShortOnly, Err: ErrShortIllegalChar})
	}
	register(long, short, options)
}

func register(long string, short byte, options Options) {
	if err := validateLongName(long); err != nil {
		panicExternal(&FlagPanic{Long: long, Short: short, Problem: LongOnly, Err: err})
	}
	if _, ok := crntRegis.long[long]; ok {
		panicExternal(&FlagPanic{Long: long, Short: short, Problem: LongOnly, Err: ErrRegisDupe})
	}
	if slices.Contains(options.ConflictsWith, long) {
		panicExternal(&FlagPanic{long, short, long, Conflict, ErrSelfConflict})
	}
	if short != 0 {
		if _, ok := crntRegis.short[short]; ok {
			panicExternal(&FlagPanic{Long: long, Short: short, Problem: ShortOnly, Err: ErrRegisDupe})
		}
	}
	for _, conflict := range options.ConflictsWith {
		if len(conflict) < 2 {
			panicExternal(&FlagPanic{long, short, conflict, Conflict, ErrShortConflict})
		}
	}
	crntResult := &Result{}
	if options.ValueType >= barrier {
		panicExternal(&ValuePanic{long, options.ValueType, ErrInvalidValueType})
	}
	if options.ValueType == ValueBoolCount {
		if options.ValueRequired {
			panicExternal(&ValuePanic{long, options.ValueType, ErrBoolCountNoTakeValue})
		} else {
			crntResult.Value = 0
		}
	}
	if options.CanBeMultiple {
		switch options.ValueType {
		case ValueBool:
			panicExternal(&ValuePanic{long, ValueBool, ErrBoolCantBeMultiple})
		case ValueInt:
			if options.Default != nil {
				if val, ok := options.Default.([]int); !ok {
					panicExternal(&ValuePanic{long, ValueInt, ErrNeedIntSliceDefault})
				} else {
					crntResult.Value = val
				}
			} else {
				crntResult.Value = []int{}
			}
		case ValueString:
			if options.Default != nil {
				if val, ok := options.Default.([]string); !ok {
					panicExternal(&ValuePanic{long, ValueString, ErrNeedStringSliceDefault})
				} else {
					crntResult.Value = val
				}
			} else {
				crntResult.Value = []string{}
			}
		case ValueSize:
			if options.Default != nil {
				if val, ok := options.Default.([]Size); !ok {
					panicExternal(&ValuePanic{long, ValueSize, ErrNeedSizeSliceDefault})
				} else {
					crntResult.Value = val
				}
			} else {
				crntResult.Value = []Size{}
			}
		}
	} else {
		switch options.ValueType {
		case ValueBool:
			if options.Default != nil {
				if val, ok := options.Default.(bool); !ok {
					panicExternal(&ValuePanic{long, ValueBool, ErrNeedBool})
				} else {
					crntResult.Value = val
				}
			} else {
				crntResult.Value = false
			}
		case ValueInt:
			if options.Default != nil {
				if val, ok := options.Default.(int); !ok {
					panicExternal(&ValuePanic{long, ValueInt, ErrNeedIntDefault})
				} else {
					crntResult.Value = val
				}
			} else {
				crntResult.Value = 0
			}
		case ValueString:
			if options.Default != nil {
				if val, ok := options.Default.(string); !ok {
					panicExternal(&ValuePanic{long, ValueString, ErrNeedStringDefault})
				} else {
					crntResult.Value = val
				}
			} else {
				crntResult.Value = ""
			}
		case ValueSize:
			if options.Default != nil {
				if val, ok := options.Default.(Size); !ok {
					panicExternal(&ValuePanic{long, ValueSize, ErrNeedSizeDefault})
				} else {
					crntResult.Value = val
				}
			} else {
				crntResult.Value = Size(0)
			}
		}
	}
	crntResults[long] = crntResult
	deDupConflicts := make(map[string]struct{})
	for _, conflict := range options.ConflictsWith {
		deDupConflicts[conflict] = struct{}{}
	}
	crntFlag := &flag{
		options.ValueRequired,
		options.ValueType,
		options.CanBeMultiple,
		deDupConflicts,
		long,
		short,
		options.Default,
	}
	crntRegis.long[long] = crntFlag
	if short != 0 {
		crntRegis.short[short] = crntFlag
	}
}

// DeleteAll deletes all registered flags and
// their corresponding [Result].
func DeleteAll() {
	crntRegis.long = longRegis{}
	crntRegis.short = shortRegis{}
	crntResults = Results{}
}

// Delete deletes a registered flag by its long
// name and its corresponding [Result]
func Delete(long string) {
	flag, ok := crntRegis.long[long]
	if !ok {
		panicExternal(&FlagPanic{Long: long, Problem: Other, Err: ErrDeleteNotRegis})
	}
	if flag.short != 0 {
		delete(crntRegis.short, flag.short)
	}
	for _, flag2 := range crntRegis.long {
		for conflict := range flag2.conflictsWith {
			if conflict == long {
				delete(flag2.conflictsWith, long)
			}
		}
	}
	delete(crntRegis.long, long)
	delete(crntResults, long)
}
