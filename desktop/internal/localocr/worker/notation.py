"""Accept only notation the handwriting renderer can represent faithfully."""
import re

GREEK = {
    "alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ε",
    "varepsilon": "ε", "theta": "θ", "lambda": "λ", "mu": "μ", "nu": "ν",
    "pi": "π", "rho": "ρ", "sigma": "σ", "tau": "τ", "phi": "φ", "omega": "ω",
    "Delta": "Δ", "Omega": "Ω", "Phi": "Φ", "Sigma": "Σ",
}


def notation(text):
    text = text.strip().strip("$").replace(" ", "")
    text = re.sub(r"\^\{?\\prime\}?", "'", text)
    for _ in range(8):
        before = text
        text = re.sub(r"\\(?:mathrm|mathit|textit|mathbf|text)\{([^{}]*)\}", r"\1", text)
        text = re.sub(r"\{\{([^{}]*)\}\}", r"{\1}", text)
        if before == text:
            break
    for name, char in sorted(GREEK.items(), key=lambda item: -len(item[0])):
        text = re.sub(r"\\" + name + r"(?![A-Za-z])", char, text)
    if "\\" in text or len(text) > 80:
        return None
    text = re.sub(r"([_^])([^{}])", r"\1{\2}", text)
    if text.startswith(("_", "^")) or re.search(r"\}[_^]", text):
        return None
    if not re.fullmatch(r"(?:[A-Za-z0-9α-ωΑ-Ω+\-=().,/'·]+|[_^]\{[A-Za-z0-9α-ωΑ-Ω+\-]+\})+", text):
        return None
    return text


def flattened(text):
    return re.sub(r"[_^{}\s]", "", text)


def contains_han(text):
    return any("\u3400" <= c <= "\u9fff" for c in text)



def han_label_body(text):
    # A leading figure-panel marker is prose, not a flattened formula or unit.
    return re.sub(r"^[（(](?:[A-Za-z]|[0-9]{1,2})[）)]\s*", "", text.strip())


def plain_han_label(text):
    body = han_label_body(text)
    # Common line/charge marks are ambiguous even with high OCR scores.
    return contains_han(body) and body not in {"一", "二", "十", "丨"} and all(
        "\u3400" <= c <= "\u9fff" or c.isdigit() or c.isspace()
        or c in "，。！？、：；,.!?():;（）" for c in body
    )


def agreed_han_label(text, score, crop_text, crop_score):
    if not plain_han_label(crop_text) or flattened(text) != flattened(crop_text):
        return False
    # A lone character has less context; require stronger agreement for it.
    minimum = .98 if len(han_label_body(crop_text)) == 1 else .92
    return min(score, crop_score) >= minimum
