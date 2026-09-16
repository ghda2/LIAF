"""External, sequential HTTP oracle for the task API benchmark (stdlib only)."""
import json
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid


def evaluate_http(launch, work, timeout, env):
    """Run the same binary across restarts, keeping its private working directory."""
    steps = []
    started = time.perf_counter()
    process = None
    logs = []
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
    base = f"http://127.0.0.1:{port}"
    (work / "public").mkdir(exist_ok=True)

    def fetch(method, path, payload=None, raw=None, request_timeout=None):
        body = raw if raw is not None else (json.dumps(payload).encode() if payload is not None else None)
        request = urllib.request.Request(base + path, data=body, method=method,
                                         headers={"Content-Type": "application/json"})
        try:
            response = opener.open(request, timeout=request_timeout or min(timeout, 5))
        except urllib.error.HTTPError as exc:
            response = exc
        with response:
            data = response.read(1024 * 1024 + 1)
            if len(data) > 1024 * 1024:
                raise AssertionError("HTTP response exceeded 1 MiB")
            return response.status, response.headers.get("Content-Type", ""), data.decode("utf-8")

    def stop():
        nonlocal process
        if process is None:
            return
        current = process
        try:
            if current.poll() is None:
                current.terminate()
            try:
                current.wait(timeout=3)
            except subprocess.TimeoutExpired:
                current.kill()
                current.wait(timeout=3)
        finally:
            entry, stdout, stderr, start = logs[-1]
            stdout.seek(0)
            stderr.seek(0)
            entry.update(exit_code=current.poll(), seconds=time.perf_counter() - start,
                         stdout=stdout.read().decode("utf-8", errors="replace"),
                         stderr=stderr.read().decode("utf-8", errors="replace"))
            stdout.close()
            stderr.close()
            process = None

    def start():
        nonlocal process
        entry = {"stage": "server", "command": launch + [str(port)], "cwd": str(work),
                 "exit_code": None, "stdout": "", "stderr": "", "seconds": 0,
                 "restart": len(logs), "stopped_by_runner": True}
        steps.append(entry)
        stdout, stderr = tempfile.TemporaryFile(), tempfile.TemporaryFile()
        start_time = time.perf_counter()
        try:
            process = subprocess.Popen(entry["command"], cwd=work, env=env,
                                       stdout=stdout, stderr=stderr)
        except OSError:
            stdout.close()
            stderr.close()
            raise
        logs.append((entry, stdout, stderr, start_time))
        deadline = time.monotonic() + min(timeout, 15)
        last = "no response"
        while time.monotonic() < deadline:
            if process.poll() is not None:
                raise AssertionError(f"Server exited during startup: {process.returncode}")
            try:
                status, _, body = fetch("GET", "/health", request_timeout=0.5)
                if status == 200 and json.loads(body) == {"status": "ok"}:
                    return
                last = f"GET /health: status={status}, body={body!r}"
            except (OSError, ValueError, urllib.error.URLError) as exc:
                last = str(exc)
            time.sleep(0.05)
        raise AssertionError(f"Server did not become ready: {last}")

    def case(name, method, path, status, expected=None, payload=None, raw=None, error=False):
        def require(condition, message):
            if not condition:
                raise AssertionError(message)

        start_time = time.perf_counter()
        entry = {"stage": "http", "case": name, "command": [method, base + path],
                 "exit_code": 1, "stdout": "", "stderr": "", "seconds": 0,
                 "request_json": payload, "request_raw": raw.decode() if raw is not None else None,
                 "expected_status": status, "expected_json": expected, "expect_error": error,
                 "passed": False}
        steps.append(entry)
        try:
            actual_status, content_type, body = fetch(method, path, payload, raw)
            entry.update(actual_status=actual_status, content_type=content_type, stdout=body)
            require(actual_status == status, f"expected HTTP {status}, got {actual_status}: {body!r}")
            require(content_type.split(";", 1)[0].strip().lower() == "application/json", "expected application/json")
            actual = json.loads(body)
            entry["actual_json"] = actual
            if error:
                require(isinstance(actual, dict) and isinstance(actual.get("error"), str) and actual["error"], "expected a nonempty JSON error string")
            else:
                # JSON comparison distinguishes booleans from integers and ignores object key order.
                require(json.dumps(actual, sort_keys=True) == json.dumps(expected, sort_keys=True), f"expected {expected!r}, got {actual!r}")
            entry.update(passed=True, exit_code=0)
        except (AssertionError, OSError, ValueError, urllib.error.URLError) as exc:
            entry["stderr"] = str(exc)
            raise AssertionError(f"{name} ({method} {path}): {exc}") from exc
        finally:
            entry["seconds"] = time.perf_counter() - start_time

    passed, feedback = False, ""
    try:
        start()
        case("health", "GET", "/health", 200, {"status": "ok"})
        case("empty list", "GET", "/tasks", 200, [])
        for name, payload, raw in [("malformed JSON", None, b"{"), ("missing title", {}, None),
                                   ("empty title", {"title": ""}, None), ("wrong title type", {"title": 9}, None)]:
            case(name, "POST", "/tasks", 400, payload=payload, raw=raw, error=True)
        case("invalid creates do not mutate", "GET", "/tasks", 200, [])
        title = 'Revisar "API" e ação ' + uuid.uuid4().hex[:8]
        first = {"id": 1, "title": title, "done": False}
        second = {"id": 2, "title": "Segunda " + uuid.uuid4().hex[:8], "done": False}
        case("create first", "POST", "/tasks", 201, first, {"title": title})
        case("create second", "POST", "/tasks", 201, second, {"title": second["title"]})
        case("list created", "GET", "/tasks", 200, [first, second])
        case("read first", "GET", "/tasks/1", 200, first)
        case("missing ID", "GET", "/tasks/999", 404, error=True)
        case("invalid ID", "GET", "/tasks/abc", 400, error=True)
        case("invalid update", "PUT", "/tasks/1", 400, payload={"title": "", "done": True}, error=True)
        case("wrong done type", "PUT", "/tasks/1", 400, payload={"title": "valid", "done": "yes"}, error=True)
        case("failed update preserves item", "GET", "/tasks/1", 200, first)
        first = {**first, "title": "Atualizada " + uuid.uuid4().hex[:8], "done": True}
        case("update", "PUT", "/tasks/1", 200, first, {"title": first["title"], "done": True})
        case("update missing ID", "PUT", "/tasks/999", 404, payload={"title": "valid", "done": True}, error=True)
        stop()
        start()
        case("list after restart", "GET", "/tasks", 200, [first, second])
        case("read after restart", "GET", "/tasks/1", 200, first)
        case("delete", "DELETE", "/tasks/2", 200, {"deleted": True})
        case("deleted item missing", "GET", "/tasks/2", 404, error=True)
        case("delete missing ID", "DELETE", "/tasks/999", 404, error=True)
        stop()
        start()
        case("deletion persisted", "GET", "/tasks", 200, [first])
        third = {"id": 3, "title": "Depois do reinício", "done": False}
        case("ID counter persisted", "POST", "/tasks", 201, third, {"title": third["title"]})
        case("final list", "GET", "/tasks", 200, [first, third])
        passed = True
    except (AssertionError, OSError, ValueError, urllib.error.URLError) as exc:
        feedback = str(exc)
    finally:
        stop()
    if not passed:
        server_errors = "\n".join(entry["stdout"] + entry["stderr"] for entry in steps if entry["stage"] == "server")
        feedback = (feedback + "\n" + server_errors)[-16000:]
    return {"passed": passed, "failed_stage": None if passed else "http", "feedback": feedback,
            "run_seconds": time.perf_counter() - started, "steps": steps}
