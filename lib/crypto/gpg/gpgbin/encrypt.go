package gpgbin

import (
	"bytes"
	"fmt"
	"io"

	"git.sr.ht/~rjarry/aerc/models"
)

// Encrypt runs gpg --encrypt [--sign] -r [recipient]
func Encrypt(r io.Reader, to []string, from string) ([]byte, error) {
	args := []string{
		"--armor",
	}
	if from != "" {
		args = append(args, "--sign", "--default-key", from)
	}
	for _, rcpt := range to {
		args = append(args, "--recipient", rcpt)
	}
	args = append(args, "--encrypt", "-")

	g := newGpg(r, args)
	_ = g.cmd.Run()
	var md models.MessageDetails
	err := parseStatusFd(bytes.NewReader(g.stderr.Bytes()), &md)
	if err != nil {
		return nil, fmt.Errorf("gpg: failure to encrypt: %w. check public key(s)", err)
	}

	return g.stdout.Bytes(), nil
}
