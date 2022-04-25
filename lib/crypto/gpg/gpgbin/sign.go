package gpgbin

import (
	"bytes"
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
	g.cmd.Run()

	outRdr := bytes.NewReader(g.stdout.Bytes())
	var md models.MessageDetails
	parse(outRdr, &md)
	var buf bytes.Buffer
	io.Copy(&buf, md.Body)
	return buf.Bytes(), md.Micalg, nil
}
