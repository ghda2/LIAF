package runtime

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/HugoSmits86/nativewebp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

// ImageToWebP converte qualquer formato de imagem suportado (PNG, JPEG, GIF, BMP, WebP)
// para WebP com compressão estritamente sem perdas (lossless VP8L), mantendo 100% da
// fidelidade dos pixels sem qualquer perda de qualidade.
func ImageToWebP(data string) Result[string, string] {
	return ImageToWebPQuality(data, 4)
}

// ImageToWebPQuality converte imagem para WebP definindo o esforço do algoritmo de compressão sem perdas (0 a 9).
// Níveis mais altos demandam mais CPU mas reduzem o tamanho final sem abrir mão da qualidade lossless.
func ImageToWebPQuality(data string, level int64) Result[string, string] {
	if len(data) == 0 {
		return Err[string, string]("image data is empty")
	}

	img, format, err := image.Decode(bytes.NewReader([]byte(data)))
	if err != nil {
		if format == "" {
			return Err[string, string](fmt.Sprintf("unsupported or corrupted image format: %v", err))
		}
		return Err[string, string](fmt.Sprintf("failed to decode %s image: %v", format, err))
	}

	if level < 0 {
		level = 0
	} else if level > 9 {
		level = 9
	}

	var buf bytes.Buffer
	opts := &nativewebp.Options{
		UseExtendedFormat: false,
		CompressionLevel:  nativewebp.CompressionLevel(level),
	}

	if err := nativewebp.Encode(&buf, img, opts); err != nil {
		return Err[string, string](fmt.Sprintf("failed to encode webp: %v", err))
	}

	return Ok[string, string](buf.String())
}

// ImageDimensions lê o cabeçalho da imagem e retorna a resolução no formato "LARGURAxALTURA" (ex: "800x600").
func ImageDimensions(data string) Result[string, string] {
	if len(data) == 0 {
		return Err[string, string]("image data is empty")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader([]byte(data)))
	if err != nil {
		return Err[string, string](fmt.Sprintf("failed to read image config: %v", err))
	}

	return Ok[string, string](fmt.Sprintf("%dx%d", cfg.Width, cfg.Height))
}
