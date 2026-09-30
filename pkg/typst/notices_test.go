package typst

import (
	"os"
	"strings"
	"testing"
)

// THIRD_PARTY_NOTICES.md (raiz do repositório, distribuído com o liafc) precisa refletir as
// licenças realmente embutidas. Para regenerar: LIAF_UPDATE_NOTICES=1 go test ./pkg/typst.
func TestThirdPartyNoticesUpToDate(t *testing.T) {
	const file = "../../THIRD_PARTY_NOTICES.md"
	want := "# Licenças de terceiros\n\n" +
		"O liafc e os programas que geram documentos (pdf-*) embutem os componentes abaixo.\n" +
		"Arquivo gerado por `LIAF_UPDATE_NOTICES=1 go test ./pkg/typst`; não edite à mão.\n\n" +
		"```text\n" + strings.TrimSpace(Notices()) + "\n```\n"
	if os.Getenv("LIAF_UPDATE_NOTICES") == "1" {
		if err := os.WriteFile(file, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(file)
	if err != nil || strings.ReplaceAll(string(got), "\r\n", "\n") != want {
		t.Fatal("THIRD_PARTY_NOTICES.md desatualizado: rode LIAF_UPDATE_NOTICES=1 go test ./pkg/typst")
	}
}
