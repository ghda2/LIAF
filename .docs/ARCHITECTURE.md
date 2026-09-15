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

## Componentes:
1. **`pkg/lexer`**: Tokenizador otimizado para vocabulários BPE sem ambiguidade de caracteres.
2. **`pkg/parser`**: Constrói a AST validando correspondência de tags de fechamento (`[fn x ... /fn x]`).
3. **`pkg/checker`**: Analisador semântico de tipos e escopos com emissor de diagnósticos JSON cirúrgicos.
4. **`pkg/codegen`**: Emite código Go concorrente e auto-suficiente.
5. **`cmd/liafc`**: CLI executável com comandos `check`, `emit`, `build` e `run`.
