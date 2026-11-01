// Package appmenu builds the native menu bar.
//
// Wails can only contribute Go callbacks, never arbitrary AppKit selectors, so
// the menus whose items drive the responder chain — the app menu, Edit and
// Window — have to come from Wails roles. Those roles ship hardcoded English
// titles; Relabel renames them to the active language once the menu is set.
//
// Menu items are never disabled: AppKit's automatic validation (enabled
// because Wails leaves autoenablesItems at its default) re-enables any item
// whose target responds to its action, which every Wails item does. So the
// frontend decides whether an action is available and reports the reason.
package appmenu

import (
	"bambuscribe/desktop/internal/i18n"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

// State is the part of the editor the menu mirrors back.
type State struct {
	Ruled  bool   `json:"ruled"`
	Dark   bool   `json:"dark"`
	Device bool   `json:"device"`
	Zoom   int    `json:"zoom"`
	Lang   string `json:"lang"`
}

// Build renders the menu in the active language. send forwards an action name
// to the frontend, which owns every decision the menu only triggers.
func Build(s State, send func(action string)) *menu.Menu {
	text := func(key, accel, action string) *menu.MenuItem {
		return menu.Text(i18n.T(key), parse(accel), func(*menu.CallbackData) { send(action) })
	}
	check := func(key, accel, action string, on bool) *menu.MenuItem {
		return menu.Checkbox(i18n.T(key), on, parse(accel), func(*menu.CallbackData) { send(action) })
	}
	radio := func(key, action, value string) *menu.MenuItem {
		return menu.Radio(i18n.T(key), s.Lang == value, nil, func(*menu.CallbackData) { send(action) })
	}
	tab := func(key, action string, on bool) *menu.MenuItem {
		return menu.Radio(i18n.T(key), on, nil, func(*menu.CallbackData) { send(action) })
	}

	result := menu.NewMenu()
	result.Append(menu.AppMenu())
	result.Append(menu.SubMenu(i18n.T("menu.file"), menu.NewMenuFromItems(
		text("menu.newProject", "cmdorctrl+n", "new"),
		text("menu.save", "cmdorctrl+s", "save"),
		text("menu.saveAs", "cmdorctrl+shift+s", "saveAs"),
		text("menu.openProject", "cmdorctrl+o", "open"),
		text("menu.reveal", "", "reveal"),
		menu.Separator(),
		text("menu.exportPage", "cmdorctrl+e", "exportPage"),
		text("menu.exportAll", "cmdorctrl+shift+e", "exportAll"),
		menu.Separator(),
		text("menu.upload", "cmdorctrl+u", "upload"),
	)))
	result.Append(menu.EditMenu())
	result.Append(menu.SubMenu(i18n.T("menu.view"), menu.NewMenuFromItems(
		check("menu.ruled", "", "ruled", s.Ruled),
		menu.Separator(),
		text("menu.zoomIn", "cmdorctrl+plus", "zoomIn"),
		text("menu.zoomOut", "cmdorctrl+-", "zoomOut"),
		text("menu.zoomReset", "cmdorctrl+0", "zoomReset"),
		menu.Separator(),
		tab("menu.writeTab", "tab:write", !s.Device),
		tab("menu.deviceTab", "tab:device", s.Device),
		menu.Separator(),
		menu.Radio(i18n.T("menu.themeLight"), !s.Dark, nil, func(*menu.CallbackData) { send("theme:light") }),
		menu.Radio(i18n.T("menu.themeDark"), s.Dark, nil, func(*menu.CallbackData) { send("theme:dark") }),
	)))
	result.Append(menu.SubMenu(i18n.T("menu.settings"), menu.NewMenuFromItems(
		menu.SubMenu(i18n.T("menu.language"), menu.NewMenuFromItems(
			radio("menu.languageSystem", "lang:system", "system"),
			radio("menu.languageZh", "lang:zh", "zh"),
			radio("menu.languageEn", "lang:en", "en"),
		)),
	)))
	result.Append(menu.WindowMenu())
	result.Append(menu.SubMenu(i18n.T("menu.help"), menu.NewMenuFromItems(
		text("menu.guide", "", "guide"),
	)))
	return result
}

// parse turns "cmdorctrl+shift+s" into an accelerator. An unparsable or empty
// spec leaves the item without a shortcut rather than failing the whole menu.
func parse(spec string) *keys.Accelerator {
	if spec == "" {
		return nil
	}
	accelerator, err := keys.Parse(spec)
	if err != nil {
		return nil
	}
	return accelerator
}
