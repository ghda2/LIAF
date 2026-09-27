# tw.webdrop.bio — frontend

Assets servidos em produção em tw.webdrop.bio.

- `public/index.html` — Live Collab Hub (kanban + docs), consome `/api/*` e `/ws/board` de
  [`../workspace_ws.liaf`](../workspace_ws.liaf).
- `public/chat.html` — chat em tempo real, consome `/rooms/{room}/size` e `/ws/chat/{room}` de
  [`../tw_api.liaf`](../tw_api.liaf) (ou [`../chat_ws.liaf`](../chat_ws.liaf)).

Os exemplos servem `./public` relativo ao diretório atual, então rode a partir desta pasta:

```powershell
cd examples/tw_webdrop
../../liafc.exe run ../workspace_ws.liaf
```
