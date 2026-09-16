# LIAF - Getting Started

Guia rápido para compilar, executar e testar o LIAF (Language for AI First).

---

## 1. Pré-requisitos
- **Go 1.22+** instalado.

---

## 2. Compilar o Compilador (`liafc`)

Na raiz do repositório:
```bash
go build -o liafc.exe ./cmd/liafc
```
*(No Linux/macOS, use `-o liafc`)*

---

## 3. Comandos Básicos

### Verificar sintaxe e tipos (Diagnostics)
```bash
./liafc check examples/loops.liaf
```

### Executar diretamente (JIT/Run)
```bash
./liafc run examples/math.liaf
```

### Compilar para executável
```bash
./liafc build examples/api_server.liaf -o server_app
```

### Iniciar Web Engine com SSG dinâmico
```bash
./liafc run examples/web_engine.liaf
```
Acesse `http://localhost:8080` no navegador.

---

## 4. Estrutura do Projeto
- `cmd/liafc`: Ponto de entrada da CLI do compilador.
- `pkg/lexer`: Tokenização e scanner rápido.
- `pkg/parser`: Parser preditivo e geração de AST.
- `pkg/checker`: Validação semântica e tipagem estática.
- `pkg/codegen`: Emissão de código em C e Go nativo.
- `pkg/web`: Web engine de ultra baixa latência com rotas dinâmicas e SSG.
- `examples/`: Exemplos práticos da linguagem.
- `docs/`: Documentação técnica completa e especificações.
