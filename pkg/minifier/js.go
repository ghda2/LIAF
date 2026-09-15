package minifier

import (
	"regexp"
	"strings"
)

var (
	jsBlockComment = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	jsLineComment  = regexp.MustCompile(`(?m)^\s*//.*$`)
	jsInlineSpaces = regexp.MustCompile(`[ \t]+`)
	jsSymbolSpace  = regexp.MustCompile(`\s*([=+\-*/%&|!<>?:;{},()\[\]])\s*`)
)

// MinifyJS strips comments and collapses whitespace in JavaScript.
func MinifyJS(input string) string {
	// 1. Remove block comments
	cleaned := jsBlockComment.ReplaceAllString(input, "")

	// 2. Remove entire line comments
	cleaned = jsLineComment.ReplaceAllString(cleaned, "")

	// 3. Normalize horizontal spaces
	cleaned = jsInlineSpaces.ReplaceAllString(cleaned, " ")

	// 4. Remove spaces around common symbols
	cleaned = jsSymbolSpace.ReplaceAllString(cleaned, "$1")

	// 5. Clean up redundant empty lines
	lines := strings.Split(cleaned, "\n")
	var validLines []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			validLines = append(validLines, trimmed)
		}
	}

	return strings.Join(validLines, "\n")
}
