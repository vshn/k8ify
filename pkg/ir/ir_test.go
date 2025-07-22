package ir

import (
	assertions "github.com/stretchr/testify/assert"
	"testing"
)

func TestServiceMonitorConfig(t *testing.T) {
	assert := assertions.New(t)
	type LabelMap map[string]string

	cases := []TestCase[LabelMap, *ServiceMonitorConfig]{
		{
			name:     "ServiceMonitorConfig_nothing_set",
			input:    LabelMap{},
			expected: &ServiceMonitorConfig{},
		},
		{
			name:  "ServiceMonitorConfig_enabled",
			input: LabelMap{"k8ify.prometheus.serviceMonitor": "true"},
			expected: &ServiceMonitorConfig{
				Enabled: true,
			},
		},
		{
			name:  "ServiceMonitorConfig_disabled",
			input: LabelMap{"k8ify.prometheus.serviceMonitor": "false"},
			expected: &ServiceMonitorConfig{
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
			expected: &ServiceMonitorConfig{
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
			expected: &ServiceMonitorConfig{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ServiceMonitorConfigPointer(tc.input)

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
