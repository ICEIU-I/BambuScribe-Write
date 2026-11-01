"""A small, non-executable scene format for the direct diagram fitting trial."""


def obj(properties):
    return {"type": "object", "properties": properties,
            "required": list(properties), "additionalProperties": False}


def array(items):
    return {"type": "array", "items": items}


NUMBER = {"type": "number"}
STRING = {"type": "string"}
BOOL = {"type": "boolean"}
POINT = obj({"x": NUMBER, "y": NUMBER})
SCHEMA = obj({
    "width": {"type": "integer"},
    "height": {"type": "integer"},
    "strokes": array(obj({"id": STRING, "role": STRING,
                           "points": array(POINT), "dashed": BOOL})),
    "circles": array(obj({"id": STRING, "cx": NUMBER, "cy": NUMBER,
                          "r": NUMBER, "dashed": BOOL})),
    "labels": array(obj({"text": STRING, "x": NUMBER, "y": NUMBER,
                         "width": NUMBER, "height": NUMBER})),
    "uncertainties": array(STRING),
})
