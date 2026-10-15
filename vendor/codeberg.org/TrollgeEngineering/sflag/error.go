package sflag

import (
	"errors"
	"fmt"
)

// FlagError is an error returned by [Parse] when there is
// an error releating to a specific flag.
type FlagError struct {
	Arg  string
	Flag string
	Err  error
}

// ArgError is an error returnd by [Parse] when there is
// an error releating to an argument as a whole.
type ArgError struct {
	Arg string
	Err error
}

// ValueError is an error returned by [Parse] when there
// is an error releating to the value passed to a flag.
type ValueError struct {
	Value string
	Flag  string
	Err   error
}

// ConflictError is an error returned by [Parse] when two
// flags in the input conflict with each other.
type ConflictError struct {
	Flag1 string
	Flag2 string
	Arg1  string
	Arg2  string
	Err   error
}

func (e *FlagError) Error() string {
	if e.Arg != "" {
		return fmt.Sprintf("argument %q: flag %s: %v", e.Arg, e.Flag, e.Err)
	} else {
		return fmt.Sprintf("flag %s: %v", e.Flag, e.Err)
	}
}

func (e *ArgError) Error() string {
	return fmt.Sprintf("argument %q: %v", e.Arg, e.Err)
}

func (e *ValueError) Error() string {
	return fmt.Sprintf("flag %s: invalid value %q: %v", e.Flag, e.Value, e.Err)
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("flag %q and flag %q contradict each other", e.Flag1, e.Flag2)
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

func (e *ConflictError) Unwrap() error {
	return e.Err
}

var (
	// ErrNoTakeValueTryHyphens is wrapped in an error returned by [Parse] when a non-flag
	// and non-value argument is passed, followed by an argument that
	// looks like a flag and the last true flag was one that cannot
	// take a value.
	ErrNoTakeValueTryHyphens = errors.New("flag cannot take a value. all arguments must come before paths. use \"--\" before paths beginning with \"-\"")
	// ErrMalformedValueTryHyphens is wrapped in an error returned by [Parse] when a
	// non-flag and non-value argument is passed, followed by an
	// argument that looks like a flag, and the last true flag
	// can optionally take a value.
	ErrMalformedValueTryHyphens = errors.New("optional values must be passed to long flags only and with an \"=\". all arguments must come before paths. use \"--\" before paths beginning with \"-\"")
	// ErrUnknownMayBeMalformed is wrapped in an error returned by [Parse] when a
	// character in a glued short flag argument is not recognized
	// as a valid short flag, but the character before it is a
	// flag that optionally takes a value.
	ErrUnknownMayBeMalformed = errors.New("unknown flag. optional values may only be passed using \"=\" to long flags only")
)

var (
	// ErrUnknown is wrapped in an error returned by [Parse] when a flag in the input was
	// not registered with [Register] or [RegisterWithShort].
	ErrUnknown = errors.New("flag not recognized")
	// ErrFlagAfterNonFlag is wrapped in an error returned by [Parse] when an argument
	// that looks like a flag is passed after an argument that is
	// not a flag, flag group, or consumed value when "--" is not
	// passed to signify the end of flags.
	ErrFlagAfterNonFlag = errors.New("all arguments must come before paths. use \"--\" before paths beginning with \"-\"")
	// ErrLongGlued is wrapped in an error returned by [Parse] when long flags (e.g. "--verbose"
	// or "--exact") are passed together in the same argument, or if
	// a value for a long flag is passed to it in the same argument
	// without being seperated by "=".
	ErrLongGlued = errors.New("long flags cannot be passed together in the same argument and cannot take values in the same argument without being seperated by \"=\"")
	// ErrNoTakeValue is wrapped in an error returned by [Parse] when a value is passed to a flag
	// that cannot take one, such as a flag of type [ValueBoolCount].
	ErrNoTakeValue = errors.New("flag cannot take a value")
	// ErrNeedValue is wrapped in an error returned by [Parse] when no value is passed to a flag
	// that requires one when [ValueRequired] is set to true.
	ErrNeedValue = errors.New("flag requires a value")
	// ErrDupe is wrapped in an error returned by [Parse] when the same flag is passed twice and
	// [CanBeMultiple] is set to false.
	ErrDupe = errors.New("flags can only be used once")
	// ErrShortTwoHyphen is wrapped in an error returned by [Parse] when a short flag (e.g. "-x", "-s")
	// is invoked using two hyphens instead of one (e.g. "--x", "--s").
	ErrShortTwoHyphen = errors.New("single character flags must be invoked using a single \"-\"")
	// ErrLongOneHyphen is wrapped in an error returned by [Parse] when a long flag (e.g. "--verbose", "--passes")
	// is invoked using one hyphen instead of two (e.g. "-passes", "-verbose").
	ErrLongOneHyphen = errors.New("long flags must be invoked using two hyphens")
	// ErrMalformedOptionalValue is wrapped in an error returned by [Parse] when an optional value is passed to
	// a short flag or is passed to a long flag without "=".
	ErrMalformedOptionalValue = errors.New("to pass an optional value to a flag, you must use the long form of the flag and connect the value with an \"=\"")
	// ErrContradict is wrapped in an error returned by [Parse] when flags that contradict each other as set with
	// [ConflictsWith] are passed.
	ErrContradict = errors.New("these two flags contradict")
	// ErrShortEquals is wrapped in an error returned by [Parse] when a short flag (e.g. "-n") is passed a
	// value using "=" (e.g. "-n=5").
	ErrShortEquals = errors.New("\"=\" cannot be used to pass values to short flags")
)

var (
	// ErrNeedInt is wrapped in an error returned by [Parse] when a flag registered with value
	// type [ValueInt] is passed a value that is not an integer.
	ErrNeedInt = errors.New("flag needs an integer value")
	// ErrNeedBool is wrapped in an error returned by [Parse] when a flag registered with [ValueType]
	// [ValueBool] is passed a value that is not either "true" or "false".
	// Not to be confused with [ErrNoNum]
	ErrNeedBool = errors.New("flag needs a value of either \"true\" or \"false\"")
	// ErrTooLarge is wrapped in an error returned by [Parse] when a numeric value part of a value
	// passed to a flag is too far above or below 0.
	ErrTooLarge = errors.New("value is too large")
	// ErrNegative is wrapped in an error returned by [Parse] when a value passed to a
	// flag with [ValueType] [ValueSize] contains a negative number.
	ErrNegative = errors.New("value cannot be negative")
	// ErrNoNum is wrapped in an error returned by [Parse] when a value passed to a flag
	// of [ValueType] [ValueSize] does not begin with an integer value.
	// Not to be confused with [ErrNeedInt].
	ErrNoNum = errors.New("value must begin with a number")
	// ErrBadSuffix is wrapped in an error returned by [Parse] when a value passed to a flag
	// of [ValueType] [ValueSize] has an invalid size suffix.
	ErrBadSuffix = errors.New("invalid suffix. suffixes must be uppercase")
)
