package minifier

import (
	"regexp"
	"strings"
)

var (
	htmlCommentRegex    = regexp.MustCompile(`<!--[\s\S]*?-->`)
	htmlWhitespaceRegex = regexp.MustCompile(`\s+`)
	htmlTagSpaceRegex   = regexp.MustCompile(`>\s+<`)
)

// MinifyHTML removes HTML comments and collapses redundant whitespace between tags.
func MinifyHTML(input string) string {
	// 1. Remove comments
	cleaned := htmlCommentRegex.ReplaceAllString(input, "")

	// 2. Collapse whitespace between tags: >   < => ><
	cleaned = htmlTagSpaceRegex.ReplaceAllString(cleaned, "><")

	// 3. Collapse multiple spaces into single space inside tags
	cleaned = htmlWhitespaceRegex.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}
