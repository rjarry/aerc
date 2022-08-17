package gpgbin

import (
	"bytes"
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
	outRdr := bytes.NewReader(g.stdout.Bytes())
	// Always parse stdout, even if there was an error running command.
	// We'll find the error in the parsing
	err = parse(outRdr, md)
	if err != nil {
		err = parseError(g.stderr.String())
		switch GPGErrors[err.Error()] {
		case ERROR_NO_PGP_DATA_FOUND:
			md.Body = bytes.NewReader(orig)
			return md, nil
		default:
			return nil, err
		}
	}
	return md, nil
}
