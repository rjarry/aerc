package gpgbin

import (
	"io"
)

// Import runs gpg --import-ownertrust and thus imports trusts for keys
func ImportOwnertrust(r io.Reader) error {
	args := []string{"--import-ownertrust"}
	g := newGpg(r, args)
	err := g.cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
