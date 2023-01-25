package util

import (
	"regexp"
	"strconv"
	"strings"
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
