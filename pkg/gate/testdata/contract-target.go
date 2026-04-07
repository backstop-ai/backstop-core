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
