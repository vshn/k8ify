package util

import (
	"regexp"
	"strings"
)

func Sanitize(str string) string {
	str = strings.ToLower(str)
	// replace all non-alphanumeric characters by "-"
	str = regexp.MustCompile("[^a-z0-9-]").ReplaceAllString(str, "-")
	// replace multiple subsequent "-" by a single "-"
	str = regexp.MustCompile("-+").ReplaceAllString(str, "-")
	// remove "-" and numbers at beginning, "-" at end of string
	str = regexp.MustCompile("^[0-9-]*([^-]*)-?$").ReplaceAllString(str, "${1}")
	return str
}
