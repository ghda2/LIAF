//go:build !liaf_doc_sem_math

package fonts

import "embed"

// New Computer Modern Math — Fórmulas ($ ... $): sem ela, qualquer fórmula falha ao compilar.
// Excluir do build: liafc build --doc-fonts sem "math" (tag liaf_doc_sem_math).
//
//go:embed math
var mathFS embed.FS

func init() { register("math", "New Computer Modern Math", mathFS, "math") }
