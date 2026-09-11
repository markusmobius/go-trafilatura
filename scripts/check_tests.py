import argparse
from collections import Counter
from contextlib import nullcontext
import json
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]
TERMINAL = {"pass", "fail", "skip"}


def load_expectations():
    manifest = json.loads((ROOT / "test-files/known-differences.json").read_text(encoding="utf-8"))
    package = manifest["package"]
    expected = {}
    for entry in manifest["failures"]:
        identity = (package, entry["test"])
        if identity in expected or type(entry["assertions"]) is not int or entry["assertions"] < 1 or not entry["reason"]:
            raise ValueError(f"Invalid or duplicate known difference: {identity}")
        expected[identity] = entry
    mappings = {}
    for path in sorted((ROOT / "test-files").glob("python-2.2.0-*.json")):
        suite = json.loads(path.read_text(encoding="utf-8"))
        for entry in suite.get("inventory", []):
            if entry.get("go_tests"):
                mappings[entry["test"]] = entry["go_tests"]
            for check in entry.get("native", []):
                if check.get("go_tests"):
                    mappings[f"{entry['test']}:line_{check['line']}"] = check["go_tests"]
    return package, expected, mappings


def check_results(events, exit_code, package, expected, mappings):
    started = set()
    packages = set()
    results = {}
    package_results = {}
    assertions = Counter()
    reported_mappings = {}
    problems = []
    for event in events:
        action = event["Action"]
        owner = event.get("Package", "")
        name = event.get("Test", "")
        identity = (owner, name)
        if action == "start":
            packages.add(owner)
        elif action == "run" and name:
            started.add(identity)
        elif action in TERMINAL:
            if name:
                results[identity] = action
            else:
                package_results[owner] = action
        elif action == "build-fail":
            problems.append(f"Build failed: {owner or event.get('ImportPath', 'unknown package')}")
        output = event.get("Output", "")
        if "Error Trace:" in output:
            assertions[identity] += 1
        if output.startswith(("panic:", "fatal error:", "WARNING: DATA RACE")):
            problems.append(output.strip())
        if "Native mapping: " in output:
            source, targets = output.split("Native mapping: ", 1)[1].strip().split(" -> ", 1)
            if source in reported_mappings:
                problems.append(f"Duplicate native mapping: {source}")
            reported_mappings[source] = targets.split(", ")

    parents = set()
    for owner, name in results:
        while "/" in name:
            name = name.rsplit("/", 1)[0]
            parents.add((owner, name))
    leaves = {identity: action for identity, action in results.items() if identity not in parents}
    failures = {identity for identity, action in leaves.items() if action == "fail"}
    for identity in sorted(failures - expected.keys()):
        problems.append(f"Unexpected failure: {identity[0]} {identity[1]}")
    for identity in sorted(assertions.keys() - expected.keys()):
        problems.append(f"Unexpected failing assertion: {identity[0]} {identity[1]}")
    for identity, entry in expected.items():
        if identity not in failures:
            problems.append(f"Known difference needs review: {identity[1]} is {results.get(identity, 'missing')}")
        elif assertions[identity] != entry["assertions"]:
            problems.append(f"Changed assertion failure count: {identity[1]} expected {entry['assertions']}, got {assertions[identity]}")
    if not packages or not leaves or packages != package_results.keys() or started != results.keys():
        problems.append("Empty or incomplete Go test run")
    for owner, action in package_results.items():
        if action == "fail" and not any(identity[0] == owner for identity in failures):
            problems.append(f"Package failed without a failing leaf: {owner}")
    if exit_code != (1 if failures else 0):
        problems.append(f"Unexpected go test exit code: {exit_code}")
    if reported_mappings != mappings:
        problems.append("Native coverage mappings are missing or differ from the imported inventory")
    for target in sorted({target for targets in mappings.values() for target in targets}):
        if results.get((package, target)) not in {"pass", "fail"}:
            problems.append(f"Mapped native test did not execute: {target}")
    return Counter(leaves.values()), problems


def main():
    parser = argparse.ArgumentParser(description="Run all Go tests and reject differences outside the reviewed compatibility manifest.")
    parser.add_argument("--go", default="go", help="Go executable")
    parser.add_argument("--timeout", default="5m", help="Go per-package test timeout")
    parser.add_argument("--log", type=Path, help="Write the unmodified go test JSON stream to this file")
    arguments = parser.parse_args()
    package, expected, mappings = load_expectations()
    command = [arguments.go, "test", "-json", "-mod=readonly", "./...", "-count=1", "-timeout", arguments.timeout]
    if arguments.log:
        arguments.log.parent.mkdir(parents=True, exist_ok=True)
    output_file = arguments.log.open("w", encoding="utf-8") if arguments.log else nullcontext()
    events = []
    with output_file as log, subprocess.Popen(command, cwd=ROOT, stdout=subprocess.PIPE, encoding="utf-8") as process:
        for line in process.stdout:
            if log is not None:
                log.write(line)
            events.append(json.loads(line))
        exit_code = process.wait()
    counts, problems = check_results(events, exit_code, package, expected, mappings)
    print(f"Go suite: {counts['pass']} pass, {counts['fail']} fail, {counts['skip']} skip; {len(mappings)} native mappings (not test results).")
    for identity, entry in expected.items():
        print(f"Known difference: {identity[1]}: {entry['reason']}")
    if problems:
        for problem in problems:
            print(problem, file=sys.stderr)
        return 1
    print("PASS: only the reviewed compatibility differences remain; go test itself still reports their failures.")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError) as error:
        sys.exit(str(error))