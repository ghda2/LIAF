# Armadilhas conhecidas

| Sintoma | Causa | Correção |
|---|---|---|
| `unable to get the current date` | `datetime.today()`: o motor é determinístico e não lê o relógio | Passe a data nos dados, já formatada (`"30/09/2026"`) |
| `dictionary does not contain key "x"` | Campo com nome diferente da struct, ou opcional ausente | Mesmo nome da struct LIAF; opcional com `d.at("x", default: "")` |
| `unknown variable: f` | Função usada antes de ser definida | Mova o `#let f(...)` para cima do ponto de uso |
| Valor sai `47.9` em vez de `R$ 47,90` | Dinheiro em `float` | Centavos em `int` + função `reais` ([typst-essencial.md](typst-essencial.md) §7) |
| `file not found (searched at /logo.png)` | Imagem não entregue ao motor | Use `pdf-typst-files` com `{"logo.png": bytes}` (ou `--arquivo` no `doc check`) |
| `pacote @preview/x:1.0.0 não embutido` | Pacote não citado onde o `liafc build` procura | Cite o `#import` no `.typ` ao lado do `.liaf` e rode `liafc run`/`build` com internet na primeira vez |
| `SHA-256 ... difere do arquivo de trava` | O pacote mudou no servidor desde a trava | Não apague a trava às cegas: confira a versão e, se for legítimo, remova só aquela linha |
| `unknown font family` (`W_TYPST`) | Fonte que não está embutida | `"Inter"`, `"Libertinus Serif"` ou `"DejaVu Sans Mono"` |
| Pix/código copiado não funciona | Hífen repetido na quebra de linha (regra do pt) | `text(lang: "en", hyphenate: false, codigo)` |
| Tamanho do PDF "errado" no log | `str-len` conta caracteres UTF-8 | `str-byte-len(pdf)` |
| PDF abre em branco ou corrompido | Gravou o `err` em vez do PDF, ou manipulou os bytes como texto | Use `try pdf-typst(...)` e grave o valor sem transformar |
| Texto do usuário "vira código" | Dado concatenado no fonte Typst | Dados só por `json-encode` → `/dados.json` |
| Build sem a fonte que o documento usa | `--doc-fonts` excluiu a família | Tire o `--doc-fonts` ou inclua a família; `serif` é a padrão do Typst |
| Primeira requisição de PDF lenta no servidor | Motor compilando na primeira execução da máquina | Normal (alguns segundos, uma vez); o aquecimento em segundo plano já começa ao subir o programa |
