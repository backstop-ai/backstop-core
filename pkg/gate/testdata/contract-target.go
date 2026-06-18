package testdata

import "context"

// DoSomething is a function that matches the declared contract signature.
func DoSomething(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return nil
}

// Widget is a type that matches the declared contract.
type Widget struct {
	Name string
}

// Runner is an interface that matches the declared contract.
type Runner interface {
	Run(ctx context.Context) error
}

// DefaultTimeout is a variable that matches the declared contract.
var DefaultTimeout int

// Mode is a defined type over a primitive underlying type, used to verify the
// contract checker renders the underlying type (so "type Mode string" matches).
type Mode string

// BindingTable is a defined map type, used to verify the contract checker
// renders a map underlying type ("type BindingTable map[string]Widget").
type BindingTable map[string]Widget
