package runtime

import (
	"encoding/json"

	"liaf/pkg/content"
	"liaf/pkg/markdown"
)

// ContentParse faz o parsing do post com frontmatter e devolve o JSON do Post.
func ContentParse(raw string) Result[string, string] {
	post, err := content.Parse(raw)
	if err != nil {
		return Err[string, string](err.Error())
	}
	bytes, err := json.Marshal(post)
	if err != nil {
		return Err[string, string]("failed to encode post to JSON: " + err.Error())
	}
	return Ok[string, string](string(bytes))
}

// ContentSlugify transforma um título em slug canônico limpo.
func ContentSlugify(title string) string {
	return content.Slugify(title)
}

// MarkdownToHTML converte Markdown em HTML higienizado.
func MarkdownToHTML(raw string) string {
	return markdown.RenderHTML(raw)
}
