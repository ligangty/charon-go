package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBlankString(t *testing.T) {
	assert.True(t, IsBlankString(""))
	assert.True(t, IsBlankString(" "))
	assert.False(t, IsBlankString("bob"))
	assert.False(t, IsBlankString("  bob  "))
}
