//go:build darwin

package application

import "os/exec"

// reveal selects the file itself rather than the folder, so the user sees
// which project was just written among its neighbours.
func reveal(path string) error {
	return exec.Command("open", "-R", path).Run()
}
