package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vshn/k8ify/pkg/util"
)

func TestIsTruthy(t *testing.T) {
	assert := assert.New(t)
	assert.True(util.IsTruthy("true"))
	assert.True(util.IsTruthy("True"))
	assert.True(util.IsTruthy("TRUE"))
	assert.True(util.IsTruthy("yes"))
	assert.True(util.IsTruthy("YES"))
	assert.True(util.IsTruthy("1"))

	assert.False(util.IsTruthy("false"))
	assert.False(util.IsTruthy("False"))
	assert.False(util.IsTruthy("FALSE"))
	assert.False(util.IsTruthy("no"))
	assert.False(util.IsTruthy("NO"))
	assert.False(util.IsTruthy("0"))
}
