package mwanachamaagency

import (
	"context"
	"errors"
)

var ErrFunctionNotFound = errors.New("mwanachamaagency: function not found")

type FunctionInput struct {
	Name string
	Type string
}

type FunctionCatalog interface {
	InputsOf(ctx context.Context, name string, version int) ([]FunctionInput, bool, error)
}

func WithFunctionCatalog(catalog FunctionCatalog) Option {
	return func(m *agencyManager) { m.functions = catalog }
}
