package util

import "fmt"

func AppendMap[KEY comparable, VALUE any](first map[KEY]VALUE, second map[KEY]VALUE) map[KEY]VALUE {
	for key, value := range second {
		if _, ok := first[key]; ok {
			panic(fmt.Sprintf("%v of second (value %v) would overwrite the value of first (%v)", key, value, first[key]))
		}
		first[key] = value
	}
	return first
}
