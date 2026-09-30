// Currículo moderno, escrito como código Typst comum: nada aqui vem de modelo pronto.
// O programa LIAF (curriculo.liaf) lê este arquivo, entrega os dados em /dados.json e
// renderiza com pdf-typst-files. Mude o que quiser: layout, cores, fontes, seções.
//
// Dados (JSON; campos ausentes ou vazios são omitidos):
//   nome, cargo, resumo, email, telefone, local, site, linkedin, github, cor ("#rrggbb")
//   foto: nome de um arquivo extra entregue ao documento, ex.: "foto.jpg"; sem ela, iniciais
//   experiencias: [{cargo, empresa, periodo, local, descricao, destaques: [str], tecnologias: [str]}]
//   formacao:     [{curso, instituicao, periodo, descricao}]
//   projetos:     [{nome, descricao, link, tecnologias: [str]}]
//   habilidades:  [{nome, nivel (1 a 5)}]
//   idiomas:      [{nome, nivel}]
//   certificacoes: [str]
//   interesses:   [str]

// ---- Tokens de design: escala, espaçamento e paleta derivada de uma cor ----
// O layout nunca usa tamanho ou cor solta: trocar só a cor de destaque (campo "cor")
// recolore o documento inteiro de forma consistente.

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
  suave: rgb("#686f7c"), // contraste >= 4,5:1 também sobre a lateral tingida (liafc doc check)
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

// ---- Componentes ----

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

// ---- Layout do currículo ----

#let largura-lateral = 190pt

#let coluna-lateral(p, d) = {
  let foto = campo(d, "foto")
  align(center, if tem(foto) {
    avatar-foto(p, "/" + foto.trim("/", at: start), tamanho: 84pt)
  } else {
    avatar-iniciais(p, d.nome, tamanho: 76pt)
  })
  v(esp.xl)

  titulo-secao(p, "Contato")
  if tem(campo(d, "email")) { rotulo-valor(p, "E-mail", d.email, destino: "mailto:" + d.email) }
  if tem(campo(d, "telefone")) { rotulo-valor(p, "Telefone", d.telefone) }
  if tem(campo(d, "local")) { rotulo-valor(p, "Local", d.local) }
  if tem(campo(d, "site")) { rotulo-valor(p, "Site", d.site, destino: "https://" + d.site.trim("https://", at: start)) }
  if tem(campo(d, "linkedin")) { rotulo-valor(p, "LinkedIn", d.linkedin, destino: "https://" + d.linkedin.trim("https://", at: start)) }
  if tem(campo(d, "github")) { rotulo-valor(p, "GitHub", d.github, destino: "https://" + d.github.trim("https://", at: start)) }

  let habilidades = lista(d, "habilidades")
  if habilidades.len() > 0 {
    titulo-secao(p, "Habilidades")
    for h in habilidades {
      block(below: esp.md + 1pt, breakable: false, {
        text(size: escala.sm, weight: 500, h.nome)
        v(esp.sm, weak: true)
        barra-nivel(p, campo(h, "nivel", padrao: 3))
      })
    }
  }

  let idiomas = lista(d, "idiomas")
  if idiomas.len() > 0 {
    titulo-secao(p, "Idiomas")
    for i in idiomas {
      block(below: esp.md, grid(
        columns: (1fr, auto),
        text(size: escala.sm, weight: 500, i.nome),
        text(size: escala.xs, fill: p.suave, campo(i, "nivel", padrao: "")),
      ))
    }
  }

  let certificacoes = lista(d, "certificacoes")
  if certificacoes.len() > 0 {
    titulo-secao(p, "Certificações")
    for c in certificacoes {
      block(below: esp.sm, text(size: escala.sm, c))
    }
  }

  let interesses = lista(d, "interesses")
  if interesses.len() > 0 {
    titulo-secao(p, "Interesses")
    set par(leading: esp.md)
    interesses.map(i => etiqueta(p, i)).join(h(esp.sm))
  }
}

#let coluna-principal(p, d) = {
  text(size: escala.xxl, weight: 700, tracking: -0.6pt, fill: p.texto, d.nome)
  if tem(campo(d, "cargo")) {
    block(above: esp.lg, below: 0pt, text(size: escala.lg, weight: 500, fill: p.destaque, d.cargo))
  }
  if tem(campo(d, "resumo")) {
    block(
      above: esp.xxl,
      width: 100%,
      inset: (left: esp.lg, y: esp.sm),
      stroke: (left: 2.5pt + p.destaque),
      par(leading: 0.7em, text(fill: p.texto.lighten(15%), d.resumo)),
    )
  }

  let experiencias = lista(d, "experiencias")
  if experiencias.len() > 0 {
    titulo-secao(p, "Experiência")
    for e in experiencias {
      item-linha-do-tempo(
        p,
        e.cargo,
        subtitulo: campo(e, "empresa"),
        quando: campo(e, "periodo"),
        onde: campo(e, "local"),
        corpo: {
          if tem(campo(e, "descricao")) { par(leading: 0.65em, e.descricao) }
          destaques(p, lista(e, "destaques"))
          etiquetas(p, lista(e, "tecnologias"))
        },
      )
    }
  }

  let projetos = lista(d, "projetos")
  if projetos.len() > 0 {
    titulo-secao(p, "Projetos")
    for pr in projetos {
      block(below: esp.xl, breakable: false, {
        grid(
          columns: (1fr, auto),
          text(weight: 600, size: escala.md, pr.nome),
          if tem(campo(pr, "link")) {
            link("https://" + pr.link.trim("https://", at: start), text(size: escala.sm, fill: p.destaque, pr.link))
          },
        )
        if tem(campo(pr, "descricao")) {
          block(above: esp.sm, below: 0pt, par(leading: 0.65em, pr.descricao))
        }
        etiquetas(p, lista(pr, "tecnologias"))
      })
    }
  }

  let formacao = lista(d, "formacao")
  if formacao.len() > 0 {
    titulo-secao(p, "Formação")
    for f in formacao {
      item-linha-do-tempo(
        p,
        f.curso,
        subtitulo: campo(f, "instituicao"),
        quando: campo(f, "periodo"),
        corpo: if tem(campo(f, "descricao")) { f.descricao },
      )
    }
  }
}

#let curriculo(d) = {
  let p = paleta(cor-de(campo(d, "cor")))
  set document(title: d.nome + " — Currículo", author: d.nome)
  set page(
    paper: "a4",
    margin: (x: 0pt, y: esp.xxxl),
    background: place(top + left, rect(width: largura-lateral, height: 100%, fill: p.lateral)),
    footer: context {
      if counter(page).final().first() > 1 {
        set text(size: escala.xs, fill: p.suave)
        pad(left: largura-lateral + 28pt, right: 32pt, grid(
          columns: (1fr, auto),
          d.nome,
          counter(page).display("1 / 1", both: true),
        ))
      }
    },
  )
  set text(font: "Inter", size: escala.base, fill: p.texto, lang: "pt", region: "BR", hyphenate: false)
  set par(leading: 0.62em, spacing: esp.md)

  grid(
    columns: (largura-lateral, 1fr),
    pad(x: 22pt, coluna-lateral(p, d)),
    pad(left: 28pt, right: 32pt, coluna-principal(p, d)),
  )
}

#curriculo(json("/dados.json"))
