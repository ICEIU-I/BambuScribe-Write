"""Conservative whole-equation routing for a single visible fraction bar."""
import re
from notation import GREEK, contains_han, flattened


def fraction_notation(text):
    text = text.strip().strip('$')
    text = re.sub(r'\\(?:tiny|scriptsize|small|displaystyle|textstyle|scriptstyle|scriptscriptstyle)(?![A-Za-z])', '', text)
    text = re.sub(r'\\(?:[,;! ]|quad\b|qquad\b)', '', text)
    text = re.sub(r'\s|~', '', text)
    for _ in range(8):
        before = text
        text = re.sub(r'\\(?:mathrm|mathit|textit|mathbf|text)\{([^{}]*)\}', r'\1', text)
        text = re.sub(r'\{\{([^{}]*)\}\}', r'{\1}', text)
        if text == before:
            break
    for name, char in sorted(GREEK.items(), key=lambda item: -len(item[0])):
        text = re.sub(r'\\' + name + r'(?![A-Za-z])', char, text)
    text = re.sub(r'([_^])([^{}])', r'\1{\2}', text)
    if len(text) > 160:
        return None
    position, fractions = 0, 0

    def sequence(depth, grouped=False):
        nonlocal position, fractions
        if depth > 4:
            raise ValueError()
        count, scriptable = 0, False
        while position < len(text) and text[position] != '}':
            if text.startswith('\\frac{', position):
                fractions += 1
                if fractions > 4:
                    raise ValueError()
                position += 5
                for _ in range(2):
                    if position >= len(text) or text[position] != '{':
                        raise ValueError()
                    position += 1
                    sequence(depth + 1, True)
                scriptable = False
            elif text[position] in '_^':
                if not scriptable:
                    raise ValueError()
                match = re.match(r'[_^]\{[A-Za-z0-9α-ωΑ-Ω+\-]+\}', text[position:])
                if not match:
                    raise ValueError()
                position += len(match[0])
                scriptable = False
            elif re.fullmatch(r"[A-Za-z0-9α-ωΑ-Ω+\-=().,/'·]", text[position]):
                scriptable = text[position].isalnum()
                position += 1
            else:
                raise ValueError()
            count += 1
        if not count:
            raise ValueError()
        if grouped:
            if position >= len(text) or text[position] != '}':
                raise ValueError()
            position += 1

    try:
        sequence(0)
        return text if position == len(text) and fractions else None
    except ValueError:
        return None


def fraction_candidate(components, labels):
    # Diagrams and mixed Chinese text keep the existing region-based pipeline.
    if not labels or len(labels) > 8 or any(contains_han(l['text']) for l in labels):
        return None
    if not any('=' in l['text'] and l['score'] >= .8 for l in labels):
        return None
    items = [tuple(map(int, s)) for s in components.stats[1:] if s[4] >= 3]
    if not items:
        return None
    x0, y0 = min(s[0] for s in items), min(s[1] for s in items)
    x1, y1 = max(s[0]+s[2] for s in items), max(s[1]+s[3] for s in items)
    if y1-y0 > 180 or x1-x0 > 1200 or x1-x0 < y1-y0:
        return None
    bars = []
    for x, y, w, h, area in items:
        if w < 8 or w < h*6 or h > max(3, (y1-y0)*.06):
            continue
        inside = [s for s in items if s[0] >= x-2 and s[0]+s[2] <= x+w+2]
        above = [s for s in inside if s[1]+s[3] <= y and s[3] > h*2]
        below = [s for s in inside if s[1] >= y+h and s[3] > h*2]
        if above and below:
            bars.append((x,y,w,h))
    return [x0,y0,x1,y1] if len(bars) == 1 else None


def agreed_fraction(result, labels):
    text = fraction_notation(result['text'])
    if text is None or result['score'] < .9 or text.count('\\frac') != 1:
        return None
    flat = flattened(text.replace('\\frac', ''))
    # The independently detected equality must survive verbatim. In particular,
    # do not accept a plausible equation that contradicts existing OCR evidence.
    equations = [flattened(l['text']) for l in labels if '=' in l['text'] and l['score'] >= .8]
    if not equations or not all(eq in flat for eq in equations):
        return None
    return text
