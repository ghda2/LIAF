package builtins

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func init() {
	// ---------------------------------------------------------
	// Web / HTTP
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "json-response",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"int", "str"},
		Return:       "Response",
		GoCall:       "web.JSONResponse",
		UnsupportedC: true,
	})

	for _, m := range []string{"http-get", "http-post", "http-put", "http-delete"} {
		method := strings.ToUpper(strings.TrimPrefix(m, "http-"))
		Register(&Builtin{
			Name:     m,
			Category: "web",
			Arity:    ExactArity(2),
			Effects:  []string{"net"},
			Return:   "void",
			GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
				return fmt.Sprintf("web.Register(%q, %s)", method, strings.Join(args, ", ")), true
			},
			UnsupportedC: true,
		})
	}

	Register(&Builtin{
		Name:         "serve-site",
		Category:     "web",
		Arity:        ExactArity(4),
		Params:       []string{"str", "str", "str", "bool"},
		Effects:      []string{"net", "fs"},
		Return:       "void",
		GoCall:       "web.ServeSite",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:     "serve-hybrid",
		Category: "web",
		Arity:    ExactArity(2),
		Params:   []string{"str", "str"},
		Effects:  []string{"net", "fs"},
		Return:   "(result bool str)",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			return "rt.Failure(web.ServeHybrid(" + strings.Join(args, ", ") + "))", true
		},
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-body",
		Category:     "web",
		Arity:        ExactArity(1),
		Params:       []string{"Request"},
		Return:       "str",
		GoTemplate:   "(%s).Body",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-path",
		Category:     "web",
		Arity:        ExactArity(1),
		Params:       []string{"Request"},
		Return:       "str",
		GoTemplate:   "(%s).Path",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-method",
		Category:     "web",
		Arity:        ExactArity(1),
		Params:       []string{"Request"},
		Return:       "str",
		GoTemplate:   "(%s).Method",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-header",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestHeader",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-query",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestQuery",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "response-set-header",
		Category:     "web",
		Arity:        ExactArity(3),
		Params:       []string{"Response", "str", "str"},
		Return:       "Response",
		GoCall:       "rt.ResponseSetHeader",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "raw-response",
		Category:     "web",
		Arity:        ExactArity(3),
		Params:       []string{"int", "str", "str"},
		Return:       "Response",
		GoCall:       "web.RawResponse",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "file-response",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result Response str)",
		GoCall:       "rt.FileResponse",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-file-data",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestFileData",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-file-name",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestFileName",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Imagem e Compressão WebP
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "image-to-webp",
		Category:     "image",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "(result str str)",
		GoCall:       "rt.ImageToWebP",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "image-to-webp-quality",
		Category:     "image",
		Arity:        ExactArity(2),
		Params:       []string{"str", "int"},
		Return:       "(result str str)",
		GoCall:       "rt.ImageToWebPQuality",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "image-dimensions",
		Category:     "image",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "(result str str)",
		GoCall:       "rt.ImageDimensions",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Storage (Estilo MinIO)
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "storage-dir-init",
		Category:     "storage",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"fs"},
		Return:       "(result bool str)",
		GoCall:       "rt.StorageInit",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-save-image",
		Category:     "storage",
		Arity:        ExactArity(3),
		Params:       []string{"str", "str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result str str)",
		GoCall:       "rt.StorageSaveImage",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-file-get",
		Category:     "storage",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result str str)",
		GoCall:       "rt.StorageGet",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-file-delete",
		Category:     "storage",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result bool str)",
		GoCall:       "rt.StorageDelete",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-file-exists",
		Category:     "storage",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "bool",
		GoCall:       "rt.StorageExists",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Cliente HTTP
	// ---------------------------------------------------------
	// http-get e afins ja sao a forma v0.2 de registrar rotas, dai o nome
	// http-fetch. O metodo literal e conferido pelo checker.
	Register(&Builtin{
		Name:         "http-fetch",
		Category:     "http-client",
		Arity:        ExactArity(4),
		Params:       []string{"str", "str", "(list str)", "str"},
		Effects:      []string{"net"},
		Return:       "(result HttpReply str)",
		GoCall:       "rt.HTTPFetch",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "reply-status",
		Category:     "http-client",
		Arity:        ExactArity(1),
		Params:       []string{"HttpReply"},
		Return:       "int",
		GoCall:       "rt.ReplyStatus",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "reply-body",
		Category:     "http-client",
		Arity:        ExactArity(1),
		Params:       []string{"HttpReply"},
		Return:       "str",
		GoCall:       "rt.ReplyBody",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "reply-header",
		Category:     "http-client",
		Arity:        ExactArity(2),
		Params:       []string{"HttpReply", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.ReplyHeader",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// WebSockets (issue #016)
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "ws-send",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "(result void str)",
		GoCall:       "rt.WSSend",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-send-json",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "(result void str)",
		GoCall:       "rt.WSSendJSON",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-close",
		Category:     "ws",
		Arity:        ExactArity(3),
		Effects:      []string{"net"},
		Return:       "(result void str)",
		GoCall:       "rt.WSCloseConn",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-broadcast",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "(result int str)",
		GoCall:       "rt.WSBroadcast",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-join",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "void",
		GoCall:       "rt.WSJoin",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-leave",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "void",
		GoCall:       "rt.WSLeave",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-topic-size",
		Category:     "ws",
		Arity:        ExactArity(1),
		Effects:      []string{"net"},
		Return:       "int",
		GoCall:       "rt.WSTopicSize",
		UnsupportedC: true,
	})
}
