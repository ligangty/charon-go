package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveFromStringSlice(t *testing.T) {
	s := []string{"abc", "efg", "hi", "j"}
	r := RemoveFromStringSlice(s, "abc")
	assert.Equal(t, 3, len(r))
	assert.Contains(t, r, "efg")
	assert.Contains(t, r, "hi")
	assert.Contains(t, r, "j")
	assert.NotContains(t, r, "abc")
}
