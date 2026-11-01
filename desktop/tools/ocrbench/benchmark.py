"""Run local OCR models on a diagram or manually annotated label crops.

This tool does not modify the desktop app, projects, or printer settings.
Model libraries download their weights on first use; image inference is local.
"""

import argparse
import hashlib
import importlib.metadata
import json
import os
from pathlib import Path
import subprocess
import time


def arguments():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("backend", choices=["vision", "ppocrv5", "ppocrv6", "pix2text"])
    parser.add_argument("--image", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--labels", type=Path, help="Crop benchmark; bypasses detection")
    parser.add_argument("--scale", type=int, default=1, choices=[1, 2])
    parser.add_argument("--formula-width", type=int, default=768)
    parser.add_argument("--model-cache", type=Path, default=Path.home() / ".cache/bambuscribe/ocr-eval/models")
    return parser.parse_args()


def serial(value):
    if hasattr(value, "tolist"):
        return value.tolist()
    if hasattr(value, "value"):
        return value.value
    raise TypeError(f"Cannot serialize {type(value)}")


def package_versions():
    names = ("rapidocr", "pix2text", "onnxruntime", "torch", "transformers", "numpy", "pillow")
    versions = {}
    for name in names:
        try:
            versions[name] = importlib.metadata.version(name)
        except importlib.metadata.PackageNotFoundError:
            pass
    return versions


def run_vision(args):
    if args.labels or args.scale != 1:
        raise ValueError("Vision probe uses the unmodified whole image")
    probe = Path(__file__).with_name("vision.swift")
    start = time.perf_counter()
    result = subprocess.run(["swift", str(probe), str(args.image.resolve())], check=True, capture_output=True, text=True)
    return {"seconds_including_swift_startup": time.perf_counter() - start, "regions": json.loads(result.stdout)}


def rapid_engine(args):
    from rapidocr import ModelType, OCRVersion, RapidOCR

    version = OCRVersion.PPOCRV5 if args.backend == "ppocrv5" else OCRVersion.PPOCRV6
    kind = ModelType.SERVER if args.backend == "ppocrv5" else ModelType.MEDIUM
    params = {
        "Det.ocr_version": version, "Rec.ocr_version": version,
        "Det.model_type": kind, "Rec.model_type": kind,
        "Global.model_root_dir": str(args.model_cache),
        "Global.use_det": not bool(args.labels), "Global.use_cls": False,
        # Keep rejected readings for diagnosis, not automatic replacement.
        "Global.text_score": 0.0,
        "EngineConfig.onnxruntime.intra_op_num_threads": 4,
        "EngineConfig.onnxruntime.inter_op_num_threads": 1,
    }
    return RapidOCR(params=params), params


def run_models(args):
    from PIL import Image, ImageOps

    image = Image.open(args.image).convert("RGB")
    if args.scale != 1:
        image = image.resize((image.width * args.scale, image.height * args.scale))
    config = {}
    start = time.perf_counter()
    if args.backend == "pix2text":
        if args.labels:
            from pix2text import LatexOCR
            engine = LatexOCR(device="cpu")
        else:
            from pix2text import TextFormulaOCR
            engine = TextFormulaOCR.from_config(device="cpu", enable_spell_checker=False)
    else:
        engine, config = rapid_engine(args)
    setup_seconds = time.perf_counter() - start
    if args.labels:
        results = []
        for label in json.loads(args.labels.read_text()):
            if args.backend == "pix2text" and label["id"] == "caption":
                continue  # Formula recognizer is not a Chinese caption model.
            box = [coordinate * args.scale for coordinate in label["box"]]
            crop = ImageOps.expand(image.crop(box), border=6 * args.scale, fill="white")
            start = time.perf_counter()
            if args.backend == "pix2text":
                output = engine(crop, rec_config={"max_new_tokens": 64})
            else:
                raw = engine(crop, use_det=False, use_cls=False)
                output = {"text": raw.txts[0] if raw.txts else "", "score": float(raw.scores[0]) if raw.scores else 0}
            results.append({**label, "recognized": output, "seconds": time.perf_counter() - start})
        return {"setup_seconds": setup_seconds, "config": config, "crops": results}

    # RapidOCR initializes sessions lazily; separate the first invocation from
    # the warmed measurement so download time is not presented as inference.
    results = []
    for state in ("first", "warm"):
        start = time.perf_counter()
        if args.backend == "pix2text":
            regions = engine.recognize(image, return_text=False, resized_shape=args.formula_width)
            regions = [{**r, "position": r["position"] / args.scale} for r in regions]
        else:
            raw = engine(image, use_cls=False)
            regions = [] if raw.boxes is None else [
                {"text": text, "score": float(score), "box": box / args.scale}
                for box, text, score in zip(raw.boxes, raw.txts, raw.scores)
            ]
        results.append({"state": state, "seconds": time.perf_counter() - start, "regions": regions})
    return {"setup_seconds": setup_seconds, "config": config, "runs": results}


def main():
    args = arguments()
    os.environ.setdefault("OMP_NUM_THREADS", "4")
    os.environ.setdefault("HF_HUB_DISABLE_TELEMETRY", "1")
    result = {
        "backend": args.backend, "image_sha256": hashlib.sha256(args.image.read_bytes()).hexdigest(),
        "scale": args.scale, "formula_width": args.formula_width, "versions": package_versions(),
        "result": run_vision(args) if args.backend == "vision" else run_models(args),
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, default=serial, ensure_ascii=False, indent=2) + "\n")
    print(f"Saved {args.output}")


if __name__ == "__main__":
    main()
