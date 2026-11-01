"""One-shot formula worker; isolated from RapidOCR's version-specific imports."""
import base64
import io
import json
import sys
from pathlib import Path
from PIL import Image
from pix2text import LatexOCR

request = json.loads(Path(sys.argv[1]).read_text())
model = LatexOCR(device="cpu", model_dir=request["model_dir"])
images = [Image.open(io.BytesIO(base64.b64decode(data))).convert("RGB") for data in request["images"]]
results = model(images, rec_config={"max_new_tokens": 64}) if images else []
Path(sys.argv[2]).write_text(json.dumps(results, ensure_ascii=False))
