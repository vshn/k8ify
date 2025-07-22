package ir

import (
	"errors"
	assertions "github.com/stretchr/testify/assert"
	"testing"
)

func TestServiceMonitorConfig(t *testing.T) {
	assert := assertions.New(t)
	type LabelMap map[string]string

	cases := []TestCase[LabelMap, *ServiceMonitorConfig]{
		{
			name:          "ServiceMonitorConfig_nothing_set",
			input:         LabelMap{},
			expectedValue: nil,
		},
		{
			name:          "ServiceMonitorConfig_enabled",
			input:         LabelMap{"k8ify.prometheus.serviceMonitor": "true"},
			expectedValue: &ServiceMonitorConfig{},
		},
		{
			name:          "ServiceMonitorConfig_disabled",
			input:         LabelMap{"k8ify.prometheus.serviceMonitor": "false"},
			expectedValue: nil,
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
			expectedValue: &ServiceMonitorConfig{
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
			expectedValue: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ServiceMonitorConfigPointer(tc.input)

			assert.Equal(tc.expectedValue, actual, "ServiceMonitorConfigPointer(%v) should return %v", tc.input, tc.expectedValue)
		})
	}
}

func TestServiceMonitorBasicAuthConfig(t *testing.T) {
	assert := assertions.New(t)
	type LabelMap map[string]string

	cases := []TestCase[LabelMap, *ServiceMonitorBasicAuthConfig]{
		{
			name:          "BasicAuthConfig_nothing_set",
			input:         LabelMap{},
			expectedValue: nil,
			expectedError: nil,
		},
		{
			name:          "BasicAuthConfig_enabled_wrongly_configured",
			input:         LabelMap{"k8ify.prometheus.serviceMonitor.endpoint.basicAuth": "true"},
			expectedValue: nil,
			expectedError: errors.New("username or password is blank, this is not allowed. username had length 0, password had length 0"),
		},
		{
			name: "BasicAuthConfig_enabled_only_username_set",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth":          "true",
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth.username": "user",
			},
			expectedValue: nil,
			expectedError: errors.New("username or password is blank, this is not allowed. username had length 4, password had length 0"),
		},
		{
			name:          "BasicAuthConfig_disabled",
			input:         LabelMap{"k8ify.prometheus.serviceMonitor.endpoint.basicAuth": "false"},
			expectedValue: nil,
		},
		{
			name: "BasicAuthConfig_values_set",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth":          "true",
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth.username": monitorUsername,
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth.password": monitorPassword,
			},
			expectedValue: &ServiceMonitorBasicAuthConfig{
				Username: monitorUsername,
				Password: monitorPassword,
			},
		},
		{
			name: "BasicAuthConfig_empty_strings",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth":          "",
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth.username": "",
				"k8ify.prometheus.serviceMonitor.endpoint.basicAuth.password": "",
			},
			expectedValue: nil,
			expectedError: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ServiceMonitorBasicAuthConfigPointer(tc.input)

			assert.Equal(tc.expectedValue, actual, "BasicAuthConfigPointer(%v) should return value %v", tc.input, tc.expectedValue)
			assert.Equal(tc.expectedError, err, "BasicAuthConfigPointer(%v) should return err %v", tc.input, tc.expectedError)
		})
	}
}

type TestCase[InParam any, OutParam any] struct {
	name          string
	input         InParam
	expectedValue OutParam
	expectedError error
}

var (
	monitorEndpointName = "default"
	monitorInterval     = "30s"
	monitorPassword     = "mypassword"
	monitorPath         = "/actuator/health"
	monitorScheme       = "http"
	monitorUsername     = "myuser"
)
