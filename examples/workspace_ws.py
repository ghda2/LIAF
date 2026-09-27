# LIAF Collab Workspace (Kanban + Docs Markdown + WebSockets + SQLite)
# Equivalente em Python de workspace_ws.liaf, para comparar tamanho e esforço.
import json
import os
import sqlite3
import sys

import uvicorn
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import JSONResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel

app = FastAPI()
DB_PATH = "kanban.db"
clients: set[WebSocket] = set()


class CreateCardInput(BaseModel):
    title: str = ""
    column: str = ""
    author: str = ""


class MoveCardInput(BaseModel):
    column: str = ""


class CreateDocInput(BaseModel):
    slug: str = ""
    title: str = ""
    content: str = ""
    author: str = ""
    updated_at: str = ""


class UpdateDocInput(BaseModel):
    title: str = ""
    content: str = ""
    updated_at: str = ""


def get_db():
    db = sqlite3.connect(DB_PATH)
    db.row_factory = sqlite3.Row
    return db


def error_response(status, message):
    return JSONResponse({"error": message}, status_code=status)


@app.exception_handler(sqlite3.Error)
async def db_error(request, exc):
    return error_response(500, str(exc))


async def broadcast(kind, message):
    payload = json.dumps({"kind": kind, "message": message})
    for ws in list(clients):
        try:
            await ws.send_text(payload)
        except Exception:
            clients.discard(ws)


def init_db():
    with get_db() as db:
        db.execute("CREATE TABLE IF NOT EXISTS cards (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, column TEXT NOT NULL, author TEXT NOT NULL)")
        db.execute("CREATE TABLE IF NOT EXISTS documents (id INTEGER PRIMARY KEY AUTOINCREMENT, slug TEXT UNIQUE NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, author TEXT NOT NULL, updated_at TEXT NOT NULL)")

        # Popula cards padrao caso tabela esteja vazia
        if db.execute("SELECT id FROM cards LIMIT 1").fetchone() is None:
            db.executemany(
                "INSERT INTO cards (title, column, author) VALUES (?, ?, ?)",
                [
                    ("Conectar SQLite e WebSockets", "done", "Gabriel"),
                    ("Editor Markdown Colaborativo", "in_progress", "LIAF AI"),
                    ("Deploy em Produção (tw.webdrop.bio)", "todo", "DevOps"),
                ],
            )

        # Popula docs padrao caso tabela esteja vazia
        if db.execute("SELECT id FROM documents LIMIT 1").fetchone() is None:
            db.execute(
                "INSERT INTO documents (slug, title, content, author, updated_at) VALUES (?, ?, ?, ?, ?)",
                (
                    "bem-vindo",
                    "Bem-vindo ao LIAF Workspace",
                    "# LIAF Live Workspace\n\nEste documento foi carregado nativamente do **SQLite** e renderizado em tempo real.\n\n### Recursos Ativos:\n- **LIAF v0.4** compilado em binário nativo\n- **WebSockets** para colaboração instantânea\n- **Banco de Dados SQLite** para persistência atômica",
                    "Gabriel",
                    "Hoje",
                ),
            )


# --- Rotas HTTP REST para Cards ---
@app.get("/api/board")
def get_board():
    with get_db() as db:
        rows = db.execute("SELECT id, title, column, author FROM cards ORDER BY id").fetchall()
    return {"cards": [dict(r) for r in rows]}


@app.post("/api/cards", status_code=201)
async def create_card(input: CreateCardInput):
    if input.title == "":
        return error_response(400, "Título obrigatório")
    col = input.column or "todo"
    author = input.author or "Anônimo"
    with get_db() as db:
        db.execute("INSERT INTO cards (title, column, author) VALUES (?, ?, ?)", (input.title, col, author))
    await broadcast("sync", "Card criado: " + input.title)
    return {"ok": True}


@app.put("/api/cards/{id}/move")
async def move_card(id: int, input: MoveCardInput):
    if id <= 0:
        return error_response(400, "ID inválido")
    with get_db() as db:
        db.execute("UPDATE cards SET column = ? WHERE id = ?", (input.column, id))
    await broadcast("sync", "Card movido para " + input.column)
    return {"ok": True}


@app.delete("/api/cards/{id}")
async def delete_card(id: int):
    if id <= 0:
        return error_response(400, "ID inválido")
    with get_db() as db:
        db.execute("DELETE FROM cards WHERE id = ?", (id,))
    await broadcast("sync", "Card excluído")
    return {"ok": True}


# --- Rotas HTTP REST para Docs (Markdown) ---
@app.get("/api/docs")
def get_docs():
    with get_db() as db:
        rows = db.execute("SELECT id, slug, title, content, author, updated_at FROM documents ORDER BY id DESC").fetchall()
    return {"docs": [dict(r) for r in rows]}


@app.post("/api/docs", status_code=201)
async def create_doc(input: CreateDocInput):
    if input.title == "":
        return error_response(400, "Título obrigatório")
    author = input.author or "Anônimo"
    with get_db() as db:
        db.execute(
            "INSERT INTO documents (slug, title, content, author, updated_at) VALUES (?, ?, ?, ?, ?)",
            (input.slug, input.title, input.content, author, input.updated_at),
        )
    await broadcast("doc_sync", "Documento criado: " + input.title)
    return {"ok": True}


@app.put("/api/docs/{id}")
async def update_doc(id: int, input: UpdateDocInput):
    if id <= 0:
        return error_response(400, "ID inválido")
    with get_db() as db:
        db.execute(
            "UPDATE documents SET title = ?, content = ?, updated_at = ? WHERE id = ?",
            (input.title, input.content, input.updated_at, id),
        )
    await broadcast("doc_sync", "Documento atualizado: " + input.title)
    return {"ok": True}


@app.delete("/api/docs/{id}")
async def delete_doc(id: int):
    if id <= 0:
        return error_response(400, "ID inválido")
    with get_db() as db:
        db.execute("DELETE FROM documents WHERE id = ?", (id,))
    await broadcast("doc_sync", "Documento excluído")
    return {"ok": True}


# Estatísticas / Presença
@app.get("/api/collaborators")
def get_collaborators():
    return {"online": len(clients)}


# --- Rota WebSocket para Sincronização em Tempo Real ---
@app.websocket("/ws/board")
async def ws_board(ws: WebSocket):
    await ws.accept()
    clients.add(ws)
    print("Colaborador conectado ao Workspace")
    await broadcast("presence", "Um colaborador conectou-se")
    try:
        while True:
            await broadcast("sync", await ws.receive_text())
    except WebSocketDisconnect:
        clients.discard(ws)
        print("Colaborador desconectado")
        await broadcast("presence", "Um colaborador desconectou-se")


app.mount("/", StaticFiles(directory="./public", html=True), name="public")

if __name__ == "__main__":
    try:
        init_db()
        print("Banco SQLite inicializado com sucesso")
    except sqlite3.Error as e:
        print("Falha ao inicializar SQLite: " + str(e))
    port = os.environ.get("PORT") or (sys.argv[1] if len(sys.argv) > 1 else "8080")
    print("LIAF Collab Workspace rodando na porta " + port)
    uvicorn.run(app, host="0.0.0.0", port=int(port))
