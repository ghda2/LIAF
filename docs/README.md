# Documentação da LIAF

| Pasta | O que tem | Comece por |
|---|---|---|
| [linguagem/](linguagem/) | Como a linguagem funciona hoje: especificação, guias, arquitetura, mapa do código | [GETTING_STARTED](linguagem/GETTING_STARTED.md) para usar, [SPEC](linguagem/SPEC.md) para a referência |
| [a-fazer/](a-fazer/) | Issues abertas ou com pendência, e a ordem de trabalho | [PROXIMOS_PASSOS](a-fazer/PROXIMOS_PASSOS.md) |
| [feito/](feito/) | Issues concluídas, mantidas como registro do que foi entregue e testado | [STATUS](STATUS.md) |
| [logs/](logs/) | Logs de sessão: o que foi feito, medido e aprendido | [LOG_2026-09-26](logs/LOG_2026-09-26.md) |
| [arquivo/](arquivo/) | Documentos superados (spec v0.1, roadmaps antigos, #014). Não descrevem o estado atual | — |
| `privado/` | Logs e notas de servidor, com IPs e hosts. **Ignorada pelo git**: só existe na máquina local | — |

[STATUS.md](STATUS.md) é o painel: o estado de cada issue, com a evidência e o que falta.

## Regras

- **Issue nova** entra em `a-fazer/` com o próximo número livre do STATUS, e é registrada no STATUS no
  mesmo commit. O formato está na seção 2 da [#031](a-fazer/ISSUE_031_GUIA_PROXIMAS_ISSUES.md).
- **Issue concluída** vai para `feito/` só quando os critérios foram verificados por teste. Código escrito
  não basta; o que ficou sem verificar continua em `a-fazer/`.
- **Mudança na linguagem** atualiza `linguagem/SPEC.md` e `linguagem/AI_GUIDE.md` na mesma entrega.
- **Nada com IP, host, token ou credencial** fora de `privado/`.
