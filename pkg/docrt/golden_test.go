package docrt

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"
)

// Regressão visual (#037): a prévia do currículo de exemplo é comparada pixel a pixel com uma
// referência aprovada. Uma mudança no motor, nas fontes ou no curriculo.typ que altere o visual
// reprova até a nova referência ser aprovada com LIAF_UPDATE_GOLDEN=1 go test ./pkg/docrt.
const goldenPNG = "testdata/curriculo.png"

// Fração máxima de pixels diferentes (antisserrilhado pode variar minimamente).
const goldenTolerance = 0.001

func TestCurriculoVisualRegression(t *testing.T) {
	src, data := curriculo(t)
	res := SourcePNG(src, data)
	if !res.OK {
		t.Fatal(res.Error)
	}
	if os.Getenv("LIAF_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPNG, []byte(res.Value), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("referência atualizada: %s", goldenPNG)
		return
	}
	want, err := os.ReadFile(goldenPNG)
	if err != nil {
		t.Fatalf("sem referência (%v): gere com LIAF_UPDATE_GOLDEN=1 go test ./pkg/docrt", err)
	}
	diff, err := pixelDiff(want, []byte(res.Value))
	if err != nil {
		t.Fatal(err)
	}
	if diff > goldenTolerance {
		save(t, "curriculo-atual.png", res.Value)
		t.Fatalf("o visual do currículo mudou (%.3f%% dos pixels). Se a mudança for intencional, aprove com LIAF_UPDATE_GOLDEN=1 go test ./pkg/docrt", diff*100)
	}
}

// pixelDiff devolve a fração de pixels com diferença perceptível entre dois PNGs.
func pixelDiff(a, b []byte) (float64, error) {
	ia, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		return 0, err
	}
	ib, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	if ia.Bounds() != ib.Bounds() {
		return 0, fmt.Errorf("tamanho mudou: %v -> %v", ia.Bounds(), ib.Bounds())
	}
	var bad, total int
	r := ia.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			total++
			if channelDelta(ia, ib, x, y) > 24 {
				bad++
			}
		}
	}
	return float64(bad) / float64(total), nil
}

func channelDelta(a, b image.Image, x, y int) int {
	r1, g1, b1, _ := a.At(x, y).RGBA()
	r2, g2, b2, _ := b.At(x, y).RGBA()
	d := func(p, q uint32) int {
		v := int(p>>8) - int(q>>8)
		if v < 0 {
			return -v
		}
		return v
	}
	return max(d(r1, r2), d(g1, g2), d(b1, b2))
}
