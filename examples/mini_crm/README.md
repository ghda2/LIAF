# LIAF Enterprise CRM (v0.4)

Exemplo completo de uma aplicação comercial em tempo real construída na linguagem **LIAF (Language for AI First)** com **SQLite**, **WebSockets** e interface SPA responsiva.

## Recursos

1. **Pipeline de Vendas (Kanban)**: Gestão visual de oportunidades por estágios (*Prospecção*, *Proposta*, *Negociação*, *Fechado/Ganho*) com Drag and Drop e sincronização instantânea.
2. **Visão 360° do Lead**: Drawer lateral exibindo timeline de atividades, reuniões, ligações e notas.
3. **Analytics & KPIs**: Taxa de conversão (*Win Rate*), total de pipeline ativo, receita ganha e ticket médio calculados diretamente no backend.
4. **Catálogo de Produtos & Serviços**: Tabela relacional de produtos vinculada ao banco SQLite.
5. **Multi-usuário em Tempo Real**: WebSocket nativo (`/ws/crm`) notificando alterações em sub-milissegundos para todos os vendedores conectados.

---

## Estrutura dos Arquivos

```
examples/mini_crm/
├── crm.liaf              # Backend nativo LIAF v0.4 (APIs REST + SQLite + WebSockets)
├── README.md             # Esta documentação
└── public/
    ├── index.html        # Estrutura HTML da SPA comercial
    ├── crm.css           # Design system Dark Mode (Grid, Kanban, Drawer, Timeline)
    └── crm.js            # Lógica SPA, WebSocket, Drag & Drop e Analytics nativo
```

---

## Como Executar

### 1. Validar Sintaxe e Tipos
```bash
liafc check examples/mini_crm/crm.liaf --json
```

### 2. Rodar em Modo Desenvolvimento
```bash
liafc run examples/mini_crm/crm.liaf
```
Acesse: [http://localhost:8080](http://localhost:8080)

### 3. Compilar em Binário Único e Autônomo
Para empacotar o backend e todos os assets estáticos (`public/`) dentro de um executável único de ~15MB que roda sem dependências:

```bash
# Windows
liafc build examples/mini_crm/crm.liaf -o crm.exe --embed=public

# Linux
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
liafc build examples/mini_crm/crm.liaf -o crm_linux --embed=public
```
