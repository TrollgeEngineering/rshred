package sflag

import "fmt"

func panicExternal(v any) {
	panic(fmt.Sprintf("sflag: %v", v))
}

func panicInternal(v any) {
	panic(fmt.Sprintf("sflag: internal bug: %v. please report this at https://codeberg.org/TrollgeEngineering/sflag/issues", v))
}
