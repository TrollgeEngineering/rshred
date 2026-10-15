package sflag

// Results is returned by [Parse] once parsing of all sflag
// completes without error. The keys of Results are the long
// names of every flag registered with [Register] and/or [RegisterWithShort].
type Results map[string]*Result

var crntResults = make(Results)

// Result is an entry in the [Results] returned by [Parse]. Each
// Result corresponds to one registered flag, matched by
// the name of the long version of the flag in [Results].
type Result struct {
	// Value is the value(s) passed to a flag. If CanBeMultiple
	// is set to true in the [Options] at the time of
	// registration, Value will be set to a slice
	// underlying type of the ValueType
	// in the [Options] of the flag.
	Value any
	// Seen is set to true if the flag associated with the
	// Result was seen at least once during parsing.
	Seen bool
	// ValueSeen is set to true if a valid value was
	// passed at least to the flag associated with the
	// Result.
	ValueSeen bool
}
