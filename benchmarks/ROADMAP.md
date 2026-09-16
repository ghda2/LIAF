# Roadmap de Benchmarks — LIAF vs Go vs Python (Issue #009)

Guia prático passo a passo para execução, coleta e validação dos benchmarks de IA.

---

## Fase 1: Preparação do Ambiente e Infraestrutura
- [x] **Runner Base**: `benchmarks/run_suite.py` funcional e testado localmente.
- [x] **Provedores de LLM**: Suporte nativo ao Gemini adicionado com carregamento do `.env`.
- [x] **Verificação de Compilador**: `liafc.exe` compilado e validado.
- [x] **Teste Dry-Run**:
  ```bash
  py benchmarks/run_suite.py --provider gemini --model gemini-2.5-flash --dry-run
  ```

---

## Fase 2: Expansão dos Cenários de Teste (`tasks.json`)
Consolidar os 3 cenários principais da proposta AI-First:
1. **Algoritmos e Controle de Fluxo**:
   - Loops com interrupção condicional e filtros (`loop-filter`).
2. **Manipulação de Dados e Erros Seguros**:
   - Listas/Mapas com tratamento explícito de ausência via `Result` (`collections-errors`).
3. **I/O e Serialização Real**:
   - Serialização de Structs para JSON, escrita em disco e roundtrip de leitura (`json-roundtrip`).
4. **(Novo) Microserviço HTTP**:
   - Rota dinâmica com parsing de payload e resposta rápida.

---

## Fase 3: Execução das Rodadas de Teste
Executar cada cenário para **LIAF**, **Go** e **Python** sob as mesmas condições:
1. **Rodada Inicial (Zero-Shot)**: Avaliar `pass_at_1` (acerto de primeira sem intervenção humana).
2. **Ciclos de Auto-Cura**: Permitir até 3 tentativas com feedback dos erros do compilador (`liafc check`).
3. **Comando de Execução (Exemplo Ollama local / API)**:
   ```bash
   python benchmarks/run_suite.py --provider ollama --model qwen2.5-coder --languages liaf go python --runs 3
   ```

---

## Fase 4: Coleta de Métricas e Análise
Registrar e tabular:
- **Taxa de Sucesso (%)**: Pass@1 e Pass@3 após auto-cura.
- **Economia de Tokens**: Total de tokens de prompt e completion gerados.
- **Ciclos Médios de Reparação**: Quantas iterações a IA precisou para compilar com sucesso.
- **Tempo de Execução e RAM**: Latência de compilação e RSS em execução.

---

## Fase 5: Publicação dos Resultados no Showcase
- [ ] Gerar tabela comparativa e gráficos em `benchmarks/README.md`.
- [ ] Atualizar o showcase público e `docs/` com dados empíricos comprovados.
