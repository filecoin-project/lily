package util

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/runes"
)

// SanitizeLabel ensures:
// - s is a valid utf8 string by removing any ill formed bytes.
// - s does not contain any nil (\x00) bytes because postgres doesn't support storing NULL (\0x00) characters in text fields.
func SanitizeLabel(s string) string {
	if s == "" {
		return s
	}
	s = strings.Replace(s, "\000", "", -1)
	if utf8.ValidString(s) {
		return s
	}

	tr := runes.ReplaceIllFormed()
	return tr.String(s)
}
