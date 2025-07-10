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

func TestServiceMonitorConfig(t *testing.T) {
	assert := assert.New(t)
	type LabelMap map[string]string

	cases := []TestCase[LabelMap, *util.ServiceMonitorConfig]{
		{
			name:     "ServiceMonitorConfig_nothing_set",
			input:    LabelMap{},
			expected: &util.ServiceMonitorConfig{},
		},
		{
			name:  "ServiceMonitorConfig_enabled",
			input: LabelMap{"k8ify.prometheus.serviceMonitor": "true"},
			expected: &util.ServiceMonitorConfig{
				Enabled: true,
			},
		},
		{
			name:  "ServiceMonitorConfig_disabled",
			input: LabelMap{"k8ify.prometheus.serviceMonitor": "false"},
			expected: &util.ServiceMonitorConfig{
				Enabled: false,
			},
		},
		{
			name: "ServiceMonitorConfig_values_set",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor":               "true",
				"k8ify.prometheus.serviceMonitor.interval":      monitorInterval,
				"k8ify.prometheus.serviceMonitor.path":          monitorPath,
				"k8ify.prometheus.serviceMonitor.scheme":        monitorScheme,
				"k8ify.prometheus.serviceMonitor.endpoint.name": monitorEndpointName,
			},
			expected: &util.ServiceMonitorConfig{
				Enabled:      true,
				Interval:     &monitorInterval,
				Path:         &monitorPath,
				Scheme:       &monitorScheme,
				EndpointName: &monitorEndpointName,
			},
		},
		{
			name: "ServiceMonitorConfig_empty_strings",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor":               "",
				"k8ify.prometheus.serviceMonitor.interval":      "",
				"k8ify.prometheus.serviceMonitor.path":          "",
				"k8ify.prometheus.serviceMonitor.scheme":        "",
				"k8ify.prometheus.serviceMonitor.endpoint.name": "",
			},
			expected: &util.ServiceMonitorConfig{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := util.ServiceMonitorConfigPointer(tc.input)

			assert.Equal(tc.expected, actual, "ServiceMonitorConfigPointer(%v) should return %v", tc.input, tc.expected)
		})
	}
}

type TestCase[InParam any, OutParam any] struct {
	name     string
	input    InParam
	expected OutParam
}

var (
	monitorInterval     = "30s"
	monitorPath         = "/actuator/health"
	monitorScheme       = "http"
	monitorEndpointName = "default"
)
