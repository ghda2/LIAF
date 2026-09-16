from fastapi import FastAPI, HTTPException
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
import json, os, uvicorn

app = FastAPI()

class CreateUserInput(BaseModel):
    name: str
    email: str

DATA_FILE = "users-data.json"

def load_state():
    if not os.path.exists(DATA_FILE):
        return {"next_id": 1, "users": []}
    with open(DATA_FILE, "r", encoding="utf-8") as f:
        return json.load(f)

def save_state(state):
    tmp = DATA_FILE + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(state, f)
    os.replace(tmp, DATA_FILE)

@app.get("/users")
def get_users():
    return load_state()["users"]

@app.post("/users", status_code=201)
def create_user(input: CreateUserInput):
    if not input.name.strip():
        raise HTTPException(status_code=400, detail="Nome eh obrigatorio")
    if not input.email.strip():
        raise HTTPException(status_code=400, detail="Email eh obrigatorio")
    
    state = load_state()
    user = {"id": state["next_id"], "name": input.name, "email": input.email}
    state["next_id"] += 1
    state["users"].append(user)
    save_state(state)
    return user

@app.get("/users/{id}")
def find_user(id: int):
    if id <= 0:
        raise HTTPException(status_code=400, detail="ID deve ser positivo")
    for user in load_state()["users"]:
        if user["id"] == id:
            return user
    raise HTTPException(status_code=404, detail="Usuario nao encontrado")

# Serve arquivos estáticos da pasta ./public
app.mount("/", StaticFiles(directory="./public", html=True), name="static")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
