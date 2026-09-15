# Especificação da Linguagem LIAF (Language for AI First)

**Versão**: 0.1.0  
**Paradigma**: Estruturado, Estático, Tipado, Concorrente (CSP/Atores), Otimizado para Transformers.

---

## 1. Princípios Mecânicos (Foco em IA)

1. **Tokens Inteiros**: Palavras-chave mapeiam para tokens únicos de alta frequência nos vocabulários BPE (tiktoken, Llama, Gemma, Claude).
2. **Fechamentos Nomeados (`/tag nome`)**: Elimina alucinação de contagem de delimitadores (`}}}`, `)))`). Se o compilador vê um fechamento com nome incorreto, o erro é determinístico e pontual.
3. **Sem Ambiguidade de Precedência**: Operadores em notação prefixada eliminam regras complexas de precedência de operadores.
4. **Auto-Cura Estruturada**: Erros de sintaxe e semântica são emitidos exclusivamente em JSON padronizado com nó da AST, localização e sugestão de correção (patch).

---

## 2. Gramática EBNF

```ebnf
Program        ::= { TopLevelDecl }
TopLevelDecl   ::= StructDecl | FuncDecl

StructDecl     ::= "[struct" Identifier { FieldDecl } "/struct" Identifier "]"
FieldDecl      ::= Identifier ":" Type

FuncDecl       ::= "[fn" Identifier "(" { ParamDecl } ")" "->" "(" [ Type ] ")" { Stmt } "/fn" Identifier "]"
ParamDecl      ::= Identifier ":" Type

Stmt           ::= LetStmt | ReturnStmt | SpawnStmt | SendStmt | RecvStmt | IfStmt | Expr
LetStmt        ::= "[let" Identifier ":" Type Expr "]"
ReturnStmt     ::= "[return" [ Expr ] "]"
SpawnStmt      ::= "[spawn" CallExpr "]"
SendStmt       ::= "[send" Expr Expr "]"
RecvStmt       ::= "[recv" Expr "]"
IfStmt         ::= "[if" Expr { Stmt } [ "[else" { Stmt } "/else]" ] "/if]"

Expr           ::= CallExpr | OpExpr | Literal | Identifier
CallExpr       ::= "(" Identifier { Expr } ")"
OpExpr         ::= "(" ( "add" | "sub" | "mul" | "div" | "eq" | "neq" | "gt" | "lt" ) Expr Expr ")"
```

---

## 3. Tipos Primitivos

- `int` (inteiro 64-bit com sinal)
- `float` (ponto flutuante 64-bit)
- `str` (string imutável)
- `bool` (`true` ou `false`)
- `chan[T]` (canal com tipagem forte para concorrência)
- `void` (sem retorno)

---

## 4. Built-ins de Runtime

- `println(args...)`: Imprime texto no stdout com quebra de linha.
- `print(args...)`: Imprime texto no stdout sem quebra de linha.
- `concat(args...)`: Concatena múltiplos valores em uma string.
- `str(val)`: Converte qualquer valor primitivo para string.
- `sleep_ms(ms)`: Pausa a rotina atual por N milissegundos.
- `http_serve(addr, handler)`: Inicia servidor HTTP nativo.
- `http_get(url)`: Faz requisição HTTP GET e retorna o corpo como string.
- `make_chan(T)`: Cria um canal com tipagem estrita para troca de mensagens concorrentes.
- `serve_site(dir, [domain], [port], [auto_tls])`: Inicializa o LIAF Web Engine autônomo com auto-minificação em memória, compressão Gzip e suporte a Auto-TLS / Let's Encrypt.

---

## 5. Schema de Auto-Cura de IA (Compiler Diagnostics)

O compilador emite diagnósticos estruturados via `liafc check <file> --json`:

```json
{
  "status": "error",
  "errors": [
    {
      "code": "TAG_NAME_MISMATCH",
      "file": "broken.liaf",
      "line": 5,
      "col": 18,
      "node": "FuncDecl",
      "message": "Nome no fechamento '/fn worker_errado' não corresponde a '/fn worker'",
      "expected": "worker",
      "received": "worker_errado",
      "suggested_patch": "/fn worker]"
    }
  ]
}
```
