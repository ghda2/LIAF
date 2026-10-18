# Roadmap de Desenvolvimento LIAF

Este documento define os marcos de evolução da linguagem, com foco no **Piloto Real: Web Engine Autônomo** e nos pilares de maturidade para uso geral por IAs e humanos.

---

## 🎯 Piloto Prioritário: Web Engine Autônomo (Estilo Caddy + Minificador)

O objetivo deste piloto é gerar um **binário único auto-suficiente** que hospeda sites com performance máxima e zero configuração de infraestrutura.

- [x] **M1. Minificação Nativa em Memória**:
  - Minificador de HTML (remoção de comentários, espaços desnecessários, fechamento opcional).
  - Minificador de CSS (colapso de whitespace, remoção de regras redundantes).
  - Minificador de JS (compressão léxica básica e preservação de semântica).
- [x] **M2. Compressão e Cache de Alta Velocidade**:
  - Suporte nativo a compressão Gzip automática via headers `Accept-Encoding`.
  - Cache em RAM de assets pré-comprimidos (zero I/O de disco após leitura inicial).
  - Validação de cache via ETags (304 Not Modified).
- [x] **M3. Empacotamento de Assets no Binário (`embed`)**:
  - Capacidade de servir pastas estáticas de frontend diretamente compiladas.
  - Deploy em 1 arquivo executável único (`liaf_web.exe`).
- [x] **M4. Auto-TLS / HTTPS Nativo (Zero-Config ACME)**:
  - Gerenciamento automático de certificados Let's Encrypt (`autocert`).
  - Redirecionamento transparente HTTP (porta 80) -> HTTPS (porta 443).
- [x] **M5. Primitivas LIAF de Alto Nível**:
  - `(serve_site dir [domain] [port] [auto_tls])`: sobe o servidor com 1 comando.

---

## 🧱 Pilares de Maturidade de Linguagem Geral

### Fase 1: Fundação Sintática e Concorrência (✓ Concluído)
- [x] Gramática determinística com operadores prefixados.
- [x] Fechamento tagged de blocos (`[fn ... /fn ...]`).
- [x] Runtime concorrente com canais e green threads (`spawn`, `chan`, `send`, `recv`).
- [x] Compilação para binário único estático via Go.
- [x] Servidor HTTP básico embutido.
- [x] Diagnóstico de erro estruturado em JSON para auto-cura por LLM.

### Fase 2: Estruturas de Dados Avançadas (Em Andamento)
- [ ] Listas/Vetores dinâmicos indexados (`list[T]`).
- [ ] Mapas chave-valor (`map[K, V]`).
- [ ] Parser e serializador JSON nativo (`json_encode`, `json_decode`).

### Fase 3: Sistema de Tipos e Tratamento de Erros
- [ ] Tipo explícito de resultado `Result[T, E]` (sem panics silenciosos).
- [ ] Tratamento forçado de erros no compilador para IA.
- [ ] Structs com métodos associados.

### Fase 4: Módulos e Sandbox de Execução
- [ ] Sistema de pacotes por diretório com namespaces limpos.
- [ ] Sandbox com limites de tempo de execução, memória e concorrência.
- [ ] Modo de auto-cura interativo em CLI (`liafc heal`).
