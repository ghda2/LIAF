package content

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Diagnostic struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
	FixHint string `json:"fix_hint"`
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("[%s] %s (Dica de correção: %s)", d.Code, d.Message, d.FixHint)
}

var (
	validSlugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	imgRefRegex    = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
)

// Validate analisa o post e devolve diagnósticos com sugestões mecânicas de auto-cura.
func Validate(post *Post, baseDir string) []Diagnostic {
	var diags []Diagnostic

	// 1. Título obrigatório
	if strings.TrimSpace(post.Title) == "" {
		diags = append(diags, Diagnostic{
			Code:    "E_CONTENT_MISSING_TITLE",
			Field:   "title",
			Message: "o campo 'title' é obrigatório no frontmatter",
			FixHint: "adicione 'title: \"Meu Título\"' dentro do cabeçalho '---'",
		})
	}

	// 2. Slug válido
	if post.Slug == "" {
		diags = append(diags, Diagnostic{
			Code:    "E_CONTENT_MISSING_SLUG",
			Field:   "slug",
			Message: "o campo 'slug' está vazio e não pôde ser gerado a partir do título",
			FixHint: "adicione 'slug: \"meu-artigo\"' ou defina um título válido",
		})
	} else if !validSlugRegex.MatchString(post.Slug) {
		diags = append(diags, Diagnostic{
			Code:    "E_CONTENT_INVALID_SLUG",
			Field:   "slug",
			Message: fmt.Sprintf("slug '%s' inválido; deve conter apenas letras minúsculas, números e hífens", post.Slug),
			FixHint: fmt.Sprintf("use o formato canônico: '%s'", Slugify(post.Slug)),
		})
	}

	// 3. Imagem de capa (se declarada)
	if post.Image != "" && baseDir != "" && !isRemoteURL(post.Image) {
		imgPath := filepath.Join(baseDir, post.Image)
		if _, err := os.Stat(imgPath); err != nil {
			diags = append(diags, Diagnostic{
				Code:    "E_CONTENT_IMAGE_NOT_FOUND",
				Field:   "image",
				Message: fmt.Sprintf("imagem de capa '%s' referenciada no frontmatter não foi encontrada", post.Image),
				FixHint: fmt.Sprintf("certifique-se de que o arquivo '%s' existe na pasta ou ajuste o caminho", post.Image),
			})
		}
	}

	// 4. Imagens embutidas no corpo do Markdown
	if baseDir != "" {
		matches := imgRefRegex.FindAllStringSubmatch(post.ContentMarkdown, -1)
		for _, m := range matches {
			if len(m) == 3 {
				src := strings.TrimSpace(m[2])
				if !isRemoteURL(src) && !strings.HasPrefix(src, "data:") {
					imgPath := filepath.Join(baseDir, src)
					if _, err := os.Stat(imgPath); err != nil {
						diags = append(diags, Diagnostic{
							Code:    "E_CONTENT_EMBED_IMAGE_NOT_FOUND",
							Field:   "content_markdown",
							Message: fmt.Sprintf("imagem '%s' usada no corpo do texto não existe no disco", src),
							FixHint: fmt.Sprintf("verifique se o arquivo '%s' está presente na pasta", src),
						})
					}
				}
			}
		}
	}

	return diags
}

// ExtractLocalImages retorna a lista de imagens locais referenciadas no post (capa + corpo).
func ExtractLocalImages(post *Post) []string {
	var images []string
	seen := make(map[string]bool)

	if post.Image != "" && !isRemoteURL(post.Image) {
		images = append(images, post.Image)
		seen[post.Image] = true
	}

	matches := imgRefRegex.FindAllStringSubmatch(post.ContentMarkdown, -1)
	for _, m := range matches {
		if len(m) == 3 {
			src := strings.TrimSpace(m[2])
			if !isRemoteURL(src) && !strings.HasPrefix(src, "data:") && !seen[src] {
				images = append(images, src)
				seen[src] = true
			}
		}
	}
	return images
}

func isRemoteURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
