"""Separate contained label components from long crossing diagram strokes."""
import cv2
import numpy as np


class Components:
    def __init__(self, image):
        rgb = np.asarray(image.convert("RGB"))
        gray = rgb @ np.array([0.299, 0.587, 0.114])
        self.height, self.width = gray.shape
        _, self.labels, self.stats, _ = cv2.connectedComponentsWithStats((gray < 230).astype("uint8"), 8)

    def label_box(self, polygon):
        points = np.asarray(polygon)
        x0, y0 = np.maximum(0, np.floor(points.min(axis=0))).astype(int)
        x1, y1 = np.minimum([self.width, self.height], np.ceil(points.max(axis=0))).astype(int)
        if x1 <= x0 or y1 <= y0:
            return None
        # Huge detections are not treated as labels; no diagram-sized erasures.
        if y1 - y0 > max(80, self.height * .12):
            return None
        selected = []
        for index in np.unique(self.labels[y0:y1, x0:x1]):
            if index == 0:
                continue
            x, y, w, h, area = self.stats[index]
            if area < 3 or x < x0 - 2 or y < y0 - 2 or x + w > x1 + 2 or y + h > y1 + 2:
                continue
            selected.append((int(x), int(y), int(x + w), int(y + h)))
        if not selected:
            return None
        return [min(b[0] for b in selected), min(b[1] for b in selected), max(b[2] for b in selected), max(b[3] for b in selected)]


def normalized(box, width, height):
    x0, y0, x1, y1 = box
    return {"x": x0 / width, "y": y0 / height, "w": (x1 - x0) / width, "h": (y1 - y0) / height}


def overlaps(a, b):
    intersection = max(0, min(a[2], b[2]) - max(a[0], b[0])) * max(0, min(a[3], b[3]) - max(a[1], b[1]))
    smaller = min((a[2] - a[0]) * (a[3] - a[1]), (b[2] - b[0]) * (b[3] - b[1]))
    return smaller > 0 and intersection > smaller * .5
