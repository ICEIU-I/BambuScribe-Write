"""Local Paddle detection/text plus cropped Pix2Text formula recognition."""
import base64
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

os.environ.setdefault("OMP_NUM_THREADS", "4")
os.environ["HF_HUB_OFFLINE"] = "1"
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"
from PIL import Image, ImageOps
from rapidocr import ModelType, OCRVersion, RapidOCR
from notation import contains_han, flattened, notation, agreed_han_label
from regions import Components, normalized, overlaps
from fraction_policy import fraction_candidate, agreed_fraction


def main():
    request = json.loads(Path(sys.argv[1]).read_text())
    image = Image.open(request["image"]).convert("RGB")
    if image.width * image.height > 20_000_000:
        raise ValueError("Image exceeds local OCR pixel limit")
    model_dir = Path(request["modelDir"])
    params = {
        "Det.ocr_version": OCRVersion.PPOCRV6, "Rec.ocr_version": OCRVersion.PPOCRV6,
        "Det.model_type": ModelType.MEDIUM, "Rec.model_type": ModelType.MEDIUM,
        "Det.model_path": str(model_dir / "PP-OCRv6_det_medium.onnx"),
        "Rec.model_path": str(model_dir / "PP-OCRv6_rec_medium.onnx"),
        "Global.model_root_dir": str(model_dir), "Global.use_cls": False,
        "Global.text_score": 0.0,
        "EngineConfig.onnxruntime.intra_op_num_threads": 4,
        "EngineConfig.onnxruntime.inter_op_num_threads": 1,
    }
    engine = RapidOCR(params=params)
    scale = 2 if max(image.size) <= 1600 else 1
    output = engine(image.resize((image.width * scale, image.height * scale)), use_cls=False)
    components = Components(image)
    labels = []
    if output.boxes is not None:
        for polygon, text, score in zip(output.boxes, output.txts, output.scores):
            polygon = polygon / scale
            box = components.label_box(polygon)
            if box is None or any(overlaps(box, item["box"]) for item in labels):
                continue
            if len(labels) >= 160:
                break
            crop = ImageOps.expand(image.crop(box), border=6, fill="white")
            candidate = engine(crop, use_det=False, use_cls=False)
            rec = candidate.txts[0] if candidate.txts else ""
            confidence = float(candidate.scores[0]) if candidate.scores else 0
            label = {"box": box, "text": str(text), "score": float(score), "cropText": rec, "cropScore": confidence, "replace": False, "textFormat": ""}
            if contains_han(text) or contains_han(rec):
                if agreed_han_label(text, float(score), rec, confidence):
                    label.update(text=rec, score=min(float(score), confidence), replace=True)
            else:
                label["formula"] = True
            labels.append(label)
    formulas = [label for label in labels if label.get("formula")]
    whole_box = fraction_candidate(components, labels)
    if whole_box is not None:
        formulas.append({"box": whole_box, "whole": True})
    if formulas:
        payload = {"model_dir": request["formulaModelDir"], "images": []}
        for label in formulas:
            crop = ImageOps.expand(image.crop(label["box"]), border=6, fill="white")
            data = io.BytesIO()
            crop.save(data, format="PNG")
            payload["images"].append(base64.b64encode(data.getvalue()).decode())
        with tempfile.TemporaryDirectory(prefix="scribe-formula-") as directory:
            inp, out = Path(directory) / "input.json", Path(directory) / "output.json"
            inp.write_text(json.dumps(payload))
            subprocess.run([request["formulaPython"], str(Path(__file__).with_name("formula.py")), str(inp), str(out)], check=True, timeout=45)
            recognized = json.loads(out.read_text())
        for label, result in zip(formulas, recognized):
            if label.get("whole"):
                text = agreed_fraction(result, labels)
                if text is not None:
                    labels = [dict(box=whole_box, text=text, score=float(result["score"]), replace=True, textFormat="math")]
                continue
            math = notation(result["text"])
            if math is None or result["score"] < .8:
                continue
            flat = flattened(math)
            agreement = any(flat == flattened(value) and score >= .8 for value, score in [(label["text"], label["score"]), (label["cropText"], label["cropScore"])])
            if agreement and any(c.isalnum() for c in flat):
                label.update(text=math, score=float(result["score"]), replace=True, textFormat="math")
    result = [dict(text=label["text"], confidence=label["score"], replace=label["replace"], textFormat=label["textFormat"], backend="local-hybrid", **normalized(label["box"], *image.size)) for label in labels]
    Path(sys.argv[2]).write_text(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
