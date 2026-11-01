#!/usr/bin/env python3
"""Summarize raw Go benchmark output. Optional argument: result directory."""
import json
from pathlib import Path
import statistics
import sys

folder = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parent
results = {}
for variant in ("baseline", "logging", "ring", "combined"):
    samples = {}
    for line in (folder / (variant + ".txt")).read_text().splitlines():
        parts = line.split()
        if not parts or not parts[0].startswith("Benchmark") or len(parts) < 8:
            continue
        name = parts[0].rsplit("-", 1)[0]
        metrics = {parts[i + 1]: float(parts[i]) for i in range(2, len(parts) - 1, 2)}
        samples.setdefault(name, []).append(metrics)
    results[variant] = {
        name: {"samples": values, "summary": {
            unit: {"median": statistics.median(v[unit] for v in values),
                   "min": min(v[unit] for v in values),
                   "max": max(v[unit] for v in values)}
            for unit in values[0]
        }} for name, values in samples.items()
    }
(folder / "summary.json").write_text(json.dumps(results, indent=2) + "\n")
for name in results["baseline"]:
    before = results["baseline"][name]["summary"]
    after = results["combined"][name]["summary"]
    a, b = before["ns/op"]["median"], after["ns/op"]["median"]
    print(f"{name}: {a:g} -> {b:g} ns/op ({(b/a-1)*100:+.1f}%), "
          f'{before["B/op"]["median"]:g} -> {after["B/op"]["median"]:g} B/op, '
          f'{before["allocs/op"]["median"]:g} -> {after["allocs/op"]["median"]:g} allocs/op')
