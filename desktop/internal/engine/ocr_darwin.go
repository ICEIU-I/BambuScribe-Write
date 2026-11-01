//go:build darwin

package engine

/*
#cgo CFLAGS: -fblocks
#cgo LDFLAGS: -framework Foundation -framework Vision
#include <stdlib.h>
char *bs_ocr_text(const void *bytes, size_t length);
*/
import "C"

import (
	"encoding/json"
	"unsafe"
)

func recognizeImageText(data []byte) []OCRBox {
	if len(data) == 0 {
		return nil
	}
	ptr := C.bs_ocr_text(unsafe.Pointer(&data[0]), C.size_t(len(data)))
	if ptr == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(ptr))
	var boxes []OCRBox
	if json.Unmarshal([]byte(C.GoString(ptr)), &boxes) != nil {
		return nil
	}
	return boxes
}
