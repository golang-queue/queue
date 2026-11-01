#!/usr/bin/env python3
"""Rebuild four variants and reproduce measurements from the report's source tree.
Run: rtk proxy python3 docs/performance/2026-11-01/run.py /tmp/queue-rerun
Requires git, Go, and the implementation/test files accompanying this report.
"""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
from datetime import datetime

ROOT = Path(__file__).resolve().parents[3]
DATA = Path(__file__).resolve().parent
DEST = Path(sys.argv[1]).resolve()
DEST.mkdir(parents=True, exist_ok=False)
MANIFEST = json.loads((DATA / "manifest.json").read_text())
FIXTURES = json.loads((DATA / "fixtures.json").read_text())
archive = subprocess.check_output(["git", "archive", MANIFEST["head"]], cwd=ROOT)
variants = ["baseline", "logging", "ring", "combined"]
for variant in variants:
    d = DEST / variant
    d.mkdir()
    subprocess.run(["tar", "-xf", "-", "-C", str(d)], input=archive, check=True)
    subprocess.run(["git", "apply", "-"], input="".join(FIXTURES["baseline"]).encode(), cwd=d, check=True)
    includes = []
    if variant in ("logging", "combined"):
        includes += ["--include=queue.go", "--include=logger.go"]
    if variant in ("ring", "combined"):
        includes += ["--include=ring.go", "--include=options.go"]
    else:
        (d / "retention_compat_test.go").write_text(
            "package queue\n// Comparison-only adapter: the baseline retains the old resizing policy.\n"
            "func WithQueueMinCapacity(int) Option { return OptionFunc(func(*Options) {}) }\n"
        )
    if includes:
        subprocess.run(["git", "apply", *includes, "-"], input="".join(FIXTURES["implementation"]).encode(), cwd=d, check=True)
    shutil.copy2(ROOT / "burst_performance_test.go", d / "burst_performance_test.go")
    for name, expected in MANIFEST["variants"][variant].items():
        actual = hashlib.sha256((d / name).read_bytes()).hexdigest()
        if actual != expected:
            raise SystemExit(f"Source mismatch: {variant}/{name}; use the source version accompanying this report")

if "--prepare-only" in sys.argv[2:]:
    print(f"Verified source snapshots: {DEST}")
    raise SystemExit(0)

if "--followup" in sys.argv[2:]:
    patterns = ["^BenchmarkQueueEndToEnd$/^workers=4$", "^BenchmarkQueueBurstProducers$/^producers=4$/^retained=false$"]
    for round_no in range(6):
        order = ["baseline", "combined"] if round_no % 2 == 0 else ["combined", "baseline"]
        for variant in order:
            with (DEST / (variant + "-followup.txt")).open("a") as output:
                for pattern in patterns:
                    cmd = ["go", "test", "-run", "^$", "-bench", pattern, "-benchmem", "-benchtime=1s", "-count=1", "-cpu=12", "."]
                    output.write(f"\n# Round {round_no + 1}, {datetime.now().isoformat()}\n# {cmd!r}\n")
                    output.flush()
                    subprocess.run(cmd, cwd=DEST / variant, stdout=output, stderr=subprocess.STDOUT, check=True)
            print(f"Follow-up {round_no + 1}: {variant}", flush=True)
    raise SystemExit(0)

pattern = "^(BenchmarkHandleRetry|BenchmarkQueueEndToEnd|BenchmarkQueueAfterFnEndToEnd|BenchmarkRingBurstDrain|BenchmarkQueueBurstProducers)$"
orders = [variants, list(reversed(variants)), ["ring", "baseline", "combined", "logging"], ["logging", "combined", "baseline", "ring"]]
for round_no, order in enumerate(orders, 1):
    for variant in order:
        cmd = ["go", "test", "-run", "^$", "-bench", pattern, "-benchmem", "-benchtime=300ms", "-count=1", "-cpu=12", "."]
        with (DEST / (variant + ".txt")).open("a") as output:
            output.write(f"\n# Round {round_no}, {datetime.now().isoformat()}\n# {cmd!r}\n")
            output.flush()
            subprocess.run(cmd, cwd=DEST / variant, stdout=output, stderr=subprocess.STDOUT, check=True)
        print(f"Round {round_no}: {variant}", flush=True)
print(f"Results: {DEST}")
