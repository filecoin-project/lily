package market

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeLabel(t *testing.T) {
	t.Run("valid utf8 string with null character", func(t *testing.T) {
		stringWithNull := "golang\000"
		goodString := SanitizeLabel(stringWithNull)
		assert.False(t, strings.Contains(goodString, "\000"))
		assert.True(t, utf8.ValidString(goodString))
	})

	t.Run("invalid utf8 string", func(t *testing.T) {
		invalidUtf8String := string([]byte("🌸")[:2])
		// sanity check
		assert.False(t, utf8.ValidString(invalidUtf8String))

		goodString := SanitizeLabel(invalidUtf8String)
		assert.False(t, strings.Contains(goodString, "\000"))
		assert.True(t, utf8.ValidString(goodString))
	})

	t.Run("invalid utf8 string with null character", func(t *testing.T) {
		invalidUtf8String := string([]byte("🌸")[:2])
		invalidUtf8String += "\000"
		// sanity check
		assert.False(t, utf8.ValidString(invalidUtf8String))

		goodString := SanitizeLabel(invalidUtf8String)
		assert.False(t, strings.Contains(goodString, "\000"))
		assert.True(t, utf8.ValidString(goodString))
	})

}
