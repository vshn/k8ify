package util

import (
	"github.com/sirupsen/logrus"
	"os"
)

func AppendMap[KEY comparable, VALUE any](first map[KEY]VALUE, second map[KEY]VALUE) map[KEY]VALUE {
	for key, value := range second {
		if _, ok := first[key]; ok {
			logrus.Errorf("%v of second (value %v) would overwrite the value of first (%v)", key, value, first[key])
			os.Exit(1)
		}
		first[key] = value
	}
	return first
}

func FilterNilErrors(list []error) []error {
	var result []error
	for _, item := range list {
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}
