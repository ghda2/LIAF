// Componentes visuais reutilizáveis pelos modelos. Todos recebem a paleta `p` (ver tema.typ)
// como primeiro argumento, para funcionarem com qualquer cor de destaque.
#import "tema.typ": escala, esp, tem

// Título de seção: rótulo em caixa alta espaçada + fio até a margem.
#let titulo-secao(p, texto, cor: auto) = {
  let c = if cor == auto { p.destaque } else { cor }
  block(above: esp.xl, below: esp.md, sticky: true, grid(
    columns: (auto, 1fr),
    column-gutter: esp.md,
    align: horizon,
    text(size: escala.sm, weight: 700, tracking: 1.4pt, fill: c, upper(texto)),
    line(length: 100%, stroke: 0.6pt + p.linha),
  ))
}

// Avatar circular com as iniciais do nome.
#let avatar-iniciais(p, nome, tamanho: 64pt) = {
  let partes = nome.split(" ").filter(w => w.len() > 0)
  let iniciais = if partes.len() == 0 { "?" } else if partes.len() == 1 {
    partes.first().first()
  } else {
    partes.first().first() + partes.last().first()
  }
  box(
    width: tamanho,
    height: tamanho,
    radius: 50%,
    fill: p.destaque,
    align(center + horizon, text(
      size: tamanho * 0.36,
      weight: 700,
      fill: p.branco,
      tracking: 0.5pt,
      upper(iniciais),
    )),
  )
}

// Avatar circular com foto (arquivo entregue ao documento, ex.: "/foto.jpg"), recortada para
// preencher o círculo, com anel na cor de destaque.
#let avatar-foto(p, caminho, tamanho: 64pt) = box(
  width: tamanho,
  height: tamanho,
  radius: 50%,
  clip: true,
  stroke: 2.5pt + p.destaque,
  image(caminho, width: 100%, height: 100%, fit: "cover"),
)

// Barra de nível (ex.: domínio de uma habilidade de 1 a 5).
#let barra-nivel(p, nivel, maximo: 5, altura: 4pt) = {
  let fracao = calc.clamp(nivel, 0, maximo) / maximo
  block(
    width: 100%,
    height: altura,
    radius: altura / 2,
    fill: p.linha,
    clip: true,
    block(width: fracao * 100%, height: altura, radius: altura / 2, fill: p.destaque),
  )
}

// Etiqueta arredondada (tecnologia, palavra-chave).
#let etiqueta(p, texto) = box(
  inset: (x: 6pt, y: 3pt),
  radius: 8pt,
  fill: p.destaque-suave,
  text(size: escala.xs, weight: 500, fill: p.destaque-forte, texto),
)

// Linha de etiquetas, com quebra automática.
#let etiquetas(p, itens) = {
  if itens.len() > 0 {
    block(above: esp.md, below: 0pt, itens.map(t => etiqueta(p, t)).join(h(esp.xs + 1pt)))
  }
}

// Par rótulo/valor empilhado (contato na coluna lateral).
#let rotulo-valor(p, rotulo, valor, destino: none) = block(below: esp.md, {
  text(size: escala.xs, weight: 600, tracking: 0.8pt, fill: p.suave, upper(rotulo))
  linebreak()
  let corpo = text(size: escala.sm, fill: p.texto, valor)
  if destino != none { link(destino, corpo) } else { corpo }
})

// Item de linha do tempo: marcador na trilha à esquerda e conteúdo ao lado.
// `titulo`, `subtitulo`, `quando` e `onde` são textos; `corpo` é conteúdo livre.
#let item-linha-do-tempo(p, titulo, subtitulo: none, quando: none, onde: none, corpo: none) = {
  block(
    breakable: true,
    above: 0pt,
    below: 0pt,
    stroke: (left: 1pt + p.linha),
    inset: (left: esp.lg, bottom: esp.xl),
    {
      place(left + top, dx: -esp.lg - 4pt, dy: 2.5pt, circle(
        radius: 3.5pt,
        fill: p.destaque,
        stroke: 2pt + p.branco,
      ))
      grid(
        columns: (1fr, auto),
        column-gutter: esp.md,
        text(size: escala.md, weight: 600, fill: p.texto, titulo),
        if tem(quando) { text(size: escala.sm, fill: p.suave, quando) },
      )
      if tem(subtitulo) or tem(onde) {
        block(above: esp.sm, below: 0pt, {
          if tem(subtitulo) { text(weight: 500, fill: p.destaque, subtitulo) }
          if tem(subtitulo) and tem(onde) { text(fill: p.suave)[ · ] }
          if tem(onde) { text(fill: p.suave, onde) }
        })
      }
      if corpo != none {
        block(above: esp.md, below: 0pt, corpo)
      }
    },
  )
}

// Lista de destaques com marcador discreto na cor do tema.
#let destaques(p, itens) = {
  if itens.len() > 0 {
    set list(marker: text(fill: p.destaque, sym.bullet), indent: 0pt, body-indent: esp.sm, spacing: esp.sm)
    block(above: esp.md, below: 0pt, list(..itens.map(i => text(fill: p.texto, i))))
  }
}
