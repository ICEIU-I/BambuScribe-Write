//go:build darwin

package windowtheme

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
void SetWindowTheme(int dark);
*/
import "C"

// Apply keeps the native title bar in the same theme as the editor. Wails'
// runtime theme setters currently do nothing on macOS.
func Apply(dark bool) {
	value := 0
	if dark {
		value = 1
	}
	C.SetWindowTheme(C.int(value))
}
