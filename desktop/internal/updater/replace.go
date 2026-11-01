package updater

import (
	"fmt"
	"os"
	"path/filepath"
)

type Prepared struct {
	Target  string
	Bundle  string
	WorkDir string
}

func (p *Prepared) Cleanup() { _ = os.RemoveAll(p.WorkDir) }

// Keep the previous bundle until the replacement has successfully launched.
// All paths share a filesystem, so both moves are atomic renames.
func (p *Prepared) replace(launch func(string) error) error {
	backup := filepath.Join(p.WorkDir, "previous.app")
	if err := os.Rename(p.Target, backup); err != nil {
		return fmt.Errorf("%w: %v", ErrInstall, err)
	}
	if err := os.Rename(p.Bundle, p.Target); err != nil {
		if restore := os.Rename(backup, p.Target); restore != nil {
			return fmt.Errorf("%w: restore from %s: %v", ErrInstall, backup, restore)
		}
		return fmt.Errorf("%w: %v", ErrInstall, err)
	}
	if err := launch(p.Target); err != nil {
		if move := os.Rename(p.Target, p.Bundle); move != nil {
			return fmt.Errorf("%w: restore from %s: %v", ErrInstall, backup, move)
		}
		if restore := os.Rename(backup, p.Target); restore != nil {
			return fmt.Errorf("%w: restore from %s: %v", ErrInstall, backup, restore)
		}
		_ = launch(p.Target)
		return fmt.Errorf("%w: %v", ErrInstall, err)
	}
	p.Cleanup()
	return nil
}
