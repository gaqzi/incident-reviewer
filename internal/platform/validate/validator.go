package validate

import (
	"context"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Struct is a thin wrapper around validator.Validate's StructCtx.
// This exists purely to ensure that we only have one validator cache.
func Struct(ctx context.Context, s any) error {
	return validate.StructCtx(ctx, s)
}
