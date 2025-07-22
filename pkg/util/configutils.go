package util

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"strconv"
	"strings"

	core "k8s.io/api/core/v1"

	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	reTrue = regexp.MustCompile("(?i)^true|yes|1$")
)

// SubConfig extracts the keys that start with a given prefix from a given
// config map
//
// If there is a key that is EQUAL to the prefix, the entry identified by
// `defaultKey` will be updated.
//
// If there is a key that is equal to the prefix, AND an entry corresponding to
// `defaultKey`, the behavior is undefined!
func SubConfig(config map[string]string, prefix string, defaultKey string) map[string]string {
	subConfig := make(map[string]string)
	for key, value := range config {
		if key == prefix {
			subConfig[defaultKey] = value
		}
		if strings.HasPrefix(key, prefix+".") && len(key) > (len(prefix)+1) {
			subKey := key[len(prefix)+1:]
			subConfig[subKey] = value
		}
	}
	return subConfig
}

// ConfigGetInt32 extracts the int32 value from the entry with the given key
//
// Returns `defaultValue` if either the entry does not exist, or is not a
// numeric value.
func ConfigGetInt32(config map[string]string, key string, defaultValue int32) int32 {
	if valStr, ok := config[key]; ok {
		if valInt, err := strconv.Atoi(valStr); err == nil {
			return int32(valInt)
		}
	}
	return defaultValue
}

// IsTruthy determines whether the given string is a representation of a "true"
// state.
//
// Concretly, it currently tests for "true", "yes" or "1", ignoring character
// cases.
func IsTruthy(s string) bool {
	return reTrue.MatchString(s)
}

func GetBoolean(labels map[string]string, key string) bool {
	if val, ok := labels[key]; ok {
		return IsTruthy(val)
	}

	return false
}

func GetOptional(labels map[string]string, key string) *string {
	if val, ok := labels[key]; ok {
		return &val
	}

	return nil
}

// IsSingleton determine whether a resource (according to its labels) should be
// treated as a singleton.
func IsSingleton(labels map[string]string) bool {
	return GetBoolean(labels, "k8ify.singleton")
}

// IsShared determines whether a volume is shared between replicas
func IsShared(labels map[string]string) bool {
	return GetBoolean(labels, "k8ify.shared")
}

// StorageClass determines a storage class from a set of labels
func StorageClass(labels map[string]string) *string {
	return GetOptional(labels, "k8ify.storageClass")
}

func StorageSizeRaw(labels map[string]string) *string {
	return GetOptional(labels, "k8ify.size")
}

func Converter(labels map[string]string) *string {
	return GetOptional(labels, "k8ify.converter")
}

func PartOf(labels map[string]string) *string {
	return GetOptional(labels, "k8ify.partOf")
}

func ImagePullSecret(labels map[string]string) *string {
	return GetOptional(labels, "k8ify.imagePullSecret")
}

// StorageSize determines the requested storage size for a volume, or a
// fallback value.
func StorageSize(labels map[string]string, fallback string) resource.Quantity {
	quantity := fallback
	if q := StorageSizeRaw(labels); q != nil {
		quantity = *q
	}

	size, err := units.RAMInBytes(quantity)
	if err != nil {
		logrus.Errorf("Invalid storage size: %q\n", quantity)
		os.Exit(1)
	}

	return *resource.NewQuantity(size, resource.BinarySI)
}

func ServiceAccountName(labels map[string]string) string {
	serviceAccountName := GetOptional(labels, "k8ify.serviceAccountName")
	if serviceAccountName == nil {
		return ""
	}
	return *serviceAccountName
}

func Annotations(labels map[string]string, kind string) map[string]string {
	annotations := SubConfig(labels, "k8ify.annotations", "")
	maps.Copy(annotations, SubConfig(labels, fmt.Sprintf("k8ify.%s.annotations", kind), ""))
	delete(annotations, "")
	return annotations
}

func ServiceType(labels map[string]string, port int32) core.ServiceType {
	subConfig := SubConfig(labels, fmt.Sprintf("k8ify.exposePlain.%d", port), "")
	if len(subConfig) == 0 {
		return ""
	}
	if serviceType, ok := subConfig["type"]; ok {
		// Go does not offer a way to list values of its "ENUMs", see https://github.com/golang/go/issues/19814
		if serviceType == string(core.ServiceTypeClusterIP) {
			return core.ServiceTypeClusterIP
		}
		if serviceType == string(core.ServiceTypeLoadBalancer) {
			return core.ServiceTypeLoadBalancer
		}
		if serviceType == string(core.ServiceTypeExternalName) {
			return core.ServiceTypeExternalName
		}
		if serviceType == string(core.ServiceTypeNodePort) {
			return core.ServiceTypeNodePort
		}
	}
	return core.ServiceTypeLoadBalancer
}

func ServiceExternalTrafficPolicy(labels map[string]string, port int32) core.ServiceExternalTrafficPolicy {
	subConfig := SubConfig(labels, fmt.Sprintf("k8ify.exposePlain.%d", port), "")
	if len(subConfig) == 0 {
		return ""
	}
	if serviceType, ok := subConfig["externalTrafficPolicy"]; ok {
		// Go does not offer a way to list values of its "ENUMs", see https://github.com/golang/go/issues/19814
		if serviceType == string(core.ServiceExternalTrafficPolicyCluster) {
			return core.ServiceExternalTrafficPolicyCluster
		}
		if serviceType == string(core.ServiceExternalTrafficPolicyLocal) {
			return core.ServiceExternalTrafficPolicyLocal
		}
	}
	return core.ServiceExternalTrafficPolicyLocal
}

func ServiceHealthCheckNodePort(labels map[string]string, port int32) int32 {
	subConfig := SubConfig(labels, fmt.Sprintf("k8ify.exposePlain.%d", port), "")
	if len(subConfig) == 0 {
		return 0
	}
	healthCheckNodePort := ConfigGetInt32(subConfig, "healthCheckNodePort", 0)
	if healthCheckNodePort > 65535 || healthCheckNodePort < 0 {
		return 0
	}
	return healthCheckNodePort
}
