// Package main is the trivial buildable target the marker commands compile.
//
// Without it every `go build -o <name>.marker ./...` fails, NO marker is ever written,
// and the five absence assertions pass for the wrong reason while the two presence
// assertions red. The module is what makes the matrix falsifiable in both directions.
package main

func main() {}
