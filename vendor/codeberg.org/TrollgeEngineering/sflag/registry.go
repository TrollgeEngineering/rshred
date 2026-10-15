package sflag

import (
	"fmt"
	"slices"
	"strings"
)

type valueType uint8

// This block of constants lists the types of values
// that can be registered in the ValueType field of [Options]
// when a flag is registered with either [Register] or
// [RegisterWithShort].
const (
	// ValueBool is a ValueType that can be set in [Options]
	// that makes the flag able to accept values of either
	// "true" or "false" during parsing. The Value field
	// in the [Result] of a flag with ValueType ValueBool
	// will have a concrete type of bool.
	ValueBool valueType = iota
	// ValueBoolCount is a ValueType that makes the Value
	// field in the [Result] of the corresponding flag
	// an int representing how many times the flag was seen.
	// Flags of type ValueBoolCount can never take values.
	ValueBoolCount
	// ValueInt is a ValueType that is able to accept a value of
	// any numeric ASCII string that can be stored as an int.
	// The concrete type as stored in the Value field of the
	// [Result] for this flag is int.
	ValueInt
	// ValueString is a ValueType that is able to accept
	// a value of a string of any characters. The concrete
	// type as stored in the varlue field of the [Result]
	// for this flag is string.
	ValueString
	// ValueSize is a ValueType that is able to accept a value
	// that represents a data size. Valid data sizes can either
	// be purely numeric ASCII strings or numeric ASCII strings
	// followed by either B (bytes), K (kibibytes), M (mebibytes),
	// G (gibibytes), T (tebibytes), or P (pebibytes) without spaces
	// between the numeric portion and the suffix. The numeric
	// portion of the value must be able to fit in an int. After
	// parsing, a [Size] representing the parsed size in bytes,
	// multiplied by powers of 1024 according to the suffix
	// will be stored in the Value field of the [Result]
	// corresponding to the registered flag. If no suffix
	// is included in the value passed to the flag, the string
	// of numbers will be converted to a [Size] as a number of bytes
	// without being multiplied by a power of 1024.
	ValueSize
	barrier
)

type flag struct {
	valueRequired bool
	valType       valueType
	canBeMultiple bool
	conflictsWith map[string]int
	formal        string
	short         byte
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
	// is set to [ValueBoolCount].
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
}

func isAlphaNumASCII(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func validateLongName(long string) error {
	if len(long) < 2 {
		return fmt.Errorf("flag %q: long flags must be at least 2 characters long", long)
	}
	if strings.HasPrefix(long, "-") || strings.HasSuffix(long, "-") {
		return fmt.Errorf("flag %q: long flag names must not have leading or trailing hyphens", long)
	}
	for i := 0; i < len(long); i++ {
		if !isAlphaNumASCII(long[i]) && long[i] != '-' {
			return fmt.Errorf("flag %q: long flag names must contain only alphanumeric ASCII characters and hyphens", long)
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
	if err := validateLongName(long); err != nil {
		panicExternal(err)
	}
	if _, ok := crntRegis.long[long]; ok {
		panicExternal(fmt.Sprintf("%s can only be registered once", long))
	}
	if slices.Contains(options.ConflictsWith, long) {
		panicExternal(fmt.Sprintf("%s cannot conflict with itself", long))
	}
	for _, conflict := range options.ConflictsWith {
		if len(conflict) < 2 {
			panicExternal(fmt.Sprintf("conflicting flag %q: conflicting flags must be registered with their long name", conflict))
		}
	}
	if options.ValueType >= barrier {
		panicExternal(fmt.Sprintf("flag %q: unreconized value type %v", long, options.ValueType))
	}
	crntResults[long] = &Result{}
	if options.ValueType == ValueBoolCount {
		if options.ValueRequired {
			panicExternal(fmt.Sprintf("flag %q of type ValueBoolCount cannot take values", long))
		} else {
			crntResults[long].Value = 0
		}
	}
	if options.CanBeMultiple {
		switch options.ValueType {
		case ValueBool:
			panicExternal(fmt.Sprintf("flag %q of type ValueBool cannot be used mutliple times. use ValueBoolCount instead", long))
		case ValueInt:
			crntResults[long].Value = []int{}
		case ValueSize:
			crntResults[long].Value = []Size{}
		case ValueString:
			crntResults[long].Value = []string{}
		default:
			panicExternal(fmt.Sprintf("flag %q: unrecognized type %v", long, options.ValueType))
		}
	}
	conflicts := make(map[string]int)
	for _, conflict := range options.ConflictsWith {
		conflicts[conflict] = 0
	}
	crntRegis.long[long] = &flag{
		options.ValueRequired,
		options.ValueType,
		options.CanBeMultiple,
		conflicts,
		long,
		0,
	}
}

// RegisterWithShort registers a flag with a long name and
// a corresponding short name using the configuration options
// supplied in the [Options]. RegisterWithShort will panic
// for the same reasons as [Register], with the addition of
// panicking when the short name supplied has already been
// registered and if the short name is not an alphanumeric
// ASCII character.
func RegisterWithShort(long string, short byte, options Options) {
	if err := validateLongName(long); err != nil {
		panicExternal(err)
	}
	if !isAlphaNumASCII(short) {
		panicExternal(fmt.Sprintf("short flag %q must be one alphanumeric ASCII character", short))
	}
	if _, ok := crntRegis.long[long]; ok {
		panicExternal(fmt.Sprintf("long flag %q can only be registered once", long))
	}
	if _, ok := crntRegis.short[short]; ok {
		panicExternal(fmt.Sprintf("short flag %q can only be registered once", short))
	}
	if slices.Contains(options.ConflictsWith, long) {
		panicExternal(fmt.Sprintf("long flag %q cannot conflict with itself", long))
	}
	for _, conflict := range options.ConflictsWith {
		if len(conflict) < 2 {
			panicExternal(fmt.Sprintf("conflicting flag %q: conflicting flags must be registered with their long name", conflict))
		}
	}
	if options.ValueType >= barrier {
		panicExternal(fmt.Sprintf("flag %q: unreconized value type %v", long, options.ValueType))
	}
	crntResults[long] = &Result{}
	if options.ValueType == ValueBoolCount {
		if options.ValueRequired {
			panicExternal(fmt.Sprintf("flag %q of type ValueBoolCount cannot take values", long))
		} else {
			crntResults[long].Value = 0
		}
	}
	if options.CanBeMultiple {
		switch options.ValueType {
		case ValueBool:
			panicExternal(fmt.Sprintf("flag %q of type ValueBool cannot be used mutliple times. use ValueBoolCount instead", long))
		case ValueInt:
			crntResults[long].Value = []int{}
		case ValueSize:
			crntResults[long].Value = []Size{}
		case ValueString:
			crntResults[long].Value = []string{}
		default:
			panicExternal(fmt.Sprintf("flag %q: unrecognized type %v", long, options.ValueType))
		}
	}
	deDupConflicts := make(map[string]int)
	for _, conflict := range options.ConflictsWith {
		deDupConflicts[conflict] = 0
	}
	flag := &flag{
		options.ValueRequired,
		options.ValueType,
		options.CanBeMultiple,
		deDupConflicts,
		long,
		short,
	}
	crntRegis.short[short] = flag
	crntRegis.long[long] = flag
}

// ResetResults will reset the each [Result] in
// the [Results] to their default values. All
// the bools in the [Result] will be set to false
// and all the Value fields in each [Result] will
// either be set to an empty slice of the underlying
// type of the associated flag's ValueType if
// CanBeMultiple is set to true, or nil if it is
// set to false. If the ValueType of the corresponding
// flag is [ValueBoolCount], the Value of the [Result]
// will be set to 0.
func ResetResults() {
	for _, flg := range crntRegis.long {
		switch {
		case flg.valType == ValueBoolCount:
			crntResults[flg.formal].Value = 0
		case flg.canBeMultiple:
			switch flg.valType {
			case ValueBool:
				panicInternal(fmt.Sprintf("flag %q was as ValueBoolCount and CanBeMultiple, but this should have been blocked at registration", flg.formal))
			case ValueInt:
				crntResults[flg.formal].Value = []int{}
			case ValueSize:
				crntResults[flg.formal].Value = []Size{}
			case ValueString:
				crntResults[flg.formal].Value = []string{}
			default:
				panicInternal(fmt.Sprintf("FlagValueType of flag %q was set to unknown value %v, but this should have been blocked at registration", flg.formal, flg.valType))
			}
		default:
			crntResults[flg.formal].Value = nil
		}
		crntResults[flg.formal].Seen = false
		crntResults[flg.formal].ValueSeen = false
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
		panicExternal(fmt.Sprintf("flag %q cannot be deleted because it is not registered", long))
	}
	if flag.short != 0 {
		delete(crntRegis.short, flag.short)
	}
	delete(crntRegis.long, long)
	delete(crntResults, long)
}
