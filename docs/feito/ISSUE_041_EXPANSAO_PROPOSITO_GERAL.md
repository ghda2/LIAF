# Issue #041: Expansão para Linguagem de Propósito Geral (Além de Backend/Web)

**Status:** Concluída  
**Componente:** `std/`, `pkg/builtins`, `pkg/runtime`, `pkg/checker`, `docs/linguagem/SPEC.md`  
**Data:** 04 de outubro de 2026  

---

## 1. Contexto e Motivação

Atualmente, a LIAF é excelente e altamente especializada em aplicações backend/web (servidores REST, WebSockets, autenticação JWT, bancos SQL/Redis, e geração de PDFs com Typst).

Contudo, para se consolidar como uma linguagem completa e de **propósito geral**, a LIAF precisa permitir a criação de:
- **Ferramentas de Linha de Comando (CLIs):** Automações, utilitários de build, scripts de manutenção.
- **Workers e Processamento em Segundo Plano:** Tarefas assíncronas, filas locais, cron jobs e pipelines de ETL sem obrigatoriedade de subir um servidor HTTP.
- **Manipulação Completa de Sistema e I/O:** Leitura/escrita de streams, buffers de bytes, subprocessos com pipes, manipulação avançada de arquivos e diretórios.

> [!IMPORTANT]
> **Decisão Arquitetural:** Mantém-se o princípio **AI-First de dependências zero externas** — todo o poder de propósito geral residirá na biblioteca padrão embutida (`std/`), sem necessidade de gerenciar pacotes externos descentralizados.

---

## 2. Proposta Técnica

### 2.1 Módulo `std/cli` e Execução sem Servidor Web
Permitir que programas LIAF executem como scripts ou utilitários de terminal puros:
- Parsing declarativo de argumentos de linha de comando, flags (`--flag`, `-f`) e comandos posicionais.
- Controle de saída padrão, erro e códigos de retorno (`exit-code`).
- Formatação de texto no terminal (cores ANSI, tabelas simples).

### 2.2 Manipulação de Arquivos, Diretórios e Subprocessos (`std/os` / `std/fs`)
- Leitura e caminhada recursiva em diretórios (`fs-walk-dir`, `fs-list-dir`, `fs-mkdir-all`, `fs-remove-all`).
- Execução de subprocessos com captura controlada de `stdin`, `stdout`, `stderr` e status de saída (`os-exec-pipe`, `os-exec-capture`).
- Manipulação de arquivos temporários e paths portáteis.

### 2.3 Formatos de Dados Adicionais na Stdlib (`std/csv`, `std/yaml`)
- Parser e gerador de CSV tipado.
- Parser e gerador de YAML/TOML para arquivos de configuração locais.

### 2.4 Workers e Concorrência Autônoma (`std/worker` / `std/cron`)
- Execução de tarefas em background com pool de concorrência nativo do Go.
- Agendamento de rotinas periódicas (expressões cron ou intervalos de tempo) em processos daemon locais.

### 2.5 Tipo e Operações com Buffers de Bytes (`bytes`)
- Suporte nativo a manipulação de dados binários brutos, hashing de streams e criptografia além de strings simples.

---

## 3. Critérios de Aceite

1. **Execução Standalone:** Um programa LIAF sem nenhuma rota `route` ou `ws-route` compila para um binário CLI leve e executa sua função `main` diretamente terminando após a execução.
2. **Exemplo de CLI Funcional:** Implementação de um programa utilitário (ex: processador de arquivos/imagens em lote via terminal) usando `std/cli` e `std/fs`.
3. **Exemplo de Worker em Background:** Implementação de um pipeline de ETL/tarefas locais sem servidor HTTP ativo.
4. **Testes de Conformidade:** Suíte de testes em `conformance/general_purpose/` validando flags de CLI, subprocessos e manipuladores de arquivos.
