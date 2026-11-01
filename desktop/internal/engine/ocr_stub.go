//go:build !darwin

package engine

func recognizeImageText([]byte) []OCRBox { return nil }
