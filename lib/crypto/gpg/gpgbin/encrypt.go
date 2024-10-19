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
	outRdr := bytes.NewReader(g.stdout.Bytes())
	var md models.MessageDetails
	err := parse(outRdr, &md)
	if err != nil {
		return nil, fmt.Errorf("gpg: failure to encrypt: %w. check public key(s)", err)
	}
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, md.Body)

	return buf.Bytes(), nil
}
