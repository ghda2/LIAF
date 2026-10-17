package content

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"liaf/pkg/markdown"
)

// Post representa um artigo ou página de conteúdo processado.
type Post struct {
	Title           string   `json:"title"`
	Slug            string   `json:"slug"`
	Date            string   `json:"date"`
	Author          string   `json:"author"`
	Tags            []string `json:"tags"`
	Summary         string   `json:"summary"`
	Image           string   `json:"image"`
	ContentMarkdown string   `json:"content_markdown"`
	ContentHTML     string   `json:"content_html"`
	ReadingTimeMin  int      `json:"reading_time_min"`
	WordCount       int      `json:"word_count"`
}

var (
	frontmatterRegex = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---\r?\n(.*)$`)
	slugCleanRegex   = regexp.MustCompile(`[^a-z0-9\-_]+`)
)

// Parse processa o conteúdo cru em Markdown com Frontmatter YAML/LIAF.
func Parse(raw string) (*Post, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return nil, fmt.Errorf("E_POST_MISSING_FRONTMATTER: o arquivo deve iniciar com bloco de metadados delimitado por '---'")
	}

	matches := frontmatterRegex.FindStringSubmatch(trimmed)
	if len(matches) != 3 {
		return nil, fmt.Errorf("E_POST_INVALID_FRONTMATTER: bloco de frontmatter '---' não foi fechado corretamente")
	}

	metaLines := strings.Split(matches[1], "\n")
	body := strings.TrimSpace(matches[2])

	post := &Post{
		ContentMarkdown: body,
		Tags:            []string{},
	}

	var curKey string
	for _, l := range metaLines {
		line := strings.TrimSpace(l)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Itens de lista com traço: - tag1
		if strings.HasPrefix(line, "- ") && curKey == "tags" {
			tagVal := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			tagVal = strings.Trim(tagVal, `"'`)
			if tagVal != "" {
				post.Tags = append(post.Tags, tagVal)
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			curKey = key

			switch key {
			case "title":
				post.Title = val
			case "slug":
				post.Slug = val
			case "date":
				post.Date = val
			case "author":
				post.Author = val
			case "summary", "description":
				post.Summary = val
			case "image", "cover", "banner":
				post.Image = val
			case "tags":
				// Suporta formato inline: [tag1, tag2]
				if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
					inner := strings.Trim(val, "[]")
					for _, item := range strings.Split(inner, ",") {
						itemClean := strings.TrimSpace(strings.Trim(item, `"' `))
						if itemClean != "" {
							post.Tags = append(post.Tags, itemClean)
						}
					}
				} else if val != "" {
					post.Tags = append(post.Tags, val)
				}
			}
		}
	}

	// Auto-slug se não fornecido
	if post.Slug == "" && post.Title != "" {
		post.Slug = Slugify(post.Title)
	}

	// Métricas de leitura
	words := strings.Fields(body)
	post.WordCount = len(words)
	post.ReadingTimeMin = post.WordCount / 200
	if post.ReadingTimeMin < 1 && post.WordCount > 0 {
		post.ReadingTimeMin = 1
	}

	// Renderiza HTML
	post.ContentHTML = markdown.RenderHTML(body)

	return post, nil
}

// Slugify converte uma string de título em um slug URL-friendly
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Normaliza acentuações básicas
	s = replaceAccents(s)
	s = slugCleanRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func replaceAccents(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ã', 'â', 'ä':
			sb.WriteRune('a')
		case 'é', 'è', 'ê', 'ë':
			sb.WriteRune('e')
		case 'í', 'ì', 'î', 'ï':
			sb.WriteRune('i')
		case 'ó', 'ò', 'õ', 'ô', 'ö':
			sb.WriteRune('o')
		case 'ú', 'ù', 'û', 'ü':
			sb.WriteRune('u')
		case 'ç':
			sb.WriteRune('c')
		case 'ñ':
			sb.WriteRune('n')
		case '.':
			sb.WriteRune('-')
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}
