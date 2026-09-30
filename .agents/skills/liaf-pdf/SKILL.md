---
name: liaf-pdf
description: Como agentes de IA geram PDFs em programas LIAF — layout escrito em Typst, dados tipados em LIAF, motor embutido no executável. Use sempre que a tarefa pedir PDF, recibo, relatório, currículo, cobrança, contrato, etiqueta, certificado ou qualquer documento impresso a partir de um programa LIAF. Cobre o fluxo, os builtins pdf-*, os comandos liafc doc (check, preview, watch), o laço de correção com avisos de layout e as armadilhas conhecidas.
---

# LIAF — Geração de PDF (skill para agentes)

Um PDF em LIAF tem **duas partes**:

| Arquivo | Linguagem | Papel |
|---|---|---|
| `doc.typ` | **Typst** | O layout: textos, cores, tabelas, imagens, páginas. Liberdade total |
| `prog.liaf` | **LIAF** | Os dados (structs tipadas), a lógica, a rota; chama `pdf-typst` |

Não existem modelos prontos: **você escreve o layout**. O motor Typst vai embutido no executável, então em
execução não há Rust, binário extra nem acesso à rede. Para a sintaxe LIAF em geral, veja a skill `liaf`.

---

## 1. Fluxo obrigatório

Siga nesta ordem. Não pule o passo 4.

1. **Modele os dados** como `struct` LIAF. Dinheiro em **centavos (`int`)**, datas como `str` já
   formatadas (`"30/09/2026"`).
2. **Escreva `dados-exemplo.json`** com os mesmos nomes de campo das structs. É com ele que você testa o
   layout sem compilar o programa.
3. **Escreva o `doc.typ`**: leia os dados com `#let d = json("/dados.json")` e monte o layout.
   Guia rápido de Typst: [references/typst-essencial.md](references/typst-essencial.md).
4. **Confira o layout** até sair sem erro e sem aviso:
   ```text
   liafc doc check doc.typ --dados dados-exemplo.json --json
   ```
   Aplique cada `fix` e rode de novo. Regras do laço: [references/laco-de-correcao.md](references/laco-de-correcao.md).
5. **Escreva o programa LIAF** que lê o `.typ`, codifica os dados e gera o PDF (§2).
6. **Valide e rode:** `liafc check prog.liaf` e depois `liafc run prog.liaf`.
7. Se você enxerga imagens, **olhe a página**: `liafc doc preview doc.typ --dados dados-exemplo.json -o p.png`.

## 2. O programa LIAF

```liaf
struct Item
  nome str
  qtd int
  preco_centavos int
end

struct Recibo
  numero int
  cliente str
  itens (list Item)
end

fn main() void effects(fs, io)
  on-err e
    println(fmt("erro: {}", e))
    exit(1)
  end
  let itens = make-list(Item)
  list-push(itens, new Item("Café", 2, 850))
  let layout = try fs-read-file("recibo.typ")
  let pdf = try pdf-typst(layout, try json-encode(new Recibo(1042, "Ana", itens)))
  try fs-write-file("recibo.pdf", pdf)
end
```

Numa API, devolva direto:

```liaf
route GET "/recibos/{id}.pdf" (id int) Response effects(fs, db)
  on-err e
    json-response(500, e)
  end
  let r = try buscar_recibo(id)
  let layout = try fs-read-file("recibo.typ")
  let pdf = try pdf-typst(layout, try json-encode(r))
  pdf-response(pdf, "recibo.pdf")
end
```

## 3. Builtins

| Builtin | Assinatura | Uso |
|---|---|---|
| `pdf-typst` | `(fonte str, dados_json str) (result str str)` | PDF. Os dados ficam em `/dados.json` |
| `pdf-typst-files` | `(fonte str, dados_json str, arquivos (map str str)) (result str str)` | PDF com imagens, logos, SVG ou CSV |
| `png-typst` | `(fonte str, dados_json str) (result str str)` | Prévia PNG da 1ª página (144 ppi) |
| `png-typst-files` | `(fonte str, dados_json str, arquivos (map str str)) (result str str)` | Prévia com arquivos |
| `pdf-response` | `(pdf str, nome str) Response` | Resposta HTTP `application/pdf`, inline |

Nenhum deles pede efeito. `fs-read-file` e `fs-write-file` pedem `fs`. Comandos, opções e variáveis de
ambiente estão em [references/comandos.md](references/comandos.md).

## 4. Regras de ouro

1. **Dados sempre por `/dados.json`.** Nunca monte o Typst concatenando strings com dados: use
   `json-encode` e leia no documento. Dados lidos assim entram como texto e nunca viram código.
2. **Dinheiro em centavos (`int`)**, formatado no `.typ` (função `reais` em
   [references/typst-essencial.md](references/typst-essencial.md)). `float` gera `47.9` em vez de
   `R$ 47,90`.
3. **Sem `datetime.today()`**: o motor é determinístico e esse comando falha ("unable to get the
   current date"). Passe a data nos dados.
4. **Use só as fontes embutidas:** `"Inter"`, `"Libertinus Serif"`, `"DejaVu Sans Mono"`. As fórmulas
   (`$ ... $`) usam a New Computer Modern Math sozinhas. Outra família cai na padrão com aviso `W_TYPST`.
5. **Arquivos extras** vão pelo mapa de `pdf-typst-files`: `{"img/logo.png": bytes}` fica em
   `/img/logo.png`. Nomes com `..`, `\`, `@`, `main.typ` ou `dados.json` são recusados.
6. **Meça bytes com `str-byte-len`**, nunca com `str-len` (que conta caracteres).
7. **Texto para copiar** (Pix copia e cola, chaves, códigos):
   `text(lang: "en", hyphenate: false, ...)`. Em português, o Typst repete o hífen na quebra de linha e
   corrompe o texto colado.
8. **Contraste mínimo WCAG AA**: 4,5:1 para texto normal. Cinza médio sobre fundo tingido costuma
   reprovar; o `doc check` aponta e sugere a cor.
9. **Pare no zero:** entregue só com `errors` e `warnings` vazios no `doc check`, ou justifique o aviso
   que ficou.

Mais armadilhas, com sintoma e correção: [references/armadilhas.md](references/armadilhas.md).

## 5. Exemplo completo

[examples/recibo/](examples/recibo/): `recibo.liaf`, `recibo.typ` e `dados-exemplo.json`. Passa no
`liafc check`, no `doc check` com 0 avisos e gera o PDF. Outros exemplos no repositório:
`curriculo.liaf` + `curriculo.typ` (raiz) e `examples/cobranca_pix_pdf/` (QR Code via pacote).
