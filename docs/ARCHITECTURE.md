# Arquitetura do Compilador e Runtime LIAF

O pipeline da LIAF é desenhado para latência ultra-baixa de compilação e execução concorrente em binário único estático.

```
+---------------+      +-------------------+      +----------------------+
| Código .liaf  | ---> | Lexer & Tokenizer | ---> | Parser (Tagged AST)  |
+---------------+      +-------------------+      +----------------------+
                                                             |
                                                             v
+-----------------------+     +-------------------+     +----------------------+
| Binário Estático .exe | <-- | Compilador Go     | <-- | Gerador de Código    |
| (Zero Dependências)   |     | (Single Binary)   |     | (Transpiler Go)      |
+-----------------------+     +-------------------+     +----------------------+
                                                                 ^
                                                                 |
                                                     [ Checker Semântico & Tipos ]
                                                                 |
                                                                 v
                                                     [ Diagnóstico JSON / Auto-Heal ]
```

## Componentes

### Pipeline de compilação

1. **`pkg/token`**: Vocabulário de tokens e palavras-chave, sem ambiguidade de caracteres para BPE.
2. **`pkg/lexer`**: Tokenizador. Identificadores ASCII; NUL no meio da entrada é erro, não EOF.
3. **`pkg/parser`**: Constrói a AST validando correspondência de tags de fechamento (`[fn x ... /fn x]`).
   Interrompe no primeiro erro estruturado. Estruturas de controle (`loop`, `match`) ficam em `control.go`.
4. **`pkg/ast`**: Nós da árvore (`ast.go`) e formatador canônico (`printer.go`).
5. **`pkg/checker`**: Analisador semântico de tipos, escopos, efeitos transitivos e tratamento
   obrigatório de `Result`.
6. **`pkg/diagnostic`**: Formato dos diagnósticos, incluindo a saída JSON consumida por agentes.
7. **`pkg/codegen`**: Emite Go concorrente e auto-suficiente. `library.go` mapeia os builtins da
   linguagem para o runtime.
8. **`pkg/codegen/c`**: Backend C alternativo (em desenvolvimento; ainda não integrado à CLI).
9. **`pkg/runtime`**: Biblioteca de apoio do código gerado — `Result`, listas, mapas, FS, JSON e strings.
10. **`cmd/liafc`**: CLI com os comandos `check`, `fmt`, `emit`, `build`, `run`, `publish`, `service` e `deploy`.

### Web e deploy

11. **`pkg/web`**: Servidor do binário final. Cache de assets em RAM com fallback para disco
    (`cache.go`), roteamento estático e dinâmico (`server.go`, `router.go`), endpoint administrativo
    autenticado de publish/reload (`admin.go`), compressão, TLS automático e acesso a arquivos
    restrito por `os.Root`.
12. **`pkg/markdown`**: Conversão de Markdown com front matter para as páginas geradas.
13. **`pkg/minifier`**: Minificação de HTML, CSS e JS dos assets servidos.
14. **`pkg/deploy`**: Geração de unidade systemd e configuração de rota via Caddy Admin API.

## Nota sobre o estado do código

O compilador é funcional e a suíte `go test ./...` cobre toda a árvore. Alguns arquivos do núcleo
(`pkg/web/cache.go`, `pkg/checker/checker.go`, `pkg/codegen/codegen.go`, `pkg/parser/parser.go`)
concentram hoje mais responsabilidades do que o ideal e estão em fila para divisão; a separação por
pacotes descrita acima é o contrato estável, e a reorganização interna desses arquivos não altera
comportamento observável.
