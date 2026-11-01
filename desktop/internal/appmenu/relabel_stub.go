//go:build !darwin
// +build !darwin

package appmenu

// Relabel is a no-op off macOS, where the menu bar is not built from roles.
func Relabel() {}
