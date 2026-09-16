"""Reference server used only to validate the external benchmark oracle."""
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
from pathlib import Path
import sys

PERSIST = True
RESET_COUNTER = False
ACCEPT_EMPTY = False
DATA = Path("state.json")
state = json.loads(DATA.read_text(encoding="utf-8")) if PERSIST and DATA.exists() else {"next": 1, "tasks": []}
if RESET_COUNTER:
    state["next"] = max([t["id"] for t in state["tasks"]] + [0]) + 1


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def reply(self, status, payload):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def handle_request(self):
        if self.path == "/health":
            return self.reply(200, {"status": "ok"})
        if self.path == "/tasks" and self.command == "GET":
            return self.reply(200, state["tasks"])
        task = None
        if self.path.startswith("/tasks/"):
            try:
                task_id = int(self.path[len("/tasks/"):])
                if task_id <= 0:
                    raise ValueError()
            except ValueError:
                return self.reply(400, {"error": "invalid ID"})
            task = next((t for t in state["tasks"] if t["id"] == task_id), None)
            if task is None:
                return self.reply(404, {"error": "missing"})
        if self.command == "GET":
            return self.reply(200, task)
        if self.command in ("POST", "PUT"):
            try:
                payload = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))
                if not isinstance(payload, dict) or not isinstance(payload.get("title"), str):
                    raise ValueError()
                if not payload["title"] and not ACCEPT_EMPTY:
                    raise ValueError()
                if self.command == "PUT" and type(payload.get("done", False)) is not bool:
                    raise ValueError()
            except (ValueError, TypeError):
                return self.reply(400, {"error": "invalid body"})
            if self.command == "POST":
                task = {"id": state["next"], "title": payload["title"], "done": False}
                state["next"] += 1
                state["tasks"].append(task)
            else:
                task.update(title=payload["title"], done=payload.get("done", False))
            result, status = task, 201 if self.command == "POST" else 200
        elif self.command == "DELETE":
            state["tasks"].remove(task)
            result, status = {"deleted": True}, 200
        else:
            return self.reply(405, {"error": "method"})
        if PERSIST:
            DATA.write_text(json.dumps(state), encoding="utf-8")
        return self.reply(status, result)

    do_GET = do_POST = do_PUT = do_DELETE = handle_request


HTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
