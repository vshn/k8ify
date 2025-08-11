package ir

import (
	"errors"
	"testing"

	prometheusTypes "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	assertions "github.com/stretchr/testify/assert"
	"github.com/vshn/k8ify/pkg/util"
)

func TestServiceMonitorConfig(t *testing.T) {
	assert := assertions.New(t)
	type LabelMap map[string]string

	cases := []TestCase[LabelMap, *ServiceMonitorConfig, error]{
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

	cases := []TestCase[LabelMap, *ServiceMonitorBasicAuthConfig, error]{
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

func TestServiceMonitorTlsConfig(t *testing.T) {
	assert := assertions.New(t)
	type LabelMap map[string]string

	cases := []TestCase[LabelMap, *ServiceMonitorTlsConfig, *[]error]{
		{
			name:          "TlsConfig_nothing_set",
			input:         LabelMap{},
			expectedValue: nil,
			expectedError: nil,
		},
		{
			name:          "TlsConfig_enabled",
			input:         LabelMap{"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig": "true"},
			expectedValue: &ServiceMonitorTlsConfig{},
			expectedError: nil,
		},
		{
			name:          "TlsConfig_disabled",
			input:         LabelMap{"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig": "false"},
			expectedValue: nil,
		},
		{
			name: "TlsConfig_invalid_maxVersion",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig":            "true",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.maxVersion": "invalid",
			},
			expectedValue: nil,
			expectedError: util.GetPointer([]error{
				errors.New("unknown TLSVersion: invalid"),
			}),
		},
		{
			name: "TlsConfig_invalid_minVersion",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig":            "true",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.minVersion": "invalid",
			},
			expectedValue: nil,
			expectedError: util.GetPointer([]error{
				errors.New("unknown TLSVersion: invalid"),
			}),
		},

		{
			name: "TlsConfig_invalid_tls_versions",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig":            "true",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.maxVersion": "invalidMax",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.minVersion": "invalidMin",
			},
			expectedValue: nil,
			expectedError: util.GetPointer([]error{
				errors.New("unknown TLSVersion: invalidMax"),
				errors.New("unknown TLSVersion: invalidMin"),
			}),
		},
		{
			name: "TlsConfig_values_set",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig":                    "true",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.ca":                 monitorCa,
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.cert":               monitorCert,
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.keySecretValue":     monitorKeySecretValue,
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.insecureSkipVerify": "true",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.maxVersion":         monitorTlsMaxVersion,
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.minVersion":         monitorTlsMinVersion,
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.serverName":         monitorServerName,
			},
			expectedValue: &ServiceMonitorTlsConfig{
				Ca:                 &monitorCa,
				Cert:               &monitorCert,
				KeySecretValue:     &monitorKeySecretValue,
				InsecureSkipVerify: util.GetPointer(true),
				MaxVersion:         util.GetPointer(prometheusTypes.TLSVersion13),
				MinVersion:         util.GetPointer(prometheusTypes.TLSVersion10),
				ServerName:         &monitorServerName,
			},
		},
		{
			name: "TlsConfig_empty_strings",
			input: LabelMap{
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig":                    "",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.ca":                 "",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.cert":               "",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.keySecretValue":     "",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.insecureSkipVerify": "",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.maxVersion":         "",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.minVersion":         "",
				"k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.serverName":         "",
			},
			expectedValue: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, tlsConfigErrors := ServiceMonitorTlsConfigPointer(tc.input)

			assert.Equal(tc.expectedValue, actual, "ServiceMonitorTlsConfigPointer(%v) should return value %v", tc.input, tc.expectedValue)
			assert.Equal(tc.expectedError, tlsConfigErrors, "ServiceMonitorTlsConfigPointer(%v) should return error %v", tc.input, tc.expectedError)
		})
	}
}

type TestCase[InParam any, OutParam any, ErrorType any] struct {
	name          string
	input         InParam
	expectedValue OutParam
	expectedError ErrorType
}

var (
	monitorCa             = "-----BEGIN CERTIFICATE-----\nMIIBhTCCASugAwIBAgIUL8fmlL3Z1OSjE+9GHNrCuDGWKZgwCgYIKoZIzj0EAwIw\nGDEWMBQGA1UEAwwNTXkgTWluaW1hbCBDQTAeFw0yNTA3MTUxMzUyMTJaFw0yNjA3\nMTAxMzUyMTJaMBgxFjAUBgNVBAMMDU15IE1pbmltYWwgQ0EwWTATBgcqhkjOPQIB\nBggqhkjOPQMBBwNCAAQ6GrfF/1dVy3v97b+c6ZWRBAmdlBNV3qxfhdWS6KIwMvCr\nDiRUhXOpcLA49HjLX9RfDpxyI8Nz/Nv12bMg5f3go1MwUTAdBgNVHQ4EFgQU7Zcx\nnhcTn8t5cdCumGg7IKL39YwwHwYDVR0jBBgwFoAU7ZcxnhcTn8t5cdCumGg7IKL3\n9YwwDwYDVR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgNIADBFAiBeHxk5JKc1JpKF\nTZU6u6Yo4ozduWSQxIH6jSzh7BOCTAIhAMDNTO4ilY+DAna/udskuXMcjsfI0kQY\nU95t8zPBdnxh\n-----END CERTIFICATE-----\n"
	monitorCert           = "-----BEGIN CERTIFICATE-----\nMIIBhTCCASugAwIBAgIUAfxOSXWbSYYnhBSQGezqo+D6ia4wCgYIKoZIzj0EAwIw\nGDEWMBQGA1UEAwwNTXkgTWluaW1hbCBDQTAeFw0yNTA3MTUxMzU3MzBaFw0yNjA3\nMTAxMzU3MzBaMBgxFjAUBgNVBAMMDU15IE1pbmltYWwgQ0EwWTATBgcqhkjOPQIB\nBggqhkjOPQMBBwNCAATxEpwQy5oTno/HH+w9lUMsQxDZWFADZzt2xuI1Q33/TsBV\nKCwmZv3ywDwP1n2rHSoR7pZrQwUvNx/gyAobTPeDo1MwUTAdBgNVHQ4EFgQUjOmk\n2Q1r4qrwPIEjnWUlcyOAmTUwHwYDVR0jBBgwFoAUjOmk2Q1r4qrwPIEjnWUlcyOA\nmTUwDwYDVR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgNIADBFAiA7A3wxscDg/3rE\nqz7dR6899fxypP+nTwVw1M9SYgwmpAIhAO72sDxX86Y6Qikv1TCEQpO5t43clkoo\nekgUTlxCHY0H\n-----END CERTIFICATE-----\n"
	monitorEndpointName   = "default"
	monitorInterval       = "30s"
	monitorKeySecretValue = "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIHA0lQcWCa/He3w/MDaQS1c/YJte4mx3Bg9gzAr4P35BoAoGCCqGSM49\nAwEHoUQDQgAEc4ivr46eO4DOxArTOGP+5sxjFHDQpF02tRnuQBa9R433GDOSvdqb\nTEmIlxovk6eif+/2yLxFIsaA8aXaMbH+wQ==\n-----END EC PRIVATE KEY-----"
	monitorPassword       = "mypassword"
	monitorPath           = "/actuator/health"
	monitorScheme         = "http"
	monitorServerName     = "service.svc"
	monitorTlsMaxVersion  = string(prometheusTypes.TLSVersion13)
	monitorTlsMinVersion  = string(prometheusTypes.TLSVersion10)
	monitorUsername       = "myuser"
)
