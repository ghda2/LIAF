// Tokens de design dos modelos da LIAF: escala tipográfica, espaçamento e paleta.
// Um modelo nunca escreve tamanho ou cor solta: usa estes tokens, para que trocar o tema
// (ou só a cor de destaque) mude o documento inteiro de forma consistente.

// Escala tipográfica de razão ~1,2 a partir de 10 pt (texto corrido).
#let escala = (
  xs: 7.5pt,
  sm: 8.5pt,
  base: 10pt,
  md: 11.5pt,
  lg: 13.5pt,
  xl: 20pt,
  xxl: 28pt,
)

// Grade de 4 pt.
#let esp = (
  xs: 2pt,
  sm: 4pt,
  md: 8pt,
  lg: 12pt,
  xl: 16pt,
  xxl: 24pt,
  xxxl: 32pt,
)

#let cor-padrao = rgb("#2563eb")

// Converte o texto de cor vindo dos dados ("#0f766e") em cor; vazio usa a padrão.
#let cor-de(valor, padrao: cor-padrao) = {
  if valor == none or valor == "" { padrao } else { rgb(valor) }
}

// Paleta derivada de uma única cor de destaque. Os neutros são fixos para manter o
// contraste de leitura (WCAG AA) com qualquer destaque.
#let paleta(destaque) = (
  destaque: destaque,
  destaque-forte: destaque.darken(25%),
  destaque-suave: destaque.lighten(88%),
  texto: rgb("#1f2937"),
  suave: rgb("#6b7280"),
  linha: rgb("#e5e7eb"),
  lateral: destaque.lighten(94%).desaturate(40%),
  branco: white,
)

// Utilitários para dados opcionais vindos de JSON.
#let tem(v) = v != none and v != "" and v != ()
#let campo(d, chave, padrao: none) = d.at(chave, default: padrao)
#let lista(d, chave) = {
  let v = d.at(chave, default: none)
  if v == none { () } else { v }
}
