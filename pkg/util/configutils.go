package util

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/go-units"
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
	return GetOptional(labels, "k8ify.storage-class")
}

func StorageSizeRaw(labels map[string]string) *string {
	return GetOptional(labels, "k8ify.size")
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
		log.Fatalf("ERROR: Invalid storage size: %q\n", quantity)
	}

	return *resource.NewQuantity(size, resource.BinarySI)
}
