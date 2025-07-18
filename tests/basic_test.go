package tests

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBasicTest(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		assert.True(t, true)
		assert.Equal(t, 1, 1)
	})
}