package minifier

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	htmlCommentRegex    = regexp.MustCompile(`<!--[\s\S]*?-->`)
	htmlTagSpaceRegex   = regexp.MustCompile(`>\s+<`)
	htmlWhitespaceRegex = regexp.MustCompile(`\s+`)
	scriptBlockRegex    = regexp.MustCompile(`(?i)(<script[^>]*>)([\s\S]*?)(</script>)`)
	styleBlockRegex     = regexp.MustCompile(`(?i)(<style[^>]*>)([\s\S]*?)(</style>)`)
)

// MinifyHTML removes HTML comments and collapses redundant whitespace without breaking <script> or <style>.
func MinifyHTML(input string) string {
	// 1. Remove comments
	cleaned := htmlCommentRegex.ReplaceAllString(input, "")

	// 2. Protect and minify <script> and <style> blocks
	type block struct {
		placeholder string
		content     string
	}
	var blocks []block

	cleaned = scriptBlockRegex.ReplaceAllStringFunc(cleaned, func(match string) string {
		parts := scriptBlockRegex.FindStringSubmatch(match)
		if len(parts) == 4 {
			openTag, jsCode, closeTag := parts[1], parts[2], parts[3]
			minifiedJS := MinifyJS(jsCode)
			placeholder := fmt.Sprintf("___LIAF_SCRIPT_%d___", len(blocks))
			blocks = append(blocks, block{
				placeholder: placeholder,
				content:     openTag + "\n" + minifiedJS + "\n" + closeTag,
			})
			return placeholder
		}
		return match
	})

	cleaned = styleBlockRegex.ReplaceAllStringFunc(cleaned, func(match string) string {
		parts := styleBlockRegex.FindStringSubmatch(match)
		if len(parts) == 4 {
			openTag, cssCode, closeTag := parts[1], parts[2], parts[3]
			minifiedCSS := MinifyCSS(cssCode)
			placeholder := fmt.Sprintf("___LIAF_STYLE_%d___", len(blocks))
			blocks = append(blocks, block{
				placeholder: placeholder,
				content:     openTag + minifiedCSS + closeTag,
			})
			return placeholder
		}
		return match
	})

	// 3. Collapse whitespace between tags: >   < => ><
	cleaned = htmlTagSpaceRegex.ReplaceAllString(cleaned, "><")

	// 4. Collapse multiple spaces into single space inside remaining HTML tags
	cleaned = htmlWhitespaceRegex.ReplaceAllString(cleaned, " ")

	// 5. Restore <script> and <style> blocks
	for _, b := range blocks {
		cleaned = strings.Replace(cleaned, b.placeholder, b.content, 1)
	}

	return strings.TrimSpace(cleaned)
}
