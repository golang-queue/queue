#!/usr/bin/env python3
"""Compare clean base with this PR. Output directory must not exist."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
DATA = Path(__file__).resolve().parent
DEST = Path(sys.argv[1]).resolve()
DEST.mkdir(parents=True, exist_ok=False)
base = json.loads((DATA / "manifest.json").read_text())["head"]
archive = subprocess.check_output(["git", "archive", base], cwd=ROOT)
versions = ["pr-base", "pr-head"]
manifest = {"base": base, "files": {}}
for version in versions:
    d = DEST / version
    d.mkdir()
    subprocess.run(["tar", "-xf", "-", "-C", str(d)], input=archive, check=True)
    if version == "pr-head":
        for name in ("queue.go", "logger.go", "ring.go", "options.go"):
            shutil.copy2(ROOT / name, d / name)
    else:
        (d / "retention_compat_test.go").write_text(
            "package queue\nfunc WithQueueMinCapacity(int) Option { return OptionFunc(func(*Options) {}) }\n")
    shutil.copy2(ROOT / "burst_performance_test.go", d / "burst_performance_test.go")
    manifest["files"][version] = {str(p.relative_to(d)): hashlib.sha256(p.read_bytes()).hexdigest()
                                   for p in sorted(d.rglob("*.go"))}
(DEST / "pr-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
pattern = "^(BenchmarkHandleRetry|BenchmarkQueueEndToEnd|BenchmarkQueueAfterFnEndToEnd|BenchmarkRingBurstDrain|BenchmarkQueueBurstProducers)$"
for round_no, order in enumerate((versions, list(reversed(versions))), 1):
    for version in order:
        cmd = ["go", "test", "-run", "^$", "-bench", pattern, "-benchmem", "-benchtime=300ms", "-count=3", "-cpu=12", "."]
        with (DEST / (version + ".txt")).open("a") as output:
            output.write(f"\n# Round {round_no}: {cmd!r}\n")
            output.flush()
            subprocess.run(cmd, cwd=DEST / version, stdout=output, stderr=subprocess.STDOUT, check=True)
        print(f"Round {round_no}: {version}", flush=True)
print(DEST)
