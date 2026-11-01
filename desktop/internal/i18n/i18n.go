// Package i18n renders the backend's user-facing messages.
//
// The interface wording lives in the frontend table; this one covers what Go
// produces: printer state, validation errors, file dialogs and the menu bar.
// Messages are keyed so the same string can be rendered in either language at
// any time — printer state in particular is re-rendered when the language
// changes, because it outlives the call that produced it.
package i18n

import (
	"fmt"
	"sync"
)

type Lang string

const (
	ZH Lang = "zh"
	EN Lang = "en"
)

var (
	mu      sync.RWMutex
	current = ZH
)

// Set switches the language used from now on.
func Set(l Lang) {
	mu.Lock()
	current = l
	mu.Unlock()
}

func Current() Lang {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Parse maps a stored preference to a language. "system" is resolved in the
// frontend, which can read the OS locale; anything unrecognised falls back to
// Chinese.
func Parse(preference string) Lang {
	if Lang(preference) == EN {
		return EN
	}
	return ZH
}

// T renders a message in the active language.
func T(key string, args ...any) string { return For(Current(), key, args...) }

// For renders a message in a specific language. An unknown key renders as the
// key itself, so a missing translation shows up instead of turning into an
// empty label.
func For(l Lang, key string, args ...any) string {
	format, ok := table(l)[key]
	if !ok {
		if format, ok = zh[key]; !ok {
			return key
		}
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func table(l Lang) map[string]string {
	if l == EN {
		return en
	}
	return zh
}
