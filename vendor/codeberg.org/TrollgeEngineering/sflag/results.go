package sflag

import "fmt"

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
		crntResults[flg.formal].Seen = false
		crntResults[flg.formal].ValueSeen = false
		if flg.valType >= barrier {
			panicInternal(fmt.Sprintf("flag %q: invalid value type %d, but this should have been a panic at registration", flg.formal, flg.valType))
		}
		if flg.valType == ValueBoolCount {
			if flg.valueRequired {
				panicInternal(fmt.Sprintf("flag %q: flags cannot be both ValueBoolCount and have ValueRequired, but this should have been a panic at registration", flg.formal))
			} else {
				crntResults[flg.formal].Value = 0
			}
		}
		if flg.canBeMultiple {
			switch flg.valType {
			case ValueBool:
				panicInternal(fmt.Sprintf("flag %q: flags of ValueBool cannot have CanBeMultiple, but this should have been a panic at registration", flg.formal))
			case ValueInt:
				if flg.defaultVal != nil {
					if val, ok := flg.defaultVal.([]int); !ok {
						panicInternal(fmt.Sprintf("flag %q: default value must be []int, but this should have been a panic at registration", flg.formal))
					} else {
						crntResults[flg.formal].Value = val
					}
				} else {
					crntResults[flg.formal].Value = []int{}
				}
			case ValueString:
				if flg.defaultVal != nil {
					if val, ok := flg.defaultVal.([]string); !ok {
						panicInternal(fmt.Sprintf("flag %q: default value must be []string, but this should have been a panic at registration", flg.formal))
					} else {
						crntResults[flg.formal].Value = val
					}
				} else {
					crntResults[flg.formal].Value = []string{}
				}
			case ValueSize:
				if flg.defaultVal != nil {
					if val, ok := flg.defaultVal.([]Size); !ok {
						panicInternal(fmt.Sprintf("flag %q: default value must be []sflag.Size but this should have been a panic at registration", flg.formal))
					} else {
						crntResults[flg.formal].Value = val
					}
				} else {
					crntResults[flg.formal].Value = []Size{}
				}
			}
		} else {
			switch flg.valType {
			case ValueBool:
				if flg.defaultVal != nil {
					if val, ok := flg.defaultVal.(bool); !ok {
						panicInternal(fmt.Sprintf("flag %q: default value must be bool, but this should have been a panic at registration", flg.formal))
					} else {
						crntResults[flg.formal].Value = val
					}
				} else {
					crntResults[flg.formal].Value = false
				}
			case ValueInt:
				if flg.defaultVal != nil {
					if val, ok := flg.defaultVal.(int); !ok {
						panicInternal(fmt.Sprintf("flag %q: default value must be int, but this should have been a panic at registration", flg.formal))
					} else {
						crntResults[flg.formal].Value = val
					}
				} else {
					crntResults[flg.formal].Value = 0
				}
			case ValueString:
				if flg.defaultVal != nil {
					if val, ok := flg.defaultVal.(string); !ok {
						panicInternal(fmt.Sprintf("flag %q: default value must be string, but this should have been a panic at registration", flg.formal))
					} else {
						crntResults[flg.formal].Value = val
					}
				} else {
					crntResults[flg.formal].Value = ""
				}
			case ValueSize:
				if flg.defaultVal != nil {
					if val, ok := flg.defaultVal.(Size); !ok {
						panicInternal(fmt.Sprintf("flag %q: default value must be sflag.Size, but this should have been a panic at registration", flg.formal))
					} else {
						crntResults[flg.formal].Value = val
					}
				} else {
					crntResults[flg.formal].Value = Size(0)
				}
			}
		}
	}
}
