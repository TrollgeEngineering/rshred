// Package sflag parses command-line flags against a registry the caller supplies.
// sflag does not have any preregistered or built-in flags such as --help, so they must be supplied by the developer.
//
// Flags are registered using the [Register] function. Optionally, a long flag may be paired
// with a short flag during registration using [RegisterWithShort].
//
// Flags may be registered with one of 5 value types, [ValueBool], [ValueBoolCount], [ValueInt], [ValueString], and [ValueSize].
package sflag

var version = "v0.2.1"
