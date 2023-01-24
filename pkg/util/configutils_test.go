package util_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vshn/k8ify/pkg/util"
)

func TestSubConfigEmpty(t *testing.T) {
	actual := util.SubConfig(
		map[string]string{},
		"myapp", "root")
	expected := map[string]string{}
	assert.Equal(t, expected, actual)
}

func TestSubConfigRoot(t *testing.T) {
	actual := util.SubConfig(
		map[string]string{"myapp": "value"},
		"myapp", "root")
	expected := map[string]string{"root": "value"}
	assert.Equal(t, expected, actual)
}

func TestSubConfigRootAnd(t *testing.T) {
	util.SubConfig(
		map[string]string{
			"myapp":      "value",
			"myapp.root": "other",
		},
		"myapp", "root")

	// This is undefined behavior
}

func TestSubConfig(t *testing.T) {
	actual := util.SubConfig(
		map[string]string{
			"myapp.one":   "foo",
			"myapp.two":   "bar",
			"myapp.three": "baz",
		},
		"myapp", "root")
	expected := map[string]string{
		"one":   "foo",
		"two":   "bar",
		"three": "baz",
	}
	assert.Equal(t, expected, actual)
}

func TestConfigGetInt32(t *testing.T) {
	config := map[string]string{
		"num": "1",
		"str": "foo",
	}

	assert.Equal(t, int32(1), util.ConfigGetInt32(config, "num", 99))
	assert.Equal(t, int32(88), util.ConfigGetInt32(config, "str", 88))
	assert.Equal(t, int32(77), util.ConfigGetInt32(config, "zzz", 77))
}

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
