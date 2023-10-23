package namelock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Lock(t *testing.T) {
	nl := NewNameLock()
	assert.True(t, nl.TryLock("foo"))
	assert.False(t, nl.TryLock("foo"))
}
