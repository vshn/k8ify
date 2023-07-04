package util

import (
	"os"
	"strings"
)

func GetEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		env[pair[0]] = pair[1]
	}
	return env
}

func GetEnvValueCaseInsensitive(caseInsensitiveKey string) string {
	for k, v := range GetEnv() {
		if strings.EqualFold(k, caseInsensitiveKey) {
			return v
		}
	}
	return ""
}
