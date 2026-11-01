package application

import (
	"bambuscribe/desktop/internal/appmenu"
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/i18n"
	"bambuscribe/desktop/internal/printer"
	"bambuscribe/desktop/internal/storage"
	"bambuscribe/desktop/internal/windowtheme"
	"context"
	"os"
	"path/filepath"
	"sync"

	wailsapp "github.com/wailsapp/wails/v2/pkg/application"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// MenuState mirrors the editor state the native menu shows back. The frontend
// owns all of it and pushes a copy whenever something checkable changes.
type MenuState struct {
	Ruled  bool   `json:"ruled"`
	Dark   bool   `json:"dark"`
	Device bool   `json:"device"`
	Zoom   int    `json:"zoom"`
	Lang   string `json:"lang"`
}

type App struct {
	persistenceMu sync.Mutex
	editorReady   bool
	closeAllowed  bool
	calculations  calculationState
	diagrams      diagramImportState
	imports       importState
	updates       updateState
	mu            sync.Mutex
	ctx           context.Context
	font          *engine.Font
	fontError     error
	store         *storage.Store
	warning       string
	fleet         *printer.Fleet
	devicesMu     sync.Mutex
	documents     map[string]*engine.Document
	documentOrder []string
	projectFile   string
	shell         *wailsapp.Application
	menuState     MenuState
}

func New(shell *wailsapp.Application) (*App, func(context.Context), func(context.Context)) {
	a := &App{documents: map[string]*engine.Document{}, shell: shell}
	a.font, a.fontError = engine.EmbeddedFont()
	a.fleet = printer.NewFleet(func() printer.Link { return &printer.Network{} }, printer.Discover, func(s printer.State) {
		a.mu.Lock()
		ctx := a.ctx
		a.mu.Unlock()
		if ctx != nil {
			runtime.EventsEmit(ctx, "printer:state", s)
		}
	})
	return a, a.startup, a.shutdown
}

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
	dir := os.Getenv("BAMBUSCRIBE_CONFIG_DIR")
	var err error
	if dir == "" {
		dir, err = os.UserConfigDir()
		if err != nil {
			a.warning = i18n.T("app.noConfigDir", err)
			return
		}
		dir = filepath.Join(dir, "BambuScribeWrite")
	}
	a.store, err = storage.Open(dir)
	if err != nil {
		a.warning = i18n.T("app.configReadFailed")
	}
	a.setProjectPath(a.store.Snapshot().ProjectPath)
	a.mu.Lock()
	a.menuState.Dark = a.store.Snapshot().Preferences.Theme == "dark"
	dark := a.menuState.Dark
	a.mu.Unlock()
	if a.shell != nil {
		windowtheme.Apply(dark)
	}
	a.startTableImport(dir)
	for _, d := range a.store.Snapshot().Devices {
		a.fleet.Add(d.ID)
	}
	// "system" needs the OS locale, which only the webview can read; the
	// frontend reports the resolved language right after it starts.
	if pref := a.store.Snapshot().Preferences.Language; pref == "zh" || pref == "en" {
		i18n.Set(i18n.Parse(pref))
		a.fleet.SetLanguage()
	}
}

func (a *App) shutdown(context.Context) {
	a.CancelUpdate()
	a.CancelDiagramRecognition()
	a.CancelCalculationRecognition()
	a.CancelTableRecognition()
	if a.imports.server != nil {
		a.imports.server.Close()
	}
	a.fleet.Close()
}

// SetLanguage switches the backend messages and rebuilds the menu bar.
// `preference` is what the user chose ("system", "zh" or "en") and is kept for
// the menu's radio state; `resolved` is the concrete language the frontend
// derived from the OS locale, since only the webview can read it.
func (a *App) SetLanguage(preference string, resolved string) {
	a.mu.Lock()
	a.menuState.Lang = preference
	a.mu.Unlock()
	i18n.Set(i18n.Parse(resolved))
	a.fleet.SetLanguage()
	a.rebuildMenu()
}

// SetMenuState rebuilds the menu so its checkmarks match the editor. It is
// called on every change the menu mirrors, so it stays cheap.
func (a *App) SetMenuState(state MenuState) {
	a.mu.Lock()
	lang := a.menuState.Lang
	themeChanged := a.menuState.Dark != state.Dark
	a.menuState = state
	a.menuState.Lang = lang
	a.mu.Unlock()
	if themeChanged && a.shell != nil {
		windowtheme.Apply(state.Dark)
	}
	a.rebuildMenu()
}

func (a *App) rebuildMenu() {
	a.mu.Lock()
	shell, state := a.shell, a.menuState
	a.mu.Unlock()
	if shell == nil {
		return
	}
	shell.SetApplicationMenu(appmenu.Build(appmenu.State{
		Ruled:  state.Ruled,
		Dark:   state.Dark,
		Device: state.Device,
		Zoom:   state.Zoom,
		Lang:   state.Lang,
	}, a.MenuAction))
	// The roles above arrive with English titles; rename them in place.
	appmenu.Relabel()
}

// MenuAction forwards a native menu selection to the frontend, which owns
// every decision the menu only triggers.
func (a *App) MenuAction(action string) {
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, "menu:action", action)
	}
}

type PrinterSettings struct {
	IP            string `json:"ip"`
	Serial        string `json:"serial"`
	HasAccessCode bool   `json:"hasAccessCode"`
}
type Bootstrap struct {
	ProjectPath      string              `json:"projectPath"`
	Preferences      storage.Preferences `json:"preferences"`
	Printer          PrinterSettings     `json:"printer"`
	State            printer.State       `json:"state"`
	Warning          string              `json:"warning"`
	Devices          []PrinterDevice     `json:"devices"`
	SelectedDeviceID string              `json:"selectedDeviceId"`
}

func (a *App) Bootstrap() Bootstrap {
	d, path, warning := a.workspaceSnapshot()
	list := a.Printers()
	return Bootstrap{ProjectPath: path, Preferences: d.Preferences, Printer: PrinterSettings{d.Printer.IP, d.Printer.Serial, d.Printer.AccessCode != ""}, State: a.PrinterState(), Warning: warning, Devices: list.Devices, SelectedDeviceID: list.SelectedDeviceID}
}
