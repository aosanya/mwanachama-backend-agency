package mwanachamaagency

import "context"

type returnedByIDKey struct{}

func WithReturnedByID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, returnedByIDKey{}, id)
}

func returnedByIDOf(ctx context.Context) string {
	id, _ := ctx.Value(returnedByIDKey{}).(string)
	return id
}
