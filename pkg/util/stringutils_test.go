package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vshn/k8ify/pkg/util"
)

func TestSanitize(t *testing.T) {
	assert.Equal(t, "foo", util.Sanitize("foo"))
	assert.Equal(t, "feat-foo1", util.Sanitize("feat/foo1"))
	assert.Equal(t, "test", util.Sanitize("1test"))
	assert.Equal(t, "feat-foo1", util.Sanitize("/feat/foo1"))
}

func TestSanitiziWithMinLength(t *testing.T) {
	assert.Equal(t, "foobar", util.SanitizeWithMinLength("foobar", 6))
	assert.Equal(t, "phpllk", util.SanitizeWithMinLength("foo", 6))
}

func TestByteArrayToAlpha(t *testing.T) {
	assert.Equal(t, "aaabacad", util.ByteArrayToAlpha([]byte{0, 1, 2, 3}))
}
