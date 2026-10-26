package sflag

import (
	"errors"
	"fmt"
)

func panicExternal(v error) {
	panic(v)
}

func panicInternal(v any) {
	panic(fmt.Sprintf("sflag: internal bug: %v. please report this at https://codeberg.org/TrollgeEngineering/sflag/issues (sflag %s)", v, version))
}

type problem uint8

const (
	// LongOnly is passed in Problem in a [FlagPanic] when
	// the issue is specific to the long name of
	// a flag.
	LongOnly problem = iota
	// ShortOnly is passed in Problem in a [FlagPanic] when
	// the issue is specific to the short name
	// of a flag.
	ShortOnly
	// Conflict is passed in Problem in a [FlagPanic] when
	// the issue pertains to a conflicting flag registered
	// on a flag.
	Conflict
	// Other in passed in Problem in a [FlagPanic] when
	// the issue does not fit into the other categories.
	Other
)

// FlagPanic is passed to panic() due to a caller error
// involving flags.
type FlagPanic struct {
	// Long is the long name of a flag involved
	// in the panic.
	Long string
	// Short is the short name of a flag involved
	// in the panic.
	Short byte
	// Conflict is the long name of a conflicting flag
	// involved in the panic.
	Conflict string
	// Problem is an enum representing te problem that
	// caused the panic.
	Problem problem
	// Err is the specific error behind the panic.
	Err error
}

func (e *FlagPanic) Error() string {
	switch e.Problem {
	case LongOnly:
		return fmt.Sprintf("sflag: long flag %q: %s", e.Long, e.Err)
	case ShortOnly:
		return fmt.Sprintf("sflag: short flag %q: %s", e.Short, e.Err)
	case Other:
		return fmt.Sprintf("sflag: flag %q: %s", e.Long, e.Err)
	case Conflict:
		return fmt.Sprintf("sflag: flag %q: conflicting flag %q: %s", e.Long, e.Conflict, e.Err)
	default:
		return fmt.Sprintf("sflag: %s. invalid Problem %d. please report this at https://codeberg.org/TrollgeEngineering/sflag/issues (sflag %s)", e.Err, e.Problem, version)
	}
	return "" // unreachable
}

func (e *FlagPanic) Unwrap() error {
	return e.Err
}

// ValuePanic is passed to panic() due to a caller
// error involving flag values.
type ValuePanic struct {
	Long    string
	ValType valueType
	Err     error
}

func (e *ValuePanic) Error() string {
	return fmt.Sprintf("sflag: flag %q: value type %s: %s", e.Long, e.ValType, e.Err)
}

func (e *ValuePanic) Unwrap() error {
	return e.Err
}

var (
	// ErrRegisDupe is wrapped in an error and passed to panic()
	// when the caller attempts to register a flag name that
	// has already been registered.
	ErrRegisDupe = errors.New("flag already registered")
	// ErrShortIllegalChar is wrapped in an error and passed to
	// panic() when the caller attempts to register a short
	// flag name that is not an alphanumeric ASCII character.
	ErrShortIllegalChar = errors.New("short flag names must be a single alphanumeric ASCII character")
	// ErrLongIllegalChar is wrapped in an error and passed to
	// panic() when the caller attempts to register a long
	// flag name that contains at least one character that
	// is not an alphanumeric ASCII character or a hyphen.
	ErrLongIllegalChar = errors.New("long flag names must only contain alphanumeric ASCII characters and hyphens")
	// ErrLeadTrailHyphen is wrapped in an error and passed
	// to panic() when the caller attempts to register
	// a long flag name that begins or ends with a hyphen.
	ErrLeadTrailHyphen = errors.New("long flag names must not contains a leading or trailing hyphen")
	// ErrShortConflict is wrapped in an error and passed
	// to panic() when the caller attempts to register
	// a conflicting flag using a name that is less than
	// two characters.
	ErrShortConflict = errors.New("conflicting flags must be registered with their long name")
	// ErrLongTooShort is wrapped in an error and passed
	// to panic() when the caller attempts to register
	// a long flag name that is less than two characters.
	ErrLongTooShort = errors.New("long flag names must be at least 2 characters")
	// ErrConflictNotRegis is wrapped in an error and
	// passed to panic() when the caller attempts to
	// call [Parse] and at least one of the registered
	// flags names a conflicting flag that was not
	// registered.
	ErrConflictNotRegis = errors.New("conflicting flag was not registered")
	// ErrSelfConflict is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag with a conflicting flag long name
	// that matches the long name of the flag currently
	// being registered.
	ErrSelfConflict = errors.New("flags cannot conflict with themselves")
	// ErrBoolCantBeMultiple is wrapped in an error
	// and passed to panic() when the caller attempts to
	// register a flag with both CanBeMultiple set to
	// true and ValueType set to [ValueBool] in the
	// passed [Options].
	ErrBoolCantBeMultiple = errors.New("flags of this value type cannot be passed multiple times. use ValueBoolCount instead")
	// ErrNeedIntSliceDefault is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag that has a ValueType of [ValueInt]
	// with CanBeMultiple set to false in [Options]
	// and the Default field contains a value that
	// is not an int and is not nil.
	ErrNeedIntDefault = errors.New("flags of this value type with CanBeMultiple set to false need a default value of nil or type int")
	// ErrNeedIntSliceDefault is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag that has a ValueType of [ValueString]
	// with CanBeMultiple set to false in [Options]
	// and the Default field contains a value that
	// is not an string and is not nil.
	ErrNeedStringDefault = errors.New("flags of this value type with CanBeMultiple set to false need a default value of nil or type string")
	// ErrNeedIntSliceDefault is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag that has a ValueType of [ValueSize]
	// with CanBeMultiple set to false in [Options]
	// and the Default field contains a value that
	// is not a [Size] and is not nil.
	ErrNeedSizeDefault = errors.New("flags of this value type with CanBeMultiple set to false need a default value of nil or type sflag.Size")
	// ErrNeedBoolDefault is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag that has a ValueType of [ValueBool]
	// in [Options] and the Default field contains a
	// value that is not a bool and is not nil.
	ErrNeedBoolDefault = errors.New("flags of this value type need a default value of nil or type bool")
	// ErrNeedIntSliceDefault is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag that has a ValueType of [ValueInt]
	// with CanBeMultiple set to true in [Options]
	// and the Default field contains a value that
	// is not an []int and is not nil.
	ErrNeedIntSliceDefault = errors.New("flags of this value type with CanBeMultiple set to true need a default value of nil or type []int")
	// ErrNeedStringSliceDefault is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag that has a ValueType of [ValueString]
	// with CanBeMultiple set to true in [Options]
	// and the Default field contains a value that
	// is not an []string and is not nil.
	ErrNeedStringSliceDefault = errors.New("flags of this value type with CanBeMultiple set to true need a default value of nil or type []string")
	// ErrNeedSizeSliceDefault is wrapped in an error and
	// passed to panic() when the caller attempts to
	// register a flag that has a ValueType of [ValueSize]
	// with CanBeMultiple set to true in [Options]
	// and the Default field contains a value that
	// is not a [][Size] and is not nil.
	ErrNeedSizeSliceDefault = errors.New("flags of this value type with CanBeMultiple set to true need a default value of nil or type []sflag.Size")
	// ErrBoolCountNoTakeValue is wrapped in an error
	// and passed to panic() when the caller attempts to
	// register a flag that has a ValueType of
	// [ValueBoolCount] and ValueRequired set to true
	// in [Options].
	ErrBoolCountNoTakeValue = errors.New("flags of this value type cannot take values. use ValueBool instead")
	// ErrInvalidValueType is wrapped in an error and
	// passed to panic() when the caller attmepts to
	// register a flag with a ValueType that is not
	// recognized.
	ErrInvalidValueType = errors.New("unrecognized value type")
	// ErrDeleteNotRegis is wrapped in an error and
	// passed to panic() when the caller attempts to
	// call [Delete] on a long flag name that was not
	// registered.
	ErrDeleteNotRegis = errors.New("flag cannot be deleted because it is not registered")
	// ErrNeedRegis is wrapped in an error and passed to
	// panic() when the caller attempts to call [Parse]
	// before at least one flag has been registered.
	ErrNeedRegis = errors.New("at least one flag must be registered before parsing")
)
