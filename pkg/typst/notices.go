package typst

import (
	"strings"

	"liaf/pkg/typst/fonts"
	"liaf/pkg/typst/wasm"
)

// Notices reúne as licenças de tudo o que o motor de documentos embute: o Typst e suas
// dependências Rust, o wazero e as famílias de fonte incluídas no binário. O texto viaja dentro
// de todo programa que gera documentos; `liafc licencas` o imprime.
func Notices() string {
	var b strings.Builder
	b.WriteString("==== Motor Typst (engines/typst) ====\n")
	b.WriteString(wasm.Notices())
	b.WriteString("\n==== wazero (github.com/tetratelabs/wazero) ====\nApache-2.0\n\n")
	b.WriteString(fonts.Notices())
	return b.String()
}
