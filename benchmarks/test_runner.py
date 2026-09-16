import json
import tempfile
import unittest
from pathlib import Path
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


if __name__ == "__main__": unittest.main()
