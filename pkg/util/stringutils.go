package util

import (
	"crypto/sha512"
	"math/big"
	"regexp"
	"strings"
)

var (
	zero  = new(big.Int).SetInt64(0)
	num26 = new(big.Int).SetInt64(26)
	num36 = new(big.Int).SetInt64(36)
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
	if minLength > 100 {
		// The hash fall-back never produces anything longer than 100 chars so minLength above 100 does not work
		minLength = 100
	}
	sanitized := Sanitize(str)
	if len(sanitized) >= minLength {
		return sanitized
	}
	// The input string does not have enough useful characters, so instead we hash the string
	sha := sha512.New()
	sha.Write([]byte(str))
	alpha := ByteArrayToAlphaNum(sha.Sum(nil))
	return alpha[0:minLength]
}

func ByteArrayToAlphaNum(bytes []byte) string {
	if len(bytes) == 0 {
		return ""
	}

	// We want to guarantee that the resulting string length is always the same for a given input length, no matter what
	// the input value.
	// Consider this: If the input is e.g. 8 bytes long but the decimal value is only e.g. 5, this is the difference
	// between producing "f" and "faaaaaaaaaaaa". If you were to feed 8 times the byte value 255 into this function you
	// would get a 13 character long output, that's why the single "f" is padded with 12 "a"s.
	// For this purpose we find the maximum value that would be possible with the given input length and keep adding
	// characters to the output until the maximum value drops to 0.
	maxValueBytes := make([]byte, len(bytes))
	for i := 0; i < len(maxValueBytes); i++ {
		maxValueBytes[i] = 255
	}
	maxValue := new(big.Int).SetBytes(maxValueBytes)

	value := new(big.Int)
	value.SetBytes(bytes)

	remainder := new(big.Int)
	str := ""

	// first iteration uses divisor 26 in order to produce a-z
	remainder.Mod(value, num26)
	str = str + string(byte(remainder.Uint64())+97)
	value.Div(value, num26)
	maxValue.Div(maxValue, num26)

	// subsequent iterations use divisor 36 in order to produce a-z, 0-9
	// As outlined above we abort when the *maximum possible input value* for the given input length reaches 0, not
	// when the *actual* value reaches 0
	for maxValue.Cmp(zero) > 0 {
		remainder.Mod(value, num36)
		if remainder.Cmp(num26) < 0 {
			str = str + string(byte(remainder.Uint64())+97)
		} else {
			str = str + string(byte(remainder.Uint64())+22)
		}
		value.Div(value, num36)
		maxValue.Div(maxValue, num36)
	}

	return str
}

func IsBlank(s *string) bool {
	if s == nil {
		return true
	}
	trimmed := strings.Trim(*s, " \t")
	return trimmed == ""
}
