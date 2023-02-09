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
	assert.Equal(t, "leading", util.Sanitize("-leading"))
	assert.Equal(t, "leading", util.Sanitize("0-leading"))
	assert.Equal(t, "trailing", util.Sanitize("trailing-"))
	assert.Equal(t, "trailing", util.Sanitize("trailing--"))
	assert.Equal(t, "both", util.Sanitize("--both--"))
}

func TestSanitiziWithMinLength(t *testing.T) {
	assert.Equal(t, "foobar", util.SanitizeWithMinLength("foobar", 6))
	assert.Equal(t, "fr54k9", util.SanitizeWithMinLength("foo", 6))
}

func TestByteArrayToAlpha(t *testing.T) {
	assert.Equal(t, "", util.ByteArrayToAlphaNum([]byte{}))
	assert.Equal(t, "aa", util.ByteArrayToAlphaNum([]byte{0}))
	assert.Equal(t, "ba", util.ByteArrayToAlphaNum([]byte{1}))
	assert.Equal(t, "za", util.ByteArrayToAlphaNum([]byte{25}))
	assert.Equal(t, "ab", util.ByteArrayToAlphaNum([]byte{26}))
	assert.Equal(t, "zb", util.ByteArrayToAlphaNum([]byte{51}))
	assert.Equal(t, "ac", util.ByteArrayToAlphaNum([]byte{52}))
	assert.Equal(t, "bc", util.ByteArrayToAlphaNum([]byte{53}))
	assert.Equal(t, "a0aa", util.ByteArrayToAlphaNum([]byte{2, 164}))
	assert.Equal(t, "a9aa", util.ByteArrayToAlphaNum([]byte{3, 142}))
	assert.Equal(t, "z9aa", util.ByteArrayToAlphaNum([]byte{3, 167}))
	assert.Equal(t, "aaba", util.ByteArrayToAlphaNum([]byte{3, 168}))
	assert.Equal(t, "faaaaaaaaaaaa", util.ByteArrayToAlphaNum([]byte{0, 0, 0, 0, 0, 0, 0, 5}))
	assert.Equal(t, "aaabacad", util.ByteArrayToAlphaNum([]byte{39, 141, 108, 23, 160}))
	maxHashValue := make([]byte, 64)
	for i := 0; i < 64; i++ {
		maxHashValue[i] = 255
	}
	assert.Equal(t, "vjyw9istw1eo0gijxmt8ogd0ittw5jzslzuqb70zyf4aes4kc2j913cu0sjpfva7hytf3gk486srdn7wpyrtoesukpqzoilrknub", util.ByteArrayToAlphaNum(maxHashValue))
}
