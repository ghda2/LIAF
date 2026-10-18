package builtins

func init() {
	Register(&Builtin{
		Name:     "content-parse",
		Category: "content",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "(result str str)",
		GoCall:   "rt.ContentParse",
	})

	Register(&Builtin{
		Name:     "content-slugify",
		Category: "content",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "str",
		GoCall:   "rt.ContentSlugify",
	})

	Register(&Builtin{
		Name:     "markdown-to-html",
		Category: "content",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "str",
		GoCall:   "rt.MarkdownToHTML",
	})
}
