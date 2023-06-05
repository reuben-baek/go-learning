package testify

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAssert(t *testing.T) {
	t.Run("assert func", func(t *testing.T) {
		expected := 1
		actual := 1
		assert.Equal(t, expected, actual)
	})
	t.Run("assertions", func(t *testing.T) {
		assertion := assert.New(t)

		expected := 1
		actual := 1
		assertion.Equal(expected, actual)
	})
}

func TestRequire(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var result bool
	result = t.Run("require func", func(t *testing.T) {
		expected := 1
		actual := 1

		assert.NotEqual(t, expected, actual)
		require.NotEqual(t, expected, actual)
		panic("ShouldNotReach")
	})
	assert.False(t, result)
	result = t.Run("require assertions", func(t *testing.T) {
		assertion := require.New(t)

		expected := 1
		actual := 1
		assertion.NotEqual(expected, actual)
		panic("ShouldNotReach")
	})
	assert.False(t, result)
}
