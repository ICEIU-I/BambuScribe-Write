"""Render model geometry verbatim; no image tracing, fitting, or point snapping."""
import argparse
import html
import json
import math
import re
from pathlib import Path

from PIL import Image, ImageDraw


def validate(scene):
    width, height = scene["width"], scene["height"]
    assert 1 <= width <= 8192 and 1 <= height <= 8192

    def point(x, y):
        assert math.isfinite(x) and math.isfinite(y), "Non-finite coordinate"
        assert 0 <= x <= width and 0 <= y <= height, "Geometry outside image"

    ids = set()
    for group in ["strokes", "circles", "labels", "uncertainties"]:
        assert len(scene[group]) <= 500, "Scene too complex"
    for stroke in scene["strokes"]:
        assert 2 <= len(stroke["points"]) <= 500
        assert stroke["id"] not in ids, "Duplicate shape ID"
        ids.add(stroke["id"])
        for p in stroke["points"]:
            point(p["x"], p["y"])
    for circle in scene["circles"]:
        assert circle["id"] not in ids, "Duplicate shape ID"
        ids.add(circle["id"])
        x, y, r = circle["cx"], circle["cy"], circle["r"]
        assert math.isfinite(r) and r > 0
        point(x - r, y - r)
        point(x + r, y + r)
    for label in scene["labels"]:
        assert len(label["text"]) <= 500 and label["text"].strip()
        assert label["width"] > 0 and label["height"] > 0
        point(label["x"], label["y"])
        point(label["x"] + label["width"], label["y"] + label["height"])


def characters(text):
    """Apply the same simple sub/superscript scale used by image_math.go."""
    position = 0
    while position < len(text):
        char = text[position]
        if char in "_^":
            match = re.match(r"([_^])\{([^{}]+)\}", text[position:])
            if not match or position == 0:
                raise ValueError("Unsupported formula: " + text)
            for value in match[2]:
                yield value, .65, 4.4 if char == "_" else -4.4
            position += len(match[0])
        else:
            if char in "{}\\":
                raise ValueError("Unsupported formula: " + text)
            yield char, 1., 0.
            position += 1


def label_strokes(label, glyphs):
    letters, total, top, bottom = [], 0., math.inf, -math.inf
    for char, size, shift in characters(label["text"]):
        if char.isspace():
            letters.append((None, 0, 0, 4.95 * size, size, shift))
            total += 4.95 * size
            continue
        glyph = glyphs[char]
        points = [p for stroke in glyph["Strokes"] for p in stroke]
        left, right = min(p["x"] * size for p in points), max(p["x"] * size for p in points)
        top = min(top, min(p["y"] * size + shift for p in points))
        bottom = max(bottom, max(p["y"] * size + shift for p in points))
        advance = max(glyph["Width"] * size, right - left) + 1.76 * size
        if 0x3400 <= ord(char) <= 0x9fff:
            advance = max(advance, 11. * size)
        letters.append((glyph, left, right - left, advance, size, shift))
        total += advance
    last = letters[-1]
    total -= last[3] - last[2]
    scale = min(label["width"] / total if total > 0 else math.inf,
                label["height"] / (bottom - top) if bottom > top else math.inf)
    assert math.isfinite(scale), "Empty glyph"
    x = label["x"]
    y = label["y"] + (label["height"] - (bottom - top) * scale) / 2
    result = []
    for glyph, left, _, advance, size, shift in letters:
        if glyph:
            for stroke in glyph["Strokes"]:
                result.append([(x + (p["x"] * size - left) * scale,
                                y + (p["y"] * size + shift - top) * scale) for p in stroke])
        x += advance * scale
    return result


def dashed_parts(points, dash=8., gap=5.):
    position = 0.
    for a, b in zip(points, points[1:]):
        distance = math.dist(a, b)
        if distance == 0:
            continue
        done = 0.
        while done < distance - 1e-8:
            phase = position % (dash + gap)
            ink = phase < dash - 1e-8
            remaining = (dash if ink else dash + gap) - phase
            if remaining < 1e-8:
                remaining = gap if ink else dash
            length = min(remaining, distance - done)
            if ink:
                yield [(a[0] + (b[0] - a[0]) * t / distance,
                        a[1] + (b[1] - a[1]) * t / distance)
                       for t in (done, done + length)]
            done += length
            position += length


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("scene", type=Path)
    parser.add_argument("--glyphs", type=Path, required=True)
    args = parser.parse_args()
    scene, glyphs = json.loads(args.scene.read_text()), json.loads(args.glyphs.read_text())
    validate(scene)
    width, height, scale = scene["width"], scene["height"], 3
    image = Image.new("RGB", (width * scale, height * scale), "white")
    draw = ImageDraw.Draw(image)
    elements, pen_strokes = [], []

    def path(points, dashed=False, role="geometry"):
        data = "M " + " L ".join(f"{x:g},{y:g}" for x, y in points)
        dash = ' stroke-dasharray="8 5"' if dashed else ""
        elements.append(f'<path d="{data}"{dash} data-role="{html.escape(role, quote=True)}"/>')
        parts = list(dashed_parts(points)) if dashed else [points]
        for part in parts:
            draw.line([(x * scale, y * scale) for x, y in part],
                      fill="black", width=5, joint="curve")
            pen_strokes.append(part)

    for stroke in scene["strokes"]:
        path([(p["x"], p["y"]) for p in stroke["points"]],
             stroke["dashed"], stroke["role"])
    for circle in scene["circles"]:
        x, y, r = circle["cx"], circle["cy"], circle["r"]
        count = max(24, math.ceil(2 * math.pi * r / .7))
        points = [(x + r * math.cos(i * 2 * math.pi / count),
                   y + r * math.sin(i * 2 * math.pi / count)) for i in range(count + 1)]
        # PNG/pen polylines sample the exact circle. SVG keeps a true circle.
        before = len(elements)
        path(points, circle["dashed"])
        dash = ' stroke-dasharray="8 5"' if circle["dashed"] else ""
        elements[before] = f'<circle cx="{x:g}" cy="{y:g}" r="{r:g}"{dash}/>'
    for label in scene["labels"]:
        for stroke in label_strokes(label, glyphs):
            path(stroke, role="text")
    svg = (f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
           f'viewBox="0 0 {width} {height}"><rect width="100%" height="100%" fill="white"/>'
           '<g fill="none" stroke="black" stroke-width="1.7" stroke-linecap="round" '
           'stroke-linejoin="round">' + "\n".join(elements) + '</g></svg>')
    args.scene.with_suffix(".svg").write_text(svg)
    image.resize((width, height), Image.Resampling.LANCZOS).save(args.scene.with_suffix(".png"))
    args.scene.with_suffix(".strokes.json").write_text(json.dumps(pen_strokes))
    print(json.dumps({"validated": True, "strokes": len(pen_strokes),
                      "png": str(args.scene.with_suffix('.png'))}))


if __name__ == "__main__":
    main()
