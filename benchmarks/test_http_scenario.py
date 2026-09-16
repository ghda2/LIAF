import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import run_suite as runner
from http_scenario import evaluate_http

REFERENCE = Path(__file__).parent / "fixtures/task_api.py"


class HttpScenarioTests(unittest.TestCase):
    def run_fixture(self, replacement=None):
        source = REFERENCE.read_text(encoding="utf-8")
        if replacement:
            source = source.replace(*replacement)
        return runner.evaluate(source, "python", {"scenario": "task-api"}, "unused", True, 5)

    def test_accepts_full_contract_and_restarts(self):
        result = self.run_fixture()
        self.assertTrue(result["passed"], result["feedback"])
        servers = [step for step in result["steps"] if step["stage"] == "server"]
        self.assertEqual(len(servers), 3)
        self.assertTrue(all(step["exit_code"] is not None for step in servers))

    def test_rejects_memory_only_persistence(self):
        result = self.run_fixture(("PERSIST = True", "PERSIST = False"))
        self.assertFalse(result["passed"])
        self.assertIn("list after restart", result["feedback"])

    def test_rejects_reusing_deleted_ids(self):
        result = self.run_fixture(("RESET_COUNTER = False", "RESET_COUNTER = True"))
        self.assertFalse(result["passed"])
        self.assertIn("ID counter persisted", result["feedback"])

    def test_rejects_invalid_input_acceptance(self):
        result = self.run_fixture(("ACCEPT_EMPTY = False", "ACCEPT_EMPTY = True"))
        self.assertFalse(result["passed"])
        self.assertIn("empty title", result["feedback"])

    def test_startup_failure_is_reported_and_process_reaped(self):
        with tempfile.TemporaryDirectory() as directory:
            result = evaluate_http([sys.executable, "-c", "raise SystemExit(7)"],
                                   Path(directory), 2, runner.child_environment())
        self.assertFalse(result["passed"])
        self.assertIn("startup", result["feedback"])
        self.assertEqual(result["steps"][0]["exit_code"], 7)

    def test_startup_timeout_stops_process(self):
        with tempfile.TemporaryDirectory() as directory:
            result = evaluate_http([sys.executable, "-c", "import time; time.sleep(60)"],
                                   Path(directory), 0.2, runner.child_environment())
        self.assertFalse(result["passed"])
        self.assertIn("did not become ready", result["feedback"])
        self.assertIsNotNone(result["steps"][0]["exit_code"])


if __name__ == "__main__":
    unittest.main()
