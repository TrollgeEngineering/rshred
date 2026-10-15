package sflag

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Size is the type stored in the Value
// field of a [Result] when the corresponding
// flag has a ValueType of [ValueSize].
type Size uint64

func checkValue(value string, flgEntry *flag) error {
	if value == "" {
		return ErrNeedValue
	}
	switch flgEntry.valType {
	case ValueBool:
		switch value {
		case "true":
			crntResults[flgEntry.formal].Value = true
		case "false":
			crntResults[flgEntry.formal].Value = false
		default:
			return ErrNeedBool
		}
	case ValueInt:
		flagNum, err := strconv.Atoi(value)
		if err != nil {
			if errors.Is(err, strconv.ErrRange) {
				return ErrTooLarge
			} else {
				return ErrNeedInt
			}
		}
		if flgEntry.canBeMultiple {
			crntResults[flgEntry.formal].Value = append(crntResults[flgEntry.formal].Value.([]int), flagNum)
		} else {
			crntResults[flgEntry.formal].Value = flagNum
		}
		return nil
	case ValueString:
		if flgEntry.canBeMultiple {
			crntResults[flgEntry.formal].Value = append(crntResults[flgEntry.formal].Value.([]string), value)
		} else {
			crntResults[flgEntry.formal].Value = value
		}
		return nil
	case ValueSize:
		bytes, err := parseSize(value)
		if err != nil {
			return err
		}
		if flgEntry.canBeMultiple {
			crntResults[flgEntry.formal].Value = append(crntResults[flgEntry.formal].Value.([]Size), bytes)
		} else {
			crntResults[flgEntry.formal].Value = bytes
		}
	default:
		panicExternal(fmt.Sprintf("invalid flag value type %v, but this should have been caught at registration", flgEntry.valType))
	}
	return nil // Shouldn't be reached.
}

func parseSize(input string) (Size, error) {
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
				if input[i] >= '0' && input[i] <= '9' {
					val = append(val, input[i])
					if !(input[i+1] >= '0' && input[i+1] <= '9') {
						allNums = true
					}
				} else {
					return 0, ErrNoNum
				}
			} else {
				baseNum, err := strconv.Atoi(string(val))
				if errors.Is(err, strconv.ErrRange) {
					return 0, ErrTooLarge
				}
				switch string(input[i]) {
				case "B":
					return Size(baseNum), nil
				case "K":
					if baseNum > math.MaxUint64>>10 {
						return 0, ErrTooLarge
					}
					return Size(baseNum) << 10, nil
				case "M":
					if baseNum > math.MaxUint64>>20 {
						return 0, ErrTooLarge
					}
					return Size(baseNum) << 20, nil
				case "G":
					if baseNum > math.MaxUint64>>30 {
						return 0, ErrTooLarge
					}
					return Size(baseNum) << 30, nil
				case "T":
					if baseNum > math.MaxUint64>>40 {
						return 0, ErrTooLarge
					}
					return Size(baseNum) << 40, nil
				case "P":
					if baseNum > math.MaxUint64>>50 {
						return 0, ErrTooLarge
					}
					return Size(baseNum) << 50, nil
				default:
					return 0, ErrBadSuffix
				}
			}
		}
		panicInternal("uncaught exception while parsing size")
	} else {
		return Size(bytes), nil
	}
	return 0, nil // unreachable
}

func markSeen(flag *flag) error {
	if _, ok := crntResults[flag.formal]; !ok {
		panicInternal(fmt.Sprintf("flag %q not found in crntResults", flag.formal))
	}
	if flag.canBeMultiple || !crntResults[flag.formal].Seen || flag.valType == ValueBoolCount {
		if flag.valType == ValueBoolCount {
			crntTimes, ok := crntResults[flag.formal].Value.(int)
			if !ok {
				panicInternal(fmt.Sprintf("could not update value for flag %q. assertion as int failed.", flag.formal))
			} else {
				crntResults[flag.formal].Value = crntTimes + 1
			}
		}
		crntResults[flag.formal].Seen = true
		return nil
	} else {
		return ErrDupe
	}
}
