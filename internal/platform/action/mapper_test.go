package action_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/platform/action"
)

type (
	testStructA struct{}
	testStructB struct{}
)

func TestMapper(t *testing.T) {
	t.Run("when no named action found return an error", func(t *testing.T) {
		mapper := &action.Mapper{}

		doer, err := mapper.Get("ComplexCalculation")

		require.ErrorContains(t, err, "no action found for: ComplexCalculation")
		require.Nil(t, doer)
	})

	t.Run("returns the saved function for a name", func(t *testing.T) {
		mapper := &action.Mapper{}
		mapper.Add("ComplexCalculation", func(a testStructA, b testStructB) error { return nil })

		doer, err := mapper.Get("ComplexCalculation")
		require.NoError(t, err)

		do, ok := doer.(func(testStructA, testStructB) error)
		require.True(t, ok)
		require.NotNil(t, do)

		require.NoError(t, do(testStructA{}, testStructB{}))
	})

	t.Run("All returns the name of each stored action", func(t *testing.T) {
		mapper := &action.Mapper{}
		require.Empty(t, mapper.All(), "expected a just initialized mapper to have nothing to show")

		mapper.Add("ComplexFunction", func() {})
		mapper.Add("SimpleFunction", func() {})
		require.ElementsMatch(
			t,
			[]string{"ComplexFunction", "SimpleFunction"},
			mapper.All(),
			"expected the names of the stored actions to be returned",
		)
	})
}
