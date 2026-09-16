# LIAF - Getting Started

Guia rápido para compilar, executar e testar o LIAF (Language for AI First).

---

## 1. Pré-requisitos
- **Go 1.22+** instalado (o backend Go compila o binário final).

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

### Formatar na forma canônica
```bash
./liafc fmt examples/math.liaf        # imprime a forma canônica
./liafc fmt -w arquivo.liaf           # grava no arquivo
./liafc fmt -l examples/*.liaf        # lista o que está fora do formato (útil em CI)
```
O formatador ainda não preserva comentários, então `-w` se recusa a gravar em
arquivo comentado a menos que você passe `--drop-comments`.

### API REST completa em v0.3
```bash
./liafc build examples/task_api_v03.liaf -o task_api
mkdir public
./task_api 8080
```
Endpoints: `GET /health`, `GET /tasks`, `POST /tasks`, `GET|PUT|DELETE /tasks/{id}`.
O exemplo mostra as três adições da v0.3 — chamadas sem `call`, `try`/`on-err` no
lugar de `match` aninhado, e rotas declarativas com parâmetros já tipados.

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
- `pkg/codegen`: Emissão de código Go; backend C em `pkg/codegen/c`.
- `pkg/runtime`: Biblioteca de apoio do código gerado (Result, listas, mapas, FS, JSON).
- `pkg/web`: Web engine de ultra baixa latência com rotas dinâmicas e SSG.
- `examples/`: Exemplos práticos da linguagem.
- `docs/`: Documentação técnica completa e especificações.
