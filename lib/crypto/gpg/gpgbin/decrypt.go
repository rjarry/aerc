package gpgbin

import (
	"bytes"
	"errors"
	"io"

	"git.sr.ht/~rjarry/aerc/models"
)

// Decrypt runs gpg --decrypt on the contents of r. If the packet is signed,
// the signature is also verified
func Decrypt(r io.Reader) (*models.MessageDetails, error) {
	md := new(models.MessageDetails)
	orig, err := io.ReadAll(r)
	if err != nil {
		return md, err
	}
	args := []string{"--decrypt"}
	g := newGpg(bytes.NewReader(orig), args)
	_ = g.cmd.Run()
	// Always parse stdout, even if there was an error running command.
	// We'll find the error in the parsing
	err = parseStatusFd(bytes.NewReader(g.stderr.Bytes()), md)

	if errors.Is(err, NoValidOpenPgpData) {
		md.Body = bytes.NewReader(orig)
		return md, nil
	} else if err != nil {
		return nil, err
	}

	md.Body = bytes.NewReader(g.stdout.Bytes())
	return md, nil
}
