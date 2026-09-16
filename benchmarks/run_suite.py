"""Reproducible LIAF/Go/Python model benchmark (Python standard library only)."""
from __future__ import annotations

import argparse
import hashlib
import html
import json
import os
from pathlib import Path
import re
import statistics
import subprocess
import sys
import tempfile
import time
import urllib.request

from http_scenario import evaluate_http

ROOT = Path(__file__).resolve().parents[1]


def load_dotenv(path=ROOT / ".env"):
    if path.exists():
        for line in path.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                k, v = line.split("=", 1)
                os.environ.setdefault(k.strip(), v.strip().strip("\"'"))


load_dotenv()


def request_model(provider, model, messages, max_tokens, timeout, endpoint=None):
    headers = {"Content-Type": "application/json"}
    if provider == "openai":
        headers["Authorization"] = "Bearer " + os.environ["OPENAI_API_KEY"]
        url = endpoint or "https://api.openai.com/v1/responses"
        body = {"model": model, "input": messages, "max_output_tokens": max_tokens,
                "store": False}
    elif provider == "anthropic":
        headers.update({"x-api-key": os.environ["ANTHROPIC_API_KEY"],
                        "anthropic-version": "2023-06-01"})
        url = endpoint or "https://api.anthropic.com/v1/messages"
        body = {"model": model, "max_tokens": max_tokens,
                "system": messages[0]["content"], "messages": messages[1:]}
    elif provider == "gemini":
        key = os.environ.get("GEMINI_API_KEY") or os.environ.get("GOOGLE_API_KEY", "")
        url = endpoint or f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={key}"
        system_parts = [{"text": m["content"]} for m in messages if m["role"] == "system"]
        contents = [{"role": "model" if m["role"] == "assistant" else "user",
                     "parts": [{"text": m["content"]}]} for m in messages if m["role"] != "system"]
        body = {
            "contents": contents,
            "generationConfig": {
                "maxOutputTokens": max_tokens,
                "temperature": 0.0,
                "thinkingConfig": {"thinkingBudget": 0}
            }
        }
        if system_parts:
            body["systemInstruction"] = {"parts": system_parts}
    else:
        url = endpoint or "http://localhost:11434/api/chat"
        body = {"model": model, "messages": messages, "stream": False,
                "options": {"temperature": 0, "seed": 1, "num_predict": max_tokens}}
    data = json.dumps(body).encode()
    req = urllib.request.Request(url, data, headers)
    start = time.perf_counter()
    with urllib.request.urlopen(req, timeout=timeout) as response:
        raw = json.load(response)
    elapsed = time.perf_counter() - start
    if provider == "openai":
        text = "".join(part.get("text", "") for item in raw.get("output", [])
                       if item.get("type") == "message" for part in item.get("content", [])
                       if part.get("type") == "output_text")
        usage = raw.get("usage", {})
    elif provider == "anthropic":
        text = "".join(part.get("text", "") for part in raw.get("content", [])
                       if part.get("type") == "text")
        usage = raw.get("usage", {})
    elif provider == "gemini":
        text = "".join(part.get("text", "") for cand in raw.get("candidates", [])
                       for part in cand.get("content", {}).get("parts", []))
        meta = raw.get("usageMetadata", {})
        usage = {"input_tokens": meta.get("promptTokenCount"),
                 "output_tokens": meta.get("candidatesTokenCount")}
    else:
        text = raw["message"]["content"]
        usage = {"input_tokens": raw.get("prompt_eval_count"),
                 "output_tokens": raw.get("eval_count")}
    if not text.strip():
        raise ValueError("Provider returned no source code")
    return text, usage, elapsed


def extract_code(text):
    text = text.strip()
    blocks = re.findall(r"```[^\n]*\n(.*?)```", text, re.S)
    if len(blocks) >= 1:
        return blocks[0].strip() + "\n"
    if text.startswith("```"):
        lines = text.splitlines()
        return "\n".join(lines[1:]).strip() + "\n"
    return text.strip() + "\n"


def child_environment():
    # Do not forward provider/deploy credentials to generated programs.
    return {k: v for k, v in os.environ.items()
            if not any(s in k.upper() for s in ("TOKEN", "SECRET", "API_KEY", "PASSWORD"))}


def command(args, cwd, timeout):
    start = time.perf_counter()
    env = child_environment()
    try:
        result = subprocess.run(args, cwd=cwd, env=env, capture_output=True,
                                text=True, encoding="utf-8", errors="replace", timeout=timeout)
        return result.returncode, result.stdout, result.stderr, time.perf_counter() - start
    except subprocess.TimeoutExpired as exc:
        def decode(value):
            return value.decode("utf-8", errors="replace") if isinstance(value, bytes) else value or ""
        return 124, decode(exc.stdout), decode(exc.stderr) + "\nexecution timeout", time.perf_counter() - start
    except OSError as exc:
        return 127, "", str(exc), time.perf_counter() - start


def evaluate(source, language, task, compiler, execute, timeout):
    steps = []

    def run_step(stage, args, cwd):
        code, out, err, duration = command(args, cwd, timeout)
        steps.append({"stage": stage, "command": args, "cwd": str(cwd),
                      "exit_code": code, "stdout": out, "stderr": err, "seconds": duration})
        return code, out, err, duration

    with tempfile.TemporaryDirectory(prefix="liaf-benchmark-") as directory:
        work = Path(directory)
        extension = {"liaf": ".liaf", "go": ".go", "python": ".py"}[language]
        path = work / ("main" + extension)
        path.write_text(source, encoding="utf-8")
        executable = work / ("program.exe" if os.name == "nt" else "program")
        check = ([compiler, "check", str(path), "--json"] if language == "liaf" else
                 [sys.executable, "-m", "py_compile", str(path)] if language == "python" else
                 ["go", "build", "-o", str(executable), str(path)])
        code, out, err, check_time = run_step("check", check, ROOT)
        result = {"check_passed": code == 0, "compiled": False, "passed": None,
                  "check_seconds": check_time, "compile_seconds": 0, "run_seconds": 0,
                  "feedback": (out + err)[-16000:], "steps": steps,
                  "failed_stage": "check" if code else None}
        if code:
            return result
        if language == "liaf":
            code, out, err, duration = run_step("compile",
                [compiler, "build", str(path), "-o", str(executable)], ROOT)
            result.update(compile_seconds=duration, feedback=(out + err)[-16000:])
            if code:
                result["failed_stage"] = "compile"
                return result
        elif language == "go":
            result["compile_seconds"] = check_time
        result["compiled"] = True
        if not execute:
            return result
        launch = [sys.executable, str(path)] if language == "python" else [str(executable)]
        if task.get("scenario") == "task-api":
            http = evaluate_http(launch, work, timeout, child_environment())
            steps.extend(http.pop("steps"))
            result.update(http)
            return result
        outcomes = []
        failures = []
        for index, case in enumerate(task.get("cases", []), start=1):
            code, out, err, duration = run_step("run", launch + case.get("args", []), work)
            result["run_seconds"] += duration
            passed = code == case.get("exit_code", 0) and out.replace("\r\n", "\n") == case["stdout"]
            outcomes.append(passed)
            steps[-1].update(case=index, expected_exit_code=case.get("exit_code", 0),
                             expected_stdout=case["stdout"], passed=passed)
            if not passed:
                result["failed_stage"] = "run"
                failures.append(f"Behavior test {index} failed: expected exit={case.get('exit_code', 0)}, "
                                f"stdout={case['stdout']!r}; actual exit={code}, stdout={out!r}; stderr={err!r}")
        if failures:
            result["feedback"] = "\n".join(failures)[-16000:]
        result["passed"] = all(outcomes) if outcomes else None
        return result


def summarize(records):
    groups = {}
    for row in records:
        groups.setdefault((row["provider"], row["model"], row["language"]), []).append(row)
    result = []
    for (provider, model, language), rows in sorted(groups.items()):
        measured = [r for r in rows if r.get("passed") is not None]
        costs = [r.get("cost_usd") for r in rows]
        result.append({"provider": provider, "model": model, "language": language,
                       "runs": len(rows), "behavior_runs": len(measured),
                       "pass_at_1": (sum(r["passed"] and r["attempts"] == 1 for r in measured) / len(measured)
                                     if measured else None),
                       "success_rate": sum(bool(r["passed"]) for r in measured) / len(measured) if measured else None,
                       "mean_attempts": statistics.mean(r["attempts"] for r in rows),
                       "cost_usd": sum(costs) if all(c is not None for c in costs) else None,
                       "cost_per_success_usd": (sum(costs) / sum(bool(r["passed"]) for r in measured)
                                                if all(c is not None for c in costs) and any(r["passed"] for r in measured)
                                                else None)})
    return result


def chart(summary, key, title, target):
    measured = [r for r in summary if r.get(key) is not None]
    height = max(120, 60 + 45 * len(measured))
    elements = [f'<svg xmlns="http://www.w3.org/2000/svg" width="900" height="{height}" role="img">',
                f'<title>{html.escape(title)}</title><rect width="100%" height="100%" fill="white"/>',
                f'<text x="20" y="28" font-family="sans-serif" font-size="18">{html.escape(title)}</text>']
    maximum = max([r[key] for r in measured] + [0.001])
    for index, row in enumerate(measured):
        y = 60 + 45 * index
        label = html.escape(row["model"] + " / " + row["language"])
        width = 400 * row[key] / maximum
        elements.extend([f'<text x="20" y="{y+18}" font-family="sans-serif">{label}</text>',
                         f'<rect x="370" y="{y}" width="{width:.2f}" height="25" fill="#245fba"/>',
                         f'<text x="{380+width:.2f}" y="{y+18}" font-family="sans-serif">{row[key]:.4f}</text>'])
    if not measured:
        elements.append('<text x="20" y="75" font-family="sans-serif">No measured data available</text>')
    target.write_text("\n".join(elements + ["</svg>"]), encoding="utf-8")


def main():
    # Windows consoles/pipes may use cp1252; diagnostics must not abort a run.
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            stream.reconfigure(errors="backslashreplace")
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--provider", choices=["openai", "anthropic", "ollama", "gemini"], default="gemini")
    p.add_argument("--model")
    p.add_argument("--language", choices=["liaf", "go", "python"], default="liaf")
    p.add_argument("--suite", type=Path, default=ROOT / "benchmarks/tasks.json")
    p.add_argument("--compiler", default=str(ROOT / ("liafc.exe" if os.name == "nt" else "liafc")))
    p.add_argument("--output", type=Path, default=ROOT / "benchmarks/results/local")
    p.add_argument("--repetitions", type=int, default=1)
    p.add_argument("--attempts", type=int, default=3)
    p.add_argument("--max-tokens", type=int, default=4096)
    p.add_argument("--timeout", type=float, default=60)
    p.add_argument("--input-price", type=float, help="USD per million tokens; record price date separately")
    p.add_argument("--output-price", type=float)
    p.add_argument("--max-calls", type=int, default=9)
    p.add_argument("--execute", action="store_true", help="execute generated code in your configured isolation")
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--endpoint")
    p.add_argument("--overwrite", action="store_true", help="overwrite existing results directory")
    p.add_argument("-v", "--verbose", action="store_true", help="imprime feedback e erros detalhados de cada tentativa")
    args = p.parse_args()
    if min(args.attempts, args.repetitions, args.max_tokens, args.max_calls) < 1:
        p.error("counts must be positive")
    tasks = json.loads(args.suite.read_text(encoding="utf-8"))
    if args.dry_run:
        print(json.dumps({"tasks": [t["id"] for t in tasks], "provider": args.provider,
                          "model": args.model, "language": args.language,
                          "maximum_calls": min(len(tasks)*args.repetitions*args.attempts,args.max_calls)}, indent=2))
        return
    if not args.model:
        p.error("--model is required; the runner never silently chooses a model")
    args.output.mkdir(parents=True, exist_ok=True)
    mode = "w" if args.overwrite else "x"
    result_file = (args.output / "records.jsonl").open(mode, encoding="utf-8")
    docs = (ROOT / ".docs/conceitos/IMPLEMENTATION.md").read_text(encoding="utf-8") if args.language == "liaf" else "Use only the standard library."
    system = f"Write a complete {args.language} program. Return only source code.\n" + docs
    records, calls = [], 0
    config = vars(args).copy()
    (args.output / "config.json").write_text(json.dumps(config, default=str, indent=2), encoding="utf-8")
    (args.output / "suite.json").write_text(json.dumps(tasks, ensure_ascii=False, indent=2), encoding="utf-8")
    for repetition in range(args.repetitions):
        for task in tasks:
            print(f"--> [{args.language}] Tarefa: {task['id']} (rep {repetition + 1}/{args.repetitions})")
            task_system = system
            if args.language == "liaf" and task.get("liaf_context"):
                task_system += "\n" + (args.suite.parent / task["liaf_context"]).read_text(encoding="utf-8")
            (args.output / f"{task['id']}-system.txt").write_text(task_system, encoding="utf-8")
            messages = [{"role": "system", "content": task_system}, {"role": "user", "content": task["prompt"]}]
            row = {"provider": args.provider, "model": args.model, "language": args.language,
                   "task": task["id"], "repetition": repetition, "attempts": 0, "passed": False,
                   "input_tokens": 0, "output_tokens": 0, "cost_usd": None, "history": [],
                   "prompt_sha256": hashlib.sha256(json.dumps(messages).encode()).hexdigest()}
            for attempt in range(args.attempts):
                if calls >= args.max_calls:
                    row["stop_reason"] = "call_limit"
                    break
                calls += 1
                row["attempts"] += 1
                print(f"    Tentativa {attempt + 1}/{args.attempts}...", end=" ", flush=True)
                try:
                    text, usage, duration = request_model(args.provider,args.model,messages,args.max_tokens,args.timeout,args.endpoint)
                except Exception as exc:
                    row["stop_reason"] = "provider_error: " + type(exc).__name__
                    print(f"Erro no provedor: {exc}")
                    break
                source = extract_code(text)
                source_name = f"{task['id']}-{repetition}-{attempt}.{args.language}"
                (args.output / source_name).write_text(source, encoding="utf-8")
                check = evaluate(source,args.language,task,args.compiler,args.execute,args.timeout)
                detail = {"source": source_name, "usage": usage, "generation_seconds": duration, **check}
                log_name = source_name + ".json"
                (args.output / log_name).write_text(json.dumps(detail, ensure_ascii=False, indent=2), encoding="utf-8")
                row["history"].append({"log": log_name, **detail})
                for key in ("input_tokens", "output_tokens"):
                    row[key] = row[key] + usage[key] if row[key] is not None and usage.get(key) is not None else None
                row["passed"] = check["passed"] if check["compiled"] else False
                status = ("PASSOU" if row["passed"] else
                          f"Falhou: {check['failed_stage']}" if check["failed_stage"] else
                          "Compilou (comportamento não avaliado)")
                print(f"{status}")
                if check["failed_stage"] or args.verbose:
                    print(f"      Fonte: {args.output / source_name}")
                    print(f"      Log: {args.output / log_name}")
                if args.verbose:
                    for step in check["steps"]:
                        print(f"      [{step['stage']}] exit={step['exit_code']} ({step['seconds']:.3f}s)")
                        print("      Comando: " + json.dumps(step["command"], ensure_ascii=False))
                        if "expected_stdout" in step:
                            print(f"      Esperado: exit={step['expected_exit_code']}; stdout={step['expected_stdout']!r}")
                        if step["stage"] == "http":
                            print(f"      Caso: {step['case']}; esperado HTTP {step['expected_status']}; passou={step['passed']}")
                        if step["stage"] == "run":
                            print(f"      stdout: {step['stdout']!r}")
                        elif step["stdout"]:
                            print(step["stdout"].rstrip())
                        if step["stderr"]:
                            print(f"      stderr: {step['stderr']!r}")
                if check["passed"] or (check["compiled"] and not args.execute):
                    break
                messages.extend([{"role":"assistant","content":text},{"role":"user","content":"Correct this program. Validation feedback:\n"+check["feedback"]}])
            if args.input_price is not None and args.output_price is not None and row["input_tokens"] is not None and row["output_tokens"] is not None:
                row["cost_usd"] = (row["input_tokens"]*args.input_price+row["output_tokens"]*args.output_price)/1e6
            records.append(row)
            result_file.write(json.dumps(row,ensure_ascii=False)+"\n");result_file.flush()
    result_file.close()
    summary = summarize(records)
    (args.output/"summary.json").write_text(json.dumps(summary,indent=2),encoding="utf-8")
    chart(summary,"cost_per_success_usd","API cost per successful task (USD)",args.output/"cost_comparison.svg")
    chart(summary,"pass_at_1","Behavioral pass@1",args.output/"pass_rate.svg")
    print(json.dumps(summary,indent=2))


if __name__ == "__main__":
    main()
