//go:build !liaf_doc_sem_inter

package fonts

import "embed"

// Inter — Sem serifa, padrão dos exemplos (6 pesos: 300 a 700 e itálico).
// Excluir do build: liafc build --doc-fonts sem "inter" (tag liaf_doc_sem_inter).
//
//go:embed inter
var interFS embed.FS

func init() { register("inter", "Inter", interFS, "inter") }
