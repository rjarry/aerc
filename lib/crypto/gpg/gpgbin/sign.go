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

	outRdr := bytes.NewReader(g.stdout.Bytes())
	var md models.MessageDetails
	err := parse(outRdr, &md)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse messagedetails: %w", err)
	}
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, md.Body)
	return buf.Bytes(), md.Micalg, nil
}
