package gpgbin

import (
	"bytes"
	"fmt"
	"io"

	"git.sr.ht/~rjarry/aerc/models"
)

// Sign creates a detached signature based on the contents of r
func Sign(r io.Reader, from string) ([]byte, string, error) {
	args := []string{
		"--armor",
		"--detach-sign",
		"--default-key", from,
	}

	g := newGpg(r, args)
	_ = g.cmd.Run()

	var md models.MessageDetails
	err := parseStatusFd(bytes.NewReader(g.stderr.Bytes()), &md)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse messagedetails: %w", err)
	}

	return g.stdout.Bytes(), md.Micalg, nil
}
