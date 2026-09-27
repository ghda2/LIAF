package runtime_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"liaf/pkg/runtime"

	"github.com/HugoSmits86/nativewebp"
)

func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Preenche com gradiente
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: 128,
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestImageToWebPLossless(t *testing.T) {
	pngData := createTestPNG(64, 64)

	res := runtime.ImageToWebP(string(pngData))
	if !res.OK {
		t.Fatalf("ImageToWebP falhou: %s", res.Error)
	}

	webpBytes := []byte(res.Value)
	if len(webpBytes) == 0 {
		t.Fatal("esperava bytes de webp não vazios")
	}

	// Decodifica o WebP gerado para verificar integridade e dimensões
	decodedImg, err := nativewebp.Decode(bytes.NewReader(webpBytes))
	if err != nil {
		t.Fatalf("falha ao decodificar WebP gerado: %v", err)
	}

	bounds := decodedImg.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Errorf("dimensões incorretas no WebP: obtido %dx%d, esperado 64x64", bounds.Dx(), bounds.Dy())
	}
}

func TestImageDimensions(t *testing.T) {
	pngData := createTestPNG(120, 80)

	res := runtime.ImageDimensions(string(pngData))
	if !res.OK {
		t.Fatalf("ImageDimensions falhou: %s", res.Error)
	}
	if res.Value != "120x80" {
		t.Errorf("esperava '120x80', obteve %q", res.Value)
	}
}

func TestImageInvalidFormat(t *testing.T) {
	res := runtime.ImageToWebP("dados_invalidos_que_nao_sao_imagem")
	if res.OK {
		t.Fatal("esperava erro ao tentar converter dados inválidos em imagem")
	}
}
