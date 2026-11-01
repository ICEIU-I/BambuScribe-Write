"""Validate and uniformly fit source stroke glyphs into 11-unit writing cells."""

import argparse
import gzip
import hashlib
import json
import math
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("source", type=Path)
parser.add_argument("output", type=Path)
args = parser.parse_args()
source = args.source.read_bytes()
data = json.loads(source)
fitted = []
for char, glyph in data.items():
    if len(char) != 1 or len(glyph) != 3:
        raise ValueError(f"Invalid glyph: {char!r}")
    width, xx, yy = glyph
    if len(xx) != len(yy) or not xx:
        raise ValueError(f"Invalid strokes: {char!r}")
    values = [width]
    for xs, ys in zip(xx, yy):
        if len(xs) != len(ys) or len(xs) < 2:
            raise ValueError(f"Invalid stroke: {char!r}")
        values.extend(xs + ys)
    if any(not math.isfinite(v) or v < 0 for v in values):
        raise ValueError(f"Invalid coordinates: {char!r}")
    extent = max(values)
    if extent > 11:
        scale = 11 / extent
        data[char] = [
            width * scale,
            [[v * scale for v in s] for s in xx],
            [[v * scale for v in s] for s in yy],
        ]
        fitted.append(char)
payload = json.dumps(data, ensure_ascii=False, separators=(",", ":")).encode()
args.output.write_bytes(gzip.compress(payload, mtime=0))
print(json.dumps({"sha256": hashlib.sha256(source).hexdigest(),
                  "glyphs": len(data), "fitted": fitted}, ensure_ascii=False))
