package validate_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/platform/validate"
)

type testStruct struct {
	Hello string `validate:"required"`
}

func TestStruct(t *testing.T) {
	ctx := context.Background()

	require.Error(t, validate.Struct(ctx, testStruct{}), "expected an error for an empty object")
	require.NoError(t, validate.Struct(ctx, testStruct{"Hello"}), "when the struct is valid don't error")
}
