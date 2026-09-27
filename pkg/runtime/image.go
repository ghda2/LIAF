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
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ImageProcess decodifica qualquer formato de imagem suportado (PNG, JPEG, GIF, BMP, WebP),
// redimensiona proporcionalmente caso exceda maxWidth ou maxHeight (preservando o aspect ratio)
// e codifica para WebP com nível de esforço de compressão definido (0 a 9).
// Se maxWidth <= 0 e maxHeight <= 0, mantém as dimensões originais intactas.
func ImageProcess(data string, maxWidth, maxHeight, level int64) Result[string, string] {
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

	origBounds := img.Bounds()
	origW := int64(origBounds.Dx())
	origH := int64(origBounds.Dy())

	targetImg := img
	if (maxWidth > 0 && origW > maxWidth) || (maxHeight > 0 && origH > maxHeight) {
		scaleW := 1.0
		if maxWidth > 0 && origW > maxWidth {
			scaleW = float64(maxWidth) / float64(origW)
		}
		scaleH := 1.0
		if maxHeight > 0 && origH > maxHeight {
			scaleH = float64(maxHeight) / float64(origH)
		}

		scale := scaleW
		if scaleH < scale {
			scale = scaleH
		}

		newW := int(float64(origW) * scale)
		newH := int(float64(origH) * scale)
		if newW < 1 {
			newW = 1
		}
		if newH < 1 {
			newH = 1
		}

		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, origBounds, draw.Over, nil)
		targetImg = dst
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

	if err := nativewebp.Encode(&buf, targetImg, opts); err != nil {
		return Err[string, string](fmt.Sprintf("failed to encode webp: %v", err))
	}

	return Ok[string, string](buf.String())
}

// ImageResize redimensiona a imagem para caber dentro de maxWidth x maxHeight proporcionalmente,
// codificando o resultado em WebP com compressão padrão (nível 4).
func ImageResize(data string, maxWidth, maxHeight int64) Result[string, string] {
	return ImageProcess(data, maxWidth, maxHeight, 4)
}

// ImageToWebP converte qualquer formato de imagem suportado (PNG, JPEG, GIF, BMP, WebP)
// para WebP com compressão estritamente sem perdas (lossless VP8L), mantendo 100% da
// fidelidade dos pixels sem qualquer perda de qualidade.
func ImageToWebP(data string) Result[string, string] {
	return ImageProcess(data, 0, 0, 4)
}

// ImageToWebPQuality converte imagem para WebP definindo o esforço do algoritmo de compressão sem perdas (0 a 9).
// Níveis mais altos demandam mais CPU mas reduzem o tamanho final sem abrir mão da qualidade lossless.
func ImageToWebPQuality(data string, level int64) Result[string, string] {
	return ImageProcess(data, 0, 0, level)
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
