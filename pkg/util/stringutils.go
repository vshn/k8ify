package util

import (
	"crypto/sha512"
	"regexp"
	"strings"
)

// Sanitize ensures a string only contains alphanumeric characters, and starts
// with a letter
func Sanitize(str string) string {
	str = strings.ToLower(str)
	// replace all non-alphanumeric characters by "-"
	str = regexp.MustCompile("[^a-z0-9-]").ReplaceAllString(str, "-")
	// replace multiple subsequent "-" by a single "-"
	str = regexp.MustCompile("-+").ReplaceAllString(str, "-")
	// remove "-" and numbers at beginning, "-" at end of string
	str = regexp.MustCompile("^[0-9-]+|-+$").ReplaceAllString(str, "")
	return str
}

// SanitizeWithMinLength applies `Sanitize`, and then ensures the result is at least `minLength` characters long.
//
// If the resulting string is too short, a stable generated value of
// `minLength` characters will be returned.
func SanitizeWithMinLength(str string, minLength int) string {
	if minLength > 128 {
		// The hash fall-back never produces anything longer than 128 chars so minLength above 128 does not work
		minLength = 128
	}
	sanitized := Sanitize(str)
	if len(sanitized) >= minLength {
		return sanitized
	}
	// The input string does not have enough useful characters, so instead we hash the string
	sha := sha512.New()
	sha.Write([]byte(str))
	alpha := ByteArrayToAlpha(sha.Sum(nil))
	return alpha[0:minLength]
}

// This is like hex encoding but instead of 0-f we use a-p
func ByteArrayToAlpha(bytes []byte) string {
	str := ""
	for i := 0; i < len(bytes); i++ {
		str = str + string(bytes[i]/16+97) + string(bytes[i]%16+97)
	}
	return str
}
