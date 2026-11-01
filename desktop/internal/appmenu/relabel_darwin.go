//go:build darwin
// +build darwin

package appmenu

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
void RelabelRoleMenus(const char *json);
*/
import "C"

import (
	"encoding/json"
	"unsafe"

	"bambuscribe/desktop/internal/i18n"
)

// Relabel renames the English titles Wails' role menus ship with. Keys are the
// AppKit selector the item drives, with the trailing colon dropped, plus the
// bare titles of the menus that hold a submenu instead of an action.
func Relabel() {
	labels := map[string]string{
		"Edit":   i18n.T("menu.edit"),
		"Window": i18n.T("menu.window"),

		"About":                 i18n.T("menu.about"),
		"Hide":                  i18n.T("menu.hide"),
		"hideOtherApplications": i18n.T("menu.hideOthers"),
		"unhideAllApplications": i18n.T("menu.showAll"),
		"Quit":                  i18n.T("menu.quit"),

		"undo":                i18n.T("menu.undo"),
		"redo":                i18n.T("menu.redo"),
		"cut":                 i18n.T("menu.cut"),
		"copy":                i18n.T("menu.copy"),
		"paste":               i18n.T("menu.paste"),
		"pasteAsRichText":     i18n.T("menu.pasteAndMatch"),
		"delete":              i18n.T("menu.delete"),
		"selectAll":           i18n.T("menu.selectAll"),
		"startSpeaking":       i18n.T("menu.startSpeaking"),
		"stopSpeaking":        i18n.T("menu.stopSpeaking"),
		"performMiniaturize":  i18n.T("menu.minimize"),
		"performZoom":         i18n.T("menu.zoom"),
		"enterFullScreenMode": i18n.T("menu.fullScreen"),
		"toggleFullScreen":    i18n.T("menu.fullScreen"),
	}
	payload, err := json.Marshal(labels)
	if err != nil {
		return
	}
	text := C.CString(string(payload))
	defer C.free(unsafe.Pointer(text))
	C.RelabelRoleMenus(text)
}
