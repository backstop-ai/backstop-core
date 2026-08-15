// Package main DELIBERATELY does not compile.
//
// It is safe to commit because the `go` tool IGNORES any directory named testdata, so it
// never enters this repository's own build or `go build ./...`.
//
// THE BREAK IS A RETURN-TYPE MISMATCH RATHER THAN A BAD `var` INITIALIZER, and that is
// deliberate: the go-standards no-global-mutable-state rule matches `var $X = ...`
// anywhere, not only at package level, so a `var` form fires it on this fixture under a
// full sweep. The compiler's diagnostic is what this fixture is for, so it must break in a
// way no rule reads as a real defect.
package main

func main() {
	println(notAnInt())
}

func notAnInt() int {
	return "this is not an int"
}
