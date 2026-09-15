package minifier

import (
	"strings"
	"testing"
)

func TestMinifyHTML(t *testing.T) {
	input := `
		<!DOCTYPE html>
		<!-- Comentario longo que deve sumir -->
		<html>
			<head>
				<title>   Titulo do Site   </title>
			</head>
			<body>
				<h1>   Ola Mundo   </h1>
			</body>
		</html>
	`
	output := MinifyHTML(input)
	if strings.Contains(output, "Comentario longo") {
		t.Errorf("Esperava remocao de comentarios HTML")
	}
	if strings.Contains(output, ">   <") {
		t.Errorf("Esperava colapso de espacos entre tags")
	}
}

func TestMinifyCSS(t *testing.T) {
	input := `
		/* Comentario CSS */
		body {
			margin : 0px ;
			padding : 20px ;
			background-color : #ffffff ;
		}
	`
	output := MinifyCSS(input)
	if strings.Contains(output, "Comentario CSS") {
		t.Errorf("Esperava remocao de comentarios CSS")
	}
	expected := "body{margin:0px;padding:20px;background-color:#ffffff}"
	if output != expected {
		t.Errorf("Esperava '%s', recebido '%s'", expected, output)
	}
}

func TestMinifyJS(t *testing.T) {
	input := `
		// Comentario de linha
		function teste() {
			/* Comentario de bloco */
			const a = 10 ;
			const b = 20 ;
			return a + b ;
		}
	`
	output := MinifyJS(input)
	if strings.Contains(output, "Comentario de linha") || strings.Contains(output, "Comentario de bloco") {
		t.Errorf("Esperava remocao de comentarios JS")
	}
}
