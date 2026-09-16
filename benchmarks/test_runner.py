import json
import contextlib
import io
import subprocess
import tempfile
import unittest
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from unittest.mock import patch
import run_suite as runner


class RunnerTests(unittest.TestCase):
    def test_summary_counts_retries_and_failures(self):
        rows = [dict(provider="local",model="m",language="liaf",passed=p,attempts=a,cost_usd=c)
                for p,a,c in [(True,1,0.01),(True,2,0.02),(False,3,0.03)]]
        row=runner.summarize(rows)[0]
        self.assertAlmostEqual(row["pass_at_1"],1/3)
        self.assertAlmostEqual(row["cost_per_success_usd"],0.03)

    def test_unmeasured_is_not_zero(self):
        row=runner.summarize([dict(provider="local",model="m",language="go",passed=None,attempts=1,cost_usd=None)])[0]
        self.assertIsNone(row["pass_at_1"])
        self.assertIsNone(row["cost_usd"])

    def test_extract(self):
        self.assertEqual(runner.extract_code("```liaf\n(module m)\n```"),"(module m)\n")

    def test_python_oracle_rejects_wrong_output(self):
        task={"cases":[{"stdout":"42\n"}]}
        self.assertTrue(runner.evaluate('print(42)',"python",task,"unused",True,10)["passed"])
        self.assertFalse(runner.evaluate('print(41)',"python",task,"unused",True,10)["passed"])

    def test_empty_chart_is_explicit(self):
        with tempfile.TemporaryDirectory() as d:
            target=Path(d)/"chart.svg"
            runner.chart([],"pass_at_1","Measured",target)
            self.assertIn("No measured data",target.read_text())

    def test_compile_failure_preserves_check_diagnostics(self):
        with patch.object(runner, "command", side_effect=[
            (0, '{"status":"success"}', "check warning", 0.1),
            (1, "", "build error", 0.2),
        ]):
            result = runner.evaluate("source", "liaf", {}, "liafc", True, 10)
        self.assertEqual(result["failed_stage"], "compile")
        self.assertEqual(result["steps"][0]["stderr"], "check warning")
        self.assertEqual(result["steps"][1]["exit_code"], 1)
        self.assertFalse(result["compiled"])

    def test_timeout_preserves_partial_output(self):
        with patch.object(runner.subprocess, "run", side_effect=subprocess.TimeoutExpired(
            "program", 1, output=b"partial output", stderr=b"partial error"
        )):
            code, out, err, _ = runner.command(["program"], runner.ROOT, 1)
        self.assertEqual(code, 124)
        self.assertEqual(out, "partial output")
        self.assertIn("partial error", err)
        self.assertIn("timeout", err)

    def test_console_handles_unicode_diagnostics_with_windows_encoding(self):
        buffer = io.BytesIO()
        console = io.TextIOWrapper(buffer, encoding="cp1252")
        with patch.object(sys, "argv", ["run_suite.py", "--dry-run"]), contextlib.redirect_stdout(console):
            runner.main()
            print("\u2713 compiled")
            console.flush()
        self.assertIn(b"\\u2713 compiled", buffer.getvalue())

    def test_attempt_log_records_expected_and_actual_output(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "results"
            suite = Path(directory) / "tasks.json"
            suite.write_text(json.dumps([{"id": "sample", "prompt": "Print 42",
                                         "cases": [{"stdout": "42\n"}]}]), encoding="utf-8")
            argv = ["run_suite.py", "--model", "test", "--language", "python",
                    "--suite", str(suite), "--output", str(output), "--execute", "--attempts", "1", "-v"]
            with patch.object(sys, "argv", argv), patch.object(runner, "request_model", return_value=(
                "print(41)", {"input_tokens": 1, "output_tokens": 1}, 0.1
            )), contextlib.redirect_stdout(io.StringIO()):
                runner.main()
            record = json.loads((output / "records.jsonl").read_text(encoding="utf-8"))
            detail = json.loads((output / record["history"][0]["log"]).read_text(encoding="utf-8"))
            self.assertEqual(detail["failed_stage"], "run")
            self.assertEqual(detail["steps"][-1]["expected_stdout"], "42\n")
            self.assertEqual(detail["steps"][-1]["stdout"], "41\n")
            self.assertIn("expected exit=0", detail["feedback"])


if __name__ == "__main__": unittest.main()
