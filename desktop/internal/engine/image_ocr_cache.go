package engine

import (
	"crypto/sha256"
	"sync"
)

// Legacy projects are kept intact on disk. Cache recognition by source bytes
// so their automatic regions can be refreshed without rerunning Vision on
// every text edit, image resize or pen-setting change. Keep at most eight images.
var imageOCRCache struct {
	sync.Mutex
	entries []imageOCRCacheEntry
}

type imageOCRCacheEntry struct {
	key   [32]byte
	boxes []OCRBox
}

func cachedImageOCR(raw []byte, recognize func() []OCRBox) []OCRBox {
	key := sha256.Sum256(raw)
	imageOCRCache.Lock()
	defer imageOCRCache.Unlock()
	for _, entry := range imageOCRCache.entries {
		if entry.key == key {
			return append([]OCRBox(nil), entry.boxes...)
		}
	}
	boxes := recognize()
	if len(imageOCRCache.entries) == 8 {
		imageOCRCache.entries = imageOCRCache.entries[1:]
	}
	imageOCRCache.entries = append(imageOCRCache.entries, imageOCRCacheEntry{key, append([]OCRBox(nil), boxes...)})
	return boxes
}
