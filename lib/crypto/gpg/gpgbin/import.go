package gpgbin

import (
	"io"
)

// Import runs gpg --import and thus imports both private and public keys
func Import(r io.Reader) error {
	args := []string{"--import"}
	g := newGpg(r, args)
	err := g.cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
