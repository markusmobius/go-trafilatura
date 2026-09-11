import unittest

from check_tests import check_results, load_expectations


PACKAGE = "example.org/extractor"
KNOWN = "Test_Extract/known"


def event(action, name="", output="", package=PACKAGE):
    return {"Action": action, "Package": package, "Test": name, "Output": output}


def known_run():
    return [
        event("start"),
        event("run", "Test_Extract"),
        event("run", KNOWN),
        event("output", KNOWN, "    Error Trace: extractor_test.go:42\n"),
        event("fail", KNOWN),
        event("fail", "Test_Extract"),
        event("fail"),
    ]


class CheckResultsTests(unittest.TestCase):
    def setUp(self):
        self.expected = {(PACKAGE, KNOWN): {"assertions": 1, "reason": "Documented backend difference"}}

    def check(self, events, exit_code=1, mappings=None):
        return check_results(events, exit_code, PACKAGE, self.expected, mappings or {})

    def test_known_failure_excludes_parent_from_counts(self):
        counts, problems = self.check(known_run())
        self.assertEqual(counts, {"fail": 1})
        self.assertEqual(problems, [])

    def test_unexpected_leaf_or_parent_assertion_is_rejected(self):
        events = known_run()
        events[5:5] = [event("run", "Test_Extract/new"), event("fail", "Test_Extract/new")]
        self.assertTrue(any("Unexpected failure" in problem for problem in self.check(events)[1]))
        events = known_run()
        events.insert(5, event("output", "Test_Extract", "Error Trace: extractor_test.go:99\n"))
        self.assertTrue(any("Unexpected failing assertion" in problem for problem in self.check(events)[1]))

    def test_changed_assertion_counts_are_rejected(self):
        for additional in (-1, 1):
            with self.subTest(additional=additional):
                events = known_run()
                if additional < 0:
                    del events[3]
                else:
                    events.insert(3, event("output", KNOWN, "Error Trace: extractor_test.go:99\n"))
                self.assertTrue(any("Changed assertion failure count" in problem for problem in self.check(events)[1]))

    def test_fixed_skipped_and_missing_differences_require_review(self):
        for outcome in ("pass", "skip", "missing"):
            with self.subTest(outcome=outcome):
                events = [event("start"), event("run", "Test_Control"), event("pass", "Test_Control"), event("pass")]
                if outcome != "missing":
                    events[1:1] = [event("run", KNOWN), event(outcome, KNOWN)]
                self.assertTrue(any("needs review" in problem for problem in self.check(events, 0)[1]))

    def test_build_failure_cannot_hide_behind_known_failures(self):
        events = known_run() + [event("start", package="example.org/other"), event("fail", package="example.org/other")]
        self.assertTrue(any("Package failed" in problem for problem in self.check(events)[1]))
        events = known_run() + [event("build-fail", package="example.org/other")]
        self.assertTrue(any("Build failed" in problem for problem in self.check(events)[1]))

    def test_crash_and_race_are_rejected(self):
        for output in ("panic: crash", "fatal error: broken runtime", "WARNING: DATA RACE"):
            with self.subTest(output=output):
                events = known_run() + [event("output", KNOWN, output)]
                self.assertIn(output, self.check(events)[1])
        self.assertTrue(any("exit code" in problem for problem in self.check(known_run(), 2)[1]))

    def test_empty_and_truncated_runs_are_rejected(self):
        for events in ([], known_run()[:-1], known_run()[:4]):
            with self.subTest(events=events):
                self.assertIn("Empty or incomplete Go test run", self.check(events)[1])

    def test_native_mappings_are_not_test_results(self):
        mappings = {"unit_tests.py/test_example:line_42": ["Test_Native"]}
        events = known_run()
        events[5:5] = [
            event("output", "Test_Extract", "Native mapping: unit_tests.py/test_example:line_42 -> Test_Native\n"),
            event("run", "Test_Native"),
            event("pass", "Test_Native"),
        ]
        counts, problems = self.check(events, mappings=mappings)
        self.assertEqual(counts, {"fail": 1, "pass": 1})
        self.assertEqual(problems, [])
        events[7] = event("skip", "Test_Native")
        self.assertTrue(any("did not execute" in problem for problem in self.check(events, mappings=mappings)[1]))
        self.assertTrue(any("missing or differ" in problem for problem in self.check(known_run(), mappings=mappings)[1]))

    def test_repository_manifest_and_mappings_are_valid(self):
        package, expected, mappings = load_expectations()
        self.assertTrue(package)
        self.assertTrue(expected)
        self.assertTrue(mappings)


if __name__ == "__main__":
    unittest.main()