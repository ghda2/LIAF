package fonts

import (
	"bytes"
	"slices"
	"testing"
)

func TestDefaultHasAllFamiliesDecompressed(t *testing.T) {
	if got := IDs(); !slices.Equal(got, []string{"inter", "math", "mono", "serif"}) {
		t.Fatalf("famílias = %v", got)
	}
	for _, p := range Default() {
		if len(p.Files) == 0 {
			t.Errorf("%s sem arquivos", p.ID)
		}
		if p.License == "" {
			t.Errorf("%s sem licença embutida", p.ID)
		}
		for _, f := range p.Files {
			// TrueType (00 01 00 00) ou OpenType CFF ("OTTO"): o gzip foi desfeito.
			if !bytes.HasPrefix(f, []byte{0, 1, 0, 0}) && !bytes.HasPrefix(f, []byte("OTTO")) {
				t.Errorf("%s: arquivo não parece fonte (%x)", p.ID, f[:4])
			}
		}
	}
}
