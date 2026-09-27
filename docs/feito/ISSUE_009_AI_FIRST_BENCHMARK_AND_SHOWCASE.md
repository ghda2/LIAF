# Issue #009: Benchmark Empírico AI-First e Showcase Kit de Lançamento

**Status:** Concluída (medição e comparação LIAF vs Python em consumo de tokens realizada)
**Componente:** `benchmarks/`, `cmd/liafc`, `docs/arquivo/showcase`  
**Data:** 26 de setembro de 2026  

---

> **Atenção (26/09/2026):** a pasta `benchmarks/` e o par de referência `examples/workspace_ws.liaf` /
> `workspace_ws.py` foram retirados do repositório para deixá-lo só com a linguagem. Estão no commit
> `c7d5ff6` (`git show c7d5ff6:benchmarks/run_suite.py`, `git checkout c7d5ff6 -- benchmarks`).
> Antes de rodar, decidir se o harness volta para cá ou vira um repositório próprio.

## 1. Contexto e Objetivo
Para divulgar a LIAF com credibilidade máxima e validar sua tese central (*"Não precisamos de modelos mais fortes; precisamos de linguagens feitas para modelos baratos"*), é indispensável um conjunto de dados empíricos irrefutáveis.

Sem números comparativos, a comunidade técnica tratará o projeto como mais uma DSL/Lisp. Com dados, a LIAF se posiciona como uma revolução na economia de tokens e confiabilidade de agentes autônomos.

---

## 2. Metodologia do Benchmark

### 2.1. Matriz de Comparação
Comparar modelos pequenos/baratos (SLMs) programando em LIAF contra modelos caros/pesados programando em linguagens tradicionais:

| Métrica | Modelo Barato + LIAF (ex: GPT-4o-mini / Haiku 3.5 / Qwen-Coder-7B) | Modelo Caro + Python/Go (ex: Claude 3.5 Sonnet / GPT-4o) |
| :--- | :--- | :--- |
| **Taxa de Acerto na 1ª Tentativa (Pass@1)** | Alvo: > 90% | Histórico: ~70-80% |
| **Custo de Tokens por Tarefa ($/task)** | Alvo: 5x a 10x menor | Linha de base |
| **Consumo de Memória do Runtime** | Alvo: ~2.4 MB RAM | ~30-50 MB (Python/Node) |
| **Velocidade de Auto-Cura (Loops de Erro)** | Alvo: 1 iteração mecânica via JSON | 2 a 4 iterações com mensagens verbosas |

### 2.2. Suite de Tarefas Mínimas (3 Cenários Reais)
1. **API REST com Validação:** Criar endpoint que recebe JSON, valida campos, grava em memória e retorna status tipado.
2. **Processamento em Lote com Efeitos:** Ler lista de URLs, buscar concorrentemente com timeout e registrar métricas sem race conditions.
3. **Serviço Web com Hot-Reload:** Subir servidor HTTP com VFS e rota de atualização atômica de assets.

---

## 3. Tarefas de Implementação
- [ ] Criar runner automatizado de benchmark em `benchmarks/run_suite.py` (executa prompts padronizados via API da OpenAI/Anthropic/Ollama).
- [ ] Implementar extrator de métricas: tokens de entrada/saída, tempo de compilação, tempo de execução e taxa de sucesso do compilador.
- [ ] Gerar gráficos comparativos automatizados (`cost_comparison.svg`, `pass_rate.svg`).
- [ ] Criar `SHOWCASE.md` pronto para postagem no Hacker News, X e Reddit, com links para o binário e demonstração ao vivo em `https://tw.webdrop.bio`.
- [ ] Empacotar prompt canônico de sistema (System Prompt / Cursor Rules) para que qualquer usuário use a LIAF no Cursor/Windsurf/Claude Code em 1 clique.


Contrato atual e ajustes de sintaxe: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
