package util

import (
	"os"
	"strings"
)

func GetEnv(prefix string) map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], prefix) {
			// put the value into the map, both with and without prefix
			env[pair[0]] = pair[1]
			varNameWithoutPrefix := pair[0][len(prefix):len(pair[0])]
			env[varNameWithoutPrefix] = pair[1]
		}
	}
	return env
}
