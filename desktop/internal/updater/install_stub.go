//go:build !darwin

package updater

import "context"

func (c *Client) Prepare(context.Context, Release, func(Progress)) (*Prepared, error) {
	return nil, ErrUnsupported
}
func (p *Prepared) Start(context.Context, string) error { return ErrUnsupported }
func HandleHelper(args []string) (bool, error) {
	if len(args) > 0 && args[0] == "--bambuscribe-apply-update" {
		return true, ErrUnsupported
	}
	return false, nil
}
