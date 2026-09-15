package minifier

import (
	"regexp"
	"strings"
)

var (
	cssCommentRegex    = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	cssWhitespaceRegex = regexp.MustCompile(`\s+`)
	cssSymbolSpace     = regexp.MustCompile(`\s*([{}:;,>+~])\s*`)
)

// MinifyCSS removes CSS comments and redundant whitespace around selectors and declarations.
func MinifyCSS(input string) string {
	// 1. Remove comments
	cleaned := cssCommentRegex.ReplaceAllString(input, "")

	// 2. Normalize whitespace to single space
	cleaned = cssWhitespaceRegex.ReplaceAllString(cleaned, " ")

	// 3. Remove spaces around operators and braces: a { color: red ; } => a{color:red;}
	cleaned = cssSymbolSpace.ReplaceAllString(cleaned, "$1")

	// 4. Remove trailing semicolons before closing braces: ;} => }
	cleaned = strings.ReplaceAll(cleaned, ";}", "}")

	return strings.TrimSpace(cleaned)
}
